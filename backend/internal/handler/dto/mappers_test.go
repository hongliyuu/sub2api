package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromService_ExposesStableAvatarAndAccountBindings(t *testing.T) {
	now := time.Now().UTC()
	user := &service.User{
		ID:        42,
		Email:     "user@example.com",
		Username:  "alice",
		Role:      service.RoleUser,
		Status:    service.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		AvatarURL: "https://cdn.example.com/avatar.webp",
		ExternalIdentities: []service.UserExternalIdentity{
			{
				Provider:         service.ExternalIdentityProviderLinuxDo,
				ProviderUserID:   "linuxdo-subject-1",
				ProviderUsername: "linuxdo_alice",
				DisplayName:      "LinuxDo Alice",
				CreatedAt:        now,
			},
		},
	}

	got := UserFromService(user)
	require.NotNil(t, got)
	require.Equal(t, "https://cdn.example.com/avatar.webp", got.AvatarURL)
	require.Contains(t, got.AccountBindings, "email")
	require.Contains(t, got.AccountBindings, "linuxdo")
	require.Contains(t, got.AccountBindings, "wechat")
	require.True(t, got.AccountBindings["email"].Bound)
	require.True(t, got.AccountBindings["linuxdo"].Bound)
	require.Equal(t, "LinuxDo Alice", got.AccountBindings["linuxdo"].Value)
	require.False(t, got.AccountBindings["wechat"].Bound)
	require.Equal(t, "/api/v1/auth/oauth/wechat/start?intent=bind&redirect=%2Fprofile%3Foauth_intent%3Dbind", got.AccountBindings["wechat"].ConnectURL)
}
