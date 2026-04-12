package service

import (
	"context"
	"encoding/json"
	"testing"
)

// TestEffectiveCopilotMaxOutputTokens_AccountLevelTakesPrecedence 验证账号级配置优先。
func TestEffectiveCopilotMaxOutputTokens_AccountLevelTakesPrecedence(t *testing.T) {
	svc := &CopilotGatewayService{}
	platformTokens := int64(4096)
	svc.platformConfigSvc = &CopilotPlatformConfigService{
		repo: &stubCopilotPlatformConfigRepo{
			entries: []CopilotPlatformConfigEntry{
				{PlanType: "individual_pro", MaxOutputTokens: &platformTokens},
			},
		},
	}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			"copilot_max_output_tokens": float64(16384),
			"plan_type":                 "individual_pro",
		},
	}
	cap, clamp := svc.effectiveCopilotMaxOutputTokensCap(context.Background(), account)
	if cap != 16384 {
		t.Errorf("expected account-level cap=16384, got %d", cap)
	}
	if !clamp {
		t.Error("expected clamp=true")
	}
}

// TestEffectiveCopilotMaxOutputTokens_FallsBackToPlatformConfig 验证账号未设置时继承平台配置。
func TestEffectiveCopilotMaxOutputTokens_FallsBackToPlatformConfig(t *testing.T) {
	svc := &CopilotGatewayService{}
	platformTokens := int64(4096)
	svc.platformConfigSvc = &CopilotPlatformConfigService{
		repo: &stubCopilotPlatformConfigRepo{
			entries: []CopilotPlatformConfigEntry{
				{PlanType: "business", MaxOutputTokens: &platformTokens},
			},
		},
	}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			"plan_type": "business",
			// copilot_max_output_tokens 未设置
		},
	}
	cap, clamp := svc.effectiveCopilotMaxOutputTokensCap(context.Background(), account)
	if cap != 4096 {
		t.Errorf("expected platform-config cap=4096, got %d", cap)
	}
	if !clamp {
		t.Error("expected clamp=true")
	}
}

// TestEffectiveCopilotMaxOutputTokens_FallsBackToSystemDefault 验证两者未设置时使用系统默认。
func TestEffectiveCopilotMaxOutputTokens_FallsBackToSystemDefault(t *testing.T) {
	svc := &CopilotGatewayService{}
	// platformConfigSvc 为 nil（未注入），模拟系统默认场景
	account := &Account{
		Platform:    PlatformCopilot,
		Credentials: map[string]any{},
	}
	cap, clamp := svc.effectiveCopilotMaxOutputTokensCap(context.Background(), account)
	if cap != defaultCopilotMaxOutputTokens {
		t.Errorf("expected system default cap=%d, got %d", defaultCopilotMaxOutputTokens, cap)
	}
	if !clamp {
		t.Error("expected clamp=true")
	}
}

// TestRewriteCopilotUpstreamModel_PlatformMappingFallback 验证当账号无 model_mapping 时，
// 平台级 model_mapping 作为 fallback 正确应用到上游请求体中。
func TestRewriteCopilotUpstreamModel_PlatformMappingFallback(t *testing.T) {
	svc := &CopilotGatewayService{}
	svc.platformConfigSvc = &CopilotPlatformConfigService{
		repo: &stubCopilotPlatformConfigRepo{
			entries: []CopilotPlatformConfigEntry{
				{
					PlanType: "business",
					ModelMapping: map[string]string{
						"claude-haiku-4.5": "claude-sonnet-4.6",
					},
				},
			},
		},
	}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			"plan_type": "business",
			// model_mapping 未设置（账号级无映射）
		},
	}
	body := []byte(`{"model":"claude-haiku-4.5","messages":[{"role":"user","content":"hi"}]}`)
	platformMapping := svc.getCopilotPlatformModelMapping(context.Background(), account)
	rewritten, logModel := rewriteCopilotUpstreamModel(body, account, platformMapping)

	gotModel := string(gjsonGetBytes(rewritten, "model"))
	if gotModel != "claude-sonnet-4.6" {
		t.Errorf("expected upstream model=claude-sonnet-4.6, got %q", gotModel)
	}
	if logModel != "claude-haiku-4.5" {
		t.Errorf("expected logModel=claude-haiku-4.5 (original), got %q", logModel)
	}
}

// TestRewriteCopilotUpstreamModel_AccountMappingTakesPrecedenceOverPlatform 验证账号级映射优先于平台级映射。
func TestRewriteCopilotUpstreamModel_AccountMappingTakesPrecedenceOverPlatform(t *testing.T) {
	svc := &CopilotGatewayService{}
	svc.platformConfigSvc = &CopilotPlatformConfigService{
		repo: &stubCopilotPlatformConfigRepo{
			entries: []CopilotPlatformConfigEntry{
				{
					PlanType: "business",
					ModelMapping: map[string]string{
						"claude-haiku-4.5": "claude-sonnet-4.6", // 平台级映射到 sonnet
					},
				},
			},
		},
	}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			"plan_type": "business",
			"model_mapping": map[string]any{
				"claude-haiku-4.5": "claude-opus-4.6", // 账号级映射到 opus（优先）
			},
		},
	}
	body := []byte(`{"model":"claude-haiku-4.5","messages":[{"role":"user","content":"hi"}]}`)
	platformMapping := svc.getCopilotPlatformModelMapping(context.Background(), account)
	rewritten, logModel := rewriteCopilotUpstreamModel(body, account, platformMapping)

	gotModel := string(gjsonGetBytes(rewritten, "model"))
	if gotModel != "claude-opus-4.6" {
		t.Errorf("expected account-level mapping wins: claude-opus-4.6, got %q", gotModel)
	}
	if logModel != "claude-haiku-4.5" {
		t.Errorf("expected logModel=claude-haiku-4.5 (original), got %q", logModel)
	}
}

// gjsonGetBytes is a test helper to extract a JSON field value as bytes.
func gjsonGetBytes(data []byte, path string) []byte {
	return []byte(gjsonGetString(data, path))
}

func gjsonGetString(data []byte, path string) string {
	// Use a simple JSON unmarshal for testing
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if v, ok := m[path]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
