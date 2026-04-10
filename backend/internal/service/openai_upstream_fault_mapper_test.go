package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpstreamFaultMapper_OpenAIResponsesContract(t *testing.T) {
	t.Parallel()

	mapper := NewUpstreamFaultMapper()
	mapped := mapper.Map(
		http.StatusUnauthorized,
		"invalid_api_key",
		"authentication_error",
		[]byte(`{"error":{"message":"SECRET"}}`),
		UpstreamFaultFormatOpenAIResponses,
	)

	require.Equal(t, UpstreamFaultCodeAuth, mapped.Code)
	require.Equal(t, http.StatusBadGateway, mapped.StatusCode)
	require.Equal(t, "upstream_error", mapped.ErrorType)
	require.Equal(t, "Upstream authentication failed, please contact administrator", mapped.Message)
	require.NotContains(t, mapped.Message, "SECRET")
}

func TestOpenAIGatewayService_ForwardAsChatCompletions_UsesMappedFault(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.2","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid_chat_fault"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"SECRET_BILLING_TEXT"}}`))),
	}}

	svc := &OpenAIGatewayService{
		cfg:                 &config.Config{},
		httpUpstream:        upstream,
		upstreamFaultMapper: NewUpstreamFaultMapper(),
	}
	account := &Account{
		ID:          1,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"type":"invalid_request_error"`)
	require.Contains(t, rec.Body.String(), `"message":"Upstream request was rejected"`)
	require.NotContains(t, rec.Body.String(), "SECRET_BILLING_TEXT")
}

func TestOpenAIGatewayService_ForwardAsAnthropic_UsesMappedFault(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.2","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid_msg_fault"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"SECRET_UPSTREAM_NOT_FOUND"}}`))),
	}}

	svc := &OpenAIGatewayService{
		cfg:                 &config.Config{},
		httpUpstream:        upstream,
		upstreamFaultMapper: NewUpstreamFaultMapper(),
	}
	account := &Account{
		ID:          2,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "")
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"type":"error"`)
	require.Contains(t, rec.Body.String(), `"type":"not_found_error"`)
	require.Contains(t, rec.Body.String(), `"message":"Upstream resource not found"`)
	require.NotContains(t, rec.Body.String(), "SECRET_UPSTREAM_NOT_FOUND")
}
