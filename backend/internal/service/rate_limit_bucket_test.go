package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelRateLimitKey(t *testing.T) {
	t.Run("canonicalizes", func(t *testing.T) {
		key, ok := ModelRateLimitKey(PlatformOpenAI, " models/GPT-4 ")
		require.True(t, ok)
		require.Equal(t, "openai:gpt-4", key)
	})

	t.Run("geminiFlashSharedBucket", func(t *testing.T) {
		key, ok := ModelRateLimitKey(PlatformGemini, "Gemini-3-Flash-B001")
		require.True(t, ok)
		require.Equal(t, "gemini:gemini-3-flash", key)
	})

	t.Run("rejectsUnsafeCharacters", func(t *testing.T) {
		key, ok := ModelRateLimitKey(PlatformOpenAI, "gpt-4{")
		require.False(t, ok)
		require.Empty(t, key)
	})

	t.Run("rejectsMissingProvider", func(t *testing.T) {
		_, ok := ModelRateLimitKey("", "gpt-4")
		require.False(t, ok)
	})
}

func TestIsSafeModelRateLimitKey(t *testing.T) {
	require.True(t, IsSafeModelRateLimitKey("openai:gpt-4"))
	require.False(t, IsSafeModelRateLimitKey("openai:gpt-4}"))
	require.False(t, IsSafeModelRateLimitKey(""))
	require.False(t, IsSafeModelRateLimitKey("openai:gp t"))
}
