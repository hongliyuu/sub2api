//go:build unit

package service

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/copilot"
)

// TestCopilotTokenProvider_ContextCancellation verifies that a cancelled context
// aborts the token exchange call and returns a context error (not a nil token).
func TestCopilotTokenProvider_ContextCancellation(t *testing.T) {
	// Server that hangs until explicitly unblocked — simulates a slow GitHub API.
	unblock := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock // block until cleanup
	}))
	// Unblock the handler first, then close the server, so server.Close() doesn't hang.
	t.Cleanup(func() {
		close(unblock)
		server.Close()
	})

	exchanger := func(ctx context.Context, httpClient *http.Client, githubToken string) (*copilot.CopilotToken, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := httpClient.Do(req) //nolint:gosec
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		return nil, nil
	}

	provider := newCopilotTokenProviderWithExchanger(exchanger)
	account := &Account{
		ID:       99,
		Platform: PlatformCopilot,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"github_token": "ghp_test",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := provider.GetAccessToken(ctx, account)
	if err == nil {
		t.Fatal("expected error on context cancellation, got nil")
	}
}

// TestCopilotTokenProvider_LogsExchangeMs verifies that a successful token exchange
// logs "exchange_ms" at the debug level for observability.
func TestCopilotTokenProvider_LogsExchangeMs(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	origLogger := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(origLogger) })

	exchanger := func(ctx context.Context, httpClient *http.Client, githubToken string) (*copilot.CopilotToken, error) {
		return &copilot.CopilotToken{
			Token:     "ghs_test",
			ExpiresAt: time.Now().Add(30 * time.Minute),
			RefreshAt: time.Now().Add(5 * time.Minute),
		}, nil
	}
	provider := newCopilotTokenProviderWithExchanger(exchanger)

	tok, err := provider.GetAccessToken(context.Background(), &Account{
		ID:       100,
		Platform: PlatformCopilot,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"github_token": "ghp_logtest",
		},
	})
	if err != nil || tok == "" {
		t.Fatalf("GetAccessToken: err=%v tok=%q", err, tok)
	}
	if !strings.Contains(logBuf.String(), "exchange_ms") {
		t.Errorf("expected exchange_ms in debug log; got: %s", logBuf.String())
	}
}
