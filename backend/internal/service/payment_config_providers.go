package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// --- Provider Instance CRUD ---

func (s *PaymentConfigService) ListProviderInstances(ctx context.Context) ([]*dbent.PaymentProviderInstance, error) {
	return s.entClient.PaymentProviderInstance.Query().Order(paymentproviderinstance.BySortOrder()).All(ctx)
}

// ProviderInstanceResponse is the API response for a provider instance.
type ProviderInstanceResponse struct {
	ID              int64             `json:"id"`
	ProviderKey     string            `json:"provider_key"`
	Name            string            `json:"name"`
	Config          map[string]string `json:"config"`
	SupportedTypes  []string          `json:"supported_types"`
	Limits          string            `json:"limits"`
	Enabled         bool              `json:"enabled"`
	RefundEnabled   bool              `json:"refund_enabled"`
	AllowUserRefund bool              `json:"allow_user_refund"`
	SortOrder       int               `json:"sort_order"`
	PaymentMode     string            `json:"payment_mode"`
}

// ListProviderInstancesWithConfig returns provider instances with decrypted config.
func (s *PaymentConfigService) ListProviderInstancesWithConfig(ctx context.Context) ([]ProviderInstanceResponse, error) {
	instances, err := s.entClient.PaymentProviderInstance.Query().
		Order(paymentproviderinstance.BySortOrder()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ProviderInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		resp := ProviderInstanceResponse{
			ID: int64(inst.ID), ProviderKey: inst.ProviderKey, Name: inst.Name,
			SupportedTypes: splitTypes(inst.SupportedTypes), Limits: inst.Limits,
			Enabled: inst.Enabled, RefundEnabled: inst.RefundEnabled,
			AllowUserRefund: inst.AllowUserRefund,
			SortOrder:       inst.SortOrder, PaymentMode: inst.PaymentMode,
		}
		resp.Config, err = s.decryptAndMaskConfig(inst.ProviderKey, inst.Config)
		if err != nil {
			return nil, fmt.Errorf("decrypt config for instance %d: %w", inst.ID, err)
		}
		result = append(result, resp)
	}
	return result, nil
}

// decryptAndMaskConfig returns the stored config with sensitive fields omitted.
// Admin UIs display masked placeholders for these; the raw values never leave
// the server. Callers that need the full config (e.g. payment runtime) must
// use decryptConfig directly.
func (s *PaymentConfigService) decryptAndMaskConfig(providerKey, encrypted string) (map[string]string, error) {
	cfg, err := s.decryptConfig(encrypted)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	masked := make(map[string]string, len(cfg))
	for k, v := range cfg {
		if isSensitiveProviderConfigField(providerKey, k) {
			continue
		}
		masked[k] = v
	}
	return masked, nil
}

// pendingOrderStatuses are order statuses considered "in progress".
var pendingOrderStatuses = []string{
	payment.OrderStatusPending,
	payment.OrderStatusPaid,
	payment.OrderStatusRecharging,
}

// providerSensitiveConfigFields is the authoritative list of config keys that
// are treated as secrets per provider. Must stay in sync with the frontend
// definition at frontend/src/components/payment/providerConfig.ts
// (PROVIDER_CONFIG_FIELDS, fields with sensitive: true).
//
// Key matching is case-insensitive. Non-listed keys (e.g. appId, notifyUrl,
// stripe publishableKey) are returned in plaintext by the admin GET API.
var providerSensitiveConfigFields = map[string]map[string]struct{}{
	payment.TypeEasyPay: {"pkey": {}},
	payment.TypeAlipay:  {"privatekey": {}, "publickey": {}, "alipaypublickey": {}},
	payment.TypeWxpay:   {"privatekey": {}, "apiv3key": {}, "publickey": {}},
	payment.TypeStripe:  {"secretkey": {}, "webhooksecret": {}},
}

var providerHistoryLockStatuses = []string{
	payment.OrderStatusPending,
	payment.OrderStatusPaid,
	payment.OrderStatusRecharging,
	payment.OrderStatusCompleted,
	payment.OrderStatusRefundRequested,
	payment.OrderStatusRefunding,
	payment.OrderStatusPartiallyRefunded,
	payment.OrderStatusRefundFailed,
}

func isSensitiveProviderConfigField(providerKey, fieldName string) bool {
	fields, ok := providerSensitiveConfigFields[providerKey]
	if !ok {
		return false
	}
	_, found := fields[strings.ToLower(fieldName)]
	return found
}

func mergeProviderConfig(providerKey string, existing, newConfig map[string]string) map[string]string {
	if len(existing) == 0 {
		if len(newConfig) == 0 {
			return nil
		}
		merged := make(map[string]string, len(newConfig))
		for k, v := range newConfig {
			if strings.TrimSpace(v) == "" && isSensitiveProviderConfigField(providerKey, k) {
				continue
			}
			merged[k] = v
		}
		return merged
	}
	merged := make(map[string]string, len(existing)+len(newConfig))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range newConfig {
		if strings.TrimSpace(v) == "" && isSensitiveProviderConfigField(providerKey, k) {
			continue
		}
		merged[k] = v
	}
	return merged
}

func (s *PaymentConfigService) countPendingOrders(ctx context.Context, providerInstanceID int64) (int, error) {
	return s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.ProviderInstanceIDEQ(strconv.FormatInt(providerInstanceID, 10)),
			paymentorder.StatusIn(pendingOrderStatuses...),
		).Count(ctx)
}

func (s *PaymentConfigService) countOrdersByInstance(ctx context.Context, providerInstanceID int64) (int, error) {
	return s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.ProviderInstanceIDEQ(strconv.FormatInt(providerInstanceID, 10)),
			paymentorder.StatusIn(providerHistoryLockStatuses...),
		).
		Count(ctx)
}

func (s *PaymentConfigService) countPendingOrdersByPlan(ctx context.Context, planID int64) (int, error) {
	return s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.PlanIDEQ(planID),
			paymentorder.StatusIn(pendingOrderStatuses...),
		).Count(ctx)
}

var validProviderKeys = map[string]bool{
	payment.TypeEasyPay: true, payment.TypeAlipay: true, payment.TypeWxpay: true, payment.TypeStripe: true,
}

func (s *PaymentConfigService) CreateProviderInstance(ctx context.Context, req CreateProviderInstanceRequest) (*dbent.PaymentProviderInstance, error) {
	typesStr := joinTypes(req.SupportedTypes)
	if err := validateProviderRequest(req.ProviderKey, req.Name, typesStr); err != nil {
		return nil, err
	}
	if err := s.validateProviderCapabilityRoute(ctx, 0, req.ProviderKey, typesStr, req.Enabled); err != nil {
		return nil, err
	}
	enc, err := s.encryptConfig(req.Config)
	if err != nil {
		return nil, err
	}
	allowUserRefund := req.AllowUserRefund && req.RefundEnabled
	return s.entClient.PaymentProviderInstance.Create().
		SetProviderKey(req.ProviderKey).SetName(req.Name).SetConfig(enc).
		SetSupportedTypes(typesStr).SetEnabled(req.Enabled).SetPaymentMode(req.PaymentMode).
		SetSortOrder(req.SortOrder).SetLimits(req.Limits).SetRefundEnabled(req.RefundEnabled).
		SetAllowUserRefund(allowUserRefund).
		Save(ctx)
}

func validateProviderRequest(providerKey, name, supportedTypes string) error {
	if strings.TrimSpace(name) == "" {
		return infraerrors.BadRequest("VALIDATION_ERROR", "provider name is required")
	}
	if !validProviderKeys[providerKey] {
		return infraerrors.BadRequest("VALIDATION_ERROR", fmt.Sprintf("invalid provider key: %s", providerKey))
	}
	return validateSupportedTypesForProvider(providerKey, supportedTypes)
}

func hasRequiredConfigValue(providerKey, key, value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	return true
}

func validateProviderConfig(providerKey string, config map[string]string) error {
	switch providerKey {
	case payment.TypeAlipay:
		if !hasRequiredConfigValue(providerKey, "appId", config["appId"]) {
			return infraerrors.BadRequest("VALIDATION_ERROR", "alipay config missing required key: appId")
		}
		if !hasRequiredConfigValue(providerKey, "privateKey", config["privateKey"]) {
			return infraerrors.BadRequest("VALIDATION_ERROR", "alipay config missing required key: privateKey")
		}
		if !hasRequiredConfigValue(providerKey, "publicKey", config["publicKey"]) &&
			!hasRequiredConfigValue(providerKey, "alipayPublicKey", config["alipayPublicKey"]) {
			return infraerrors.BadRequest("VALIDATION_ERROR", "alipay config missing required key: publicKey (or alipayPublicKey)")
		}
	}
	return nil
}

// UpdateProviderInstance updates a provider instance by ID (patch semantics).
// NOTE: This function exceeds 30 lines due to per-field nil-check patch update
// boilerplate and pending-order safety checks.
func (s *PaymentConfigService) UpdateProviderInstance(ctx context.Context, id int64, req UpdateProviderInstanceRequest) (*dbent.PaymentProviderInstance, error) {
	current, err := s.entClient.PaymentProviderInstance.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load provider instance: %w", err)
	}
	finalSupportedTypes := current.SupportedTypes
	if req.SupportedTypes != nil {
		finalSupportedTypes = joinTypes(req.SupportedTypes)
		if err := validateSupportedTypesForProvider(current.ProviderKey, finalSupportedTypes); err != nil {
			return nil, err
		}
	}
	finalEnabled := current.Enabled
	if req.Enabled != nil {
		finalEnabled = *req.Enabled
	}
	if err := s.validateProviderCapabilityRoute(ctx, id, current.ProviderKey, finalSupportedTypes, finalEnabled); err != nil {
		return nil, err
	}
	if req.Config != nil {
		hasSensitive := false
		for k, v := range req.Config {
			if strings.TrimSpace(v) != "" && isSensitiveProviderConfigField(current.ProviderKey, k) {
				hasSensitive = true
				break
			}
		}
		if hasSensitive {
			count, err := s.countOrdersByInstance(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("check provider order history: %w", err)
			}
			if count > 0 {
				return nil, infraerrors.Conflict("ORDER_HISTORY_EXISTS", "instance has settled or in-progress orders and cannot change sensitive merchant credentials").
					WithMetadata(map[string]string{"count": strconv.Itoa(count)})
			}
		}
	}
	if req.Enabled != nil && !*req.Enabled {
		count, err := s.countPendingOrders(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("check pending orders: %w", err)
		}
		if count > 0 {
			return nil, infraerrors.Conflict("PENDING_ORDERS", "instance has pending orders").
				WithMetadata(map[string]string{"count": strconv.Itoa(count)})
		}
	}
	u := s.entClient.PaymentProviderInstance.UpdateOneID(id)
	if req.Name != nil {
		u.SetName(*req.Name)
	}
	if req.Config != nil {
		merged, err := s.mergeConfig(ctx, id, req.Config)
		if err != nil {
			return nil, err
		}
		if err := validateProviderConfig(current.ProviderKey, merged); err != nil {
			return nil, err
		}
		enc, err := s.encryptConfig(merged)
		if err != nil {
			return nil, err
		}
		u.SetConfig(enc)
	}
	if req.SupportedTypes != nil {
		// Check pending orders before removing payment types
		count, err := s.countPendingOrders(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("check pending orders: %w", err)
		}
		if count > 0 {
			// Load current instance to compare types
			oldTypes := strings.Split(current.SupportedTypes, ",")
			newTypes := req.SupportedTypes
			for _, ot := range oldTypes {
				ot = strings.TrimSpace(ot)
				if ot == "" {
					continue
				}
				found := false
				for _, nt := range newTypes {
					if strings.TrimSpace(nt) == ot {
						found = true
						break
					}
				}
				if !found {
					return nil, infraerrors.Conflict("PENDING_ORDERS", "cannot remove payment types while instance has pending orders").
						WithMetadata(map[string]string{"count": strconv.Itoa(count)})
				}
			}
		}
		u.SetSupportedTypes(joinTypes(req.SupportedTypes))
	}
	if req.Enabled != nil {
		u.SetEnabled(*req.Enabled)
	}
	if req.SortOrder != nil {
		u.SetSortOrder(*req.SortOrder)
	}
	if req.Limits != nil {
		u.SetLimits(*req.Limits)
	}
	if req.RefundEnabled != nil {
		u.SetRefundEnabled(*req.RefundEnabled)
		// Cascade: turning off refund_enabled also disables allow_user_refund
		if !*req.RefundEnabled {
			u.SetAllowUserRefund(false)
		}
	}
	if req.AllowUserRefund != nil {
		// Only allow enabling when refund_enabled is (or will be) true
		if *req.AllowUserRefund {
			refundEnabled := false
			if req.RefundEnabled != nil {
				refundEnabled = *req.RefundEnabled
			} else {
				refundEnabled = current.RefundEnabled
			}
			if refundEnabled {
				u.SetAllowUserRefund(true)
			}
		} else {
			u.SetAllowUserRefund(false)
		}
	}
	if req.PaymentMode != nil {
		u.SetPaymentMode(*req.PaymentMode)
	}
	return u.Save(ctx)
}

func validateSupportedTypesForProvider(providerKey, supportedTypes string) error {
	allowed := allowedStoredTypesForProvider(providerKey)
	if len(allowed) == 0 {
		return nil
	}
	for _, typ := range splitTypes(supportedTypes) {
		if _, ok := allowed[typ]; !ok {
			return infraerrors.BadRequest("VALIDATION_ERROR",
				fmt.Sprintf("provider %s only supports %v", providerKey, orderedAllowedTypes(allowed)))
		}
	}
	return nil
}

func allowedStoredTypesForProvider(providerKey string) map[string]struct{} {
	switch providerKey {
	case payment.TypeAlipay:
		return map[string]struct{}{payment.TypeAlipay: {}}
	case payment.TypeWxpay:
		return map[string]struct{}{payment.TypeWxpay: {}}
	case payment.TypeEasyPay:
		return map[string]struct{}{payment.TypeAlipay: {}, payment.TypeWxpay: {}}
	case payment.TypeStripe:
		return map[string]struct{}{
			payment.TypeStripe: {},
			payment.TypeCard:   {},
			payment.TypeLink:   {},
			payment.TypeAlipay: {},
			payment.TypeWxpay:  {},
		}
	default:
		return nil
	}
}

func orderedAllowedTypes(allowed map[string]struct{}) []string {
	result := make([]string, 0, len(allowed))
	for typ := range allowed {
		result = append(result, typ)
	}
	slices.Sort(result)
	return result
}

func (s *PaymentConfigService) validateProviderCapabilityRoute(ctx context.Context, selfID int64, providerKey, supportedTypes string, enabled bool) error {
	instances, err := s.entClient.PaymentProviderInstance.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("query provider instances: %w", err)
	}
	return validateCapabilityRouteConflicts(instances, selfID, providerKey, supportedTypes, enabled)
}

func validateCapabilityRouteConflicts(instances []*dbent.PaymentProviderInstance, selfID int64, providerKey, supportedTypes string, enabled bool) error {
	typeProviders := make(map[string]map[string]struct{})
	add := func(instID int64, instProviderKey, instSupportedTypes string, instEnabled bool) {
		if !instEnabled || (selfID > 0 && instID == selfID) {
			return
		}
		for _, capability := range payment.VisiblePaymentTypesForProvider(instProviderKey, instSupportedTypes) {
			if capability == payment.TypeStripe {
				continue
			}
			key := string(capability)
			if typeProviders[key] == nil {
				typeProviders[key] = make(map[string]struct{})
			}
			typeProviders[key][instProviderKey] = struct{}{}
		}
	}

	for _, inst := range instances {
		add(int64(inst.ID), inst.ProviderKey, inst.SupportedTypes, inst.Enabled)
	}
	add(selfID, providerKey, supportedTypes, enabled)

	for _, capability := range []string{payment.TypeAlipay, payment.TypeWxpay} {
		providerTypes := orderedAllowedTypes(typeProviders[capability])
		if len(providerTypes) > 1 {
			return infraerrors.Conflict("PAYMENT_CAPABILITY_CONFLICT",
				fmt.Sprintf("%s capability conflict: enabled provider types %v", capability, providerTypes))
		}
	}
	return nil
}

// GetUserRefundEligibleInstanceIDs returns provider instance IDs that allow user refund.
func (s *PaymentConfigService) GetUserRefundEligibleInstanceIDs(ctx context.Context) ([]string, error) {
	instances, err := s.entClient.PaymentProviderInstance.Query().
		Where(
			paymentproviderinstance.RefundEnabledEQ(true),
			paymentproviderinstance.AllowUserRefundEQ(true),
		).Select(paymentproviderinstance.FieldID).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, strconv.FormatInt(int64(inst.ID), 10))
	}
	return ids, nil
}

func (s *PaymentConfigService) mergeConfig(ctx context.Context, id int64, newConfig map[string]string) (map[string]string, error) {
	inst, err := s.entClient.PaymentProviderInstance.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load existing provider: %w", err)
	}
	existing, err := s.decryptConfig(inst.Config)
	if err != nil {
		return nil, fmt.Errorf("decrypt existing config for instance %d: %w", id, err)
	}
	if existing == nil {
		existing = nil
	}
	return mergeProviderConfig(inst.ProviderKey, existing, newConfig), nil
}

// decryptConfig parses a stored provider config.
// New records are plaintext JSON; legacy records are AES-256-GCM ciphertext
// ("iv:authTag:ciphertext"). Values that cannot be parsed as either — including
// legacy ciphertext with no/invalid TOTP_ENCRYPTION_KEY — are treated as empty,
// letting the admin re-enter the config via the UI to complete the migration.
//
// TODO(deprecated-legacy-ciphertext): The AES fallback branch is a transitional
// shim for pre-plaintext records. Remove it (and the encryptionKey field) after
// a few releases once all live deployments have re-saved their provider configs.
func (s *PaymentConfigService) decryptConfig(stored string) (map[string]string, error) {
	if stored == "" {
		return nil, nil
	}
	var cfg map[string]string
	if err := json.Unmarshal([]byte(stored), &cfg); err == nil {
		return cfg, nil
	}
	// Deprecated: legacy AES-256-GCM ciphertext fallback — scheduled for removal.
	if len(s.encryptionKey) == payment.AES256KeySize {
		//nolint:staticcheck // SA1019: intentional legacy fallback, scheduled for removal
		if plaintext, err := payment.Decrypt(stored, s.encryptionKey); err == nil {
			if err := json.Unmarshal([]byte(plaintext), &cfg); err == nil {
				return cfg, nil
			}
		}
	}
	slog.Warn("payment provider config unreadable, treating as empty for re-entry",
		"stored_len", len(stored))
	return nil, nil
}

func (s *PaymentConfigService) DeleteProviderInstance(ctx context.Context, id int64) error {
	count, err := s.countPendingOrders(ctx, id)
	if err != nil {
		return fmt.Errorf("check pending orders: %w", err)
	}
	if count > 0 {
		return infraerrors.Conflict("PENDING_ORDERS",
			fmt.Sprintf("this instance has %d in-progress orders and cannot be deleted — wait for orders to complete or disable the instance first", count))
	}
	totalOrders, err := s.countOrdersByInstance(ctx, id)
	if err != nil {
		return fmt.Errorf("check historical orders: %w", err)
	}
	if totalOrders > 0 {
		return infraerrors.Conflict("ORDER_HISTORY_EXISTS",
			fmt.Sprintf("this instance has %d historical orders and cannot be deleted because refunds and reconciliation must keep the original merchant binding", totalOrders))
	}
	return s.entClient.PaymentProviderInstance.DeleteOneID(id).Exec(ctx)
}

// encryptConfig serialises a provider config for storage.
// New records are written as plaintext JSON; the historical AES-GCM wrapping
// has been dropped but decryptConfig still accepts old ciphertext during migration.
func (s *PaymentConfigService) encryptConfig(cfg map[string]string) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	return string(data), nil
}
