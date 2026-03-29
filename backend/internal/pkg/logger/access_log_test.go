//go:build unit

package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccessLogger_EmitWritesJSONLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")

	al := NewAccessLogger(AccessLogOptions{
		FilePath:    path,
		IncludeBody: false,
	})
	defer al.Close()

	al.Emit(AccessLogEntry{
		RequestID:       "req-123",
		ClientRequestID: "client-456",
		SessionHash:     "sess-abc",
		Platform:        "anthropic",
		Model:           "claude-sonnet-4-6",
		StatusCode:      200,
		DurationMs:      1234,
	})

	al.Close() // flush

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read access log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, `"request_id":"req-123"`) {
		t.Errorf("expected request_id in output, got: %s", line)
	}
	if !strings.Contains(line, `"session_hash":"sess-abc"`) {
		t.Errorf("expected session_hash in output, got: %s", line)
	}
	if !strings.Contains(line, `"platform":"anthropic"`) {
		t.Errorf("expected platform in output, got: %s", line)
	}
}

func TestAccessLogger_NilSafeEmit(t *testing.T) {
	var al *AccessLogger
	al.Emit(AccessLogEntry{RequestID: "should-not-panic"}) // must not panic
}
