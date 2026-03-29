package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// AccessLogParams collects all data needed for an access log entry.
type AccessLogParams struct {
	RequestID       string
	ClientRequestID string
	SessionHash     string
	Platform        string
	Model           string
	UpstreamModel   string
	AccountID       int64
	GroupID         int64
	UserID          int64
	APIKeyID        int64
	Stream          bool
	RequestType     string
	InboundEndpoint string
	ClientIP        string
	UserAgent       string
	StatusCode      int
	DurationMs      int
	FirstTokenMs    int
	AuthLatencyMs   int
	RouteLatencyMs  int
	InputTokens     int
	OutputTokens    int
	CacheCreation   int
	CacheRead       int
	RetryCount      int
	AccountSwitches int
	RequestBody     string
	ResponseBody    string
}

// BuildAccessLogEntry converts params into a logger.AccessLogEntry.
func BuildAccessLogEntry(p AccessLogParams) logger.AccessLogEntry {
	return logger.AccessLogEntry{
		RequestID:           p.RequestID,
		ClientRequestID:     p.ClientRequestID,
		SessionHash:         p.SessionHash,
		Platform:            p.Platform,
		Model:               p.Model,
		UpstreamModel:       p.UpstreamModel,
		AccountID:           p.AccountID,
		GroupID:             p.GroupID,
		UserID:              p.UserID,
		APIKeyID:            p.APIKeyID,
		Stream:              p.Stream,
		RequestType:         p.RequestType,
		InboundEndpoint:     p.InboundEndpoint,
		ClientIP:            p.ClientIP,
		UserAgent:           p.UserAgent,
		StatusCode:          p.StatusCode,
		Success:             p.StatusCode >= 200 && p.StatusCode < 400,
		DurationMs:          p.DurationMs,
		FirstTokenMs:        p.FirstTokenMs,
		AuthLatencyMs:       p.AuthLatencyMs,
		RouteLatencyMs:      p.RouteLatencyMs,
		InputTokens:         p.InputTokens,
		OutputTokens:        p.OutputTokens,
		CacheCreationTokens: p.CacheCreation,
		CacheReadTokens:     p.CacheRead,
		RetryCount:          p.RetryCount,
		AccountSwitchCount:  p.AccountSwitches,
		RequestBody:         p.RequestBody,
		ResponseBody:        p.ResponseBody,
	}
}

// CollectAccessLogParamsFromContext extracts common fields from gin context.
func CollectAccessLogParamsFromContext(c *gin.Context) AccessLogParams {
	ctx := c.Request.Context()
	var params AccessLogParams

	if v, ok := ctx.Value(ctxkey.RequestID).(string); ok {
		params.RequestID = v
	}
	if v, ok := ctx.Value(ctxkey.ClientRequestID).(string); ok {
		params.ClientRequestID = v
	}
	if v, ok := ctx.Value(ctxkey.Platform).(string); ok {
		params.Platform = v
	}
	if v, ok := ctx.Value(ctxkey.Model).(string); ok {
		params.Model = v
	}
	if v, ok := ctx.Value(ctxkey.AccountID).(int64); ok {
		params.AccountID = v
	}

	if count, ok := service.AccountSwitchCountFromContext(ctx); ok {
		params.AccountSwitches = count
	}
	if v, ok := ctx.Value(ctxkey.RetryCount).(int); ok {
		params.RetryCount = v
	}

	if v, ok := getOpsInt64(c, "ops_auth_latency_ms"); ok {
		params.AuthLatencyMs = int(v)
	}
	if v, ok := getOpsInt64(c, "ops_routing_latency_ms"); ok {
		params.RouteLatencyMs = int(v)
	}

	return params
}

// stringPtrIfNotEmpty returns a pointer to s if it is non-empty, otherwise nil.
func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func getOpsInt64(c *gin.Context, key string) (int64, bool) {
	v, exists := c.Get(key)
	if !exists {
		return 0, false
	}
	if i, ok := v.(int64); ok {
		return i, true
	}
	return 0, false
}
