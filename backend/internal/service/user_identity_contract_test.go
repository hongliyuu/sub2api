//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildUserIdentityState_InfersLegacySyntheticProviders(t *testing.T) {
	t.Run("linuxdo synthetic email maps to linuxdo identity", func(t *testing.T) {
		state := BuildUserIdentityState(&User{
			Email:        "linuxdo-123" + LinuxDoConnectSyntheticEmailDomain,
			PasswordHash: "generated-but-unusable",
		}, nil)

		require.Len(t, state.ExternalIdentities, 1)
		require.Equal(t, ExternalIdentityProviderLinuxDo, state.ExternalIdentities[0].Provider)
		require.Equal(t, "123", state.ExternalIdentities[0].ProviderUserID)
		require.False(t, state.HasUsableLocalLogin())
	})

	t.Run("oidc synthetic email maps to oidc identity", func(t *testing.T) {
		state := BuildUserIdentityState(&User{
			Email:        "oidc-abcdef" + OIDCConnectSyntheticEmailDomain,
			PasswordHash: "generated-but-unusable",
		}, nil)

		require.Len(t, state.ExternalIdentities, 1)
		require.Equal(t, ExternalIdentityProviderOIDC, state.ExternalIdentities[0].Provider)
		require.Equal(t, "abcdef", state.ExternalIdentities[0].ProviderUserID)
		require.False(t, state.HasUsableLocalLogin())
	})

	t.Run("wechat synthetic email maps to wechat identity", func(t *testing.T) {
		state := BuildUserIdentityState(&User{
			Email:        "wechat-union-1" + WeChatConnectSyntheticEmailDomain,
			PasswordHash: "generated-but-unusable",
		}, nil)

		require.Len(t, state.ExternalIdentities, 1)
		require.Equal(t, ExternalIdentityProviderWeChat, state.ExternalIdentities[0].Provider)
		require.Equal(t, "union-1", state.ExternalIdentities[0].ProviderUserID)
		require.False(t, state.HasUsableLocalLogin())
	})
}

func TestCanDisconnectExternalIdentity_RejectsLastUsableLoginMethodWhenProviderDisabled(t *testing.T) {
	state := BuildUserIdentityState(&User{
		Email:        "wechat-abc" + WeChatConnectSyntheticEmailDomain,
		PasswordHash: "generated-but-unusable",
	}, nil)

	require.False(t, CanDisconnectExternalIdentity(state, ExternalIdentityProviderWeChat, map[ExternalIdentityProvider]ProviderAvailability{
		ExternalIdentityProviderWeChat: {Enabled: false, ConfigValid: true},
	}))
}

func TestCanDisconnectExternalIdentity_AllowsDisconnectWithUsableLocalPasswordLogin(t *testing.T) {
	state := BuildUserIdentityState(&User{
		Email:        "user@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuv",
	}, []UserExternalIdentity{
		{Provider: ExternalIdentityProviderLinuxDo, ProviderUserID: "linuxdo-1"},
	})

	require.True(t, CanDisconnectExternalIdentity(state, ExternalIdentityProviderLinuxDo, map[ExternalIdentityProvider]ProviderAvailability{
		ExternalIdentityProviderLinuxDo: {Enabled: true, ConfigValid: true},
	}))
}

func TestCanDisconnectExternalIdentity_AllowsDisconnectWithAnotherUsableProvider(t *testing.T) {
	state := BuildUserIdentityState(&User{
		Email:        "wechat-union-1" + WeChatConnectSyntheticEmailDomain,
		PasswordHash: "generated-but-unusable",
	}, []UserExternalIdentity{
		{Provider: ExternalIdentityProviderWeChat, ProviderUserID: "union-1"},
		{Provider: ExternalIdentityProviderOIDC, ProviderUserID: "oidc-1"},
	})

	require.True(t, CanDisconnectExternalIdentity(state, ExternalIdentityProviderWeChat, map[ExternalIdentityProvider]ProviderAvailability{
		ExternalIdentityProviderWeChat: {Enabled: true, ConfigValid: true},
		ExternalIdentityProviderOIDC:   {Enabled: true, ConfigValid: true},
	}))
	require.False(t, CanDisconnectExternalIdentity(state, ExternalIdentityProviderWeChat, map[ExternalIdentityProvider]ProviderAvailability{
		ExternalIdentityProviderWeChat: {Enabled: true, ConfigValid: true},
		ExternalIdentityProviderOIDC:   {Enabled: true, ConfigValid: false},
	}))
}

func TestBuildUserIdentityState_NormalizesExplicitIdentityProviders(t *testing.T) {
	state := BuildUserIdentityState(&User{
		Email:        "user@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuv",
	}, []UserExternalIdentity{
		{Provider: ExternalIdentityProvider(" OIDC "), ProviderUserID: "  sub-oidc  "},
		{Provider: ExternalIdentityProvider("WeChat"), ProviderUserID: " union-1 "},
	})

	require.Len(t, state.ExternalIdentities, 2)
	require.Equal(t, ExternalIdentityProviderOIDC, state.ExternalIdentities[0].Provider)
	require.Equal(t, "sub-oidc", state.ExternalIdentities[0].ProviderUserID)
	require.Equal(t, ExternalIdentityProviderWeChat, state.ExternalIdentities[1].Provider)
	require.Equal(t, "union-1", state.ExternalIdentities[1].ProviderUserID)
}
