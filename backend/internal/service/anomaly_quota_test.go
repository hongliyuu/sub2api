//go:build unit

package service

import (
	"testing"
)

func TestDetectAnomalies_QuotaExhaustion(t *testing.T) {
	settings := &AnomalySettings{
		SlowRequestThresholdMs: 20000,
		TimeoutThresholdMs:     60000,
		DetectZeroToken:        true,
	}

	// Zero tokens + fast upstream latency signals quota exhaustion, not a slow request.
	upstreamLatencyMs := int(200)
	types := detectAnomalies(AnomalySignal{
		InputTokens:       0,
		OutputTokens:      0,
		DurationMs:        500,
		UpstreamLatencyMs: &upstreamLatencyMs,
		StatusCode:        200,
	}, settings)

	found := map[string]bool{}
	for _, at := range types {
		found[string(at)] = true
	}
	if !found[string(AnomalyQuotaExhaustionSuspected)] {
		t.Errorf("expected quota_exhaustion_suspected; got %v", types)
	}
	// zero_token should NOT fire when quota exhaustion is detected (same root cause)
	if found[string(AnomalyZeroToken)] {
		t.Errorf("zero_token should not fire when quota_exhaustion_suspected; got %v", types)
	}
}
