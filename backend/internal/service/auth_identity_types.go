package service

import (
	"context"
	"time"
)

const (
	PendingAuthIntentLogin                    = "login"
	PendingAuthIntentBindCurrentUser          = "bind_current_user"
	PendingAuthIntentAdoptExistingUserByEmail = "adopt_existing_user_by_email"
	pendingAuthSessionTokenPurpose            = "pending_auth_session"
)

type PendingAuthSessionInput struct {
	Intent          string
	ProviderType    string
	ProviderKey     string
	ProviderSubject string
	TargetUserID    *int64
	RedirectTo      string
	Metadata        map[string]any
}

type PendingAuthSessionRecord struct {
	ID                  string
	Token               string
	Intent              string
	ProviderType        string
	ProviderKey         string
	ProviderSubject     string
	TargetUserID        *int64
	ResolvedEmail       string
	PendingPasswordHash string
	Metadata            map[string]any
	EmailVerifiedAt     *time.Time
	PasswordVerifiedAt  *time.Time
	TOTPVerifiedAt      *time.Time
	ExpiresAt           time.Time
	ConsumedAt          *time.Time
	RedirectTo          string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PendingAuthEmailResolutionResult struct {
	Intent           string
	TargetUserID     *int64
	Requires2FA      bool
	PendingAuthToken string
}

type IdentityAdoptionDecisionRef struct {
	ProviderType    string
	ProviderKey     string
	ProviderSubject string
}

type UpsertIdentityAdoptionDecisionInput struct {
	Ref              IdentityAdoptionDecisionRef
	AdoptDisplayName *bool
	AdoptAvatar      *bool
}

type IdentityAdoptionDecision struct {
	Ref              IdentityAdoptionDecisionRef
	AdoptDisplayName bool
	AdoptAvatar      bool
	DecidedAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PendingAuthBindCompletion struct {
	UserID           int64
	Intent           string
	PendingAuthToken string
}

type pendingAuthSessionStore interface {
	CreatePendingAuthSession(ctx context.Context, input PendingAuthSessionInput) (*PendingAuthSessionRecord, error)
	GetPendingAuthSessionByID(ctx context.Context, sessionID string) (*PendingAuthSessionRecord, error)
	UpdatePendingAuthSession(ctx context.Context, session *PendingAuthSessionRecord) error
}

type pendingAuthIdentityBinder interface {
	BindPendingAuthIdentity(ctx context.Context, session *PendingAuthSessionRecord, userID int64) error
}

type providerDefaultBindGrantStore interface {
	TryCreateProviderDefaultBindGrant(ctx context.Context, userID int64, providerType string) (bool, error)
	DeleteProviderDefaultBindGrant(ctx context.Context, userID int64, providerType string) error
}

type identityAdoptionDecisionStore interface {
	GetIdentityAdoptionDecision(ctx context.Context, userID int64, ref IdentityAdoptionDecisionRef) (*IdentityAdoptionDecision, error)
	UpsertIdentityAdoptionDecision(ctx context.Context, userID int64, input UpsertIdentityAdoptionDecisionInput) (*IdentityAdoptionDecision, error)
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func clonePendingAuthSessionRecord(in *PendingAuthSessionRecord) *PendingAuthSessionRecord {
	if in == nil {
		return nil
	}
	out := *in
	out.Metadata = cloneStringAnyMap(in.Metadata)
	if in.TargetUserID != nil {
		target := *in.TargetUserID
		out.TargetUserID = &target
	}
	if in.EmailVerifiedAt != nil {
		v := *in.EmailVerifiedAt
		out.EmailVerifiedAt = &v
	}
	if in.PasswordVerifiedAt != nil {
		v := *in.PasswordVerifiedAt
		out.PasswordVerifiedAt = &v
	}
	if in.TOTPVerifiedAt != nil {
		v := *in.TOTPVerifiedAt
		out.TOTPVerifiedAt = &v
	}
	if in.ConsumedAt != nil {
		v := *in.ConsumedAt
		out.ConsumedAt = &v
	}
	return &out
}
