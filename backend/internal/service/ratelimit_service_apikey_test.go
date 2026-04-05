//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRateLimitService_HandleUpstreamError_OpenAIAPIKey403ModelAccessIgnored(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       104,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"model not allowed for this project","code":"forbidden"}}`),
	)

	require.False(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
}

func TestRateLimitService_HandleUpstreamError_GeminiAPIKey400InvalidDisables(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       105,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusBadRequest,
		http.Header{},
		[]byte(`{"error":{"message":"API key not valid. Please pass a valid API key.","status":"API_KEY_INVALID"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrorCalls)
}

func TestRateLimitService_HandleUpstreamError_APIKey429UsesTemporaryCooldown(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       106,
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
	}

	before := time.Now()
	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"rate limited"}}`),
	)
	after := time.Now()

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.rateLimitedCalls)
	require.Equal(t, 0, repo.overloadedCalls)
	require.Equal(t, 0, repo.tempCalls)
	require.NotNil(t, repo.lastRateLimitResetAt)
	require.WithinDuration(t, before.Add(apiKey429Cooldown), *repo.lastRateLimitResetAt, after.Sub(before)+time.Second)
}

func TestRateLimitService_HandleUpstreamError_APIKey529UsesTemporaryCooldown(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       108,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}

	before := time.Now()
	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		529,
		http.Header{},
		[]byte(`{"error":{"message":"overloaded"}}`),
	)
	after := time.Now()

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 0, repo.rateLimitedCalls)
	require.Equal(t, 1, repo.overloadedCalls)
	require.Equal(t, 0, repo.tempCalls)
	require.NotNil(t, repo.lastOverloadedUntil)
	require.WithinDuration(t, before.Add(apiKey529Cooldown), *repo.lastOverloadedUntil, after.Sub(before)+time.Second)
}

func TestRateLimitService_HandleUpstreamError_APIKey503UsesTemporaryCooldown(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       107,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}

	before := time.Now()
	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusServiceUnavailable,
		http.Header{},
		[]byte(`{"error":{"message":"service temporarily unavailable"}}`),
	)
	after := time.Now()

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 0, repo.rateLimitedCalls)
	require.Equal(t, 0, repo.overloadedCalls)
	require.Equal(t, 1, repo.tempCalls)
	require.NotNil(t, repo.lastTempUntil)
	require.WithinDuration(t, before.Add(apiKeyServerErrorCooldown), *repo.lastTempUntil, after.Sub(before)+time.Second)
}
