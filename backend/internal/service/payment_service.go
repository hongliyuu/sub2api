package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/payment/provider"
)

// --- Order Status Constants ---

const (
	OrderStatusPending           = payment.OrderStatusPending
	OrderStatusPaid              = payment.OrderStatusPaid
	OrderStatusRecharging        = payment.OrderStatusRecharging
	OrderStatusCompleted         = payment.OrderStatusCompleted
	OrderStatusExpired           = payment.OrderStatusExpired
	OrderStatusCancelled         = payment.OrderStatusCancelled
	OrderStatusFailed            = payment.OrderStatusFailed
	OrderStatusRefundRequested   = payment.OrderStatusRefundRequested
	OrderStatusRefunding         = payment.OrderStatusRefunding
	OrderStatusPartiallyRefunded = payment.OrderStatusPartiallyRefunded
	OrderStatusRefunded          = payment.OrderStatusRefunded
	OrderStatusRefundFailed      = payment.OrderStatusRefundFailed
)

const (
	// defaultMaxPendingOrders and defaultOrderTimeoutMin are defined in
	// payment_config_service.go alongside other payment configuration defaults.
	paymentGraceMinutes = 5

	defaultPageSize    = 20
	maxPageSize        = 100
	topUsersLimit      = 10
	amountToleranceCNY = 0.01

	orderIDPrefix = "sub2_"
)

// --- Types ---

// generateOutTradeNo creates a unique external order ID for payment providers.
// Format: sub2_20250409aB3kX9mQ (prefix + date + 8-char random)
func generateOutTradeNo() string {
	date := time.Now().Format("20060102")
	rnd := generateRandomString(8)
	return orderIDPrefix + date + rnd
}

func generateRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

func (s *PaymentService) getOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*dbent.PaymentOrder, error) {
	if outTradeNo == "" {
		return s.entClient.PaymentOrder.Query().
			Where(paymentorder.IDEQ(-1)).
			First(ctx)
	}

	orders, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.OutTradeNoEQ(outTradeNo)).
		Order(dbent.Desc(paymentorder.FieldCreatedAt), dbent.Desc(paymentorder.FieldID)).
		Limit(2).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return s.entClient.PaymentOrder.Query().
			Where(paymentorder.IDEQ(-1)).
			First(ctx)
	}
	if len(orders) > 1 {
		slog.Error("[PaymentService] duplicate out_trade_no detected; refusing ambiguous lookup", "outTradeNo", outTradeNo, "newestOrderID", orders[0].ID)
		return nil, fmt.Errorf("duplicate out_trade_no detected: %s", outTradeNo)
	}
	return orders[0], nil
}

type CreateOrderRequest struct {
	UserID      int64
	Amount      float64
	PaymentType string
	ClientIP    string
	IsMobile    bool
	SrcHost     string
	SrcURL      string
	OrderType   string
	PlanID      int64
}

type CreateOrderResponse struct {
	OrderID              int64     `json:"order_id"`
	Amount               float64   `json:"amount"`
	PayAmount            float64   `json:"pay_amount"`
	FeeRate              float64   `json:"fee_rate"`
	Status               string    `json:"status"`
	PaymentType          string    `json:"payment_type"`
	PayURL               string    `json:"pay_url,omitempty"`
	QRCode               string    `json:"qr_code,omitempty"`
	ClientSecret         string    `json:"client_secret,omitempty"`
	StripePublishableKey string    `json:"stripe_publishable_key,omitempty"`
	ExpiresAt            time.Time `json:"expires_at"`
	PaymentMode          string    `json:"payment_mode,omitempty"`
}

type stripePublishableKeyProvider interface {
	GetPublishableKey() string
}

type OrderListParams struct {
	Page        int
	PageSize    int
	Status      string
	OrderType   string
	PaymentType string
	Keyword     string
}

type RefundPlan struct {
	OrderID         int64
	Order           *dbent.PaymentOrder
	RefundAmount    float64
	GatewayAmount   float64
	Reason          string
	Force           bool
	DeductBalance   bool
	DeductionType   string
	BalanceToDeduct float64
	SubDaysToDeduct int
	SubscriptionID  int64
	SubSnapshot     *UserSubscription
	SubRevoked      bool
}

type RefundResult struct {
	Success         bool    `json:"success"`
	Warning         string  `json:"warning,omitempty"`
	RequireForce    bool    `json:"require_force,omitempty"`
	BalanceDeducted float64 `json:"balance_deducted,omitempty"`
	SubDaysDeducted int     `json:"subscription_days_deducted,omitempty"`
}

type DashboardStats struct {
	TodayAmount   float64 `json:"today_amount"`
	TotalAmount   float64 `json:"total_amount"`
	TodayCount    int     `json:"today_count"`
	TotalCount    int     `json:"total_count"`
	AvgAmount     float64 `json:"avg_amount"`
	PendingOrders int     `json:"pending_orders"`

	DailySeries    []DailyStats        `json:"daily_series"`
	PaymentMethods []PaymentMethodStat `json:"payment_methods"`
	TopUsers       []TopUserStat       `json:"top_users"`
}

type DailyStats struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type PaymentMethodStat struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type TopUserStat struct {
	UserID int64   `json:"user_id"`
	Email  string  `json:"email"`
	Amount float64 `json:"amount"`
}

// --- Service ---

type PaymentService struct {
	providerMu      sync.Mutex
	providersLoaded bool
	entClient       *dbent.Client
	registry        *payment.Registry
	loadBalancer    payment.LoadBalancer
	redeemService   *RedeemService
	subscriptionSvc *SubscriptionService
	configService   *PaymentConfigService
	userRepo        UserRepository
	groupRepo       GroupRepository
}

type webhookProviderCandidate struct {
	instanceID string
	provider   payment.Provider
}

func NewPaymentService(entClient *dbent.Client, registry *payment.Registry, loadBalancer payment.LoadBalancer, redeemService *RedeemService, subscriptionSvc *SubscriptionService, configService *PaymentConfigService, userRepo UserRepository, groupRepo GroupRepository) *PaymentService {
	return &PaymentService{entClient: entClient, registry: registry, loadBalancer: loadBalancer, redeemService: redeemService, subscriptionSvc: subscriptionSvc, configService: configService, userRepo: userRepo, groupRepo: groupRepo}
}

// --- Provider Registry ---

// EnsureProviders lazily initializes the provider registry on first call.
func (s *PaymentService) EnsureProviders(ctx context.Context) {
	s.providerMu.Lock()
	defer s.providerMu.Unlock()
	if !s.providersLoaded {
		s.loadProviders(ctx)
		s.providersLoaded = true
	}
}

// RefreshProviders clears and re-registers all providers from the database.
func (s *PaymentService) RefreshProviders(ctx context.Context) {
	s.providerMu.Lock()
	defer s.providerMu.Unlock()
	s.registry.Clear()
	s.loadProviders(ctx)
	s.providersLoaded = true
}

func (s *PaymentService) loadProviders(ctx context.Context) {
	instances, err := s.entClient.PaymentProviderInstance.Query().
		Where(paymentproviderinstance.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		slog.Error("[PaymentService] failed to query provider instances", "error", err)
		return
	}
	for _, inst := range instances {
		cfg, err := s.loadBalancer.GetInstanceConfig(ctx, int64(inst.ID))
		if err != nil {
			slog.Warn("[PaymentService] failed to decrypt config for instance", "instanceID", inst.ID, "error", err)
			continue
		}
		if inst.PaymentMode != "" {
			cfg["paymentMode"] = inst.PaymentMode
		}
		instID := fmt.Sprintf("%d", inst.ID)
		p, err := provider.CreateProvider(inst.ProviderKey, instID, cfg)
		if err != nil {
			slog.Warn("[PaymentService] failed to create provider for instance", "instanceID", inst.ID, "key", inst.ProviderKey, "error", err)
			continue
		}
		s.registry.Register(p)
	}
}

// VerifyWebhookNotification verifies a provider webhook against the correct
// provider instance and ensures the verified callback belongs to the order that
// originally created the payment.
func (s *PaymentService) VerifyWebhookNotification(ctx context.Context, providerKey, hintedOutTradeNo, rawBody string, headers map[string]string) (*payment.PaymentNotification, error) {
	if hintedOutTradeNo != "" {
		order, err := s.getOrderByOutTradeNo(ctx, hintedOutTradeNo)
		if err == nil {
			return s.verifyWebhookForOrder(ctx, providerKey, order, rawBody, headers)
		}
	}

	candidates, err := s.listWebhookProviderCandidates(ctx, providerKey)
	if err != nil {
		return nil, err
	}

	var firstErr error
	for _, candidate := range candidates {
		notification, verifyErr := candidate.provider.VerifyNotification(ctx, rawBody, headers)
		if verifyErr != nil {
			if firstErr == nil {
				firstErr = verifyErr
			}
			continue
		}
		if notification == nil {
			return nil, nil
		}

		order, orderErr := s.getOrderByOutTradeNo(ctx, notification.OrderID)
		if orderErr != nil {
			return nil, fmt.Errorf("order not found for verified notification: %w", orderErr)
		}
		if err := s.validateWebhookOrderBinding(order, providerKey, candidate.instanceID); err != nil {
			return nil, err
		}
		return notification, nil
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return nil, payment.ErrProviderNotFound
}

// GetWebhookProvider returns the provider instance that should verify a webhook.
// It extracts out_trade_no from the raw body, looks up the order to find the
// original provider instance, and creates a provider with that instance's credentials.
// Falls back to the registry provider when the order cannot be found.
func (s *PaymentService) GetWebhookProvider(ctx context.Context, providerKey, outTradeNo string) (payment.Provider, error) {
	if outTradeNo != "" {
		order, err := s.getOrderByOutTradeNo(ctx, outTradeNo)
		if err == nil {
			p, pErr := s.getOrderProvider(ctx, order)
			if pErr == nil {
				return p, nil
			}
			slog.Warn("[Webhook] order provider creation failed, falling back to registry", "outTradeNo", outTradeNo, "error", pErr)
		}
	}
	s.EnsureProviders(ctx)
	return s.registry.GetProviderByKey(providerKey)
}

func (s *PaymentService) verifyWebhookForOrder(ctx context.Context, providerKey string, order *dbent.PaymentOrder, rawBody string, headers map[string]string) (*payment.PaymentNotification, error) {
	if order == nil {
		return nil, fmt.Errorf("nil order")
	}
	expectedProviderKey := s.expectedProviderKeyForOrder(order)
	if expectedProviderKey != "" && expectedProviderKey != providerKey {
		return nil, fmt.Errorf("provider key mismatch: got %s want %s", providerKey, expectedProviderKey)
	}

	candidate, err := s.webhookCandidateForOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	notification, err := candidate.provider.VerifyNotification(ctx, rawBody, headers)
	if err != nil {
		return nil, err
	}
	if notification == nil {
		return nil, nil
	}
	if notification.OrderID != "" && notification.OrderID != order.OutTradeNo {
		return nil, fmt.Errorf("verified webhook order mismatch: got %s want %s", notification.OrderID, order.OutTradeNo)
	}
	if err := s.validateWebhookOrderBinding(order, providerKey, candidate.instanceID); err != nil {
		return nil, err
	}
	return notification, nil
}

func (s *PaymentService) webhookCandidateForOrder(ctx context.Context, order *dbent.PaymentOrder) (*webhookProviderCandidate, error) {
	if order.ProviderInstanceID != nil && *order.ProviderInstanceID != "" {
		provider, err := s.getOrderProviderFromInstance(ctx, order)
		if err != nil {
			return nil, err
		}
		return &webhookProviderCandidate{
			instanceID: *order.ProviderInstanceID,
			provider:   provider,
		}, nil
	}

	provider, err := s.getOrderProvider(ctx, order)
	if err != nil {
		return nil, err
	}
	return &webhookProviderCandidate{provider: provider}, nil
}

func (s *PaymentService) listWebhookProviderCandidates(ctx context.Context, providerKey string) ([]webhookProviderCandidate, error) {
	instances, err := s.entClient.PaymentProviderInstance.Query().
		Where(
			paymentproviderinstance.EnabledEQ(true),
			paymentproviderinstance.ProviderKeyEQ(providerKey),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query provider instances: %w", err)
	}

	candidates := make([]webhookProviderCandidate, 0, len(instances))
	for _, inst := range instances {
		cfg, cfgErr := s.loadBalancer.GetInstanceConfig(ctx, int64(inst.ID))
		if cfgErr != nil {
			slog.Warn("[Webhook] failed to load provider config", "instanceID", inst.ID, "error", cfgErr)
			continue
		}
		if inst.PaymentMode != "" {
			cfg["paymentMode"] = inst.PaymentMode
		}
		instanceID := fmt.Sprintf("%d", inst.ID)
		provider, providerErr := provider.CreateProvider(providerKey, instanceID, cfg)
		if providerErr != nil {
			slog.Warn("[Webhook] failed to create provider", "instanceID", inst.ID, "provider", providerKey, "error", providerErr)
			continue
		}
		candidates = append(candidates, webhookProviderCandidate{
			instanceID: instanceID,
			provider:   provider,
		})
	}
	if len(candidates) > 0 {
		return candidates, nil
	}

	s.EnsureProviders(ctx)
	provider, err := s.registry.GetProviderByKey(providerKey)
	if err != nil {
		return nil, err
	}
	return []webhookProviderCandidate{{provider: provider}}, nil
}

func (s *PaymentService) validateWebhookOrderBinding(order *dbent.PaymentOrder, providerKey, instanceID string) error {
	if order == nil {
		return fmt.Errorf("nil order")
	}
	expectedProviderKey := s.expectedProviderKeyForOrder(order)
	if expectedProviderKey != "" && expectedProviderKey != providerKey {
		return fmt.Errorf("provider key mismatch: got %s want %s", providerKey, expectedProviderKey)
	}
	if order.ProviderInstanceID != nil && *order.ProviderInstanceID != "" {
		if instanceID == "" {
			return fmt.Errorf("missing provider instance for order %d", order.ID)
		}
		if *order.ProviderInstanceID != instanceID {
			return fmt.Errorf("provider instance mismatch: got %s want %s", instanceID, *order.ProviderInstanceID)
		}
	}
	return nil
}

func (s *PaymentService) expectedProviderKeyForOrder(order *dbent.PaymentOrder) string {
	if order == nil {
		return ""
	}
	if key := s.registry.GetProviderKey(order.PaymentType); key != "" {
		return key
	}
	return payment.GetBasePaymentType(order.PaymentType)
}

// --- Helpers ---

func psIsRefundStatus(s string) bool {
	switch s {
	case OrderStatusRefundRequested, OrderStatusRefunding, OrderStatusPartiallyRefunded, OrderStatusRefunded, OrderStatusRefundFailed:
		return true
	}
	return false
}

func psErrMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func psNilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func psSliceContains(sl []string, s string) bool {
	for _, v := range sl {
		if v == s {
			return true
		}
	}
	return false
}

// Subscription validity period unit constants.
const (
	validityUnitWeek  = "week"
	validityUnitMonth = "month"
)

func psComputeValidityDays(days int, unit string) int {
	switch unit {
	case validityUnitWeek:
		return days * 7
	case validityUnitMonth:
		return days * 30
	default:
		return days
	}
}

func (s *PaymentService) getFeeRate(_ string) float64 { return 0 }

func psStartOfDayUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func applyPagination(pageSize, page int) (size, pg int) {
	size = pageSize
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	pg = page
	if pg < 1 {
		pg = 1
	}
	return size, pg
}
