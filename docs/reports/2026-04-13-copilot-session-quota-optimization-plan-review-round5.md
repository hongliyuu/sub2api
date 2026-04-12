# Copilot Session Quota Optimization 方案评审报告（Round 5）

## 基本信息
- 复审日期：2026-04-13
- 复审对象：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md`
- 对应上一轮报告：`docs/reports/2026-04-13-copilot-session-quota-optimization-plan-review-round4.md`
- 复审类型：测试 fixture 修订后的最终 follow-up review
- 复审方式：静态审阅更新后的计划文档，并对照当前 session key 解析契约与分支选择逻辑核对可实施性

## 复审结论（摘要）
- Round 4 指出的 `ViaMessagesBranch` session fixture 非法问题，这一轮已经修正：
  - `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:1220`
  - 现在使用的 session UUID `aaaabbbb-cccc-dddd-eeee-ffff11112222` 符合当前 `ParseMetadataUserID` 的 legacy 正则契约：`backend/internal/service/metadata_userid.go:24`
- 我同时复核了文档内其余 `account__session_...` fixtures，未再发现新的格式问题。
- 之前几轮指出的范围定义、Responses 链路、测试覆盖和计数一致性问题，到这一版都已收敛。
- 本轮未发现新的阻断级问题，计划文档已达到可进入实现阶段的质量。

---

## 本轮核验点

### 1. `ViaMessagesBranch` fixture 已修正为合法 session key

上一轮的问题是 `TestCopilotSessionCache_ViaMessagesBranch` 里的 `sessionUser` 含有非十六进制字符 `g`，导致：

- `ParseMetadataUserID(...)` 无法匹配
- `extractSessionKeyFromOpenAIBody(...)` 返回空字符串
- session cache 实际不会命中

这一轮对应位置已修正为：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:1220`

并且当前解析契约仍然是：

- `backend/internal/service/metadata_userid.go:24`

所以这条测试数据现在能够真实参与 session cache 验证。

### 2. 其余 fixtures 也与 legacy 解析契约一致

我检查了本计划文档里所有 `user_{64hex}_account_{...}_session_{uuid}` 形式的 fixture：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:161`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:177`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:778`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:883`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:1007`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:1113`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:1220`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:1323`

在我审查到的范围内，这些 fixture 的 64 位前缀和 36 位 session segment 都符合当前正则约束，没有再看到新的输入契约问题。

### 3. 分支级测试与真实路由条件现已对齐

本轮再次复核后，Task 4 的 6 个测试与真实代码的分支选择条件是吻合的：

- file parts + `/models => ["/responses"]` + 无 `base_url`
  - 对应 `forwardChatCompletionsViaResponses`
  - 见 `backend/internal/service/copilot_gateway_service.go:190`
- file parts + `/models => ["/v1/messages"]` + 无 `base_url`
  - 对应 `forwardChatCompletionsViaMessages`
  - 见 `backend/internal/service/copilot_gateway_service.go:195`
- `ForwardMessages` + `/models => ["/responses"]` only
  - 对应 `forwardMessagesViaResponses`
  - 见 `backend/internal/service/copilot_gateway_service.go:1767`
  - 见 `backend/internal/service/copilot_gateway_service.go:2045`

结合前几轮已补齐的：

- `ForwardResponses` 真正接入 session cache
- `result.Initiator` 统一回传
- Task 2 / Task 3 的“六个调用点”计数对齐

当前计划在静态层面已经闭环。

---

## 残余提醒

以下不构成当前计划的阻断项，但实现阶段仍建议注意：

1. 端到端测试落地时，尽量复用现有 `TestXInitiatorHeader_*` / file-routing 测试夹具，减少 SSE 事件手写错误。
2. 实现完成后，优先核验 `result.Initiator` 与实际发出的 `X-Initiator` 是否在 6 个调用点上都一致，而不只看 header。

---

## 复审结论

### Recommendation
`APPROVE`

### 原因
- Round 4 的唯一阻断问题已经修复。
- 本轮未发现新的实现阻断点。
- 这份计划现在可以进入代码实施阶段。

### 备注
- 本次仍为静态 review。
- 未修改业务代码。
- 未执行构建、类型检查或测试，因为复审对象仍是计划文档而非已实现代码。
