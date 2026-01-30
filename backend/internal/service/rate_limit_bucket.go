package service

import (
	"strings"
)

// normalizeModelID produces a stable identifier for bucketing.
// This must match existing normalization patterns used elsewhere.
func normalizeModelID(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return ""
	}
	return strings.TrimPrefix(m, "models/")
}

// quotaBucketID returns the bucket id used for cooldown decisions.
// In the common case, it is the exact model id, but some model families share quota.
func quotaBucketID(providerID, exactModelID string) (string, bool) {
	provider := strings.ToLower(strings.TrimSpace(providerID))
	model := normalizeModelID(exactModelID)
	if provider == "" || model == "" {
		return "", false
	}

	switch provider {
	case PlatformGemini:
		// Gemini 3 families share quota within the family.
		// Keep Flash and Pro isolated from each other.
		if strings.HasPrefix(model, "gemini-3-flash") {
			return "gemini-3-flash", true
		}
		if strings.HasPrefix(model, "gemini-3-pro") {
			return "gemini-3-pro", true
		}
		return model, true
	case PlatformAnthropic:
		// Preserve existing scoped behavior for Sonnet as a shared bucket.
		if strings.Contains(model, "sonnet") {
			return modelRateLimitScopeClaudeSonnet, true
		}
		return model, true
	default:
		return model, true
	}
}

// IsSafeModelRateLimitKey ensures the provided key can be safely used as a
// JSONB path element (e.g. inside `jsonb_set('{model_rate_limits,<key>}', ...)`).
//
// It enforces a strict whitelist to keep JSON paths stable and reject
// characters (quotes, braces, carriage returns, etc.) that might break SQL.
func IsSafeModelRateLimitKey(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	// Prevent unbounded keys / path literal breakage.
	// This is deliberately strict; if it fails, callers should fall back to global cooldown.
	if len(s) > 160 {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', ':':
			continue
		default:
			return false
		}
	}
	return true
}

// ModelRateLimitKey returns a stable key used under accounts.extra.model_rate_limits.
// Format: "<provider_id>:<quota_bucket_id>".
func ModelRateLimitKey(providerID, exactModelID string) (string, bool) {
	bucket, ok := quotaBucketID(providerID, exactModelID)
	if !ok {
		return "", false
	}
	key := strings.ToLower(strings.TrimSpace(providerID)) + ":" + bucket
	if !IsSafeModelRateLimitKey(key) {
		return "", false
	}
	return key, true
}
