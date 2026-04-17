package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ExternalIdentityProviderLinuxDo = "linuxdo"
	ExternalIdentityProviderWeChat  = "wechat"

	DefaultUserAvatarMaxBytes int64 = 100 * 1024
)

var (
	ErrExternalIdentityNotFound       = infraerrors.NotFound("USER_EXTERNAL_IDENTITY_NOT_FOUND", "user external identity not found")
	ErrExternalIdentityAlreadyBound   = infraerrors.Conflict("USER_EXTERNAL_IDENTITY_ALREADY_BOUND", "external identity is already bound")
	ErrInvalidExternalIdentity        = infraerrors.BadRequest("INVALID_EXTERNAL_IDENTITY", "invalid external identity")
	ErrInvalidExternalIdentitySubject = infraerrors.BadRequest("INVALID_EXTERNAL_IDENTITY_SUBJECT", "external identity subject is required")
	ErrLastLoginMethodRequired        = infraerrors.BadRequest("LAST_LOGIN_METHOD_REQUIRED", "cannot remove the last available login method")
	ErrUserAvatarNotFound             = infraerrors.NotFound("USER_AVATAR_NOT_FOUND", "user avatar not found")
	ErrUserAvatarStorageProviderEmpty = infraerrors.BadRequest("USER_AVATAR_STORAGE_PROVIDER_REQUIRED", "avatar storage provider is required")
	ErrUserAvatarStorageKeyEmpty      = infraerrors.BadRequest("USER_AVATAR_STORAGE_KEY_REQUIRED", "avatar storage key is required")
	ErrUserAvatarContentTypeEmpty     = infraerrors.BadRequest("USER_AVATAR_CONTENT_TYPE_REQUIRED", "avatar content type is required")
	ErrUserAvatarTooLarge             = infraerrors.BadRequest("USER_AVATAR_TOO_LARGE", "avatar exceeds 100KB limit")
	ErrPendingOAuthBindOnly           = infraerrors.BadRequest("PENDING_OAUTH_BIND_ONLY", "pending oauth identity can only be bound to the current account")
	ErrPendingOAuthEmailRequired      = infraerrors.BadRequest("PENDING_OAUTH_EMAIL_REQUIRED", "email is required to finish oauth login")
	ErrPendingOAuthVerifyCodeRequired = infraerrors.BadRequest("PENDING_OAUTH_VERIFY_CODE_REQUIRED", "email verification code is required to finish oauth login")
)

type UserExternalIdentity struct {
	ID               int64
	UserID           int64
	Provider         string
	ProviderUserID   string
	ProviderUnionID  *string
	ProviderUsername string
	DisplayName      string
	ProfileURL       string
	AvatarURL        string
	Metadata         map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UpsertUserExternalIdentityInput struct {
	Provider         string
	ProviderUserID   string
	ProviderUnionID  *string
	ProviderUsername string
	DisplayName      string
	ProfileURL       string
	AvatarURL        string
	Metadata         map[string]any
}

type UserAvatar struct {
	ID              int64
	UserID          int64
	StorageProvider string
	StorageKey      string
	URL             string
	ContentType     string
	ByteSize        int64
	SHA256          string
	Width           *int
	Height          *int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type UpsertUserAvatarInput struct {
	StorageProvider string
	StorageKey      string
	URL             string
	ContentType     string
	ByteSize        int64
	SHA256          string
	Width           *int
	Height          *int
}

type UserIdentityRepository interface {
	ListExternalIdentities(ctx context.Context, userID int64) ([]UserExternalIdentity, error)
	UpsertExternalIdentity(ctx context.Context, userID int64, input UpsertUserExternalIdentityInput) (*UserExternalIdentity, error)
	DeleteExternalIdentity(ctx context.Context, userID int64, provider string) error
	GetAvatar(ctx context.Context, userID int64) (*UserAvatar, error)
	UpsertAvatar(ctx context.Context, userID int64, input UpsertUserAvatarInput) (*UserAvatar, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

func NormalizeExternalProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ExternalIdentityProviderLinuxDo:
		return ExternalIdentityProviderLinuxDo
	case ExternalIdentityProviderWeChat:
		return ExternalIdentityProviderWeChat
	default:
		return ""
	}
}

func (in UpsertUserExternalIdentityInput) Validate() error {
	if NormalizeExternalProvider(in.Provider) == "" {
		return ErrInvalidExternalIdentity
	}
	if strings.TrimSpace(in.ProviderUserID) == "" {
		return ErrInvalidExternalIdentitySubject
	}
	return nil
}

func (in UpsertUserAvatarInput) Validate(maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultUserAvatarMaxBytes
	}
	if strings.TrimSpace(in.StorageProvider) == "" {
		return ErrUserAvatarStorageProviderEmpty
	}
	if strings.TrimSpace(in.StorageKey) == "" {
		return ErrUserAvatarStorageKeyEmpty
	}
	if strings.TrimSpace(in.ContentType) == "" {
		return ErrUserAvatarContentTypeEmpty
	}
	if in.ByteSize < 0 || in.ByteSize > maxBytes {
		return ErrUserAvatarTooLarge.WithMetadata(map[string]string{"max_bytes": fmt.Sprintf("%d", maxBytes)})
	}
	return nil
}

func preferredAvatarURL(custom *UserAvatar, identities []UserExternalIdentity) string {
	if custom != nil && strings.TrimSpace(custom.URL) != "" {
		return strings.TrimSpace(custom.URL)
	}
	for _, identity := range identities {
		if avatar := strings.TrimSpace(identity.AvatarURL); avatar != "" {
			return avatar
		}
	}
	return ""
}

func ResolvePreferredUserAvatarURL(custom *UserAvatar, identities []UserExternalIdentity) string {
	return preferredAvatarURL(custom, identities)
}

func enrichUserWithIdentityData(user *User, avatar *UserAvatar, identities []UserExternalIdentity) *User {
	if user == nil {
		return nil
	}
	user.ExternalIdentities = append([]UserExternalIdentity(nil), identities...)
	user.Avatar = avatar
	user.AvatarURL = preferredAvatarURL(avatar, identities)
	user.HasCustomAvatar = avatar != nil && (strings.TrimSpace(avatar.StorageKey) != "" || strings.TrimSpace(avatar.URL) != "")
	if avatar != nil {
		user.AvatarUpdatedAt = &avatar.UpdatedAt
	}
	return user
}

func ApplyIdentityProfileData(user *User, avatar *UserAvatar, identities []UserExternalIdentity) *User {
	return enrichUserWithIdentityData(user, avatar, identities)
}

func hasBoundExternalIdentity(user *User, provider string) bool {
	if user == nil {
		return false
	}
	normalized := NormalizeExternalProvider(provider)
	if normalized == "" {
		return false
	}
	for _, identity := range user.ExternalIdentities {
		if NormalizeExternalProvider(identity.Provider) == normalized {
			return true
		}
	}
	return false
}

func hasUsableLocalLogin(user *User) bool {
	if user == nil {
		return false
	}
	email := strings.ToLower(strings.TrimSpace(user.Email))
	if email == "" || strings.TrimSpace(user.PasswordHash) == "" {
		return false
	}
	return !strings.HasSuffix(email, LinuxDoConnectSyntheticEmailDomain) &&
		!strings.HasSuffix(email, WeChatConnectSyntheticEmailDomain) &&
		!strings.HasSuffix(email, OIDCConnectSyntheticEmailDomain)
}

func CanDisconnectExternalIdentity(user *User, provider string) bool {
	if !hasBoundExternalIdentity(user, provider) {
		return false
	}
	if hasUsableLocalLogin(user) {
		return true
	}

	normalized := NormalizeExternalProvider(provider)
	for _, identity := range user.ExternalIdentities {
		if NormalizeExternalProvider(identity.Provider) != normalized {
			return true
		}
	}
	return false
}

func (s *UserService) ListExternalIdentities(ctx context.Context, userID int64) ([]UserExternalIdentity, error) {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	identities, err := s.userRepo.ListExternalIdentities(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list external identities: %w", err)
	}
	return identities, nil
}

func (s *UserService) UpsertExternalIdentity(ctx context.Context, userID int64, input UpsertUserExternalIdentityInput) (*UserExternalIdentity, error) {
	input.Provider = NormalizeExternalProvider(input.Provider)
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	identity, err := s.userRepo.UpsertExternalIdentity(ctx, userID, input)
	if err != nil {
		return nil, fmt.Errorf("upsert external identity: %w", err)
	}
	return identity, nil
}

func (s *UserService) DeleteExternalIdentity(ctx context.Context, userID int64, provider string) error {
	provider = NormalizeExternalProvider(provider)
	if provider == "" {
		return ErrInvalidExternalIdentity
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user != nil && !hasBoundExternalIdentity(user, provider) {
		identities, listErr := s.userRepo.ListExternalIdentities(ctx, userID)
		if listErr != nil {
			return fmt.Errorf("list external identities: %w", listErr)
		}
		user.ExternalIdentities = identities
	}
	if hasBoundExternalIdentity(user, provider) && !CanDisconnectExternalIdentity(user, provider) {
		return ErrLastLoginMethodRequired
	}
	if err := s.userRepo.DeleteExternalIdentity(ctx, userID, provider); err != nil {
		return fmt.Errorf("delete external identity: %w", err)
	}
	return nil
}

func (s *UserService) GetAvatar(ctx context.Context, userID int64) (*UserAvatar, error) {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	avatar, err := s.userRepo.GetAvatar(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get avatar: %w", err)
	}
	return avatar, nil
}

func (s *UserService) UpsertAvatar(ctx context.Context, userID int64, input UpsertUserAvatarInput) (*UserAvatar, error) {
	if err := input.Validate(DefaultUserAvatarMaxBytes); err != nil {
		return nil, err
	}
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	avatar, err := s.userRepo.UpsertAvatar(ctx, userID, input)
	if err != nil {
		return nil, fmt.Errorf("upsert avatar: %w", err)
	}
	return avatar, nil
}

func (s *UserService) DeleteAvatar(ctx context.Context, userID int64) error {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if err := s.userRepo.DeleteAvatar(ctx, userID); err != nil {
		return fmt.Errorf("delete avatar: %w", err)
	}
	return nil
}

func ensureVerifiedEmailForOAuth(ctx context.Context, emailService *EmailService, email, verifyCode string) error {
	if strings.TrimSpace(email) == "" {
		return ErrPendingOAuthEmailRequired
	}
	if strings.TrimSpace(verifyCode) == "" {
		return ErrPendingOAuthVerifyCodeRequired
	}
	if emailService == nil {
		return ErrServiceUnavailable
	}
	if err := emailService.VerifyCode(ctx, email, verifyCode); err != nil {
		return fmt.Errorf("verify code: %w", err)
	}
	return nil
}

func isExternalIdentityNotFound(err error) bool {
	return errors.Is(err, ErrExternalIdentityNotFound)
}
