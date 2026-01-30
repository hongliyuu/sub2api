package service

import (
	"strings"
	"time"
)

const modelRateLimitsKey = "model_rate_limits"
const modelRateLimitScopeClaudeSonnet = "claude_sonnet"

func (a *Account) isModelRateLimited(requestedModel string) bool {
	if a == nil {
		return false
	}
	key, ok := ModelRateLimitKey(a.Platform, requestedModel)
	if !ok {
		// Fallback to legacy behavior (pre-provider bucketing).
		legacy := normalizeModelID(requestedModel)
		if strings.Contains(legacy, "sonnet") {
			key = modelRateLimitScopeClaudeSonnet
			ok = true
		}
	}
	if !ok {
		return false
	}
	resetAt := a.modelRateLimitResetAt(key)
	if resetAt == nil {
		return false
	}
	return time.Now().Before(*resetAt)
}

func (a *Account) modelRateLimitResetAt(key string) *time.Time {
	if a == nil || a.Extra == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	rawLimits, ok := a.Extra[modelRateLimitsKey].(map[string]any)
	if !ok {
		return nil
	}
	rawLimit, ok := rawLimits[key].(map[string]any)
	if !ok {
		// Backward compat: if key is provider-scoped, allow reading bucket-only.
		if idx := strings.IndexByte(key, ':'); idx > 0 {
			bucketOnly := key[idx+1:]
			rawLimit, ok = rawLimits[bucketOnly].(map[string]any)
		}
	}
	if !ok {
		return nil
	}
	resetAtRaw, ok := rawLimit["rate_limit_reset_at"].(string)
	if !ok || strings.TrimSpace(resetAtRaw) == "" {
		return nil
	}
	resetAt, err := time.Parse(time.RFC3339, resetAtRaw)
	if err != nil {
		return nil
	}
	return &resetAt
}
