//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type pendingAuthSettingRepoStub struct {
	values map[string]string
	err    error
}

func (s *pendingAuthSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *pendingAuthSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *pendingAuthSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *pendingAuthSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *pendingAuthSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *pendingAuthSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *pendingAuthSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type pendingAuthSessionStoreStub struct {
	sessions          map[string]*PendingAuthSessionRecord
	nextID            int
	bindCalls         []pendingAuthBindCall
	adoptionDecisions map[string]*IdentityAdoptionDecision
	createUser        *User
	bindErr           error
}

type pendingAuthBindCall struct {
	SessionID string
	UserID    int64
}

func newPendingAuthSessionStoreStub() *pendingAuthSessionStoreStub {
	return &pendingAuthSessionStoreStub{
		sessions:          make(map[string]*PendingAuthSessionRecord),
		adoptionDecisions: make(map[string]*IdentityAdoptionDecision),
	}
}

func (s *pendingAuthSessionStoreStub) CreatePendingAuthSession(_ context.Context, input PendingAuthSessionInput) (*PendingAuthSessionRecord, error) {
	s.nextID++
	session := &PendingAuthSessionRecord{
		ID:              string(rune('a' + s.nextID - 1)),
		Intent:          input.Intent,
		ProviderType:    input.ProviderType,
		ProviderKey:     input.ProviderKey,
		ProviderSubject: input.ProviderSubject,
		Metadata:        cloneStringAnyMap(input.Metadata),
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		RedirectTo:      input.RedirectTo,
	}
	if input.TargetUserID != nil {
		v := *input.TargetUserID
		session.TargetUserID = &v
	}
	s.sessions[session.ID] = clonePendingAuthSessionRecord(session)
	return clonePendingAuthSessionRecord(session), nil
}

func (s *pendingAuthSessionStoreStub) GetPendingAuthSessionByID(_ context.Context, sessionID string) (*PendingAuthSessionRecord, error) {
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return clonePendingAuthSessionRecord(session), nil
}

func (s *pendingAuthSessionStoreStub) UpdatePendingAuthSession(_ context.Context, session *PendingAuthSessionRecord) error {
	if session == nil {
		return errors.New("nil session")
	}
	s.sessions[session.ID] = clonePendingAuthSessionRecord(session)
	return nil
}

func (s *pendingAuthSessionStoreStub) BindPendingAuthIdentity(_ context.Context, session *PendingAuthSessionRecord, userID int64) error {
	s.bindCalls = append(s.bindCalls, pendingAuthBindCall{
		SessionID: session.ID,
		UserID:    userID,
	})
	if s.bindErr != nil {
		return s.bindErr
	}
	return nil
}

func (s *pendingAuthSessionStoreStub) GetIdentityAdoptionDecision(_ context.Context, userID int64, ref IdentityAdoptionDecisionRef) (*IdentityAdoptionDecision, error) {
	decision, ok := s.adoptionDecisions[pendingAuthDecisionKey(userID, ref)]
	if !ok {
		return nil, ErrUserNotFound
	}
	copied := *decision
	return &copied, nil
}

func (s *pendingAuthSessionStoreStub) UpsertIdentityAdoptionDecision(_ context.Context, userID int64, input UpsertIdentityAdoptionDecisionInput) (*IdentityAdoptionDecision, error) {
	key := pendingAuthDecisionKey(userID, input.Ref)
	decision, ok := s.adoptionDecisions[key]
	if !ok {
		decision = &IdentityAdoptionDecision{
			Ref:       input.Ref,
			CreatedAt: time.Now(),
		}
	}
	if input.AdoptDisplayName != nil {
		decision.AdoptDisplayName = *input.AdoptDisplayName
	}
	if input.AdoptAvatar != nil {
		decision.AdoptAvatar = *input.AdoptAvatar
	}
	decision.DecidedAt = time.Now()
	decision.UpdatedAt = decision.DecidedAt
	s.adoptionDecisions[key] = decision

	copied := *decision
	return &copied, nil
}

type pendingAuthUserRepoStub struct {
	*pendingAuthSessionStoreStub
	users       map[int64]*User
	usersByMail map[string]*User
	nextUserID  int64
	createCount int
	deleteCount int
}

func newPendingAuthUserRepoStub() *pendingAuthUserRepoStub {
	return &pendingAuthUserRepoStub{
		pendingAuthSessionStoreStub: newPendingAuthSessionStoreStub(),
		users:                       make(map[int64]*User),
		usersByMail:                 make(map[string]*User),
		nextUserID:                  100,
	}
}

func (s *pendingAuthUserRepoStub) Create(_ context.Context, user *User) error {
	s.createCount++
	s.nextUserID++
	if user.ID == 0 {
		user.ID = s.nextUserID
	}
	copied := *user
	s.users[user.ID] = &copied
	s.usersByMail[user.Email] = &copied
	return nil
}

func (s *pendingAuthUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	copied := *user
	return &copied, nil
}

func (s *pendingAuthUserRepoStub) GetByEmail(_ context.Context, email string) (*User, error) {
	user, ok := s.usersByMail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	copied := *user
	return &copied, nil
}

func (s *pendingAuthUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}

func (s *pendingAuthUserRepoStub) Update(_ context.Context, user *User) error {
	copied := *user
	s.users[user.ID] = &copied
	s.usersByMail[user.Email] = &copied
	return nil
}

func (s *pendingAuthUserRepoStub) Delete(_ context.Context, id int64) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	delete(s.users, id)
	delete(s.usersByMail, user.Email)
	s.deleteCount++
	return nil
}

func (s *pendingAuthUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *pendingAuthUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *pendingAuthUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *pendingAuthUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}

func (s *pendingAuthUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *pendingAuthUserRepoStub) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := s.usersByMail[email]
	return ok, nil
}

func (s *pendingAuthUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *pendingAuthUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *pendingAuthUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *pendingAuthUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *pendingAuthUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}

func (s *pendingAuthUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

type pendingAuthRefreshTokenCacheStub struct{}

func (pendingAuthRefreshTokenCacheStub) StoreRefreshToken(context.Context, string, *RefreshTokenData, time.Duration) error {
	return nil
}

func (pendingAuthRefreshTokenCacheStub) GetRefreshToken(context.Context, string) (*RefreshTokenData, error) {
	return nil, ErrRefreshTokenInvalid
}

func (pendingAuthRefreshTokenCacheStub) DeleteRefreshToken(context.Context, string) error {
	return nil
}

func (pendingAuthRefreshTokenCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}

func (pendingAuthRefreshTokenCacheStub) DeleteTokenFamily(context.Context, string) error {
	return nil
}

func (pendingAuthRefreshTokenCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}

func (pendingAuthRefreshTokenCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}

func (pendingAuthRefreshTokenCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (pendingAuthRefreshTokenCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}

func (pendingAuthRefreshTokenCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return true, nil
}

func newAuthServiceForPendingSessionTest(t *testing.T) (*AuthService, *pendingAuthUserRepoStub) {
	t.Helper()

	repo := newPendingAuthUserRepoStub()
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
		nil,
	)

	return svc, repo
}

func pendingAuthDecisionKey(userID int64, ref IdentityAdoptionDecisionRef) string {
	return fmt.Sprintf("%d|%s|%s|%s", userID, ref.ProviderType, ref.ProviderKey, ref.ProviderSubject)
}

func mustCreatePendingAuthSession(t *testing.T, svc *AuthService, intent, providerType, providerKey, providerSubject string) *PendingAuthSessionRecord {
	t.Helper()

	token, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          intent,
		ProviderType:    providerType,
		ProviderKey:     providerKey,
		ProviderSubject: providerSubject,
	})
	require.NoError(t, err)

	session, err := svc.GetPendingAuthSessionForProgress(context.Background(), token, nil)
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, token, session.Token)

	return session
}

func mustCreateExistingEmailUser(t *testing.T, repo *pendingAuthUserRepoStub, email string) *User {
	t.Helper()

	user := &User{
		ID:           7,
		Email:        email,
		PasswordHash: "$2a$10$2b2oGqGKlMImZ/Sc7VcRb.rBeYjf4eDqRLVQjaxAg/P070nxsVXni",
		Role:         RoleUser,
		Status:       StatusActive,
	}
	repo.users[user.ID] = user
	repo.usersByMail[email] = user
	return user
}

func mustCreateAdoptExistingSession(t *testing.T, svc *AuthService, repo *pendingAuthUserRepoStub, email string) *PendingAuthSessionRecord {
	t.Helper()
	existing := mustCreateExistingEmailUser(t, repo, email)
	result, err := svc.ResolvePendingAuthSessionEmail(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "oidc",
		ProviderKey:     "issuer",
		ProviderSubject: "sub-1",
	}, email, "123456", "")
	require.NoError(t, err)
	require.Equal(t, PendingAuthIntentAdoptExistingUserByEmail, result.Intent)
	require.NotNil(t, result.TargetUserID)
	require.Equal(t, existing.ID, *result.TargetUserID)

	session, err := svc.GetPendingAuthSessionForProgress(context.Background(), result.PendingAuthToken, nil)
	require.NoError(t, err)
	return session
}

func TestCreatePendingAuthSession_PreservesRedirectAndMetadataForProgress(t *testing.T) {
	svc, _ := newAuthServiceForPendingSessionTest(t)

	token, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
		RedirectTo:      "/settings/profile",
		Metadata: map[string]any{
			"channel":                "mp",
			"suggested_display_name": "wechat-user",
		},
	})
	require.NoError(t, err)

	session, err := svc.GetPendingAuthSessionForProgress(context.Background(), token, nil)
	require.NoError(t, err)
	require.Equal(t, "/settings/profile", session.RedirectTo)
	require.Equal(t, "mp", session.Metadata["channel"])
	require.Equal(t, "wechat-user", session.Metadata["suggested_display_name"])
	require.Nil(t, session.TargetUserID)
	require.Nil(t, session.EmailVerifiedAt)
	require.Nil(t, session.PasswordVerifiedAt)
	require.Nil(t, session.TOTPVerifiedAt)
	require.Nil(t, session.ConsumedAt)
}

func TestResolvePendingAuthSessionEmail_PersistsEmailVerificationAndPendingPasswordState(t *testing.T) {
	svc, _ := newAuthServiceForPendingSessionTest(t)

	result, err := svc.ResolvePendingAuthSessionEmail(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "oidc",
		ProviderKey:     "https://issuer.example.com",
		ProviderSubject: "sub-1",
		RedirectTo:      "/auth/callback",
		Metadata: map[string]any{
			"issuer": "https://issuer.example.com",
		},
	}, "Owner@Example.com", "123456", "password-123")
	require.NoError(t, err)
	require.Equal(t, PendingAuthIntentLogin, result.Intent)

	session, err := svc.GetPendingAuthSessionForProgress(context.Background(), result.PendingAuthToken, nil)
	require.NoError(t, err)
	require.Equal(t, "/auth/callback", session.RedirectTo)
	require.Equal(t, "https://issuer.example.com", session.Metadata["issuer"])
	require.Equal(t, "owner@example.com", session.ResolvedEmail)
	require.NotNil(t, session.EmailVerifiedAt)
	require.NotEmpty(t, session.PendingPasswordHash)
	require.NotEqual(t, "password-123", session.PendingPasswordHash)
	require.Nil(t, session.PasswordVerifiedAt)
	require.Nil(t, session.TOTPVerifiedAt)
	require.Nil(t, session.ConsumedAt)
}

func TestPendingAuthSession_TracksPasswordAndTotpVerificationBeforeConsume(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	passwordHash, err := svc.HashPassword("password-123")
	require.NoError(t, err)
	repo.users[7] = &User{
		ID:           7,
		Email:        "owner@example.com",
		PasswordHash: passwordHash,
		TotpEnabled:  true,
		Role:         RoleUser,
		Status:       StatusActive,
	}
	repo.usersByMail["owner@example.com"] = repo.users[7]

	result, err := svc.ResolvePendingAuthSessionEmail(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "oidc",
		ProviderKey:     "issuer",
		ProviderSubject: "sub-1",
		Metadata: map[string]any{
			"adopt_display_name": true,
		},
	}, "owner@example.com", "123456", "")
	require.NoError(t, err)
	require.Equal(t, PendingAuthIntentAdoptExistingUserByEmail, result.Intent)
	require.NotNil(t, result.TargetUserID)
	require.True(t, result.Requires2FA)

	session, err := svc.GetPendingAuthSessionForProgress(context.Background(), result.PendingAuthToken, nil)
	require.NoError(t, err)
	require.NotNil(t, session.EmailVerifiedAt)
	require.Nil(t, session.PasswordVerifiedAt)
	require.Nil(t, session.TOTPVerifiedAt)
	require.Nil(t, session.ConsumedAt)

	verifyResult, err := svc.VerifyPendingAuthBindPassword(context.Background(), result.PendingAuthToken, "password-123")
	require.NoError(t, err)
	require.Equal(t, result.PendingAuthToken, verifyResult.PendingAuthToken)
	require.True(t, verifyResult.Requires2FA)

	session, err = svc.GetPendingAuthSessionForProgress(context.Background(), result.PendingAuthToken, nil)
	require.NoError(t, err)
	require.NotNil(t, session.TargetUserID)
	require.Equal(t, int64(7), *session.TargetUserID)
	require.NotNil(t, session.PasswordVerifiedAt)
	require.Nil(t, session.TOTPVerifiedAt)
	require.Nil(t, session.ConsumedAt)

	_, err = svc.MarkPendingAuthSessionTOTPVerified(context.Background(), result.PendingAuthToken, 7)
	require.NoError(t, err)

	session, err = svc.GetPendingAuthSessionForProgress(context.Background(), result.PendingAuthToken, nil)
	require.NoError(t, err)
	require.NotNil(t, session.TOTPVerifiedAt)
	require.Nil(t, session.ConsumedAt)

	completion, err := svc.CompletePendingAuthSessionBind(context.Background(), result.PendingAuthToken, 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), completion.UserID)

	sessionID, err := svc.verifyPendingSessionToken(result.PendingAuthToken)
	require.NoError(t, err)
	stored := repo.sessions[sessionID]
	require.NotNil(t, stored)
	require.NotNil(t, stored.ConsumedAt)
}

func TestPendingAuthSession_SurvivesBindLoginUntilFinal2FACompletion(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.users[7] = &User{ID: 7, Email: "owner@example.com", Role: RoleUser, Status: StatusActive}
	session := mustCreatePendingAuthSession(t, svc, PendingAuthIntentLogin, "linuxdo", "linuxdo", "linuxdo-user-1")

	_, err := svc.MarkPendingAuthSessionPasswordVerified(context.Background(), session.Token, 7)
	require.NoError(t, err)

	completion, err := svc.CompletePendingAuthSessionBind(context.Background(), session.Token, 7)
	require.NoError(t, err)
	require.NotNil(t, completion)
	require.Equal(t, int64(7), completion.UserID)
}

func TestPendingAuthSession_RejectsReplayAfterFinalCompletion(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.users[7] = &User{ID: 7, Email: "owner@example.com", Role: RoleUser, Status: StatusActive}
	targetUserID := int64(7)

	token, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
		TargetUserID:    &targetUserID,
	})
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), token, 7)
	require.NoError(t, err)

	_, err = svc.CompletePendingAuthSessionBind(context.Background(), token, 7)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestCreateAccountFromPendingAuthSession_TransitionsToAdoptExistingWhenEmailAlreadyExists(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	mustCreateExistingEmailUser(t, repo, "owner@example.com")

	result, err := svc.ResolvePendingAuthSessionEmail(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "oidc",
		ProviderKey:     "https://issuer.example.com",
		ProviderSubject: "sub-1",
	}, "owner@example.com", "123456", "")

	require.NoError(t, err)
	require.Equal(t, PendingAuthIntentAdoptExistingUserByEmail, result.Intent)
	require.NotNil(t, result.TargetUserID)
	require.Equal(t, int64(7), *result.TargetUserID)
}

func TestCompletePendingAuthSessionBind_RequiresPasswordOr2FAForAdoptExisting(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	session := mustCreateAdoptExistingSession(t, svc, repo, "owner@example.com")

	_, err := svc.CompletePendingAuthSessionBind(context.Background(), session.Token, 7)
	require.Error(t, err)
}

func TestCompletePendingAuthSessionBind_PersistsAdoptionDecisionAndSuggestedDisplayName(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.users[7] = &User{
		ID:       7,
		Email:    "owner@example.com",
		Username: "legacy-name",
		Role:     RoleUser,
		Status:   StatusActive,
	}
	repo.usersByMail["owner@example.com"] = repo.users[7]

	token, err := svc.CreatePendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
		Metadata: map[string]any{
			"adopt_display_name":     true,
			"adopt_avatar":           false,
			"suggested_display_name": "wechat-user",
		},
	})
	require.NoError(t, err)

	completion, err := svc.CompletePendingAuthSessionBind(context.Background(), token, 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), completion.UserID)
	require.Len(t, repo.bindCalls, 1)

	decision, err := svc.GetIdentityAdoptionDecision(context.Background(), 7, IdentityAdoptionDecisionRef{
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
	})
	require.NoError(t, err)
	require.True(t, decision.AdoptDisplayName)
	require.False(t, decision.AdoptAvatar)

	user, err := repo.GetByID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "wechat-user", user.Username)

	_, err = svc.GetPendingAuthSessionForProgress(context.Background(), token, nil)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestCreateAccountFromPendingAuthSession_ReplayReturnsAdoptionFlowInsteadOfCreatingDuplicateUser(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)

	_, firstUser, err := svc.CreateAccountFromPendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
	}, "owner@example.com", "123456", "password-123")
	require.NoError(t, err)
	require.NotNil(t, firstUser)
	require.Equal(t, 1, repo.createCount)

	_, secondUser, err := svc.CreateAccountFromPendingAuthSession(context.Background(), PendingAuthSessionInput{
		Intent:          PendingAuthIntentLogin,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
	}, "owner@example.com", "123456", "password-123")
	require.ErrorIs(t, err, ErrPendingAuthVerificationRequired)
	require.Nil(t, secondUser)
	require.Equal(t, 1, repo.createCount)
}

func TestLoginOrRegisterSyntheticOAuthAndBindPendingSession_DeletesCreatedSyntheticUserWhenBindFails(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.bindErr = errors.New("bind failed")

	session := mustCreatePendingAuthSession(t, svc, PendingAuthIntentLogin, "oidc", "issuer", "sub-1")
	email := "oidc-sub-1" + OIDCConnectSyntheticEmailDomain

	tokenPair, user, err := svc.LoginOrRegisterSyntheticOAuthAndBindPendingSession(
		context.Background(),
		session.Token,
		email,
		"oidc-user",
		"",
	)
	require.Error(t, err)
	require.Nil(t, tokenPair)
	require.Nil(t, user)
	require.Equal(t, 1, repo.createCount)
	require.Equal(t, 1, repo.deleteCount)

	_, getErr := repo.GetByEmail(context.Background(), email)
	require.ErrorIs(t, getErr, ErrUserNotFound)
}

func TestLoginOrRegisterSyntheticOAuthAndBindPendingSession_DoesNotDeleteExistingSyntheticUserWhenBindFails(t *testing.T) {
	svc, repo := newAuthServiceForPendingSessionTest(t)
	repo.bindErr = errors.New("bind failed")

	email := "wechat-union-1" + WeChatConnectSyntheticEmailDomain
	existing := &User{
		ID:           7,
		Email:        email,
		Username:     "wechat-user",
		PasswordHash: "synthetic-password-hash",
		Role:         RoleUser,
		Status:       StatusActive,
	}
	repo.users[existing.ID] = existing
	repo.usersByMail[email] = existing

	session := mustCreatePendingAuthSession(t, svc, PendingAuthIntentLogin, "wechat", "wechat-main", "union-1")

	tokenPair, user, err := svc.LoginOrRegisterSyntheticOAuthAndBindPendingSession(
		context.Background(),
		session.Token,
		email,
		"wechat-user",
		"",
	)
	require.Error(t, err)
	require.Nil(t, tokenPair)
	require.Nil(t, user)
	require.Zero(t, repo.createCount)
	require.Zero(t, repo.deleteCount)

	stored, getErr := repo.GetByEmail(context.Background(), email)
	require.NoError(t, getErr)
	require.Equal(t, existing.ID, stored.ID)
}
