package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_UpdateSettings_ForcesWeChatAdminPreset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &adminSettingRepoStub{
		values: map[string]string{
			service.SettingKeyWeChatConnectAppSecret: "existing-secret",
			service.SettingKeyWeChatConnectMode:      "open",
			service.SettingKeyWeChatConnectScopes:    "snsapi_login",
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
		"wechat_connect_mode":         "open",
		"wechat_connect_scopes":       "snsapi_login",
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
	require.Equal(t, weChatConnectAdminFixedMode, repo.updates[service.SettingKeyWeChatConnectMode])
	require.Equal(t, weChatConnectAdminFixedScopes, repo.updates[service.SettingKeyWeChatConnectScopes])
	require.Equal(t, weChatConnectDefaultFrontendRedirectURL, repo.updates[service.SettingKeyWeChatConnectFrontendRedirectURL])
}
