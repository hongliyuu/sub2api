package service

import (
	"net/http"
	"strings"
)

// UpstreamFaultFormat identifies which downstream API contract to render.
type UpstreamFaultFormat string

const (
	UpstreamFaultFormatOpenAIResponses       UpstreamFaultFormat = "openai_responses"
	UpstreamFaultFormatOpenAIChatCompletions UpstreamFaultFormat = "openai_chat_completions"
	UpstreamFaultFormatAnthropicMessages     UpstreamFaultFormat = "anthropic_messages"
)

// UpstreamFaultCode is a provider-neutral internal fault code.
type UpstreamFaultCode string

const (
	UpstreamFaultCodeUnknown     UpstreamFaultCode = "upstream_unknown_error"
	UpstreamFaultCodeAuth        UpstreamFaultCode = "upstream_auth_error"
	UpstreamFaultCodePayment     UpstreamFaultCode = "upstream_payment_error"
	UpstreamFaultCodeForbidden   UpstreamFaultCode = "upstream_forbidden_error"
	UpstreamFaultCodeRateLimited UpstreamFaultCode = "upstream_rate_limited"
	UpstreamFaultCodeInvalid     UpstreamFaultCode = "upstream_invalid_request"
	UpstreamFaultCodeNotFound    UpstreamFaultCode = "upstream_not_found"
)

// UpstreamMappedFault is the normalized output contract used by handlers.
type UpstreamMappedFault struct {
	Code       UpstreamFaultCode
	StatusCode int
	ErrorType  string
	Message    string
}

// UpstreamFaultMapper maps upstream status/code/type/body into unified client-facing errors.
// It must never expose raw upstream message text to end users.
type UpstreamFaultMapper struct{}

var defaultUpstreamFaultMapper = &UpstreamFaultMapper{}

func NewUpstreamFaultMapper() *UpstreamFaultMapper {
	return &UpstreamFaultMapper{}
}

func (m *UpstreamFaultMapper) Map(
	statusCode int,
	upstreamCode string,
	upstreamType string,
	responseBody []byte,
	format UpstreamFaultFormat,
) UpstreamMappedFault {
	normalizedCode := strings.ToLower(strings.TrimSpace(upstreamCode))
	normalizedType := strings.ToLower(strings.TrimSpace(upstreamType))
	normalizedBody := strings.ToLower(string(responseBody))
	quotaExhausted := statusCode == http.StatusTooManyRequests &&
		(normalizedCode == "insufficient_quota" ||
			normalizedType == "insufficient_quota" ||
			strings.Contains(normalizedBody, "insufficient_quota"))

	switch format {
	case UpstreamFaultFormatOpenAIResponses:
		return mapOpenAIResponsesFault(statusCode, quotaExhausted)
	case UpstreamFaultFormatAnthropicMessages:
		return mapAnthropicMessagesFault(statusCode, quotaExhausted)
	default:
		return mapOpenAIChatCompletionsFault(statusCode, quotaExhausted)
	}
}

func mapOpenAIResponsesFault(statusCode int, quotaExhausted bool) UpstreamMappedFault {
	switch statusCode {
	case http.StatusUnauthorized:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeAuth,
			StatusCode: http.StatusBadGateway,
			ErrorType:  "upstream_error",
			Message:    "Upstream authentication failed, please contact administrator",
		}
	case http.StatusPaymentRequired:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodePayment,
			StatusCode: http.StatusBadGateway,
			ErrorType:  "upstream_error",
			Message:    "Upstream payment required: insufficient balance or billing issue",
		}
	case http.StatusForbidden:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeForbidden,
			StatusCode: http.StatusBadGateway,
			ErrorType:  "upstream_error",
			Message:    "Upstream access forbidden, please contact administrator",
		}
	case http.StatusTooManyRequests:
		internalCode := UpstreamFaultCodeRateLimited
		if quotaExhausted {
			internalCode = UpstreamFaultCodePayment
		}
		return UpstreamMappedFault{
			Code:       internalCode,
			StatusCode: http.StatusTooManyRequests,
			ErrorType:  "rate_limit_error",
			Message:    "Upstream rate limit exceeded, please retry later",
		}
	default:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeUnknown,
			StatusCode: http.StatusBadGateway,
			ErrorType:  "upstream_error",
			Message:    "Upstream request failed",
		}
	}
}

func mapOpenAIChatCompletionsFault(statusCode int, quotaExhausted bool) UpstreamMappedFault {
	switch statusCode {
	case http.StatusBadRequest:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeInvalid,
			StatusCode: http.StatusBadRequest,
			ErrorType:  "invalid_request_error",
			Message:    "Upstream request was rejected",
		}
	case http.StatusNotFound:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeNotFound,
			StatusCode: http.StatusNotFound,
			ErrorType:  "not_found_error",
			Message:    "Upstream resource not found",
		}
	case http.StatusTooManyRequests:
		internalCode := UpstreamFaultCodeRateLimited
		if quotaExhausted {
			internalCode = UpstreamFaultCodePayment
		}
		return UpstreamMappedFault{
			Code:       internalCode,
			StatusCode: http.StatusTooManyRequests,
			ErrorType:  "rate_limit_error",
			Message:    "Upstream rate limit exceeded, please retry later",
		}
	default:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeUnknown,
			StatusCode: statusCode,
			ErrorType:  "api_error",
			Message:    "Upstream request failed",
		}
	}
}

func mapAnthropicMessagesFault(statusCode int, quotaExhausted bool) UpstreamMappedFault {
	switch statusCode {
	case http.StatusBadRequest:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeInvalid,
			StatusCode: http.StatusBadRequest,
			ErrorType:  "invalid_request_error",
			Message:    "Upstream request was rejected",
		}
	case http.StatusNotFound:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeNotFound,
			StatusCode: http.StatusNotFound,
			ErrorType:  "not_found_error",
			Message:    "Upstream resource not found",
		}
	case http.StatusTooManyRequests:
		internalCode := UpstreamFaultCodeRateLimited
		if quotaExhausted {
			internalCode = UpstreamFaultCodePayment
		}
		return UpstreamMappedFault{
			Code:       internalCode,
			StatusCode: http.StatusTooManyRequests,
			ErrorType:  "rate_limit_error",
			Message:    "Upstream rate limit exceeded, please retry later",
		}
	default:
		return UpstreamMappedFault{
			Code:       UpstreamFaultCodeUnknown,
			StatusCode: statusCode,
			ErrorType:  "api_error",
			Message:    "Upstream request failed",
		}
	}
}
