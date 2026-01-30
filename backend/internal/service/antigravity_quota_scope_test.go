package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsSchedulableForModel_TableDriven(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name           string
		account        *Account
		requestedModel string
		want           bool
	}{
		{
			name: "direct_bucket_limited",
			account: &Account{
				Platform:    PlatformAnthropic,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"anthropic:claude-3-opus": map[string]any{
							"rate_limit_reset_at": future.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "claude-3-opus",
			want:           false,
		},
		{
			name: "direct_bucket_eligible_different_model",
			account: &Account{
				Platform:    PlatformAnthropic,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"anthropic:claude-3-opus": map[string]any{
							"rate_limit_reset_at": future.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "claude-3-haiku",
			want:           true,
		},
		{
			name: "anthropic_family_bucket_limited",
			account: &Account{
				Platform:    PlatformAnthropic,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"anthropic:claude_sonnet": map[string]any{
							"rate_limit_reset_at": future.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "claude-3-5-sonnet",
			want:           false,
		},
		{
			name: "gemini_family_isolation_flash_limited_pro_eligible",
			account: &Account{
				Platform:    PlatformGemini,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"gemini:gemini-3-flash": map[string]any{
							"rate_limit_reset_at": future.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "gemini-3-pro",
			want:           true,
		},
		{
			name: "gemini_family_flash_limited_blocks_other_flash",
			account: &Account{
				Platform:    PlatformGemini,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"gemini:gemini-3-flash": map[string]any{
							"rate_limit_reset_at": future.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "gemini-3-flash-001",
			want:           false,
		},
		{
			name: "legacy_bucket_limited",
			account: &Account{
				Platform:    PlatformAnthropic,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"claude_sonnet": map[string]any{
							"rate_limit_reset_at": future.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "claude-3-sonnet-20240229",
			want:           false,
		},
		{
			name: "expired_limit_is_eligible",
			account: &Account{
				Platform:    PlatformAnthropic,
				Status:      StatusActive,
				Schedulable: true,
				Extra: map[string]any{
					"model_rate_limits": map[string]any{
						"anthropic:claude-3-5-sonnet": map[string]any{
							"rate_limit_reset_at": past.Format(time.RFC3339),
						},
					},
				},
			},
			requestedModel: "claude-3-5-sonnet",
			want:           true,
		},
		{
			name: "global_limited_blocks_all",
			account: &Account{
				Platform:         PlatformAnthropic,
				Status:           StatusActive,
				Schedulable:      true,
				RateLimitResetAt: &future,
			},
			requestedModel: "claude-3-opus",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.IsSchedulableForModel(tt.requestedModel))
		})
	}
}
