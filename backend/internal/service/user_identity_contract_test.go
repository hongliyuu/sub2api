//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type identityDeleteRepoStub struct {
	mockUserRepo
	user        *User
	deleteCalls int
	deleteErr   error
}

func (s *identityDeleteRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

func (s *identityDeleteRepoStub) DeleteExternalIdentity(context.Context, int64, string) error {
	s.deleteCalls++
	return s.deleteErr
}

func TestUserService_DeleteExternalIdentity_RejectsLastOAuthLogin(t *testing.T) {
	repo := &identityDeleteRepoStub{
		user: &User{
			ID:           7,
			Email:        "linuxdo-abc" + LinuxDoConnectSyntheticEmailDomain,
			PasswordHash: "generated-but-unusable",
			ExternalIdentities: []UserExternalIdentity{
				{Provider: ExternalIdentityProviderLinuxDo, ProviderUserID: "linuxdo-abc"},
			},
		},
	}
	svc := NewUserService(repo, nil, nil, nil)

	err := svc.DeleteExternalIdentity(context.Background(), 7, ExternalIdentityProviderLinuxDo)

	require.ErrorIs(t, err, ErrLastLoginMethodRequired)
	require.Zero(t, repo.deleteCalls)
}

func TestUserService_DeleteExternalIdentity_AllowsLocalUserFallback(t *testing.T) {
	repo := &identityDeleteRepoStub{
		user: &User{
			ID:           9,
			Email:        "user@example.com",
			PasswordHash: "local-password-hash",
			ExternalIdentities: []UserExternalIdentity{
				{Provider: ExternalIdentityProviderLinuxDo, ProviderUserID: "linuxdo-abc"},
			},
		},
	}
	svc := NewUserService(repo, nil, nil, nil)

	err := svc.DeleteExternalIdentity(context.Background(), 9, ExternalIdentityProviderLinuxDo)

	require.NoError(t, err)
	require.Equal(t, 1, repo.deleteCalls)
}

func TestCanDisconnectExternalIdentity_AllowsWhenAnotherOAuthBindingExists(t *testing.T) {
	user := &User{
		Email:        "wechat-abc" + WeChatConnectSyntheticEmailDomain,
		PasswordHash: "generated-but-unusable",
		ExternalIdentities: []UserExternalIdentity{
			{Provider: ExternalIdentityProviderLinuxDo, ProviderUserID: "linuxdo-abc"},
			{Provider: ExternalIdentityProviderWeChat, ProviderUserID: "wechat-abc"},
		},
	}

	require.True(t, CanDisconnectExternalIdentity(user, ExternalIdentityProviderLinuxDo))
	require.True(t, CanDisconnectExternalIdentity(user, ExternalIdentityProviderWeChat))
}
