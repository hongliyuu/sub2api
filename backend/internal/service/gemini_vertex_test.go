package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type vertexTokenCacheStub struct {
	token string
}

func (s *vertexTokenCacheStub) GetAccessToken(context.Context, string) (string, error) {
	if strings.TrimSpace(s.token) == "" {
		return "", errors.New("cache miss")
	}
	return s.token, nil
}

func (s *vertexTokenCacheStub) SetAccessToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *vertexTokenCacheStub) DeleteAccessToken(context.Context, string) error {
	return nil
}

func (s *vertexTokenCacheStub) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return true, nil
}

func (s *vertexTokenCacheStub) ReleaseRefreshLock(context.Context, string) error {
	return nil
}

func TestBuildGeminiVertexURL(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeVertex,
		Credentials: map[string]any{
			"project_id": "demo-project",
			"location":   "asia-northeast1",
		},
	}

	fullURL, err := buildGeminiVertexURL(account, "gemini-2.5-pro", "streamGenerateContent", true)
	require.NoError(t, err)
	require.Equal(t,
		"https://asia-northeast1-aiplatform.googleapis.com/v1beta1/projects/demo-project/locations/asia-northeast1/publishers/google/models/gemini-2.5-pro:streamGenerateContent?alt=sse",
		fullURL,
	)
}

func TestBuildGeminiVertexURL_GlobalEndpoint(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeVertex,
		Credentials: map[string]any{
			"project_id": "demo-project",
			"location":   "global",
		},
	}

	fullURL, err := buildGeminiVertexURL(account, "gemini-2.5-pro", "streamGenerateContent", true)
	require.NoError(t, err)
	require.Equal(t,
		"https://aiplatform.googleapis.com/v1beta1/projects/demo-project/locations/global/publishers/google/models/gemini-2.5-pro:streamGenerateContent?alt=sse",
		fullURL,
	)
}

func TestRewriteGeminiVertexGetURL(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeVertex,
		Credentials: map[string]any{
			"project_id": "demo-project",
			"location":   "us-central1",
		},
	}

	fullURL, err := rewriteGeminiVertexGetURL(account, "/v1beta/models/gemini-2.5-flash")
	require.NoError(t, err)
	require.Equal(t,
		"https://us-central1-aiplatform.googleapis.com/v1beta1/projects/demo-project/locations/us-central1/publishers/google/models/gemini-2.5-flash",
		fullURL,
	)
}

func TestRewriteGeminiVertexGetURL_GlobalEndpoint(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeVertex,
		Credentials: map[string]any{
			"project_id": "demo-project",
			"location":   "global",
		},
	}

	fullURL, err := rewriteGeminiVertexGetURL(account, "/v1beta/models/gemini-2.5-flash")
	require.NoError(t, err)
	require.Equal(t,
		"https://aiplatform.googleapis.com/v1beta1/projects/demo-project/locations/global/publishers/google/models/gemini-2.5-flash",
		fullURL,
	)
}

func TestGeminiMessagesCompatServiceForward_UsesVertexEndpoint(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"vertex-req-1"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`)),
		},
	}

	provider := &GeminiTokenProvider{
		tokenCache: &vertexTokenCacheStub{token: "vertex-access-token"},
	}
	svc := &GeminiMessagesCompatService{
		httpUpstream:  httpStub,
		tokenProvider: provider,
		cfg:           &config.Config{},
	}

	account := &Account{
		ID:       9,
		Platform: PlatformGemini,
		Type:     AccountTypeVertex,
		Credentials: map[string]any{
			"project_id": "demo-project",
			"location":   "us-central1",
		},
	}
	body := []byte(`{"model":"gemini-2.5-pro","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, httpStub.lastReq)
	require.Equal(t, "Bearer vertex-access-token", httpStub.lastReq.Header.Get("Authorization"))
	require.Contains(t, httpStub.lastReq.URL.String(), "/v1beta1/projects/demo-project/locations/us-central1/publishers/google/models/gemini-2.5-pro:generateContent")
}
