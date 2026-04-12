package service

import (
	"context"
	"testing"
)

// stubCopilotPlatformConfigRepo 是 CopilotPlatformConfigRepository 的最小 stub。
type stubCopilotPlatformConfigRepo struct {
	entries  []CopilotPlatformConfigEntry
	upsertFn func(planType string, patch CopilotPlatformConfigPatch) (*CopilotPlatformConfigEntry, error)
}

func (s *stubCopilotPlatformConfigRepo) GetAll(ctx context.Context) ([]CopilotPlatformConfigEntry, error) {
	return s.entries, nil
}

func (s *stubCopilotPlatformConfigRepo) GetByPlanType(ctx context.Context, planType string) (*CopilotPlatformConfigEntry, error) {
	for _, e := range s.entries {
		if e.PlanType == planType {
			return &e, nil
		}
	}
	return nil, ErrCopilotPlatformConfigNotFound
}

func (s *stubCopilotPlatformConfigRepo) Upsert(ctx context.Context, planType string, patch CopilotPlatformConfigPatch) (*CopilotPlatformConfigEntry, error) {
	return s.upsertFn(planType, patch)
}

func TestCopilotPlatformConfigService_GetAll(t *testing.T) {
	maxTokens := int64(8192)
	repo := &stubCopilotPlatformConfigRepo{
		entries: []CopilotPlatformConfigEntry{
			{PlanType: "individual_free"},
			{PlanType: "individual_pro", MaxOutputTokens: &maxTokens},
		},
	}
	svc := NewCopilotPlatformConfigService(repo)
	results, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[1].MaxOutputTokens == nil || *results[1].MaxOutputTokens != 8192 {
		t.Fatalf("expected MaxOutputTokens=8192 for individual_pro")
	}
}

func TestCopilotPlatformConfigService_GetByPlanType_NotFound(t *testing.T) {
	repo := &stubCopilotPlatformConfigRepo{entries: nil}
	svc := NewCopilotPlatformConfigService(repo)
	_, err := svc.GetByPlanType(context.Background(), "nonexistent")
	if err != ErrCopilotPlatformConfigNotFound {
		t.Fatalf("expected ErrCopilotPlatformConfigNotFound, got %v", err)
	}
}

func TestCopilotPlatformConfigService_UpdateByPlanType(t *testing.T) {
	called := false
	repo := &stubCopilotPlatformConfigRepo{
		upsertFn: func(planType string, patch CopilotPlatformConfigPatch) (*CopilotPlatformConfigEntry, error) {
			called = true
			if planType != "business" {
				t.Errorf("expected planType=business, got %s", planType)
			}
			maxTokens := int64(8192)
			return &CopilotPlatformConfigEntry{PlanType: planType, MaxOutputTokens: &maxTokens}, nil
		},
	}
	svc := NewCopilotPlatformConfigService(repo)
	maxTokens := int64(8192)
	result, err := svc.UpdateByPlanType(context.Background(), "business", CopilotPlatformConfigPatch{
		MaxOutputTokens:    &maxTokens,
		SetMaxOutputTokens: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("repo.Upsert was not called")
	}
	if result.MaxOutputTokens == nil || *result.MaxOutputTokens != 8192 {
		t.Fatalf("expected MaxOutputTokens=8192 in result")
	}
}
