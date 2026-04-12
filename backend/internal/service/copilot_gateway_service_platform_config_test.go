package service

import (
	"context"
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
