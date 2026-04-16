package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminSettingRepoStub struct {
	values  map[string]string
	updates map[string]string
}

func (s *adminSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *adminSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *adminSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *adminSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *adminSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.updates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.values[key] = value
		s.updates[key] = value
	}
	return nil
}

func (s *adminSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *adminSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingHandler_UpdateSettings_PreservesWeChatSecretAndDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminSettingRepoStub{
		values: map[string]string{
			service.SettingKeyWeChatConnectAppSecret: "existing-secret",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Mode:                "open",
			Scopes:              "snsapi_login",
			FrontendRedirectURL: "/auth/wechat/callback",
		},
	})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"wechat_connect_enabled":      true,
		"wechat_connect_app_id":       "wx1234567890abcdef",
		"wechat_connect_redirect_url": "https://example.com/api/v1/auth/oauth/wechat/callback",
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "existing-secret", repo.updates[service.SettingKeyWeChatConnectAppSecret])
	require.Equal(t, "open", repo.updates[service.SettingKeyWeChatConnectMode])
	require.Equal(t, "snsapi_login", repo.updates[service.SettingKeyWeChatConnectScopes])
	require.Equal(t, "/auth/wechat/callback", repo.updates[service.SettingKeyWeChatConnectFrontendRedirectURL])
}
