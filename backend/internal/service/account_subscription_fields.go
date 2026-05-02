package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (a *Account) ApplySubscriptionFieldsFromCredentials() {
	if a == nil {
		return
	}
	a.PlanType = stringPtrFromCredential(a.Credentials, "plan_type")
	a.SubscriptionStatus = stringPtrFromCredential(a.Credentials, "subscription_status")
	a.SubscriptionExpiresAt = timePtrFromCredential(a.Credentials, "subscription_expires_at")
}

func stringPtrFromCredential(values map[string]any, key string) *string {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return nil
	}
	value := strings.TrimSpace(credentialValueToString(raw))
	if value == "" {
		return nil
	}
	return &value
}

func timePtrFromCredential(values map[string]any, key string) *time.Time {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case time.Time:
		return &v
	case string:
		return parseCredentialTimePtr(v)
	case json.Number:
		return unixCredentialTimePtr(v.String())
	case int64:
		return unixCredentialTimeFromIntPtr(v)
	case int:
		return unixCredentialTimeFromIntPtr(int64(v))
	case float64:
		return unixCredentialTimeFromIntPtr(int64(v))
	default:
		return parseCredentialTimePtr(credentialValueToString(v))
	}
}

func parseCredentialTimePtr(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &t
	}
	return unixCredentialTimePtr(value)
}

func unixCredentialTimePtr(value string) *time.Time {
	unix, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || unix <= 0 {
		return nil
	}
	return unixCredentialTimeFromIntPtr(unix)
}

func unixCredentialTimeFromIntPtr(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	if value > 1000000000000 {
		value = value / 1000
	}
	t := time.Unix(value, 0)
	return &t
}

func credentialValueToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
