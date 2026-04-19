//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type pendingAuthAvatarRepoStub struct {
	*pendingAuthUserRepoStub
	avatarUpserts []pendingAuthAvatarUpsertCall
}

type pendingAuthAvatarUpsertCall struct {
	UserID int64
	Input  UpsertUserAvatarInput
}

type bindDefaultsBalanceUpdateCall struct {
	UserID int64
	Delta  float64
}

type bindDefaultsConcurrencyUpdateCall struct {
	UserID int64
	Delta  int
}

type pendingAuthBindDefaultsRepoStub struct {
	*pendingAuthUserRepoStub
	bindDefaultGrants  map[string]struct{}
	balanceUpdates     []bindDefaultsBalanceUpdateCall
	concurrencyUpdates []bindDefaultsConcurrencyUpdateCall
	deductBalanceCalls []bindDefaultsBalanceUpdateCall
	balanceErr         error
	concurrencyErr     error
}

func newPendingAuthBindDefaultsRepoStub() *pendingAuthBindDefaultsRepoStub {
	return &pendingAuthBindDefaultsRepoStub{
		pendingAuthUserRepoStub: newPendingAuthUserRepoStub(),
		bindDefaultGrants:       make(map[string]struct{}),
	}
}

func (s *pendingAuthBindDefaultsRepoStub) UpdateBalance(ctx context.Context, userID int64, amount float64) error {
	return s.updateBalance(ctx, userID, amount, true)
}

func (s *pendingAuthBindDefaultsRepoStub) CreditBalanceWithoutRechargeTracking(ctx context.Context, userID int64, amount float64) error {
	return s.updateBalance(ctx, userID, amount, false)
}

func (s *pendingAuthBindDefaultsRepoStub) updateBalance(ctx context.Context, userID int64, amount float64, trackRecharge bool) error {
	if s.balanceErr != nil {
		return s.balanceErr
	}
	user, ok := s.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.Balance += amount
	if trackRecharge && ShouldTrackTotalRecharged(ctx) {
		user.TotalRecharged += amount
	}
	s.balanceUpdates = append(s.balanceUpdates, bindDefaultsBalanceUpdateCall{
		UserID: userID,
		Delta:  amount,
	})
	return nil
}

func (s *pendingAuthBindDefaultsRepoStub) DeductBalance(_ context.Context, userID int64, amount float64) error {
	user, ok := s.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.Balance -= amount
	s.deductBalanceCalls = append(s.deductBalanceCalls, bindDefaultsBalanceUpdateCall{
		UserID: userID,
		Delta:  amount,
	})
	return nil
}

func (s *pendingAuthBindDefaultsRepoStub) UpdateConcurrency(_ context.Context, userID int64, amount int) error {
	if s.concurrencyErr != nil {
		return s.concurrencyErr
	}
	user, ok := s.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.Concurrency += amount
	s.concurrencyUpdates = append(s.concurrencyUpdates, bindDefaultsConcurrencyUpdateCall{
		UserID: userID,
		Delta:  amount,
	})
	return nil
}

func (s *pendingAuthBindDefaultsRepoStub) TryCreateProviderDefaultBindGrant(_ context.Context, userID int64, providerType string) (bool, error) {
	key := strconv.FormatInt(userID, 10) + ":" + providerType
	if _, exists := s.bindDefaultGrants[key]; exists {
		return false, nil
	}
	s.bindDefaultGrants[key] = struct{}{}
	return true, nil
}

func (s *pendingAuthBindDefaultsRepoStub) DeleteProviderDefaultBindGrant(_ context.Context, userID int64, providerType string) error {
	delete(s.bindDefaultGrants, strconv.FormatInt(userID, 10)+":"+providerType)
	return nil
}

func (s *pendingAuthAvatarRepoStub) UpsertAvatar(_ context.Context, userID int64, input UpsertUserAvatarInput) (*UserAvatar, error) {
	s.avatarUpserts = append(s.avatarUpserts, pendingAuthAvatarUpsertCall{
		UserID: userID,
		Input:  input,
	})
	return &UserAvatar{
		ID:              int64(len(s.avatarUpserts)),
		UserID:          userID,
		StorageProvider: input.StorageProvider,
		StorageKey:      input.StorageKey,
		URL:             input.URL,
		ContentType:     input.ContentType,
		ByteSize:        input.ByteSize,
		SHA256:          input.SHA256,
	}, nil
}

func newAuthServiceForPendingSessionRepo(repo UserRepository) *AuthService {
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

	settings := NewSettingService(&pendingAuthSettingRepoStub{values: map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}}, cfg)

	return NewAuthService(
		nil,
		repo,
		nil,
		pendingAuthRefreshTokenCacheStub{},
		cfg,
		settings,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func TestCompletePendingAuthSessionBind_AdoptExistingRequiresVerifiedEmail(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.users[7] = &User{
		ID:           7,
		Email:        "owner@example.com",
		PasswordHash: "$2a$10$2b2oGqGKlMImZ/Sc7VcRb.rBeYjf4eDqRLVQjaxAg/P070nxsVXni",
		Role:         RoleUser,
		Status:       StatusActive,
	}
	repo.usersByMail["owner@example.com"] = repo.users[7]

	targetUserID := int64(7)
	sessionToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentAdoptExistingUserByEmail,
		ProviderType:    "oidc",
		ProviderKey:     "issuer",
		ProviderSubject: "sub-1",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.MarkPendingAuthSessionPasswordVerified(context.Background(), sessionToken, 7)
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), sessionToken, 7)
	require.ErrorIs(t, err, ErrPendingAuthVerificationRequired)
	require.Empty(t, repo.bindCalls)
}

func TestRegisterAndBindEmailIdentity_RejectSyntheticEmails(t *testing.T) {
	testCases := []struct {
		name  string
		email string
	}{
		{name: "linuxdo", email: "linuxdo-user-1" + LinuxDoConnectSyntheticEmailDomain},
		{name: "oidc", email: "oidc-user-1" + OIDCConnectSyntheticEmailDomain},
		{name: "wechat", email: "wechat-user-1" + WeChatConnectSyntheticEmailDomain},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"/register", func(t *testing.T) {
			svc, _ := newAuthServiceForPendingSessionTest(t)
			_, _, err := svc.RegisterWithVerification(context.Background(), tc.email, "password-123", "", "", "")
			require.ErrorIs(t, err, ErrEmailReserved)
		})

		t.Run(tc.name+"/bind", func(t *testing.T) {
			svc, repo := newAuthServiceForPendingSessionTest(t)
			repo.users[9] = &User{
				ID:     9,
				Email:  "oauth-only@example.com",
				Role:   RoleUser,
				Status: StatusActive,
			}
			repo.usersByMail["oauth-only@example.com"] = repo.users[9]
			_, err := svc.BindEmailIdentity(context.Background(), 9, tc.email, "123456", "password-123")
			require.ErrorIs(t, err, ErrEmailReserved)
		})
	}
}

func TestCompletePendingAuthSessionBind_AppliesAdoptionDecisionAcrossProviders(t *testing.T) {
	testCases := []struct {
		name            string
		providerType    string
		providerKey     string
		providerSubject string
	}{
		{name: "linuxdo", providerType: "linuxdo", providerKey: "linuxdo", providerSubject: "linuxdo-user-1"},
		{name: "wechat", providerType: "wechat", providerKey: "wechat-main", providerSubject: "union-1"},
		{name: "oidc", providerType: "oidc", providerKey: "https://issuer.example.com", providerSubject: "sub-1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &pendingAuthAvatarRepoStub{pendingAuthUserRepoStub: newPendingAuthUserRepoStub()}
			repo.users[7] = &User{
				ID:       7,
				Email:    "owner@example.com",
				Username: "legacy-name",
				Role:     RoleUser,
				Status:   StatusActive,
			}
			repo.usersByMail["owner@example.com"] = repo.users[7]
			svc := newAuthServiceForPendingSessionRepo(repo)

			avatarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write([]byte("png"))
			}))
			defer avatarServer.Close()

			targetUserID := int64(7)
			token, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
				Intent:          PendingAuthIntentBindCurrentUser,
				ProviderType:    tc.providerType,
				ProviderKey:     tc.providerKey,
				ProviderSubject: tc.providerSubject,
				TargetUserID:    &targetUserID,
				Metadata: map[string]any{
					"adopt_display_name":     true,
					"adopt_avatar":           true,
					"suggested_display_name": tc.name + "-user",
					"suggested_avatar_url":   avatarServer.URL + "/avatar.png",
				},
			})
			require.NoError(t, err)

			completion, err := svc.CompletePendingAuthSessionBind(context.Background(), token, 7)
			require.NoError(t, err)
			require.Equal(t, int64(7), completion.UserID)
			require.Len(t, repo.bindCalls, 1)
			require.Len(t, repo.avatarUpserts, 1)
			require.Equal(t, int64(7), repo.avatarUpserts[0].UserID)
			require.Equal(t, UserAvatarStorageProviderInline, repo.avatarUpserts[0].Input.StorageProvider)
			require.Equal(t, "image/png", repo.avatarUpserts[0].Input.ContentType)

			decision, err := svc.GetIdentityAdoptionDecision(context.Background(), 7, IdentityAdoptionDecisionRef{
				ProviderType:    tc.providerType,
				ProviderKey:     tc.providerKey,
				ProviderSubject: tc.providerSubject,
			})
			require.NoError(t, err)
			require.True(t, decision.AdoptDisplayName)
			require.True(t, decision.AdoptAvatar)

			user, err := repo.GetByID(context.Background(), 7)
			require.NoError(t, err)
			require.Equal(t, tc.name+"-user", user.Username)
		})
	}
}

func TestCompletePendingAuthSessionBind_AppliesProviderDefaultsOnceWhenEnabled(t *testing.T) {
	repo := newPendingAuthBindDefaultsRepoStub()
	repo.users[7] = &User{
		ID:          7,
		Email:       "owner@example.com",
		Role:        RoleUser,
		Status:      StatusActive,
		Balance:     10,
		Concurrency: 2,
	}
	repo.usersByMail["owner@example.com"] = repo.users[7]

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
	settings := NewSettingService(&pendingAuthSettingRepoStub{values: map[string]string{
		SettingKeyRegistrationEnabled:        "true",
		"default_apply_on_bind_wechat":       "true",
		SettingKeyDefaultBalanceWeChat:       "31.5",
		SettingKeyDefaultConcurrencyWeChat:   "5",
		SettingKeyDefaultSubscriptionsWeChat: `[{"group_id":31,"validity_days":90}]`,
	}}, cfg)

	svc := NewAuthService(
		nil,
		repo,
		nil,
		pendingAuthRefreshTokenCacheStub{},
		cfg,
		settings,
		nil,
		nil,
		nil,
		nil,
		assigner,
	)

	targetUserID := int64(7)
	firstToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	completion, err := svc.CompletePendingAuthSessionBind(context.Background(), firstToken, 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), completion.UserID)
	require.Len(t, repo.bindCalls, 1)
	require.Len(t, repo.balanceUpdates, 1)
	require.Equal(t, 31.5, repo.balanceUpdates[0].Delta)
	require.Len(t, repo.concurrencyUpdates, 1)
	require.Equal(t, 5, repo.concurrencyUpdates[0].Delta)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(31), assigner.calls[0].GroupID)
	require.Equal(t, 90, assigner.calls[0].ValidityDays)

	user, err := repo.GetByID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 41.5, user.Balance)
	require.Equal(t, 7, user.Concurrency)
	require.Len(t, repo.bindDefaultGrants, 1)

	secondToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-2",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), secondToken, 7)
	require.NoError(t, err)
	require.Len(t, repo.bindCalls, 2)
	require.Len(t, repo.balanceUpdates, 1)
	require.Len(t, repo.concurrencyUpdates, 1)
	require.Len(t, assigner.calls, 1)

	user, err = repo.GetByID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 41.5, user.Balance)
	require.Equal(t, 7, user.Concurrency)
	require.Zero(t, user.TotalRecharged)
}

func TestCompletePendingAuthSessionBind_RollsBackProviderDefaultsWhenConcurrencyUpdateFails(t *testing.T) {
	repo := newPendingAuthBindDefaultsRepoStub()
	repo.users[7] = &User{
		ID:             7,
		Email:          "owner@example.com",
		Role:           RoleUser,
		Status:         StatusActive,
		Balance:        10,
		Concurrency:    2,
		TotalRecharged: 0,
	}
	repo.usersByMail["owner@example.com"] = repo.users[7]
	repo.concurrencyErr = errors.New("update concurrency failed")

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
	settings := NewSettingService(&pendingAuthSettingRepoStub{values: map[string]string{
		SettingKeyRegistrationEnabled:      "true",
		SettingKeyDefaultApplyOnBindWeChat: "true",
		SettingKeyDefaultBalanceWeChat:     "31.5",
		SettingKeyDefaultConcurrencyWeChat: "5",
	}}, cfg)

	svc := NewAuthService(nil, repo, nil, pendingAuthRefreshTokenCacheStub{}, cfg, settings, nil, nil, nil, nil, assigner)

	targetUserID := int64(7)
	firstToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-rollback-1",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), firstToken, 7)
	require.ErrorContains(t, err, "update concurrency failed")

	user, getErr := repo.GetByID(context.Background(), 7)
	require.NoError(t, getErr)
	require.Equal(t, 10.0, user.Balance)
	require.Equal(t, 2, user.Concurrency)
	require.Zero(t, user.TotalRecharged)
	require.Len(t, repo.balanceUpdates, 1)
	require.Len(t, repo.deductBalanceCalls, 1)
	require.Empty(t, repo.bindDefaultGrants)
	require.Empty(t, assigner.calls)

	repo.concurrencyErr = nil
	secondToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-rollback-2",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), secondToken, 7)
	require.NoError(t, err)

	user, getErr = repo.GetByID(context.Background(), 7)
	require.NoError(t, getErr)
	require.Equal(t, 41.5, user.Balance)
	require.Equal(t, 7, user.Concurrency)
	require.Zero(t, user.TotalRecharged)
	require.Len(t, repo.bindDefaultGrants, 1)
}

func TestCompletePendingAuthSessionBind_RollsBackProviderDefaultsWhenSubscriptionAssignFails(t *testing.T) {
	repo := newPendingAuthBindDefaultsRepoStub()
	repo.users[7] = &User{
		ID:             7,
		Email:          "owner@example.com",
		Role:           RoleUser,
		Status:         StatusActive,
		Balance:        10,
		Concurrency:    2,
		TotalRecharged: 0,
	}
	repo.usersByMail["owner@example.com"] = repo.users[7]

	assigner := &defaultSubscriptionAssignerStub{err: errors.New("assign subscription failed")}
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
	settings := NewSettingService(&pendingAuthSettingRepoStub{values: map[string]string{
		SettingKeyRegistrationEnabled:        "true",
		SettingKeyDefaultApplyOnBindWeChat:   "true",
		SettingKeyDefaultBalanceWeChat:       "31.5",
		SettingKeyDefaultConcurrencyWeChat:   "5",
		SettingKeyDefaultSubscriptionsWeChat: `[{"group_id":31,"validity_days":90}]`,
	}}, cfg)

	svc := NewAuthService(nil, repo, nil, pendingAuthRefreshTokenCacheStub{}, cfg, settings, nil, nil, nil, nil, assigner)

	targetUserID := int64(7)
	firstToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-sub-fail-1",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), firstToken, 7)
	require.ErrorContains(t, err, "assign subscription failed")

	user, getErr := repo.GetByID(context.Background(), 7)
	require.NoError(t, getErr)
	require.Equal(t, 10.0, user.Balance)
	require.Equal(t, 2, user.Concurrency)
	require.Zero(t, user.TotalRecharged)
	require.Len(t, repo.balanceUpdates, 1)
	require.Len(t, repo.deductBalanceCalls, 1)
	require.Len(t, repo.concurrencyUpdates, 2)
	require.Empty(t, repo.bindDefaultGrants)

	assigner.err = nil
	secondToken, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-sub-fail-2",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), secondToken, 7)
	require.NoError(t, err)

	user, getErr = repo.GetByID(context.Background(), 7)
	require.NoError(t, getErr)
	require.Equal(t, 41.5, user.Balance)
	require.Equal(t, 7, user.Concurrency)
	require.Zero(t, user.TotalRecharged)
	require.Len(t, repo.bindDefaultGrants, 1)
	require.Len(t, assigner.calls, 2)
}
