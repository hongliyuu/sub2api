package service_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestAppendOpsSpan_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	now := time.Now()
	service.AppendOpsSpan(c, service.OpsSpan{
		Name:        "auth.verify",
		StartUnixMs: now.UnixMilli(),
		DurationMs:  12,
		Status:      "ok",
	})

	spans := service.GetOpsSpans(c)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "auth.verify" {
		t.Errorf("expected name auth.verify, got %s", spans[0].Name)
	}
}

func TestMarshalOpsSpans_EmptyIsNil(t *testing.T) {
	result := service.MarshalOpsSpans(nil)
	if result != nil {
		t.Errorf("expected nil for empty spans, got %v", result)
	}
}

func TestMarshalOpsSpans_Valid(t *testing.T) {
	spans := []*service.OpsSpan{
		{Name: "routing.select_account", StartUnixMs: 1000, DurationMs: 5, Status: "ok"},
	}
	result := service.MarshalOpsSpans(spans)
	if result == nil {
		t.Fatal("expected non-nil JSON string")
	}

	var parsed []*service.OpsSpan
	if err := json.Unmarshal([]byte(*result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "routing.select_account" {
		t.Errorf("parsed span mismatch: %+v", parsed)
	}
}
