//go:build unit

package handler

import (
	"testing"
)

func TestBuildAccessLogEntry_BasicFields(t *testing.T) {
	params := AccessLogParams{
		RequestID:       "req-1",
		ClientRequestID: "client-1",
		SessionHash:     "sess-1",
		Platform:        "anthropic",
		Model:           "claude-sonnet-4-6",
		AccountID:       42,
		UserID:          7,
		APIKeyID:        3,
		Stream:          true,
		StatusCode:      200,
		DurationMs:      500,
	}

	entry := BuildAccessLogEntry(params)

	if entry.RequestID != "req-1" {
		t.Errorf("expected request_id=req-1, got %s", entry.RequestID)
	}
	if entry.SessionHash != "sess-1" {
		t.Errorf("expected session_hash=sess-1, got %s", entry.SessionHash)
	}
	if !entry.Success {
		t.Error("expected success=true for status 200")
	}
}

func TestBuildAccessLogEntry_FailureStatus(t *testing.T) {
	params := AccessLogParams{StatusCode: 500}
	entry := BuildAccessLogEntry(params)
	if entry.Success {
		t.Error("expected success=false for status 500")
	}
}
