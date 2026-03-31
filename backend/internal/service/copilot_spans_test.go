//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRecordUsage_CopilotSpansPersistedToUsageLog(t *testing.T) {
	usageRepo := &openAIRecordUsageBestEffortLogRepoStub{}
	userRepo := &openAIRecordUsageUserRepoStub{}
	subRepo := &openAIRecordUsageSubRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, subRepo)

	spans := []*OpsSpan{
		{Name: "token.fetch", StartUnixMs: 1000, DurationMs: 50, Status: "ok"},
		{Name: "upstream.post", StartUnixMs: 1050, DurationMs: 800, Status: "ok"},
	}

	_, _, err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			Model:    "gpt-4o",
			Duration: time.Second,
			Usage:    ClaudeUsage{InputTokens: 10, OutputTokens: 5},
		},
		APIKey:  &APIKey{ID: 1001},
		User:    &User{ID: 2001},
		Account: &Account{ID: 3001, Platform: PlatformCopilot},
		Spans:   spans,
	})
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if usageRepo.lastLog == nil {
		t.Fatal("UsageLog was not saved")
	}
	if usageRepo.lastLog.Spans == nil {
		t.Fatal("expected Spans non-nil in saved UsageLog")
	}
	if !strings.Contains(*usageRepo.lastLog.Spans, "token.fetch") {
		t.Errorf("expected token.fetch in spans; got %s", *usageRepo.lastLog.Spans)
	}
}
