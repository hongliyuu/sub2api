# 更新记录

## 2026-03-20

- 版本号：`0.1.107` → `0.1.108`（合并上游 Wei-Shaw/sub2api `main` 至 `a225a241`）。
- 合并上游：OpenAI `gpt-5.4-mini` / `gpt-5.4-nano` 定价与模型配置；日/周配额重置后用量展示不再沿用累计旧值；Anthropic→OpenAI 推理级别映射与 Codex 转换；OpenAI 默认模型转发；`UseKeyModal` 配额展示与单测等（对应上游 PR #1172、#1176、#1171、#1162 等）。

- Copilot 计费：`/copilot/v1`、`/copilot` 路由组挂载 `InboundEndpointMiddleware`；异步 `RecordUsage` 之前在请求协程内快照 `inbound`（禁止在 `go func()` 里读 `GetInboundEndpoint(c)`，避免 Gin 回收 Context 后入站路径为空）；上游路径与全局常量对齐为 `/v1/chat/completions`、`/v1/responses`。
- 运维 / 请求排查：左右分栏改为组件内 `scoped` 媒体查询（`grid-template-columns`），避免 Tailwind 任意 `grid-cols-[minmax(...,...)]` 因逗号解析失败而始终单列。
- 运维 / Copilot：`usage_logs` 写入时补齐 `inbound_endpoint`、`upstream_endpoint`（`/chat/completions`、`/responses` 等）及 `ForwardResult.UpstreamModel`（转发体中的 model）；`ops` 请求列表 CTE 中成功行 `status_code` 固定为 `200`。请求排查页自 `md` 断点起稳定左右分栏；详情与列表对成功行无状态码时显示 `200`；usage 卡片「上游模型」在库中未单独存（与客户端相同）时回退为客户端模型。
- 运维：新增管理端「请求排查」页 `/admin/ops/request-inspect`，列表与运维大盘一致的请求明细，右侧失败请求展示 ops 入库的客户端 `request_body` / `request_headers`、上游关联条目的转发请求体；成功请求调用新接口 `GET /admin/ops/usage-inspect` 展示 `usage_logs` 中的 `model` / `upstream_model` / `inbound_endpoint` / `upstream_endpoint` 等元数据（不含原始 JSON body）。侧栏在开启运维监控时增加入口；`/admin/ops` 菜单高亮改为仅精确匹配避免与子路径冲突。
- 修复 CI：`account_handler_available_models_test.go` 中 `NewAccountHandler` 缺少 `CopilotGatewayService` 参数导致单元测试无法编译。
- Copilot 网关：转发前对 `model` 应用账号 `model_mapping`，并将 Anthropic 风格 id（如 `claude-haiku-4-5-20251001`、短横线版）规范为 Copilot API 接受的点分版（如 `claude-haiku-4.5`）；覆盖 `/chat/completions`、`/responses` 与 Anthropic `/v1/messages` 翻译路径。实现见 `backend/internal/pkg/copilot/model_normalize.go` 与 `CopilotGatewayService.rewriteCopilotUpstreamModel`。
