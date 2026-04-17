//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingWeChatRepoStub struct {
	values map[string]string
}

func (s *settingWeChatRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingWeChatRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingWeChatRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *settingWeChatRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingWeChatRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingWeChatRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingWeChatRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetWeChatConnectOAuthConfig(t *testing.T) {
	t.Run("uses db overrides", func(t *testing.T) {
		svc := NewSettingService(&settingWeChatRepoStub{
			values: map[string]string{
				SettingKeyWeChatConnectEnabled:             "true",
				SettingKeyWeChatConnectAppID:               "wx-db-app",
				SettingKeyWeChatConnectAppSecret:           "db-secret",
				SettingKeyWeChatConnectMode:                "open",
				SettingKeyWeChatConnectScopes:              "snsapi_base snsapi_userinfo",
				SettingKeyWeChatConnectRedirectURL:         "https://example.com/api/v1/auth/oauth/wechat/callback",
				SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
			},
		}, &config.Config{
			WeChat: config.WeChatConnectConfig{
				Enabled:             true,
				AppID:               "wx-config",
				AppSecret:           "cfg-secret",
				Scopes:              "snsapi_login",
				RedirectURL:         "https://cfg.example.com/callback",
				FrontendRedirectURL: "/auth/wechat/callback",
			},
		})

		got, err := svc.GetWeChatConnectOAuthConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, "wx-db-app", got.AppID)
		require.Equal(t, "db-secret", got.AppSecret)
		require.Equal(t, "snsapi_base snsapi_userinfo", got.Scopes)
		require.Equal(t, "https://example.com/api/v1/auth/oauth/wechat/callback", got.RedirectURL)
	})

	t.Run("falls back to config and defaults frontend callback", func(t *testing.T) {
		svc := NewSettingService(&settingWeChatRepoStub{values: map[string]string{}}, &config.Config{
			WeChat: config.WeChatConnectConfig{
				Enabled:     true,
				AppID:       "wx-config",
				AppSecret:   "cfg-secret",
				RedirectURL: "https://cfg.example.com/api/v1/auth/oauth/wechat/callback",
			},
		})

		got, err := svc.GetWeChatConnectOAuthConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, "wx-config", got.AppID)
		require.Equal(t, "/auth/wechat/callback", got.FrontendRedirectURL)
		require.Equal(t, "snsapi_login", got.Scopes)
		require.Equal(t, "open", got.Mode)
	})

	t.Run("uses defaults when db values are blank strings", func(t *testing.T) {
		svc := NewSettingService(&settingWeChatRepoStub{
			values: map[string]string{
				SettingKeyWeChatConnectEnabled:             "true",
				SettingKeyWeChatConnectAppID:               "wx-db-app",
				SettingKeyWeChatConnectAppSecret:           "db-secret",
				SettingKeyWeChatConnectMode:                "   ",
				SettingKeyWeChatConnectScopes:              "",
				SettingKeyWeChatConnectRedirectURL:         "https://example.com/api/v1/auth/oauth/wechat/callback",
				SettingKeyWeChatConnectFrontendRedirectURL: " ",
			},
		}, &config.Config{})

		got, err := svc.GetWeChatConnectOAuthConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, "open", got.Mode)
		require.Equal(t, "snsapi_login", got.Scopes)
		require.Equal(t, "/auth/wechat/callback", got.FrontendRedirectURL)
	})

	t.Run("rejects disabled config", func(t *testing.T) {
		svc := NewSettingService(&settingWeChatRepoStub{
			values: map[string]string{SettingKeyWeChatConnectEnabled: "false"},
		}, &config.Config{
			WeChat: config.WeChatConnectConfig{
				Enabled:     true,
				AppID:       "wx-config",
				AppSecret:   "cfg-secret",
				RedirectURL: "https://cfg.example.com/api/v1/auth/oauth/wechat/callback",
			},
		})

		_, err := svc.GetWeChatConnectOAuthConfig(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "oauth login is disabled")
	})
}
