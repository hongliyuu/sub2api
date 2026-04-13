package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// GetAvailableMethodLimits collects all payment types from enabled provider
// instances and returns limits for each, plus the global widest range.
// Stripe sub-types (card, link) are aggregated under "stripe".
func (s *PaymentConfigService) GetAvailableMethodLimits(ctx context.Context) (*MethodLimitsResponse, error) {
	instances, err := s.entClient.PaymentProviderInstance.Query().
		Where(paymentproviderinstance.EnabledEQ(true)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query provider instances: %w", err)
	}
	cfg, err := s.GetPaymentConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get payment config: %w", err)
	}
	typeInstances := pcGroupByPaymentType(instances)
	resp := &MethodLimitsResponse{
		Methods: make(map[string]MethodLimits, len(typeInstances)),
	}
	for pt, insts := range typeInstances {
		ml := pcAggregateMethodLimits(pt, insts)
		resp.Methods[ml.PaymentType] = ml
	}
	if usageMap, usageErr := s.loadMethodUsageMap(ctx, instances); usageErr == nil {
		resp.Methods = applyMethodAvailability(resp.Methods, typeInstances, usageMap)
	}
	resp.Methods = filterMethodLimitsByEnabledTypes(resp.Methods, cfg)
	resp.GlobalMin, resp.GlobalMax = pcComputeGlobalRange(resp.Methods)
	return resp, nil
}

// GetMethodLimits returns per-payment-type limits from enabled provider instances.
func (s *PaymentConfigService) GetMethodLimits(ctx context.Context, types []string) ([]MethodLimits, error) {
	resp, err := s.GetAvailableMethodLimits(ctx)
	if err != nil {
		return nil, err
	}
	return selectRequestedMethodLimits(resp.Methods, types), nil
}

// pcGroupByPaymentType groups instances by user-facing payment type.
// For Stripe providers, ALL sub-types (card, link, alipay, wxpay) map to "stripe"
// because the user sees a single "Stripe" button, not individual sub-methods.
// Uses a seen set to avoid counting one instance twice.
func pcGroupByPaymentType(instances []*dbent.PaymentProviderInstance) map[string][]*dbent.PaymentProviderInstance {
	typeInstances := make(map[string][]*dbent.PaymentProviderInstance)
	seen := make(map[string]map[int64]bool)
	add := func(key string, inst *dbent.PaymentProviderInstance) {
		if seen[key] == nil {
			seen[key] = make(map[int64]bool)
		}
		if !seen[key][int64(inst.ID)] {
			seen[key][int64(inst.ID)] = true
			typeInstances[key] = append(typeInstances[key], inst)
		}
	}
	for _, inst := range instances {
		// Stripe provider: all sub-types → single "stripe" group
		if inst.ProviderKey == payment.TypeStripe {
			add(payment.TypeStripe, inst)
			continue
		}
		for _, t := range splitTypes(inst.SupportedTypes) {
			add(t, inst)
		}
	}
	return typeInstances
}

// pcInstanceTypeLimits extracts per-type limits from a provider instance.
// Returns (limits, true) if configured; (zero, false) if unlimited.
// For Stripe instances, limits are stored under "stripe" key regardless of sub-types.
func pcInstanceTypeLimits(inst *dbent.PaymentProviderInstance, pt string) (payment.ChannelLimits, bool) {
	if inst.Limits == "" {
		return payment.ChannelLimits{}, false
	}
	var limits payment.InstanceLimits
	if err := json.Unmarshal([]byte(inst.Limits), &limits); err != nil {
		return payment.ChannelLimits{}, false
	}
	cl, ok := limits[pt]
	return cl, ok
}

// unionFloat merges a single limit value into the aggregate using UNION semantics.
//   - For "min" fields (wantMin=true): keeps the lowest non-zero value
//   - For "max"/"cap" fields (wantMin=false): keeps the highest non-zero value
//   - If any value is 0 (unlimited), the result is unlimited.
//
// Returns (aggregated value, still limited).
func unionFloat(agg float64, limited bool, val float64, wantMin bool) (float64, bool) {
	if val == 0 {
		return agg, false
	}
	if !limited {
		return agg, false
	}
	if agg == 0 {
		return val, true
	}
	if wantMin && val < agg {
		return val, true
	}
	if !wantMin && val > agg {
		return val, true
	}
	return agg, true
}

// pcAggregateMethodLimits computes the UNION (least restrictive) of limits
// across all provider instances for a given payment type.
//
// Since the load balancer can route an order to any available instance,
// the user should see the widest possible range:
//   - SingleMin: lowest floor across instances; 0 if any is unlimited
//   - SingleMax: highest ceiling across instances; 0 if any is unlimited
//   - DailyLimit: highest cap across instances; 0 if any is unlimited
func pcAggregateMethodLimits(pt string, instances []*dbent.PaymentProviderInstance) MethodLimits {
	ml := MethodLimits{PaymentType: pt}
	minLimited, maxLimited, dailyLimited := true, true, true

	for _, inst := range instances {
		cl, hasLimits := pcInstanceTypeLimits(inst, pt)
		if !hasLimits {
			return MethodLimits{PaymentType: pt} // any unlimited instance → all zeros
		}
		ml.SingleMin, minLimited = unionFloat(ml.SingleMin, minLimited, cl.SingleMin, true)
		ml.SingleMax, maxLimited = unionFloat(ml.SingleMax, maxLimited, cl.SingleMax, false)
		ml.DailyLimit, dailyLimited = unionFloat(ml.DailyLimit, dailyLimited, cl.DailyLimit, false)
	}

	if !minLimited {
		ml.SingleMin = 0
	}
	if !maxLimited {
		ml.SingleMax = 0
	}
	if !dailyLimited {
		ml.DailyLimit = 0
	}
	return ml
}

// pcComputeGlobalRange computes the widest [min, max] across all methods.
// Uses the same union logic: lowest min, highest max, 0 if any is unlimited.
func pcComputeGlobalRange(methods map[string]MethodLimits) (globalMin, globalMax float64) {
	minLimited, maxLimited := true, true
	for _, ml := range methods {
		globalMin, minLimited = unionFloat(globalMin, minLimited, ml.SingleMin, true)
		globalMax, maxLimited = unionFloat(globalMax, maxLimited, ml.SingleMax, false)
	}
	if !minLimited {
		globalMin = 0
	}
	if !maxLimited {
		globalMax = 0
	}
	return globalMin, globalMax
}

func filterMethodLimitsByEnabledTypes(methods map[string]MethodLimits, cfg *PaymentConfig) map[string]MethodLimits {
	if len(methods) == 0 {
		return methods
	}
	if cfg == nil || len(cfg.EnabledTypes) == 0 {
		return methods
	}
	filtered := make(map[string]MethodLimits, len(methods))
	for key, ml := range methods {
		if isPaymentTypeEnabled(cfg, key) {
			filtered[key] = ml
		}
	}
	return filtered
}

func selectRequestedMethodLimits(methods map[string]MethodLimits, types []string) []MethodLimits {
	result := make([]MethodLimits, 0, len(types))
	for _, pt := range types {
		key := payment.GetBasePaymentType(strings.ToLower(strings.TrimSpace(pt)))
		if ml, ok := methods[key]; ok {
			result = append(result, ml)
			continue
		}
		result = append(result, MethodLimits{PaymentType: key})
	}
	return result
}

type methodUsageSnapshot struct {
	dailyUsed      float64
	dailyRemaining float64
	available      bool
}

func (s *PaymentConfigService) loadMethodUsageMap(ctx context.Context, instances []*dbent.PaymentProviderInstance) (map[int64]float64, error) {
	if len(instances) == 0 {
		return map[int64]float64{}, nil
	}
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, fmt.Sprintf("%d", inst.ID))
	}
	type row struct {
		InstanceID string  `json:"provider_instance_id"`
		Sum        float64 `json:"sum"`
	}
	var rows []row
	err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.ProviderInstanceIDIn(ids...),
			paymentorder.StatusIn(
				OrderStatusPending,
				OrderStatusPaid,
				OrderStatusCompleted,
				OrderStatusRecharging,
			),
			paymentorder.CreatedAtGTE(startOfDay(time.Now())),
		).
		GroupBy(paymentorder.FieldProviderInstanceID).
		Aggregate(dbent.Sum(paymentorder.FieldPayAmount)).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("query method usage: %w", err)
	}
	usageMap := make(map[int64]float64, len(rows))
	for _, row := range rows {
		var id int64
		_, _ = fmt.Sscan(row.InstanceID, &id)
		if id > 0 {
			usageMap[id] = row.Sum
		}
	}
	return usageMap, nil
}

func applyMethodAvailability(methods map[string]MethodLimits, grouped map[string][]*dbent.PaymentProviderInstance, usageMap map[int64]float64) map[string]MethodLimits {
	for pt, ml := range methods {
		snapshot := computeMethodUsageSnapshot(pt, grouped[pt], usageMap)
		ml.DailyUsed = snapshot.dailyUsed
		ml.DailyRemaining = snapshot.dailyRemaining
		ml.Available = snapshot.available
		methods[pt] = ml
	}
	return methods
}

func computeMethodUsageSnapshot(pt string, instances []*dbent.PaymentProviderInstance, usageMap map[int64]float64) methodUsageSnapshot {
	if len(instances) == 0 {
		return methodUsageSnapshot{}
	}
	bestRemaining := -1.0
	bestUsed := 0.0
	for _, inst := range instances {
		cl, hasLimits := pcInstanceTypeLimits(inst, pt)
		if !hasLimits || cl.DailyLimit <= 0 {
			return methodUsageSnapshot{available: true}
		}
		used := usageMap[int64(inst.ID)]
		remaining := cl.DailyLimit - used
		if remaining > bestRemaining {
			bestRemaining = remaining
			bestUsed = used
		}
	}
	if bestRemaining < 0 {
		bestRemaining = 0
	}
	return methodUsageSnapshot{
		dailyUsed:      bestUsed,
		dailyRemaining: bestRemaining,
		available:      bestRemaining > 0,
	}
}
