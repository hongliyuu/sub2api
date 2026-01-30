package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseRetryAfterSeconds(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	headers := http.Header{}
	headers.Set("Retry-After", "45")
	result := parseRetryAfter(headers, now)
	require.NotNil(t, result)
	require.Equal(t, now.Add(45*time.Second), *result)
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	now := time.Date(2026, time.February, 2, 3, 4, 5, 0, time.UTC)
	target := now.Add(3 * time.Minute)
	headers := http.Header{}
	headers.Set("Retry-After", target.Format(http.TimeFormat))
	result := parseRetryAfter(headers, now)
	require.NotNil(t, result)
	require.Equal(t, target, *result)
}

func TestParseRetryAfterInvalidValues(t *testing.T) {
	now := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	headers := http.Header{}
	for _, value := range []string{"-1", "0", "not-a-number"} {
		headers.Set("Retry-After", value)
		require.Nil(t, parseRetryAfter(headers, now), "value=%s should be ignored", value)
	}
}

func TestClampResetAtBounds(t *testing.T) {
	now := time.Date(2026, time.March, 3, 4, 5, 6, 0, time.UTC)
	tooSoon := clampResetAt(now.Add(-time.Hour), now)
	require.Equal(t, now.Add(time.Second), tooSoon)
	tooLate := clampResetAt(now.Add(48*time.Hour), now)
	require.Equal(t, now.Add(24*time.Hour), tooLate)
	within := clampResetAt(now.Add(12*time.Hour), now)
	require.Equal(t, now.Add(12*time.Hour), within)
}

func TestApplyResetJitterRange(t *testing.T) {
	now := time.Date(2026, time.March, 3, 4, 5, 6, 0, time.UTC)
	base := now.Add(10 * time.Minute)
	jittered := applyResetJitter(base, now)
	delta := jittered.Sub(base)
	require.True(t, delta >= 0)
	require.True(t, delta < rateLimitResetJitter)
}
