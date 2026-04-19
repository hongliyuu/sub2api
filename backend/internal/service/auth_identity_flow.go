package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrPendingAuthVerificationRequired = infraerrors.BadRequest("PENDING_AUTH_VERIFICATION_REQUIRED", "pending auth session requires additional verification")
	ErrPendingAuthTargetMismatch       = infraerrors.Forbidden("PENDING_AUTH_TARGET_MISMATCH", "pending auth session target mismatch")
	ErrPendingAuthUnavailable          = infraerrors.ServiceUnavailable("PENDING_AUTH_UNAVAILABLE", "pending auth session storage unavailable")
)

const pendingAuthSessionTTL = 10 * time.Minute

type pendingAuthSessionClaims struct {
	SessionID string `json:"sid"`
	Purpose   string `json:"purpose"`
	jwt.RegisteredClaims
}

func (s *AuthService) CreatePendingAuthSession(ctx context.Context, input PendingAuthSessionInput) (string, error) {
	store, err := s.pendingAuthStore()
	if err != nil {
		return "", err
	}

	session, err := store.CreatePendingAuthSession(ctx, input)
	if err != nil {
		return "", err
	}
	if session == nil {
		return "", ErrPendingAuthUnavailable
	}
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(pendingAuthSessionTTL)
	}

	token, err := s.signPendingSessionToken(session.ID, session.ExpiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) GetPendingAuthSessionForProgress(ctx context.Context, pendingToken string, expectedUserID *int64) (*PendingAuthSessionRecord, error) {
	sessionID, err := s.verifyPendingSessionToken(pendingToken)
	if err != nil {
		return nil, err
	}

	store, err := s.pendingAuthStore()
	if err != nil {
		return nil, err
	}

	session, err := store.GetPendingAuthSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if session == nil {
		return nil, ErrInvalidToken
	}
	if session.ConsumedAt != nil || time.Now().After(session.ExpiresAt) {
		return nil, ErrInvalidToken
	}
	if expectedUserID != nil && session.TargetUserID != nil && *session.TargetUserID != *expectedUserID {
		return nil, ErrPendingAuthTargetMismatch
	}
	session.Token = pendingToken
	return session, nil
}

func (s *AuthService) UpdatePendingAuthSessionAdoptionDecision(
	ctx context.Context,
	pendingToken string,
	adoptDisplayName bool,
	adoptAvatar bool,
) (*PendingAuthSessionRecord, error) {
	session, err := s.GetPendingAuthSessionForProgress(ctx, pendingToken, nil)
	if err != nil {
		return nil, err
	}
	if session.Metadata == nil {
		session.Metadata = make(map[string]any, 2)
	}
	session.Metadata["adopt_display_name"] = adoptDisplayName
	session.Metadata["adopt_avatar"] = adoptAvatar
	if err := s.updatePendingAuthSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *AuthService) GetIdentityAdoptionDecision(
	ctx context.Context,
	userID int64,
	ref IdentityAdoptionDecisionRef,
) (*IdentityAdoptionDecision, error) {
	store, ok := s.userRepo.(identityAdoptionDecisionStore)
	if !ok {
		return nil, ErrPendingAuthUnavailable
	}
	return store.GetIdentityAdoptionDecision(ctx, userID, ref)
}

func (s *AuthService) UpsertIdentityAdoptionDecision(
	ctx context.Context,
	userID int64,
	input UpsertIdentityAdoptionDecisionInput,
) (*IdentityAdoptionDecision, error) {
	store, ok := s.userRepo.(identityAdoptionDecisionStore)
	if !ok {
		return nil, ErrPendingAuthUnavailable
	}
	return store.UpsertIdentityAdoptionDecision(ctx, userID, input)
}

func (s *AuthService) ResolvePendingAuthSessionEmail(ctx context.Context, input PendingAuthSessionInput, email, verifyCode, password string) (*PendingAuthEmailResolutionResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if strings.TrimSpace(verifyCode) == "" {
		return nil, ErrEmailVerifyRequired
	}
	if err := s.verifyPendingAuthEmailCode(ctx, email, verifyCode); err != nil {
		return nil, err
	}

	sessionToken, err := s.CreatePendingAuthSession(ctx, input)
	if err != nil {
		return nil, err
	}
	session, err := s.GetPendingAuthSessionForProgress(ctx, sessionToken, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session.ResolvedEmail = email
	session.EmailVerifiedAt = &now

	existing, err := s.userRepo.GetByEmail(ctx, email)
	switch {
	case err == nil && existing != nil:
		targetUserID := existing.ID
		session.Intent = PendingAuthIntentAdoptExistingUserByEmail
		session.TargetUserID = &targetUserID
		if err := s.updatePendingAuthSession(ctx, session); err != nil {
			return nil, err
		}
		return &PendingAuthEmailResolutionResult{
			Intent:           session.Intent,
			TargetUserID:     &targetUserID,
			Requires2FA:      existing.TotpEnabled,
			PendingAuthToken: sessionToken,
		}, nil
	case err != nil && !errors.Is(err, ErrUserNotFound):
		return nil, err
	}

	if strings.TrimSpace(password) == "" {
		return nil, ErrPasswordRequired
	}
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	session.PendingPasswordHash = hashedPassword
	if err := s.updatePendingAuthSession(ctx, session); err != nil {
		return nil, err
	}

	return &PendingAuthEmailResolutionResult{
		Intent:           session.Intent,
		PendingAuthToken: sessionToken,
	}, nil
}

func (s *AuthService) CreateAccountFromPendingAuthSession(ctx context.Context, input PendingAuthSessionInput, email, verifyCode, password string) (*TokenPair, *User, error) {
	result, err := s.ResolvePendingAuthSessionEmail(ctx, input, email, verifyCode, password)
	if err != nil {
		return nil, nil, err
	}
	if result.Intent != PendingAuthIntentLogin {
		return nil, nil, ErrPendingAuthVerificationRequired
	}

	session, err := s.GetPendingAuthSessionForProgress(ctx, result.PendingAuthToken, nil)
	if err != nil {
		return nil, nil, err
	}
	if session.EmailVerifiedAt == nil {
		return nil, nil, ErrEmailVerifyRequired
	}
	if strings.TrimSpace(session.PendingPasswordHash) == "" {
		return nil, nil, ErrPasswordRequired
	}

	signupSource := normalizeDefaultSettingsSignupSource(input.ProviderType)
	defaultSettings := DefaultUserSettings{
		Balance:     s.cfg.Default.UserBalance,
		Concurrency: s.cfg.Default.UserConcurrency,
	}
	if s.settingService != nil {
		defaultSettings = s.settingService.GetDefaultUserSettingsBySignupSource(ctx, signupSource)
	}

	user := &User{
		Email:        session.ResolvedEmail,
		PasswordHash: session.PendingPasswordHash,
		SignupSource: signupSource,
		Role:         RoleUser,
		Balance:      defaultSettings.Balance,
		Concurrency:  defaultSettings.Concurrency,
		Status:       StatusActive,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}
	if writer, ok := s.userRepo.(UserSignupSourceRepository); ok {
		if err := writer.UpdateSignupSource(ctx, user.ID, signupSource); err != nil {
			return nil, nil, err
		}
	}
	if err := s.finalizePendingAuthSession(ctx, session, user.ID, false); err != nil {
		return nil, nil, err
	}
	s.assignDefaultSubscriptionsForSettings(ctx, user.ID, defaultSettings.Subscriptions)

	tokenPair, err := s.GenerateTokenPair(ctx, user, "")
	if err != nil {
		return nil, nil, fmt.Errorf("generate token pair: %w", err)
	}
	return tokenPair, user, nil
}

func (s *AuthService) LoginOrRegisterSyntheticOAuthAndBindPendingSession(ctx context.Context, pendingToken, email, username, invitationCode string) (*TokenPair, *User, error) {
	tokenPair, user, created, err := s.LoginOrRegisterOAuthWithTokenPairDetailed(ctx, email, username, invitationCode)
	if err != nil {
		return nil, nil, err
	}

	if _, err := s.CompletePendingAuthSessionBind(ctx, pendingToken, user.ID); err != nil {
		if created {
			if cleanupErr := s.deleteSyntheticOAuthUser(ctx, user.ID, email); cleanupErr != nil {
				logger.LegacyPrintf(
					"service.auth",
					"[Auth] Failed to cleanup synthetic oauth signup after bind failure: user_id=%d email=%s err=%v",
					user.ID,
					email,
					cleanupErr,
				)
			}
		}
		return nil, nil, err
	}

	return tokenPair, user, nil
}

func (s *AuthService) BindEmailIdentity(ctx context.Context, userID int64, email, verifyCode, password string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if isReservedEmail(email) {
		return nil, ErrEmailReserved
	}
	if strings.TrimSpace(verifyCode) == "" {
		return nil, ErrEmailVerifyRequired
	}
	if strings.TrimSpace(password) == "" {
		return nil, ErrPasswordRequired
	}
	if err := s.verifyPendingAuthEmailCode(ctx, email, verifyCode); err != nil {
		return nil, err
	}
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if exists && !strings.EqualFold(user.Email, email) {
		return nil, ErrEmailExists
	}
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user.Email = email
	user.PasswordHash = hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) VerifyPendingAuthBindPassword(ctx context.Context, pendingToken, password string) (*PendingAuthEmailResolutionResult, error) {
	session, err := s.GetPendingAuthSessionForProgress(ctx, pendingToken, nil)
	if err != nil {
		return nil, err
	}
	if session.TargetUserID == nil {
		return nil, ErrPendingAuthVerificationRequired
	}
	user, err := s.userRepo.GetByID(ctx, *session.TargetUserID)
	if err != nil {
		return nil, err
	}
	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	now := time.Now()
	session.PasswordVerifiedAt = &now
	if err := s.updatePendingAuthSession(ctx, session); err != nil {
		return nil, err
	}
	return &PendingAuthEmailResolutionResult{
		Intent:           session.Intent,
		TargetUserID:     session.TargetUserID,
		Requires2FA:      user.TotpEnabled,
		PendingAuthToken: pendingToken,
	}, nil
}

func (s *AuthService) MarkPendingAuthSessionPasswordVerified(ctx context.Context, pendingToken string, userID int64) (*PendingAuthSessionRecord, error) {
	session, err := s.GetPendingAuthSessionForProgress(ctx, pendingToken, nil)
	if err != nil {
		return nil, err
	}
	if session.TargetUserID != nil && *session.TargetUserID != userID {
		return nil, ErrPendingAuthTargetMismatch
	}
	now := time.Now()
	session.TargetUserID = pendingAuthInt64Ptr(userID)
	session.PasswordVerifiedAt = &now
	if err := s.updatePendingAuthSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *AuthService) MarkPendingAuthSessionTOTPVerified(ctx context.Context, pendingToken string, userID int64) (*PendingAuthSessionRecord, error) {
	session, err := s.GetPendingAuthSessionForProgress(ctx, pendingToken, pendingAuthInt64Ptr(userID))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	session.TOTPVerifiedAt = &now
	if err := s.updatePendingAuthSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *AuthService) CompletePendingAuthSessionBind(ctx context.Context, pendingToken string, userID int64) (*PendingAuthBindCompletion, error) {
	session, err := s.GetPendingAuthSessionForProgress(ctx, pendingToken, nil)
	if err != nil {
		return nil, err
	}
	if session.TargetUserID != nil && *session.TargetUserID != userID {
		return nil, ErrPendingAuthTargetMismatch
	}

	targetUserID := userID
	if session.TargetUserID != nil {
		targetUserID = *session.TargetUserID
	} else {
		session.TargetUserID = &targetUserID
	}

	if session.Intent == PendingAuthIntentAdoptExistingUserByEmail &&
		(session.EmailVerifiedAt == nil || session.PasswordVerifiedAt == nil) {
		return nil, ErrPendingAuthVerificationRequired
	}

	user, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user.TotpEnabled && session.TOTPVerifiedAt == nil {
		return nil, ErrPendingAuthVerificationRequired
	}
	if err := s.finalizePendingAuthSession(ctx, session, targetUserID, true); err != nil {
		return nil, err
	}

	return &PendingAuthBindCompletion{
		UserID:           targetUserID,
		Intent:           session.Intent,
		PendingAuthToken: pendingToken,
	}, nil
}

func (s *AuthService) finalizePendingAuthSession(ctx context.Context, session *PendingAuthSessionRecord, userID int64, applyBindDefaults bool) error {
	if binder, ok := s.userRepo.(pendingAuthIdentityBinder); ok {
		if err := binder.BindPendingAuthIdentity(ctx, session, userID); err != nil {
			return err
		}
	}
	if err := s.applyPendingAuthAdoptionDecision(ctx, session, userID); err != nil {
		return err
	}
	if applyBindDefaults {
		if err := s.applyProviderDefaultSettingsOnFirstBind(ctx, userID, session.ProviderType); err != nil {
			return err
		}
	}
	now := time.Now()
	session.ConsumedAt = &now
	return s.updatePendingAuthSession(ctx, session)
}

func (s *AuthService) applyProviderDefaultSettingsOnFirstBind(ctx context.Context, userID int64, providerType string) error {
	if s.settingService == nil {
		return nil
	}

	signupSource := normalizeDefaultSettingsSignupSource(providerType)
	if signupSource == SignupSourceEmail {
		return nil
	}

	defaultSettings := s.settingService.GetDefaultUserSettingsBySignupSource(ctx, signupSource)
	if !defaultSettings.ApplyOnBind {
		return nil
	}

	if s.entClient != nil {
		return s.applyProviderDefaultSettingsOnFirstBindInTx(ctx, userID, signupSource, defaultSettings)
	}
	return s.applyProviderDefaultSettingsOnFirstBindWithCompensation(ctx, userID, signupSource, defaultSettings)
}

func (s *AuthService) applyProviderDefaultSettingsOnFirstBindInTx(ctx context.Context, userID int64, signupSource string, defaultSettings DefaultUserSettings) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return fmt.Errorf("begin provider default bind transaction: %w", err)
	}

	txCtx := ctx
	if err == nil {
		txCtx = dbent.NewTxContext(ctx, tx)
		defer func() { _ = tx.Rollback() }()
	}

	if err := s.applyProviderDefaultSettingsOnFirstBindStep(txCtx, userID, signupSource, defaultSettings); err != nil {
		return err
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit provider default bind transaction: %w", err)
		}
	}
	return nil
}

func (s *AuthService) applyProviderDefaultSettingsOnFirstBindWithCompensation(ctx context.Context, userID int64, signupSource string, defaultSettings DefaultUserSettings) error {
	grantStore, ok := s.userRepo.(providerDefaultBindGrantStore)
	if !ok {
		return nil
	}

	created, err := grantStore.TryCreateProviderDefaultBindGrant(ctx, userID, signupSource)
	if err != nil || !created {
		return err
	}

	applied := false
	appliedBalance := 0.0
	appliedConcurrency := 0
	defer func() {
		if applied {
			return
		}
		if appliedConcurrency != 0 {
			_ = s.userRepo.UpdateConcurrency(ctx, userID, -appliedConcurrency)
		}
		if appliedBalance > 0 {
			_ = s.userRepo.DeductBalance(ctx, userID, appliedBalance)
		}
		_ = grantStore.DeleteProviderDefaultBindGrant(context.WithoutCancel(ctx), userID, signupSource)
	}()

	if defaultSettings.Balance > 0 {
		if err := s.userRepo.UpdateBalance(WithSkipTotalRechargedTracking(ctx), userID, defaultSettings.Balance); err != nil {
			return err
		}
		appliedBalance = defaultSettings.Balance
	}
	if defaultSettings.Concurrency > 0 {
		if err := s.userRepo.UpdateConcurrency(ctx, userID, defaultSettings.Concurrency); err != nil {
			return err
		}
		appliedConcurrency = defaultSettings.Concurrency
	}
	if err := s.assignDefaultSubscriptionsForSettingsStrict(ctx, userID, defaultSettings.Subscriptions); err != nil {
		return err
	}

	applied = true
	return nil
}

func (s *AuthService) applyProviderDefaultSettingsOnFirstBindStep(ctx context.Context, userID int64, signupSource string, defaultSettings DefaultUserSettings) error {
	grantStore, ok := s.userRepo.(providerDefaultBindGrantStore)
	if !ok {
		return nil
	}

	created, err := grantStore.TryCreateProviderDefaultBindGrant(ctx, userID, signupSource)
	if err != nil || !created {
		return err
	}

	applied := false
	defer func() {
		if applied {
			return
		}
		_ = grantStore.DeleteProviderDefaultBindGrant(context.WithoutCancel(ctx), userID, signupSource)
	}()

	if defaultSettings.Balance > 0 {
		if err := s.userRepo.UpdateBalance(WithSkipTotalRechargedTracking(ctx), userID, defaultSettings.Balance); err != nil {
			return err
		}
	}
	if defaultSettings.Concurrency > 0 {
		if err := s.userRepo.UpdateConcurrency(ctx, userID, defaultSettings.Concurrency); err != nil {
			return err
		}
	}
	if err := s.assignDefaultSubscriptionsForSettingsStrict(ctx, userID, defaultSettings.Subscriptions); err != nil {
		return err
	}
	applied = true
	return nil
}

func (s *AuthService) updatePendingAuthSession(ctx context.Context, session *PendingAuthSessionRecord) error {
	store, err := s.pendingAuthStore()
	if err != nil {
		return err
	}
	return store.UpdatePendingAuthSession(ctx, clonePendingAuthSessionRecord(session))
}

func (s *AuthService) pendingAuthStore() (pendingAuthSessionStore, error) {
	store, ok := s.userRepo.(pendingAuthSessionStore)
	if !ok {
		return nil, ErrPendingAuthUnavailable
	}
	return store, nil
}

func (s *AuthService) signPendingSessionToken(sessionID string, expiresAt time.Time) (string, error) {
	now := time.Now()
	claims := &pendingAuthSessionClaims{
		SessionID: sessionID,
		Purpose:   pendingAuthSessionTokenPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

func (s *AuthService) verifyPendingSessionToken(tokenStr string) (string, error) {
	if len(tokenStr) > maxTokenLength {
		return "", ErrInvalidToken
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, err := parser.ParseWithClaims(tokenStr, &pendingAuthSessionClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}
	claims, ok := token.Claims.(*pendingAuthSessionClaims)
	if !ok || !token.Valid || claims.Purpose != pendingAuthSessionTokenPurpose || strings.TrimSpace(claims.SessionID) == "" {
		return "", ErrInvalidToken
	}
	return claims.SessionID, nil
}

func (s *AuthService) verifyPendingAuthEmailCode(ctx context.Context, email, verifyCode string) error {
	if s.emailService == nil {
		return nil
	}
	if err := s.emailService.VerifyCode(ctx, email, verifyCode); err != nil {
		return err
	}
	return nil
}

func pendingAuthInt64Ptr(v int64) *int64 {
	return &v
}

func (s *AuthService) assignDefaultSubscriptionsForSettingsStrict(ctx context.Context, userID int64, items []DefaultSubscriptionSetting) error {
	if s.defaultSubAssigner == nil || userID <= 0 {
		return nil
	}
	for _, item := range items {
		if _, _, err := s.defaultSubAssigner.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      item.GroupID,
			ValidityDays: item.ValidityDays,
			Notes:        "auto assigned by default user subscriptions setting",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthService) assignDefaultSubscriptionsForSettings(ctx context.Context, userID int64, items []DefaultSubscriptionSetting) {
	if err := s.assignDefaultSubscriptionsForSettingsStrict(ctx, userID, items); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to assign default subscription: user_id=%d err=%v", userID, err)
	}
}

func (s *AuthService) deleteSyntheticOAuthUser(ctx context.Context, userID int64, email string) error {
	if userID <= 0 || !isSyntheticOAuthEmail(email) {
		return nil
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil
		}
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
		return nil
	}
	if user.HasLocalIdentity() {
		return nil
	}

	return s.userRepo.Delete(ctx, userID)
}

func (s *AuthService) applyPendingAuthAdoptionDecision(ctx context.Context, session *PendingAuthSessionRecord, userID int64) error {
	if session == nil {
		return nil
	}

	adoptDisplayName, hasDisplayNameDecision := pendingAuthMetadataBool(session.Metadata, "adopt_display_name")
	adoptAvatar, hasAvatarDecision := pendingAuthMetadataBool(session.Metadata, "adopt_avatar")
	if !hasDisplayNameDecision && !hasAvatarDecision {
		return nil
	}

	if store, ok := s.userRepo.(identityAdoptionDecisionStore); ok {
		input := UpsertIdentityAdoptionDecisionInput{
			Ref: IdentityAdoptionDecisionRef{
				ProviderType:    session.ProviderType,
				ProviderKey:     session.ProviderKey,
				ProviderSubject: session.ProviderSubject,
			},
		}
		if hasDisplayNameDecision {
			input.AdoptDisplayName = &adoptDisplayName
		}
		if hasAvatarDecision {
			input.AdoptAvatar = &adoptAvatar
		}
		if _, err := store.UpsertIdentityAdoptionDecision(ctx, userID, input); err != nil {
			return err
		}
	}

	if adoptDisplayName {
		displayName := pendingAuthMetadataString(session.Metadata, "suggested_display_name")
		if displayName != "" {
			user, err := s.userRepo.GetByID(ctx, userID)
			if err != nil {
				return err
			}
			if user.Username != displayName {
				user.Username = displayName
				if err := s.userRepo.Update(ctx, user); err != nil {
					return err
				}
			}
		}
	}

	if adoptAvatar {
		avatarURL := pendingAuthMetadataString(session.Metadata, "suggested_avatar_url")
		if avatarURL != "" {
			_ = s.storePendingAuthSuggestedAvatar(ctx, userID, avatarURL)
		}
	}

	return nil
}

func (s *AuthService) storePendingAuthSuggestedAvatar(ctx context.Context, userID int64, avatarURL string) error {
	store, ok := s.userRepo.(interface {
		UpsertAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error)
	})
	if !ok {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(avatarURL), nil)
	if err != nil {
		return fmt.Errorf("build avatar request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch avatar: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("fetch avatar: unexpected status %d", resp.StatusCode)
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return fmt.Errorf("fetch avatar: unsupported content type %q", contentType)
	}
	if semi := strings.Index(contentType, ";"); semi >= 0 {
		contentType = strings.TrimSpace(contentType[:semi])
	}

	payload, err := io.ReadAll(io.LimitReader(resp.Body, DefaultUserAvatarMaxBytes+1))
	if err != nil {
		return fmt.Errorf("read avatar payload: %w", err)
	}
	if int64(len(payload)) > DefaultUserAvatarMaxBytes {
		return ErrUserAvatarTooLarge
	}

	_, err = store.UpsertAvatar(ctx, userID, BuildInlineUserAvatarInput(payload, contentType))
	return err
}

func pendingAuthMetadataBool(metadata map[string]any, key string) (bool, bool) {
	if len(metadata) == 0 {
		return false, false
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return false, false
	}
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true, true
		case "0", "false", "no", "off":
			return false, true
		}
	}
	return false, false
}

func pendingAuthMetadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
