package service

import "testing"

// TestGatewayServiceIsModelSupportedByAccount_CopilotNoMappingAllowsAll verifies
// that a Copilot account with no model_mapping accepts every model.
func TestGatewayServiceIsModelSupportedByAccount_CopilotNoMappingAllowsAll(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform:    PlatformCopilot,
		Credentials: map[string]any{},
	}

	for _, model := range []string{"claude-sonnet-4.6", "gpt-4o", "claude-sonnet-4-5", "claude-opus-4-6"} {
		if !svc.isModelSupportedByAccount(account, model) {
			t.Fatalf("expected model %q to be supported when model_mapping is empty", model)
		}
	}
}

// TestGatewayServiceIsModelSupportedByAccount_CopilotMappingDoesNotActAsAllowlist verifies
// that adding a model_mapping to a Copilot account does NOT restrict which models
// can be routed to it. The mapping only controls upstream model name rewriting.
func TestGatewayServiceIsModelSupportedByAccount_CopilotMappingDoesNotActAsAllowlist(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"claude-sonnet-4-5": "claude-sonnet-4.6",
				"claude-sonnet-4.5": "claude-sonnet-4.6",
			},
		},
	}

	// These models are NOT in the mapping keys but must still be routable.
	for _, model := range []string{"claude-sonnet-4.6", "gpt-4o", "claude-opus-4-6", "claude-haiku-4.5"} {
		if !svc.isModelSupportedByAccount(account, model) {
			t.Fatalf("model %q should be supported even though it is not a model_mapping key; "+
				"Copilot model_mapping must not act as an allowlist", model)
		}
	}

	// The mapped models themselves should also be routable.
	for _, model := range []string{"claude-sonnet-4-5", "claude-sonnet-4.5"} {
		if !svc.isModelSupportedByAccount(account, model) {
			t.Fatalf("model %q (a mapping key) should be supported", model)
		}
	}
}

// TestGatewayServiceIsModelSupportedByAccount_CopilotWhitelistFilters 验证
// 当账号设置了 model_whitelist 时，只有白名单内的模型被允许。
func TestGatewayServiceIsModelSupportedByAccount_CopilotWhitelistFilters(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			"model_whitelist": []interface{}{"claude-sonnet-4.6", "gpt-4o"},
		},
	}
	allowed := []string{"claude-sonnet-4.6", "gpt-4o"}
	blocked := []string{"claude-opus-4.6", "claude-haiku-4.5"}
	for _, m := range allowed {
		if !svc.isModelSupportedByAccount(account, m) {
			t.Errorf("model %q should be allowed by whitelist", m)
		}
	}
	for _, m := range blocked {
		if svc.isModelSupportedByAccount(account, m) {
			t.Errorf("model %q should be blocked by whitelist", m)
		}
	}
}

// TestGatewayServiceIsModelSupportedByAccount_CopilotEmptyWhitelistAllowsAll 验证
// 当账号 model_whitelist 为空时允许所有模型。
func TestGatewayServiceIsModelSupportedByAccount_CopilotEmptyWhitelistAllowsAll(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformCopilot,
		Credentials: map[string]any{
			// model_whitelist 未设置
		},
	}
	for _, m := range []string{"claude-sonnet-4.6", "gpt-4o", "claude-opus-4.6"} {
		if !svc.isModelSupportedByAccount(account, m) {
			t.Errorf("model %q should be allowed when whitelist is empty", m)
		}
	}
}
