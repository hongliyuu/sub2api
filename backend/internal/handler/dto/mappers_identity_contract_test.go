//go:build unit

package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromService_PrefersCustomAvatarURL(t *testing.T) {
	user := &service.User{
		Avatar: &service.UserAvatar{
			URL:        "https://cdn.example.com/custom.webp",
			StorageKey: "avatars/u1.webp",
		},
		ExternalIdentities: []service.UserExternalIdentity{
			{Provider: service.ExternalIdentityProviderWeChat, AvatarURL: "https://cdn.example.com/provider.png"},
		},
	}

	got := UserFromService(user)

	require.NotNil(t, got)
	require.Equal(t, "https://cdn.example.com/custom.webp", got.AvatarURL)
}

func TestUserFromService_FallsBackToProviderAvatarURL(t *testing.T) {
	user := &service.User{
		ExternalIdentities: []service.UserExternalIdentity{
			{Provider: service.ExternalIdentityProviderWeChat, AvatarURL: "https://cdn.example.com/provider.png"},
		},
	}

	got := UserFromService(user)

	require.NotNil(t, got)
	require.Equal(t, "https://cdn.example.com/provider.png", got.AvatarURL)
}

func TestAccountBindingsFromService_UsesBindIntentRedirectAndDisconnectGuard(t *testing.T) {
	user := &service.User{
		Email:        "wechat-abc" + service.WeChatConnectSyntheticEmailDomain,
		PasswordHash: "generated-but-unusable",
		ExternalIdentities: []service.UserExternalIdentity{
			{Provider: service.ExternalIdentityProviderWeChat, ProviderUserID: "wechat-abc"},
		},
	}

	bindings := AccountBindingsFromService(user)

	require.Equal(t, "/api/v1/auth/oauth/linuxdo/start?redirect=%2Fprofile%3Fintent%3Dbind", bindings["linuxdo"].ConnectURL)
	require.Equal(t, "/api/v1/auth/oauth/wechat/start?redirect=%2Fprofile%3Fintent%3Dbind", bindings["wechat"].ConnectURL)
	require.False(t, bindings["wechat"].CanDisconnect)
}
