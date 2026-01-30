package service

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitScope string

const (
	RateLimitScopeUnknown     RateLimitScope = "unknown"
	RateLimitScopeAccount     RateLimitScope = "account"
	RateLimitScopeModelBucket RateLimitScope = "model_bucket"
)

// RateLimitSignal describes the metadata emitted with a 429 response.
type RateLimitSignal struct {
	Scope   RateLimitScope
	Key     string
	ResetAt time.Time
}

// setRateLimitResponseHeaders writes the stable headers that describe the throttled bucket.
func setRateLimitResponseHeaders(c *gin.Context, signal *RateLimitSignal) {
	if signal == nil || signal.ResetAt.IsZero() {
		return
	}
	retryAfter := int(time.Until(signal.ResetAt).Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	c.Header("Retry-After", strconv.Itoa(retryAfter))
	c.Header("X-Sub2API-RateLimit-Scope", string(signal.Scope))
	if signal.Scope == RateLimitScopeModelBucket && signal.Key != "" {
		c.Header("X-Sub2API-RateLimit-Key", signal.Key)
	}
}
