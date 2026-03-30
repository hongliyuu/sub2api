package service

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// OpsSpansKey is the gin context key for the accumulated span slice.
	OpsSpansKey = "ops_spans"
)

// OpsSpan represents a single timed phase within a gateway request.
// Stored as a JSONB array in usage_logs.spans / ops_error_logs.spans.
//
// Phase name conventions:
//
//	auth.verify          — API key lookup + permission check
//	routing.select       — account selection (first attempt)
//	token.fetch          — Copilot access-token fetch
//	translate.req        — client→upstream body translation
//	upstream.post        — single upstream HTTP attempt (per failover attempt)
//	failover.select      — account re-selection after upstream error
//	translate.resp       — upstream→client body translation
//	body.truncation      — large request body truncation event
type OpsSpan struct {
	// Name identifies the phase (e.g. "auth.verify", "upstream.post").
	Name string `json:"name"`

	// StartUnixMs is the wall-clock start of this span (Unix milliseconds).
	StartUnixMs int64 `json:"start_unix_ms"`

	// DurationMs is the elapsed time for this span in milliseconds.
	DurationMs int64 `json:"duration_ms"`

	// Status is "ok", "error", or "skipped".
	Status string `json:"status,omitempty"`

	// Attrs holds span-specific key-value attributes (e.g. account_id, model, attempt).
	// Keep values small — strings and numbers only, no nested objects.
	Attrs map[string]any `json:"attrs,omitempty"`
}

// NewOpsSpan creates a span with StartUnixMs set to now.
// Call span.End() when the phase finishes.
func NewOpsSpan(name string) *OpsSpan {
	return &OpsSpan{
		Name:        name,
		StartUnixMs: time.Now().UnixMilli(),
	}
}

// End closes a span, setting DurationMs from StartUnixMs to now.
func (s *OpsSpan) End(status string) {
	if s == nil {
		return
	}
	s.DurationMs = time.Now().UnixMilli() - s.StartUnixMs
	if status != "" {
		s.Status = status
	}
}

// AppendOpsSpan appends a span to the gin context span slice.
// Safe to call with a nil context (no-op).
func AppendOpsSpan(c *gin.Context, span OpsSpan) {
	if c == nil {
		return
	}
	if span.StartUnixMs <= 0 {
		span.StartUnixMs = time.Now().UnixMilli()
	}
	var existing []*OpsSpan
	if v, ok := c.Get(OpsSpansKey); ok {
		if arr, ok := v.([]*OpsSpan); ok {
			existing = arr
		}
	}
	spanCopy := span
	existing = append(existing, &spanCopy)
	c.Set(OpsSpansKey, existing)
}

// GetOpsSpans reads the accumulated spans from the gin context.
// Returns nil if none were recorded.
func GetOpsSpans(c *gin.Context) []*OpsSpan {
	if c == nil {
		return nil
	}
	v, ok := c.Get(OpsSpansKey)
	if !ok {
		return nil
	}
	arr, _ := v.([]*OpsSpan)
	return arr
}

// MarshalOpsSpans serialises span slice to a JSON string for DB storage.
// Returns nil when spans is empty.
func MarshalOpsSpans(spans []*OpsSpan) *string {
	if len(spans) == 0 {
		return nil
	}
	raw, err := json.Marshal(spans)
	if err != nil || len(raw) == 0 {
		return nil
	}
	s := string(raw)
	return &s
}
