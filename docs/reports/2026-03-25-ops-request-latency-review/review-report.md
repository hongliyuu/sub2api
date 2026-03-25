# 2026-03-25 请求排查耗时改造 Code Review 报告

## 1. 审查目标
围绕你的需求评审 cc 本次改动是否达标：

1. 当前请求排查页面“耗时”到底指什么。
2. 能否把链路耗时拆分得更细。
3. 是否做到“每个时间段都入库，并在前端展示”，用于运维定位。

## 2. 审查范围
本次主要审查未提交变更中与该需求相关的文件：

- 后端埋点与持久化：
  - `backend/internal/handler/gateway_handler.go`
  - `backend/internal/handler/openai_gateway_handler.go`
  - `backend/internal/handler/copilot_gateway_handler.go`
  - `backend/internal/service/gateway_service.go`
  - `backend/internal/service/openai_gateway_service.go`
  - `backend/internal/service/copilot_gateway_service.go`
  - `backend/internal/repository/usage_log_repo.go`
  - `backend/internal/repository/ops_repo_usage_inspect.go`
  - `backend/internal/service/ops_request_details.go`
  - `backend/migrations/078_add_usage_log_stage_latencies.sql`
- 前端展示：
  - `frontend/src/views/admin/ops/OpsRequestInspectView.vue`
  - `frontend/src/views/admin/ops/components/OpsRequestDetailPanel.vue`
  - `frontend/src/views/admin/ops/components/OpsLatencyBreakdownCard.vue`
  - `frontend/src/views/admin/ops/components/DurationBadge.vue`
  - `frontend/src/api/admin/ops.ts`

## 3. 结论（先给结论）
**当前改动“部分满足”，但还不能满足你提出的“可精确定位运维问题”的目标。**

- 优点：已经新增了分段耗时字段和排查页可视化卡片。
- 关键不足：存在高优先级正确性问题（含测试已失败）、Copilot 上游阶段未采集、总耗时口径与分段图分母不一致，导致展示结果可能误导排查。

---

## 4. 关键发现（按严重级别）

## [严重] F1：`usage_logs` 单条写入 SQL 与参数数量不一致，存在运行时写库失败风险

- 证据：`prepareUsageLogInsert` 已新增 4 个阶段参数，最终参数数为 44。
  - `backend/internal/repository/usage_log_repo.go:1190`
  - `backend/internal/repository/usage_log_repo.go:1222`
- 但 `createSingle` 仍是 40 列/40 占位符 SQL（未包含新增 4 列）。
  - `backend/internal/repository/usage_log_repo.go:279`
  - `backend/internal/repository/usage_log_repo.go:327`
- 已有测试直接失败验证该问题：
  - `TestUsageLogRepositoryCreateSyncRequestTypeAndLegacyFields`
  - `TestUsageLogRepositoryCreate_PersistsServiceTier`
  - 报错核心：`expected 40, but got 44 arguments`

影响：
- 单条写入路径（如事务路径、`request_id` 为空路径）可能直接写库失败，导致你想要的分段耗时“并非稳定入库”。

---

## [严重] F2：Copilot 路径没有实际写入 `upstream_latency_ms`，导致分段不完整且语义失真

- Copilot handler 读取 `OpsUpstreamLatencyMsKey` 并尝试计算响应阶段：
  - `backend/internal/handler/copilot_gateway_handler.go:232`
  - `backend/internal/handler/copilot_gateway_handler.go:607`
  - `backend/internal/handler/copilot_gateway_handler.go:945`
- 但 Copilot service 发起上游请求处未见 `SetOpsLatencyMs(..., OpsUpstreamLatencyMsKey, ...)`：
  - 请求发送点：
    - `backend/internal/service/copilot_gateway_service.go:155`
    - `backend/internal/service/copilot_gateway_service.go:674`
    - `backend/internal/service/copilot_gateway_service.go:838`

影响：
- Copilot 请求的 `upstream_latency_ms` 大概率长期为空。
- 前端看到的“上游阶段”缺失；`response_latency_ms` 也会被迫退化为整个 forward 时长，阶段边界不准。

---

## [高] F3：前端“总耗时”与分段阶段不是同一口径，时间轴百分比可能误导

- 前端以 `duration_ms` 作为总耗时分母：
  - `frontend/src/views/admin/ops/components/OpsLatencyBreakdownCard.vue:108`
  - `frontend/src/views/admin/ops/components/OpsLatencyBreakdownCard.vue:135`
- 但后端 `duration_ms`（usage_logs）来自 `ForwardResult.Duration`，起点是进入 `Forward`，不包含前置认证/路由等待：
  - `backend/internal/service/gateway_service.go:4008`
  - `backend/internal/service/gateway_service.go:4555`
  - `backend/internal/service/openai_gateway_service.go:1645`
  - `backend/internal/service/openai_gateway_service.go:2275`
- 同时 `auth_latency_ms`、`routing_latency_ms` 在 handler 侧计时，属于 `Forward` 之前阶段。

影响：
- 你看到的“总耗时 vs 分段占比”不是同一时间窗，可能出现阶段和总耗时对不上（甚至百分比被 clamp 到 100%）。
- 对运维排查会造成错误直觉。

---

## [中] F4：分段口径跨入口不一致，横向对比会失真

- `gateway_handler` 中，`auth_latency_ms` 在“用户并发等待 + 二次计费检查”之后才打点：
  - 用户槽位获取：`backend/internal/handler/gateway_handler.go:223`
  - 二次计费：`backend/internal/handler/gateway_handler.go:241`
  - auth 赋值：`backend/internal/handler/gateway_handler.go:249`
- `openai_gateway_handler` 与 `copilot_gateway_handler` 中，`auth_latency_ms` 在用户并发等待之前就打点：
  - `backend/internal/handler/openai_gateway_handler.go:200`
  - `backend/internal/handler/copilot_gateway_handler.go:174`

影响：
- 同样是“auth_latency_ms”，不同入口含义不同，平台/路径横向比较不可靠。

---

## [中] F5：错误请求虽然后端有阶段字段，但请求排查错误详情页未展示这些字段

- 错误详情 API 类型已包含阶段字段：
  - `frontend/src/api/admin/ops.ts:1021`
  - `frontend/src/api/admin/ops.ts:1025`
- 但 `OpsErrorDetailPanel` 页面未渲染这些字段（仅展示 request/status/message/body 等）。
  - `frontend/src/views/admin/ops/components/OpsErrorDetailPanel.vue:1`

影响：
- 失败请求是运维排查重点，但前端缺少对应分段耗时展示，和你的目标不一致。

---

## [低] F6：通用 usage 查询链路未补齐新字段读取

- `usageLogSelectColumns` 仍未包含新增 4 个阶段字段：
  - `backend/internal/repository/usage_log_repo.go:31`
- `scanUsageLog` 也未扫描这 4 个字段：
  - `backend/internal/repository/usage_log_repo.go:3933`

影响：
- 虽然 request-inspect 专用查询已补字段，但其他使用 usage repository 的读取路径拿不到新字段，数据可见性不一致。

---

## 5. 你提的“当前耗时到底是什么”——基于代码的明确回答

当前请求排查列表中的 `duration_ms`：

- 成功请求来自 `usage_logs.duration_ms`（`ops_repo_request_details` 把它直接取出）。
  - `backend/internal/repository/ops_repo_request_details.go:95`
- 这个值不是“用户请求进入 sub2api 到返回”的全链路耗时；它主要是 **Forward 阶段耗时**（上游请求 + 响应处理）。
  - `backend/internal/service/gateway_service.go:4008`
  - `backend/internal/service/gateway_service.go:4555`

因此：
- **不是纯“上游等待响应”**（因为还包含响应传输/读取等）。
- **也不是完整“用户到 sub2api 的端到端总耗时”**（认证、排队、路由等待通常不在 `duration_ms` 内）。

## 6. 需求满足度评估

- “耗时定义清晰可解释”：**部分满足**（代码里可追溯，但页面上仍容易误解）。
- “流程更详细展示”：**部分满足**（成功请求新增分解卡片）。
- “每个阶段入库并前端展示”：**未满足**（至少存在 F1/F2/F5 三个阻断问题）。
- “可精确辅助运维定位”：**当前不达标**（分母口径与阶段不一致 + Copilot 上游阶段缺失）。

## 7. 验证记录

已执行：

```bash
cd backend && go test ./internal/handler ./internal/service ./internal/repository
```

结果：
- `internal/handler` 通过
- `internal/service` 通过
- `internal/repository` 失败（核心失败即 F1）

