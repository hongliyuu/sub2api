//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingWeChatAdminPresetRepoStub struct {
	values  map[string]string
	updates map[string]string
}

func (s *settingWeChatAdminPresetRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingWeChatAdminPresetRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingWeChatAdminPresetRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *settingWeChatAdminPresetRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingWeChatAdminPresetRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.updates[key] = value
	}
	return nil
}

func (s *settingWeChatAdminPresetRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *settingWeChatAdminPresetRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestSettingService_UpdateSettings_ForcesWeChatAdminPreset(t *testing.T) {
	repo := &settingWeChatAdminPresetRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		WeChatConnectEnabled:     true,
		WeChatConnectAppID:       "wx1234567890abcdef",
		WeChatConnectAppSecret:   "wechat-secret",
		WeChatConnectMode:        "open",
		WeChatConnectScopes:      "snsapi_login",
		WeChatConnectRedirectURL: "https://example.com/api/v1/auth/oauth/wechat/callback",
	})
	require.NoError(t, err)
	require.Equal(t, defaultWeChatConnectMode, repo.updates[SettingKeyWeChatConnectMode])
	require.Equal(t, defaultWeChatConnectScopes, repo.updates[SettingKeyWeChatConnectScopes])
	require.Equal(t, defaultWeChatConnectFrontendRedirectURL, repo.updates[SettingKeyWeChatConnectFrontendRedirectURL])
}

func TestSettingService_GetAllSettings_UsesWeChatAdminPresetDefaultsWhenBlank(t *testing.T) {
	repo := &settingWeChatAdminPresetRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:     "true",
			SettingKeyWeChatConnectAppID:       "wx1234567890abcdef",
			SettingKeyWeChatConnectAppSecret:   "wechat-secret",
			SettingKeyWeChatConnectRedirectURL: "https://example.com/api/v1/auth/oauth/wechat/callback",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	got, err := svc.GetAllSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, defaultWeChatConnectMode, got.WeChatConnectMode)
	require.Equal(t, defaultWeChatConnectScopes, got.WeChatConnectScopes)
	require.Equal(t, defaultWeChatConnectFrontendRedirectURL, got.WeChatConnectFrontendRedirectURL)
}

func TestSettingService_GetWeChatConnectOAuthConfig_UsesWeChatAdminPresetDefaultsWhenBlank(t *testing.T) {
	repo := &settingWeChatAdminPresetRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:     "true",
			SettingKeyWeChatConnectAppID:       "wx1234567890abcdef",
			SettingKeyWeChatConnectAppSecret:   "wechat-secret",
			SettingKeyWeChatConnectRedirectURL: "https://example.com/api/v1/auth/oauth/wechat/callback",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	got, err := svc.GetWeChatConnectOAuthConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, defaultWeChatConnectMode, got.Mode)
	require.Equal(t, defaultWeChatConnectScopes, got.Scopes)
	require.Equal(t, defaultWeChatConnectFrontendRedirectURL, got.FrontendRedirectURL)
}
