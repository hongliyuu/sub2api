package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockAccountRepoForSignal struct {
	AccountRepository
	modelRateLimits map[string]time.Time
}

func (m *mockAccountRepoForSignal) SetModelRateLimit(ctx context.Context, id int64, key string, resetAt time.Time) error {
	m.modelRateLimits[key] = resetAt
	return nil
}

func (m *mockAccountRepoForSignal) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	return nil
}

func (m *mockAccountRepoForSignal) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	return nil
}

func TestRateLimitService_HandleUpstreamError_EmitsSignal(t *testing.T) {
	repo := &mockAccountRepoForSignal{
		modelRateLimits: make(map[string]time.Time),
	}
	s := &RateLimitService{
		accountRepo: repo,
	}

	account := &Account{
		ID:       1,
		Platform: PlatformAnthropic,
	}

	// 429 with model should produce a model-scoped signal.
	headers := http.Header{}
	headers.Set("Retry-After", "60")
	_, signal := s.HandleUpstreamError(context.Background(), account, "claude-3-opus", 429, headers, []byte("{}"))

	require.NotNil(t, signal)
	require.Equal(t, RateLimitScopeModelBucket, signal.Scope)
	require.Equal(t, "anthropic:claude-3-opus", signal.Key)
	require.True(t, signal.ResetAt.After(time.Now().Add(55*time.Second)))

	require.Contains(t, repo.modelRateLimits, "anthropic:claude-3-opus")
}

func TestRateLimitService_HandleUpstreamError_GlobalFallback(t *testing.T) {
	repo := &mockAccountRepoForSignal{
		modelRateLimits: make(map[string]time.Time),
	}
	s := &RateLimitService{
		accountRepo: repo,
	}

	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
	}

	// 429 without model should produce an account-scoped signal.
	headers := http.Header{}
	headers.Set("Retry-After", "120")
	_, signal := s.HandleUpstreamError(context.Background(), account, "", 429, headers, []byte("{}"))

	require.NotNil(t, signal)
	require.Equal(t, RateLimitScopeAccount, signal.Scope)
	require.Empty(t, signal.Key)
	require.True(t, signal.ResetAt.After(time.Now().Add(115*time.Second)))

	require.Empty(t, repo.modelRateLimits)
}
