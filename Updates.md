# 更新记录

## 2026-03-20

- 修复 CI：`account_handler_available_models_test.go` 中 `NewAccountHandler` 缺少 `CopilotGatewayService` 参数导致单元测试无法编译。
- Copilot 网关：转发前对 `model` 应用账号 `model_mapping`，并将 Anthropic 风格 id（如 `claude-haiku-4-5-20251001`、短横线版）规范为 Copilot API 接受的点分版（如 `claude-haiku-4.5`）；覆盖 `/chat/completions`、`/responses` 与 Anthropic `/v1/messages` 翻译路径。实现见 `backend/internal/pkg/copilot/model_normalize.go` 与 `CopilotGatewayService.rewriteCopilotUpstreamModel`。
- 版本号：`0.1.105` → `0.1.106`。
