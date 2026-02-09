package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultExpiryReminderAdvanceDays = 7

// accountExpiryInfo holds info about an account nearing expiry.
type accountExpiryInfo struct {
	Name      string
	Email     string
	ExpiresAt string
	DaysLeft  int
}

// AccountExpiryReminderScheduler checks Claude OAuth accounts nearing expiry
// and sends a daily summary email to the configured admin address.
type AccountExpiryReminderScheduler struct {
	accountRepo    AccountRepository
	emailService   *EmailService
	settingService *SettingService
	redisClient    *redis.Client

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewAccountExpiryReminderScheduler creates a new scheduler instance.
func NewAccountExpiryReminderScheduler(
	accountRepo AccountRepository,
	emailService *EmailService,
	settingService *SettingService,
	redisClient *redis.Client,
) *AccountExpiryReminderScheduler {
	return &AccountExpiryReminderScheduler{
		accountRepo:    accountRepo,
		emailService:   emailService,
		settingService: settingService,
		redisClient:    redisClient,
		stopCh:         make(chan struct{}),
	}
}

// Start begins the background check loop.
func (s *AccountExpiryReminderScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("[AccountExpiryReminder] Started")
}

// Stop gracefully stops the scheduler.
func (s *AccountExpiryReminderScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
	log.Printf("[AccountExpiryReminder] Stopped")
}

func (s *AccountExpiryReminderScheduler) run() {
	defer s.wg.Done()

	// Delay first check by 5 minutes after startup
	select {
	case <-time.After(5 * time.Minute):
	case <-s.stopCh:
		return
	}

	s.checkAndNotify()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndNotify()
		}
	}
}

func (s *AccountExpiryReminderScheduler) checkAndNotify() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Get admin email from settings
	adminEmail, err := s.settingService.GetSettingValue(ctx, SettingKeyAccountExpiryReminderEmail)
	if err != nil || strings.TrimSpace(adminEmail) == "" {
		return // Not configured, skip silently
	}
	adminEmail = strings.TrimSpace(adminEmail)

	// 2. Read advance reminder days (default 7)
	advanceDays := defaultExpiryReminderAdvanceDays
	if raw, err := s.settingService.GetSettingValue(ctx, SettingKeyAccountExpiryReminderAdvanceDays); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && v > 0 && v <= 30 {
			advanceDays = v
		}
	}

	// 3. Redis SetNX dedup: one email per day
	today := time.Now().Format("2006-01-02")
	redisKey := fmt.Sprintf("account_expiry_reminder_sent:%s", today)
	isFirst, err := s.redisClient.SetNX(ctx, redisKey, "1", 25*time.Hour).Result()
	if err != nil {
		log.Printf("[AccountExpiryReminder] Redis SetNX error: %v", err)
		return
	}
	if !isFirst {
		return // Already sent today
	}

	// 4. List all Anthropic accounts
	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformAnthropic)
	if err != nil {
		log.Printf("[AccountExpiryReminder] Error listing accounts: %v", err)
		s.redisClient.Del(ctx, redisKey)
		return
	}

	// 5. Filter expiring accounts
	now := time.Now()
	var expiring []accountExpiryInfo

	for _, acct := range accounts {
		if !acct.IsAnthropicOAuthOrSetupToken() || !acct.IsActive() {
			continue
		}

		var daysLeft int
		var expiresAtStr string
		var shouldNotify bool

		if acct.ExpiresAt != nil {
			// Has explicit ExpiresAt
			remaining := time.Until(*acct.ExpiresAt)
			daysLeft = int(math.Ceil(remaining.Hours() / 24))
			expiresAtStr = acct.ExpiresAt.Format("2006-01-02 15:04")
			shouldNotify = daysLeft <= advanceDays && daysLeft > 0
		} else {
			// Fallback: estimate from CreatedAt (30-day cycle)
			daysSinceCreated := now.Sub(acct.CreatedAt).Hours() / 24
			if daysSinceCreated >= float64(30-advanceDays) {
				estimatedExpiry := acct.CreatedAt.AddDate(0, 0, 30)
				remaining := time.Until(estimatedExpiry)
				daysLeft = int(math.Ceil(remaining.Hours() / 24))
				expiresAtStr = estimatedExpiry.Format("2006-01-02 15:04") + " (estimated)"
				shouldNotify = daysLeft > 0
			}
		}

		if !shouldNotify {
			continue
		}

		email := acct.GetCredential("email_address")
		expiring = append(expiring, accountExpiryInfo{
			Name:      acct.Name,
			Email:     email,
			ExpiresAt: expiresAtStr,
			DaysLeft:  daysLeft,
		})
	}

	if len(expiring) == 0 {
		return // No accounts expiring soon
	}

	// 6. Build and send email
	siteName := s.settingService.GetSiteName(ctx)
	subject := fmt.Sprintf("[%s] Claude OAuth 账号即将到期提醒", siteName)
	body := s.buildEmailBody(siteName, expiring)

	if err := s.emailService.SendEmail(ctx, adminEmail, subject, body); err != nil {
		log.Printf("[AccountExpiryReminder] Failed to send email: %v", err)
		// Allow retry by deleting the dedup key
		s.redisClient.Del(ctx, redisKey)
		return
	}

	log.Printf("[AccountExpiryReminder] Sent expiry reminder for %d account(s) to %s", len(expiring), adminEmail)
}

func (s *AccountExpiryReminderScheduler) buildEmailBody(siteName string, accounts []accountExpiryInfo) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>`)
	b.WriteString(fmt.Sprintf(`<h2>%s - Claude OAuth 账号到期提醒</h2>`, siteName))
	b.WriteString(fmt.Sprintf(`<p>以下 %d 个 Claude OAuth 账号即将到期，请及时处理：</p>`, len(accounts)))

	b.WriteString(`<table border="1" cellpadding="8" cellspacing="0" style="border-collapse:collapse;font-family:sans-serif;">`)
	b.WriteString(`<tr style="background:#f2f2f2;"><th>账号名称</th><th>邮箱</th><th>到期时间</th><th>剩余天数</th></tr>`)

	for _, acct := range accounts {
		color := "#333"
		if acct.DaysLeft <= 3 {
			color = "#e53e3e"
		} else if acct.DaysLeft <= 5 {
			color = "#dd6b20"
		}

		email := acct.Email
		if email == "" {
			email = "-"
		}

		b.WriteString(fmt.Sprintf(
			`<tr><td>%s</td><td>%s</td><td>%s</td><td style="color:%s;font-weight:bold;">%d 天</td></tr>`,
			acct.Name, email, acct.ExpiresAt, color, acct.DaysLeft,
		))
	}

	b.WriteString(`</table>`)
	b.WriteString(`<p style="color:#666;margin-top:16px;">请及时续费或替换即将到期的账号，避免服务中断。</p>`)
	b.WriteString(`</body></html>`)

	return b.String()
}
