package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// CopilotGatewayHandler handles GitHub Copilot API gateway requests.
// It exposes /copilot/v1/chat/completions and /copilot/v1/responses endpoints
// and can also be invoked through the automatic platform routing on /v1/*.
type CopilotGatewayHandler struct {
	copilotService        *service.CopilotGatewayService
	gatewayService        *service.GatewayService
	billingCacheService   *service.BillingCacheService
	apiKeyService         *service.APIKeyService
	usageRecordWorkerPool *service.UsageRecordWorkerPool
	concurrencyHelper     *ConcurrencyHelper
	maxAccountSwitches    int
}

// NewCopilotGatewayHandler creates a new CopilotGatewayHandler.
func NewCopilotGatewayHandler(
	copilotService *service.CopilotGatewayService,
	gatewayService *service.GatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
) *CopilotGatewayHandler {
	return &CopilotGatewayHandler{
		copilotService:        copilotService,
		gatewayService:        gatewayService,
		billingCacheService:   billingCacheService,
		apiKeyService:         apiKeyService,
		usageRecordWorkerPool: usageRecordWorkerPool,
		concurrencyHelper:     NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, 0),
		maxAccountSwitches:    5,
	}
}

// ChatCompletions handles POST /copilot/v1/chat/completions
func (h *CopilotGatewayHandler) ChatCompletions(c *gin.Context) {
	h.handleRequest(c, "chat_completions", func(ctx *copilotRequestContext) (*service.CopilotForwardResult, error) {
		return ctx.service.ForwardChatCompletions(c.Request.Context(), c, ctx.account, ctx.body)
	})
}

// Responses handles POST /copilot/v1/responses
func (h *CopilotGatewayHandler) Responses(c *gin.Context) {
	h.handleRequest(c, "responses", func(ctx *copilotRequestContext) (*service.CopilotForwardResult, error) {
		return ctx.service.ForwardResponses(c.Request.Context(), c, ctx.account, ctx.body)
	})
}

// Models handles GET /copilot/v1/models
func (h *CopilotGatewayHandler) Models(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(c, "handler.copilot_gateway.models",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
	)

	// Select a Copilot account to query models from.
	account, err := h.selectAccount(c, apiKey, "")
	if err != nil {
		reqLog.Warn("copilot: no available account for models list", zap.Error(err))
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Copilot accounts")
		return
	}

	modelsJSON, err := h.copilotService.ListModels(c.Request.Context(), account)
	if err != nil {
		reqLog.Error("copilot: failed to list models", zap.Error(err))
		h.errorResponse(c, http.StatusBadGateway, "api_error", "Failed to list Copilot models")
		return
	}

	c.Data(http.StatusOK, "application/json", modelsJSON)
}

// copilotRequestContext holds the pre-validated request state.
type copilotRequestContext struct {
	service *service.CopilotGatewayService
	account *service.Account
	body    []byte
}

// handleRequest is the shared request processing logic for both
// ChatCompletions and Responses endpoints. It handles authentication,
// account selection, failover, and concurrency control.
func (h *CopilotGatewayHandler) handleRequest(
	c *gin.Context,
	endpoint string,
	forwardFn func(ctx *copilotRequestContext) (*service.CopilotForwardResult, error),
) {
	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(c, "handler.copilot_gateway."+endpoint,
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	// Read and validate request body.
	body, err := httputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}

	reqModel := gjson.GetBytes(body, "model").String()
	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	reqLog = reqLog.With(zap.String("model", reqModel))

	// Acquire user concurrency slot.
	reqStream := gjson.GetBytes(body, "stream").Bool()
	streamStarted := false
	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted)
	if err != nil {
		reqLog.Warn("copilot: user slot acquire failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", streamStarted)
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// Check billing eligibility after acquiring slot.
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("copilot: billing eligibility check failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.errorResponse(c, status, code, message)
		return
	}

	// Account selection with failover loop.
	var failedAccountIDs []int64
	for attempt := 0; attempt <= h.maxAccountSwitches; attempt++ {
		account, err := h.selectAccountExcluding(c, apiKey, reqModel, failedAccountIDs)
		if err != nil {
			reqLog.Warn("copilot: no available accounts",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.Int64s("failed_ids", failedAccountIDs),
			)
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Copilot accounts: "+err.Error())
			return
		}

		result, fwdErr := forwardFn(&copilotRequestContext{
			service: h.copilotService,
			account: account,
			body:    body,
		})

		if fwdErr == nil {
			reqLog.Info("copilot: request forwarded successfully",
				zap.Int64("account_id", account.ID),
				zap.Duration("duration", time.Since(requestStart)),
			)

			// Submit usage record asynchronously.
			// Note: Copilot streaming responses are forwarded as-is without
			// token usage parsing, so we record a basic usage entry for
			// billing/audit purposes. Token counts will be zero.
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			requestPayloadHash := service.HashUsageRequestPayload(body)
			inboundEndpoint := GetInboundEndpoint(c)
			upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
			h.submitUsageRecordTask(func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
					Result: &service.ForwardResult{
						Model:         reqModel,
						UpstreamModel: result.UpstreamModel,
					},
					APIKey:             apiKey,
					User:               apiKey.User,
					Account:            account,
					Subscription:       subscription,
					InboundEndpoint:    inboundEndpoint,
					UpstreamEndpoint:   upstreamEndpoint,
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: requestPayloadHash,
					APIKeyService:      h.apiKeyService,
				}); err != nil {
					logger.L().With(
						zap.String("component", "handler.copilot_gateway."+endpoint),
						zap.Int64("user_id", subject.UserID),
						zap.Int64("api_key_id", apiKey.ID),
						zap.Any("group_id", apiKey.GroupID),
						zap.String("model", reqModel),
						zap.Int64("account_id", account.ID),
					).Error("copilot: record usage failed", zap.Error(err))
				}
			})
			return
		}

		// Check if the error is eligible for failover.
		var upstreamErr *service.CopilotUpstreamError
		if errors.As(fwdErr, &upstreamErr) && service.ShouldFailoverCopilotUpstreamError(upstreamErr.StatusCode) {
			reqLog.Info("copilot: upstream error, trying next account",
				zap.Int64("account_id", account.ID),
				zap.Int("status", upstreamErr.StatusCode),
				zap.Int("attempt", attempt),
			)
			failedAccountIDs = append(failedAccountIDs, account.ID)
			continue
		}

		// Non-failover error: return to client.
		reqLog.Error("copilot: forward failed (non-failover)",
			zap.Int64("account_id", account.ID),
			zap.Error(fwdErr),
		)
		if !c.Writer.Written() {
			if upstreamErr != nil {
				c.Data(upstreamErr.StatusCode, "application/json", upstreamErr.Body)
			} else {
				h.errorResponse(c, http.StatusBadGateway, "api_error", "Copilot request failed")
			}
		}
		return
	}

	// All accounts exhausted.
	if !c.Writer.Written() {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "All Copilot accounts exhausted after failover")
	}
}

// selectAccount picks an available Copilot account for the given API key.
func (h *CopilotGatewayHandler) selectAccount(c *gin.Context, apiKey *service.APIKey, model string) (*service.Account, error) {
	return h.selectAccountExcluding(c, apiKey, model, nil)
}

// selectAccountExcluding picks an available Copilot account, excluding
// the specified account IDs (used during failover).
func (h *CopilotGatewayHandler) selectAccountExcluding(c *gin.Context, apiKey *service.APIKey, model string, excludeIDs []int64) (*service.Account, error) {
	var excludeSet map[int64]struct{}
	if len(excludeIDs) > 0 {
		excludeSet = make(map[int64]struct{}, len(excludeIDs))
		for _, id := range excludeIDs {
			excludeSet[id] = struct{}{}
		}
	}

	return h.gatewayService.SelectAccountForModelWithExclusions(
		c.Request.Context(),
		apiKey.GroupID,
		"", // no sticky session for copilot
		model,
		excludeSet,
	)
}

func (h *CopilotGatewayHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

func (h *CopilotGatewayHandler) handleConcurrencyError(c *gin.Context, err error, scope string, streamStarted bool) {
	logger.L().Warn("copilot: concurrency error",
		zap.String("scope", scope),
		zap.Error(err),
	)
	if streamStarted {
		return
	}
	h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many concurrent requests")
}

// submitUsageRecordTask submits a usage recording task to the bounded worker
// pool. Falls back to synchronous execution if the pool is not injected.
func (h *CopilotGatewayHandler) submitUsageRecordTask(task service.UsageRecordTask) {
	if task == nil {
		return
	}
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)
		return
	}
	// Fallback: synchronous execution with timeout + panic recovery.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().With(
				zap.String("component", "handler.copilot_gateway"),
				zap.Any("panic", recovered),
			).Error("copilot: usage record task panic recovered")
		}
	}()
	task(ctx)
}
