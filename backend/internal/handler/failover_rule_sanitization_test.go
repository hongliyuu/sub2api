package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type errorPassthroughRepoStub struct {
	rules []*model.ErrorPassthroughRule
}

func (s *errorPassthroughRepoStub) List(context.Context) ([]*model.ErrorPassthroughRule, error) {
	return s.rules, nil
}
func (s *errorPassthroughRepoStub) GetByID(_ context.Context, id int64) (*model.ErrorPassthroughRule, error) {
	for _, r := range s.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}
func (s *errorPassthroughRepoStub) Create(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (s *errorPassthroughRepoStub) Update(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (s *errorPassthroughRepoStub) Delete(context.Context, int64) error { return nil }

func newRuleServiceForHandlerTest(rule *model.ErrorPassthroughRule) *service.ErrorPassthroughService {
	return service.NewErrorPassthroughService(&errorPassthroughRepoStub{
		rules: []*model.ErrorPassthroughRule{rule},
	}, nil)
}

func TestOpenAIHandleFailoverExhausted_RuleNeverLeaksUpstreamBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	ruleSvc := newRuleServiceForHandlerTest(&model.ErrorPassthroughRule{
		ID:              1,
		Name:            "legacy-openai-rule",
		Enabled:         true,
		Priority:        1,
		ErrorCodes:      []int{http.StatusBadRequest},
		MatchMode:       model.MatchModeAny,
		Platforms:       []string{"openai"},
		PassthroughCode: true,
		PassthroughBody: true, // Deprecated legacy value, should be ignored.
	})

	h := &OpenAIGatewayHandler{errorPassthroughService: ruleSvc}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusBadRequest,
		ResponseBody: []byte(`{"error":{"message":"SECRET_OPENAI_UPSTREAM"}}`),
	}, false)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errObj := payload["error"].(map[string]any)
	require.Equal(t, "upstream_error", errObj["type"])
	require.Equal(t, "Upstream request failed", errObj["message"])
	require.NotContains(t, rec.Body.String(), "SECRET_OPENAI_UPSTREAM")
}

func TestGatewayHandleFailoverExhausted_RuleNeverLeaksUpstreamBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	ruleSvc := newRuleServiceForHandlerTest(&model.ErrorPassthroughRule{
		ID:              2,
		Name:            "legacy-anthropic-rule",
		Enabled:         true,
		Priority:        1,
		ErrorCodes:      []int{http.StatusForbidden},
		MatchMode:       model.MatchModeAny,
		Platforms:       []string{service.PlatformAnthropic},
		PassthroughCode: true,
		PassthroughBody: true,
	})

	h := &GatewayHandler{errorPassthroughService: ruleSvc}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusForbidden,
		ResponseBody: []byte(`{"error":{"message":"SECRET_ANTHROPIC_UPSTREAM"}}`),
	}, service.PlatformAnthropic, false)

	require.Equal(t, http.StatusForbidden, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errObj := payload["error"].(map[string]any)
	require.Equal(t, "upstream_error", errObj["type"])
	require.Equal(t, "Upstream access forbidden, please contact administrator", errObj["message"])
	require.NotContains(t, rec.Body.String(), "SECRET_ANTHROPIC_UPSTREAM")
}

func TestGeminiHandleFailoverExhausted_RuleNeverLeaksUpstreamBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", nil)

	ruleSvc := newRuleServiceForHandlerTest(&model.ErrorPassthroughRule{
		ID:              3,
		Name:            "legacy-gemini-rule",
		Enabled:         true,
		Priority:        1,
		ErrorCodes:      []int{http.StatusTooManyRequests},
		MatchMode:       model.MatchModeAny,
		Platforms:       []string{service.PlatformGemini},
		PassthroughCode: true,
		PassthroughBody: true,
	})

	h := &GatewayHandler{errorPassthroughService: ruleSvc}
	h.handleGeminiFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"message":"SECRET_GEMINI_UPSTREAM"}}`),
	})

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errObj := payload["error"].(map[string]any)
	require.Equal(t, float64(http.StatusTooManyRequests), errObj["code"])
	require.Equal(t, "Upstream rate limit exceeded, please retry later", errObj["message"])
	require.NotContains(t, rec.Body.String(), "SECRET_GEMINI_UPSTREAM")
}
