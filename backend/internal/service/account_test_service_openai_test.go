//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIAccountTestRepo struct {
	mockAccountRepoForGemini
	updatedExtra  map[string]any
	rateLimitedID int64
	rateLimitedAt *time.Time
}

func (r *openAIAccountTestRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updatedExtra = updates
	return nil
}

func (r *openAIAccountTestRepo) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	r.rateLimitedID = id
	r.rateLimitedAt = &resetAt
	return nil
}

func TestAccountTestService_OpenAISuccessPersistsSnapshotFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newSoraTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	resp.Header.Set("x-codex-primary-used-percent", "88")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "42")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          89,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4")
	require.NoError(t, err)
	require.NotEmpty(t, repo.updatedExtra)
	require.Equal(t, 42.0, repo.updatedExtra["codex_5h_used_percent"])
	require.Equal(t, 88.0, repo.updatedExtra["codex_7d_used_percent"])
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAI429PersistsSnapshotAndRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newSoraTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached"}}`)
	resp.Header.Set("x-codex-primary-used-percent", "100")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "100")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          88,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4")
	require.Error(t, err)
	require.NotEmpty(t, repo.updatedExtra)
	require.Equal(t, 100.0, repo.updatedExtra["codex_5h_used_percent"])
	require.Equal(t, int64(88), repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.NotNil(t, account.RateLimitResetAt)
	if account.RateLimitResetAt != nil && repo.rateLimitedAt != nil {
		require.WithinDuration(t, *repo.rateLimitedAt, *account.RateLimitResetAt, time.Second)
	}
}

func TestAccountTestService_OpenAIApiKeyUsesV1ResponsesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newSoraTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.openai.com"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://api.openai.com/v1/responses", upstream.requests[0].URL.String())
	require.Contains(t, recorder.Body.String(), "test_complete")
}

// openAIStreamTextErrorRepo tracks SetError and SetSchedulable calls.
type openAIStreamTextErrorRepo struct {
	mockAccountRepoForGemini
	setErrorCalls       int
	lastErrorMsg        string
	setSchedulableCalls int
	lastSchedulable     bool
}

func (r *openAIStreamTextErrorRepo) SetError(_ context.Context, _ int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorMsg = errorMsg
	return nil
}

func (r *openAIStreamTextErrorRepo) SetSchedulable(_ context.Context, _ int64, schedulable bool) error {
	r.setSchedulableCalls++
	r.lastSchedulable = schedulable
	return nil
}

func TestAccountTestService_OpenAIApiKey_StreamBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name        string
		sseBody     string
		wantDisable bool // expect SetError+SetSchedulable(false)
		wantSuccess bool // expect test_complete in output
	}{
		{
			// Single delta matching known error phrase, no response.completed → disable
			name: "single_delta_error_phrase_no_completed",
			sseBody: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Your account is not active, please check your billing details on our website.\"}\n\n" +
				"data: [DONE]\n\n",
			wantDisable: true,
			wantSuccess: false,
		},
		{
			// Same error phrase but stream ends with response.completed → normal reply, no disable
			name: "error_phrase_with_completed_event_not_flagged",
			sseBody: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Your account is not active, please check your billing details on our website.\"}\n\n" +
				"data: {\"type\":\"response.completed\"}\n\n",
			wantDisable: false,
			wantSuccess: true,
		},
		{
			// Multiple deltas → normal multi-token reply, no disable even if one delta matches
			name: "multiple_deltas_not_flagged",
			sseBody: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Your account is not active\"}\n\n" +
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\", please check your billing details.\"}\n\n" +
				"data: [DONE]\n\n",
			wantDisable: false,
			wantSuccess: true,
		},
		{
			// Normal AI reply: no disable
			name: "normal_response_not_flagged",
			sseBody: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello! How can I help you?\"}\n\n" +
				"data: {\"type\":\"response.completed\"}\n\n",
			wantDisable: false,
			wantSuccess: true,
		},
		{
			// The original bug: "Hi — what can I help with?" must never trigger disable
			name: "hi_response_not_flagged",
			sseBody: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hi \\u2014 what can I help with?\"}\n\n" +
				"data: {\"type\":\"response.completed\"}\n\n",
			wantDisable: false,
			wantSuccess: true,
		},
		{
			// Single delta, no completed, but text does not match any known error phrase → success
			name: "single_delta_unknown_text_not_flagged",
			sseBody: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Sure, happy to help!\"}\n\n" +
				"data: [DONE]\n\n",
			wantDisable: false,
			wantSuccess: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := newSoraTestContext()

			resp := newJSONResponse(http.StatusOK, "")
			resp.Body = io.NopCloser(strings.NewReader(tc.sseBody))

			repo := &openAIStreamTextErrorRepo{}
			upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
			svc := &AccountTestService{
				httpUpstream: upstream,
				accountRepo:  repo,
				cfg:          &config.Config{},
			}
			account := &Account{
				ID:          91,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.openai.com"},
			}

			_ = svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4")

			if tc.wantDisable {
				require.Equal(t, 1, repo.setErrorCalls, "SetError should be called")
				require.Equal(t, 1, repo.setSchedulableCalls, "SetSchedulable should be called")
				require.False(t, repo.lastSchedulable)
				require.NotContains(t, recorder.Body.String(), "test_complete")
			} else {
				require.Equal(t, 0, repo.setErrorCalls, "SetError should NOT be called")
				require.Equal(t, 0, repo.setSchedulableCalls, "SetSchedulable should NOT be called")
				if tc.wantSuccess {
					require.Contains(t, recorder.Body.String(), "test_complete")
				}
			}
		})
	}
}

func TestIsStreamOnlyErrorText(t *testing.T) {
	cases := []struct {
		text          string
		deltaCount    int
		completedSeen bool
		want          bool
	}{
		// Known error phrase, single delta, no completed → detect
		{"Your account is not active, please check your billing details on our website.", 1, false, true},
		{"account is not active", 1, false, true},
		// response.completed seen → never detect
		{"Your account is not active, please check your billing details on our website.", 1, true, false},
		// Multiple deltas → normal reply
		{"Your account is not active", 2, false, false},
		// Text doesn't match any phrase
		{"Hello! How can I help you?", 1, false, false},
		{"Hi — what can I help with?", 1, false, false},
		{"Sure, happy to help!", 1, false, false},
		{"", 1, false, false},
	}

	for _, tc := range cases {
		got := isStreamOnlyErrorText(tc.text, tc.deltaCount, tc.completedSeen)
		if got != tc.want {
			t.Errorf("text=%q deltaCount=%d completedSeen=%v: want %v, got %v",
				tc.text, tc.deltaCount, tc.completedSeen, tc.want, got)
		}
	}
}
