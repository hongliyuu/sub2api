package service

import (
	"testing"
)

func TestDetectAnomalies_ZeroToken(t *testing.T) {
	settings := &AnomalySettings{
		SlowRequestThresholdMs: 20000,
		TimeoutThresholdMs:     60000,
		DetectZeroToken:        true,
	}
	types := detectAnomalies(0, 0, 5000, 200, settings)
	if len(types) != 1 || types[0] != AnomalyZeroToken {
		t.Errorf("expected [zero_token], got %v", types)
	}
}

func TestDetectAnomalies_SlowRequest(t *testing.T) {
	settings := &AnomalySettings{
		SlowRequestThresholdMs: 20000,
		TimeoutThresholdMs:     60000,
		DetectZeroToken:        true,
	}
	types := detectAnomalies(100, 200, 25000, 200, settings)
	if len(types) != 1 || types[0] != AnomalySlowRequest {
		t.Errorf("expected [slow_request], got %v", types)
	}
}

func TestDetectAnomalies_Timeout(t *testing.T) {
	settings := &AnomalySettings{
		SlowRequestThresholdMs: 20000,
		TimeoutThresholdMs:     60000,
		DetectZeroToken:        true,
	}
	types := detectAnomalies(0, 0, 70000, 200, settings)
	found := map[string]bool{}
	for _, at := range types {
		found[string(at)] = true
	}
	if !found["zero_token"] || !found["timeout"] || found["slow_request"] {
		t.Errorf("expected [zero_token, timeout], got %v", types)
	}
}

func TestDetectAnomalies_Error(t *testing.T) {
	settings := &AnomalySettings{SlowRequestThresholdMs: 20000, TimeoutThresholdMs: 60000}
	types := detectAnomalies(100, 200, 1000, 500, settings)
	if len(types) != 1 || types[0] != AnomalyError {
		t.Errorf("expected [error], got %v", types)
	}
}

func TestDetectAnomalies_Normal(t *testing.T) {
	settings := &AnomalySettings{SlowRequestThresholdMs: 20000, TimeoutThresholdMs: 60000, DetectZeroToken: true}
	types := detectAnomalies(100, 200, 5000, 200, settings)
	if len(types) != 0 {
		t.Errorf("expected no anomalies, got %v", types)
	}
}

func TestDetectAnomalies_ZeroTokenDisabled(t *testing.T) {
	settings := &AnomalySettings{SlowRequestThresholdMs: 20000, TimeoutThresholdMs: 60000, DetectZeroToken: false}
	types := detectAnomalies(0, 0, 5000, 200, settings)
	if len(types) != 0 {
		t.Errorf("expected no anomalies when detect_zero_token=false, got %v", types)
	}
}
