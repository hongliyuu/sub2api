# Copilot Session Quota Optimization 方案评审报告（Round 3）

## 基本信息
- 复审日期：2026-04-13
- 复审对象：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md`
- 对应上一轮报告：`docs/reports/2026-04-13-copilot-session-quota-optimization-plan-review-round2.md`
- 复审类型：Task 4 / M1 / M2 修订后的 follow-up review
- 复审方式：静态审阅更新后的计划文档，并对照当前 Copilot gateway 的真实分支结构与现有测试底座核对覆盖面

## 复审结论（摘要）
- 上一轮指出的 “helper-closure 伪集成测试” 这次已经修掉了：Task 4 确实改成了调用真实 service 入口，并开始同时断言上游 `X-Initiator` 与 `result.Initiator`。
- M1 这轮也基本收干净了：cache 实现注释、数据结构、后台清理策略现在是一致的。
- 但当前计划仍有 1 个新的阻断级问题：Task 4 只锁住了 3 个顶层入口，没有真正覆盖 Task 2 / Task 3 声称要修改的 6 个分支调用点。尤其 `forwardChatCompletionsViaResponses`、`forwardChatCompletionsViaMessages`、`forwardMessagesViaResponses` 仍然没有 session-cache 级别的回归护栏。
- 当前结论仍为：`REQUEST CHANGES`。

---

## 主要发现

### HIGH

#### H1. Task 4 虽然改成了真实 service 调用，但实际只覆盖了 3 个顶层入口，仍然没有锁住 Task 2 声称的 6 个分支调用点

#### 问题
这版文档已经把 scope 明确写成了 6 个调用点：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:65`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:440`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:636`

也就是计划明确要求接入 session cache / `Initiator` 填充的调用点包括：

1. `forwardChatCompletionsDirect`
2. `forwardChatCompletionsViaResponses`
3. `forwardChatCompletionsViaMessages`
4. `ForwardResponses`
5. `ForwardMessages`
6. `forwardMessagesViaResponses`

但 Task 4 新写的 3 个测试，实际只会触发 3 个顶层入口：

- `TestCopilotSessionCache_ChatCompletions` → `ForwardChatCompletions`
- `TestCopilotSessionCache_ResponsesEndpoint` → `ForwardResponses`
- `TestCopilotSessionCache_MessagesEndpoint` → `ForwardMessages`

对应文档位置：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:728`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:750`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:856`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:972`

问题在于，这 3 个测试的输入设计并没有把顶层入口导向那些内部变体分支：

1. `TestCopilotSessionCache_ChatCompletions` 使用的是普通文本 body，并给账号设置了 `base_url`，它只会走 direct path，不会进入：
   - `forwardChatCompletionsViaResponses`：`backend/internal/service/copilot_gateway_service.go:355`
   - `forwardChatCompletionsViaMessages`：`backend/internal/service/copilot_gateway_service.go:513`

2. `TestCopilotSessionCache_MessagesEndpoint` 还把 `/models` stub 成了只有 `["/chat/completions"]`：
   - `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:980`
   这意味着它只会走 `ForwardMessages` 默认 chat/completions 分支，不会进入：
   - `forwardMessagesViaResponses`：`backend/internal/service/copilot_gateway_service.go:2147`

所以现在的 Task 4 虽然比上一版好很多，但它真正锁住的其实只是：

- `forwardChatCompletionsDirect`
- `ForwardResponses`
- `ForwardMessages` 默认分支

而不是文档 headline 里宣称的完整 6 处。

#### 风险
- 如果实现时漏改了 `forwardChatCompletionsViaResponses`、`forwardChatCompletionsViaMessages` 或 `forwardMessagesViaResponses` 中任意一个，当前 Task 4 依然可能全绿。
- 这会让“文件上传 / responses-only model / messages bridge”这些更边缘但真实存在的路径继续保留 quota 归因偏差。
- 同样地，这 3 个内部分支上的 `result.Initiator` 漏填也不会被当前测试发现。

#### 建议
- 如果文档 scope 继续坚持“六个调用点”，Task 4 也必须补齐对应的分支级测试，而不是只测 3 个顶层入口。
- 当前仓库已经有可以复用的分支测试底座，最适合直接在这些测试上追加 session 场景断言：
  - `TestForwardChatCompletions_FilePartsViaResponsesStreaming`：`backend/internal/service/copilot_gateway_service_test.go:1156`
  - `TestForwardChatCompletions_FilePartsViaResponsesNonStreaming`：`backend/internal/service/copilot_gateway_service_test.go:1253`
  - `TestForwardChatCompletions_UnsupportedAPIForModel_FallbackToChatCompletions`：`backend/internal/service/copilot_gateway_service_test.go:1321`
- 建议至少新增或改造以下 3 类测试：
  1. **ChatCompletions → viaResponses 分支**
     - 用 file parts + `/models => ["/responses"]`
     - 连续两次同 session 断言第一次 `"user"`、第二次 `"agent"`
     - 同时断言 `result.Initiator`
  2. **ChatCompletions → viaMessages 分支**
     - 用 file parts + 支持 `/v1/messages` 的模型响应
     - 断言 session cache 与 `result.Initiator`
  3. **Messages → forwardMessagesViaResponses 分支**
     - 让 `/models` 返回 responses-only
     - 断言 metadata.user_id 驱动的 session cache 与 account isolation

如果团队不想补这些分支测试，那就要反过来收紧文档 scope，把 Task 2 / Task 3 明确降为“只保证 3 个入口级路径”，否则当前“6 处都已被测试锁住”的表述仍然过宽。

---

## 非阻断提醒

### MEDIUM

#### M1. M2 还没完全收干净，Task 3 的若干标题和注释仍残留旧计数

虽然 Task 2 headline 已经改成“六个调用点”，但 Task 3 里还残留一些旧计数文案：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:609`
  - 这里仍写着“`四个转发函数` 的 return 语句”
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:663`
  - 这里仍写“检查其余四个函数”
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:665`
  - checkbox 文案仍写“`五个函数` 的返回路径补填 Initiator”

这些都不是功能性阻断，但会继续增加执行时的歧义。建议把 Task 3 整段也统一改成“六个调用点 / 六个返回路径”。

---

## 复审结论

### Recommendation
`REQUEST CHANGES`

### 原因
- 上一轮指出的伪集成测试问题，这次已经修到位了，说明计划整体是在收敛的。
- 但当前 Task 4 仍然没有覆盖文档自己声称的全部 6 个修改分支，尤其 3 个内部分支还缺真实 session-cache 回归测试。
- 先把 Task 4 的覆盖面和 Task 2 / Task 3 的 scope 对齐，再进入实现会更稳。

### 备注
- 本次仍为静态 review。
- 未修改业务代码。
- 未执行构建、类型检查或测试，因为复审对象仍是计划文档而非已实现代码。
