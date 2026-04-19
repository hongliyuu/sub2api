//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type signupSourceTrackingRepoStub struct {
	*pendingAuthUserRepoStub
	calls []signupSourceUpdateCall
}

type signupSourceUpdateCall struct {
	UserID       int64
	SignupSource string
}

func newSignupSourceTrackingRepoStub() *signupSourceTrackingRepoStub {
	return &signupSourceTrackingRepoStub{
		pendingAuthUserRepoStub: newPendingAuthUserRepoStub(),
	}
}

func (s *signupSourceTrackingRepoStub) UpdateSignupSource(_ context.Context, userID int64, signupSource string) error {
	s.calls = append(s.calls, signupSourceUpdateCall{
		UserID:       userID,
		SignupSource: signupSource,
	})
	return nil
}

func TestCreateAccountFromPendingAuthSession_RequiresVerifiedEmailAndPassword(t *testing.T) {
	svc, _ := newAuthServiceForPendingSessionTest(t)

	_, _, err := svc.CreateAccountFromPendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
	}, "owner@example.com", "", "")
	require.ErrorIs(t, err, ErrEmailVerifyRequired)

	_, _, err = svc.CreateAccountFromPendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
	}, "owner@example.com", "123456", "")
	require.ErrorIs(t, err, ErrPasswordRequired)
}

func TestBindEmailIdentity_RequiresVerifyCodeAndPassword(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.users[9] = &User{ID: 9, Email: "oauth-only@example.com", Role: RoleUser, Status: StatusActive}

	_, err := svc.BindEmailIdentity(context.Background(), 9, "owner@example.com", "", "new-password")
	require.ErrorIs(t, err, ErrEmailVerifyRequired)

	_, err = svc.BindEmailIdentity(context.Background(), 9, "owner@example.com", "123456", "")
	require.ErrorIs(t, err, ErrPasswordRequired)
}

func TestUserService_UpdateSignupSource_NormalizesWeChatSource(t *testing.T) {
	repo := newSignupSourceTrackingRepoStub()
	svc := NewUserService(repo, nil, nil, nil)

	err := svc.UpdateSignupSource(context.Background(), 7, " WeChat ")
	require.NoError(t, err)
	require.Len(t, repo.calls, 1)
	require.Equal(t, int64(7), repo.calls[0].UserID)
	require.Equal(t, SignupSourceWeChat, repo.calls[0].SignupSource)
}

func TestUserService_UpdateSignupSource_IsNoOpWithoutCompatibilityWriter(t *testing.T) {
	svc := NewUserService(newPendingAuthUserRepoStub(), nil, nil, nil)

	err := svc.UpdateSignupSource(context.Background(), 7, SignupSourceWeChat)
	require.NoError(t, err)
}

func TestCreateAccountFromPendingAuthSession_AppliesProviderScopedDefaultsAndSignupSource(t *testing.T) {
	repo := newSignupSourceTrackingRepoStub()
	assigner := &defaultSubscriptionAssignerStub{}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "pending-auth-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
		Default: config.DefaultConfig{
			UserBalance:     1,
			UserConcurrency: 1,
		},
	}
	settingSvc := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyRegistrationEnabled:         "true",
		SettingKeyDefaultBalanceLinuxDo:       "21.5",
		SettingKeyDefaultConcurrencyLinuxDo:   "9",
		SettingKeyDefaultSubscriptionsLinuxDo: `[{"group_id":21,"validity_days":60}]`,
	}}, cfg)

	svc := NewAuthService(
		nil,
		repo,
		nil,
		pendingAuthRefreshTokenCacheStub{},
		cfg,
		settingSvc,
		nil,
		nil,
		nil,
		nil,
		assigner,
	)

	_, user, err := svc.CreateAccountFromPendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "linuxdo",
		ProviderKey:     "linuxdo-main",
		ProviderSubject: "linuxdo-user-1",
	}, "owner@example.com", "123456", "password-123")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, 21.5, user.Balance)
	require.Equal(t, 9, user.Concurrency)
	require.Equal(t, SignupSourceLinuxDo, user.SignupSource)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(21), assigner.calls[0].GroupID)
	require.Equal(t, 60, assigner.calls[0].ValidityDays)
	require.Len(t, repo.calls, 1)
	require.Equal(t, user.ID, repo.calls[0].UserID)
	require.Equal(t, SignupSourceLinuxDo, repo.calls[0].SignupSource)
}
