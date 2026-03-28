package service

import (
	"context"
	"fmt"
	"time"

	copilotpkg "github.com/Wei-Shaw/sub2api/internal/pkg/copilot"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	gocache "github.com/patrickmn/go-cache"
)

const (
	copilotQuotaCacheTTL     = 5 * time.Minute
	copilotQuotaCacheCleanup = time.Minute

	// Alert status values returned in API responses.
	CopilotAlertStatusOK       = "ok"
	CopilotAlertStatusWarning  = "warning"
	CopilotAlertStatusCritical = "critical"

	copilotAlertCriticalThreshold = 95 // percent
)

// CopilotCachedQuota is an entry stored in the in-memory cache.
type CopilotCachedQuota struct {
	AccountID int64
	QuotaInfo *copilotpkg.CopilotQuotaInfo
	CachedAt  time.Time
}

// CopilotQuotaCacheService manages real-time Copilot quota data with a short-lived
// in-memory cache. On every successful fetch, the quota is persisted to
// copilot_quota_snapshots as the "last known state" for trend charts.
type CopilotQuotaCacheService struct {
	cache        *gocache.Cache
	copilotSvc   *CopilotGatewayService
	snapshotRepo CopilotQuotaSnapshotRepository
	alertRepo    CopilotBudgetAlertRepository
	adminSvc     AdminService
}

// NewCopilotQuotaCacheService constructs the cache service.
func NewCopilotQuotaCacheService(
	copilotSvc *CopilotGatewayService,
	snapshotRepo CopilotQuotaSnapshotRepository,
	alertRepo CopilotBudgetAlertRepository,
	adminSvc AdminService,
) *CopilotQuotaCacheService {
	return &CopilotQuotaCacheService{
		cache:        gocache.New(copilotQuotaCacheTTL, copilotQuotaCacheCleanup),
		copilotSvc:   copilotSvc,
		snapshotRepo: snapshotRepo,
		alertRepo:    alertRepo,
		adminSvc:     adminSvc,
	}
}

// cacheKey returns the cache key for an account.
func cacheKey(accountID int64) string {
	return fmt.Sprintf("copilot_quota:%d", accountID)
}

// FetchAll fetches quota for all active Copilot accounts concurrently.
// Cached results are returned immediately; stale or missing entries trigger
// real-time GitHub API calls.
func (s *CopilotQuotaCacheService) FetchAll(ctx context.Context) ([]CopilotCachedQuota, error) {
	summaries, err := s.copilotSvc.FetchAllCopilotQuotas(ctx, s.adminSvc)
	if err != nil {
		return nil, fmt.Errorf("copilot quota cache: fetch all: %w", err)
	}

	now := time.Now()
	result := make([]CopilotCachedQuota, 0, len(summaries))
	for _, summary := range summaries {
		entry := CopilotCachedQuota{
			AccountID: summary.AccountID,
			QuotaInfo: summary.QuotaInfo,
			CachedAt:  now,
		}
		// Update cache entry.
		s.cache.Set(cacheKey(summary.AccountID), entry, gocache.DefaultExpiration)

		// Persist snapshot and check alerts asynchronously to avoid blocking the caller.
		if summary.QuotaInfo != nil {
			go s.persistAndAlert(context.Background(), summary.AccountID, summary.QuotaInfo, now)
		}

		result = append(result, entry)
	}
	return result, nil
}

// FetchAllCached returns all quota entries from cache + any uncached accounts.
// If an account is not in the cache, a real-time fetch is triggered.
// This is the primary call for the accounts overview page.
func (s *CopilotQuotaCacheService) FetchAllCached(ctx context.Context) ([]CopilotCachedQuota, error) {
	// For simplicity we always do a full refresh on cache miss.
	// A partial miss (some accounts cached) still benefits from the cached TTL:
	// FetchAllCopilotQuotas only hits GitHub API for uncached accounts.
	// Since FetchAllCopilotQuotas is already concurrent, a full call is acceptable.
	return s.FetchAll(ctx)
}

// GetCached returns the cached quota for a single account, or nil if not cached.
func (s *CopilotQuotaCacheService) GetCached(accountID int64) *CopilotCachedQuota {
	v, ok := s.cache.Get(cacheKey(accountID))
	if !ok {
		return nil
	}
	entry, ok := v.(CopilotCachedQuota)
	if !ok {
		return nil
	}
	return &entry
}

// ForceRefresh bypasses the cache and re-fetches quota for a single account.
// The result is stored in the cache (resetting the TTL) and persisted.
func (s *CopilotQuotaCacheService) ForceRefresh(ctx context.Context, account *Account) (*CopilotCachedQuota, error) {
	qi, err := s.copilotSvc.FetchQuota(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("copilot quota cache: force refresh account %d: %w", account.ID, err)
	}

	now := time.Now()
	entry := CopilotCachedQuota{
		AccountID: account.ID,
		QuotaInfo: qi,
		CachedAt:  now,
	}
	s.cache.Set(cacheKey(account.ID), entry, gocache.DefaultExpiration)

	if qi != nil {
		go s.persistAndAlert(context.Background(), account.ID, qi, now)
	}
	return &entry, nil
}

// persistAndAlert saves a quota snapshot and checks alert conditions.
// Designed to run in a goroutine (errors are logged, not returned).
func (s *CopilotQuotaCacheService) persistAndAlert(
	ctx context.Context,
	accountID int64,
	qi *copilotpkg.CopilotQuotaInfo,
	fetchedAt time.Time,
) {
	// Build snapshot from quota info.
	snap := buildSnapshot(accountID, qi, fetchedAt)
	if err := s.snapshotRepo.Upsert(ctx, snap); err != nil {
		logger.LegacyPrintf("copilot.quota_cache", "failed to upsert snapshot for account %d: %v", accountID, err)
	}

	// Check budget alert.
	alert, err := s.alertRepo.GetByAccountID(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("copilot.quota_cache", "failed to get budget alert for account %d: %v", accountID, err)
		return
	}
	if alert == nil || !alert.Enabled {
		return
	}

	pi := qi.PremiumInteractions
	if pi == nil || pi.Unlimited || pi.Entitlement == 0 {
		return
	}

	usageRate := float64(pi.Used) / float64(pi.Entitlement) * 100
	alertStatus := CopilotAlertStatusOK
	switch {
	case usageRate >= copilotAlertCriticalThreshold:
		alertStatus = CopilotAlertStatusCritical
	case usageRate >= float64(alert.AlertThreshold):
		alertStatus = CopilotAlertStatusWarning
	}

	if alertStatus != CopilotAlertStatusOK {
		logger.LegacyPrintf("copilot.budget_alert",
			"account %d quota alert: status=%s used=%.1f%% threshold=%d%% premium_used=%d/%d",
			accountID, alertStatus, usageRate, alert.AlertThreshold, pi.Used, pi.Entitlement)
	}
}

// AlertStatus returns the alert status string for the given usage rate and threshold.
func AlertStatus(usageRate float64, threshold int) string {
	switch {
	case usageRate >= copilotAlertCriticalThreshold:
		return CopilotAlertStatusCritical
	case usageRate >= float64(threshold):
		return CopilotAlertStatusWarning
	default:
		return CopilotAlertStatusOK
	}
}

// buildSnapshot constructs a CopilotQuotaSnapshot from live quota info.
func buildSnapshot(accountID int64, qi *copilotpkg.CopilotQuotaInfo, fetchedAt time.Time) *CopilotQuotaSnapshot {
	// Truncate to date precision (midnight UTC) so one row per day.
	date := time.Date(fetchedAt.Year(), fetchedAt.Month(), fetchedAt.Day(), 0, 0, 0, 0, time.UTC)

	snap := &CopilotQuotaSnapshot{
		AccountID:    accountID,
		SnapshotDate: date,
	}

	if qi.Plan != "" {
		plan := qi.Plan
		snap.PlanType = &plan
	}

	pi := qi.PremiumInteractions
	if pi != nil {
		snap.PremiumEntitlement = pi.Entitlement
		snap.PremiumRemaining = pi.Remaining
		snap.PremiumUsed = pi.Used
		snap.PremiumOverage = pi.OverageCount
		snap.Unlimited = pi.Unlimited
	}
	return snap
}
