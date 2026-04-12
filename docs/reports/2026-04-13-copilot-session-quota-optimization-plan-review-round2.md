# Copilot Session Quota Optimization 方案评审报告（Round 2）

## 基本信息
- 复审日期：2026-04-13
- 复审对象：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md`
- 对应上一轮报告：`docs/reports/2026-04-13-copilot-session-quota-optimization-plan-review.md`
- 复审类型：H1/H2 修订后的 follow-up review
- 复审方式：静态审阅更新后的计划文档，并对照当前 Copilot gateway 的真实调用点与现有测试面核对可实施性

## 复审结论（摘要）
- 上一轮指出的两个阻断问题，这一版都已经补对了：
  - session cache key 明确加上 `account.ID` namespace，跨账号污染问题已收敛
  - `ForwardResponses` 已被纳入真实修改范围，且 `result.Initiator` / handler analytics 的一致性也写进了任务
- 但当前计划里仍有 1 个新的阻断级问题：Task 4 所谓的“集成测试”仍然只是手工复制 session cache 逻辑，不能锁住真实的六个调用点，也锁不住这轮刚修过的 `ForwardResponses` 链路。
- 当前结论仍为：`REQUEST CHANGES`。

---

## 主要发现

### HIGH

#### H1. Task 4 的“集成测试”没有走真实转发函数，只是在测试里手工复制了一遍 cache 覆盖逻辑，无法为实际调用点提供回归保护

#### 问题
这轮文档已经把真实修改点补到了 service 层：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:446`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:579`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:640`

也就是当前实现计划实际要覆盖的是这些真实链路：

- `forwardChatCompletionsDirect`：`backend/internal/service/copilot_gateway_service.go:209`
- `forwardChatCompletionsViaResponses`：`backend/internal/service/copilot_gateway_service.go:355`
- `forwardChatCompletionsViaMessages`：`backend/internal/service/copilot_gateway_service.go:513`
- `ForwardResponses`：`backend/internal/service/copilot_gateway_service.go:1789`
- `ForwardMessages`：`backend/internal/service/copilot_gateway_service.go:1950`
- `forwardMessagesViaResponses`：`backend/internal/service/copilot_gateway_service.go:2148`

但 Task 4 里的新增测试并没有调用这些真实函数路径，而是在测试里重新写了本地 helper：

- `applySessionCache(...)`：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:749`
- `applySession(...)`：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:807`
- `apply(...)`：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:846`

这些 helper 只是把 `copilotInitiator(...) + markAndCheckSeen(...)` 在测试里手抄了一遍，并没有验证：

1. 真实 service 方法是否真的在每个调用点都接入了 session cache
2. `ForwardResponses` 是否真的把 `initiatorResp` 用到了上游 header
3. `result.Initiator` 是否真的在真实返回路径上被填充
4. handler 侧改成 `result.Initiator` 后，链路是否与真实上游值一致

#### 风险
- 这类测试即使全绿，也不能证明真实链路没有漏改。
- 上一轮 H2 的根因正是“文档以为覆盖了 Responses，实际上真实 `ForwardResponses` 没改到”；而当前 Step 4 仍然无法防住同类问题复发。
- 如果实现时漏掉任意一个真实调用点，尤其是 `ForwardResponses` 或 `forwardMessagesViaResponses`，这组测试仍然可能全部通过。

#### 建议
- 不要把 helper-closure 测试当作集成测试。
- 直接复用当前已经存在的 HTTP 层 header 回归测试骨架：
  - `TestXInitiatorHeader_ChatCompletions`：`backend/internal/service/copilot_gateway_service_test.go:1544`
  - `TestXInitiatorHeader_ResponsesEndpoint`：`backend/internal/service/copilot_gateway_service_test.go:1618`
  - `TestXInitiatorHeader_MessagesEndpoint`：`backend/internal/service/copilot_gateway_service_test.go:1702`
- 在这些真实端到端测试上追加 session 场景：
  - 同一 session 第一次请求应为 `"user"`
  - 同一 session 第二次请求应为 `"agent"`
  - 不同 `account.ID` 即使 raw session key 相同也必须互不污染
- 至少在真实 service 调用后同时断言两件事：
  - 上游捕获到的 `X-Initiator`
  - 返回的 `result.Initiator`

---

## 非阻断提醒

### MEDIUM

#### M1. cache 生命周期描述前后不一致，容易让实现落到与文档注释不一致的版本

Step 0.3 的实现注释明确写的是：

- `sync.Map`
- “不启动 background goroutine”
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:217`

但紧接着示例代码实际用的是：

- `sync.Mutex + map`
- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:221`

随后 Task 1.2 又要求在构造函数里启动常驻 ticker goroutine：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:379`

这不是功能性阻断，但会让实现者在“lazy eviction”与“后台清理线程”之间来回摇摆。建议文档只保留一种明确策略，并把注释、示例代码、生命周期说明写成同一个版本。

### MEDIUM

#### M2. Task 2 / Task 3 的调用点计数仍然有歧义，建议把 `forwardMessagesViaResponses` 单独列出来，避免再次出现“文档写了，实施时漏了一处”的问题

当前文档在 Task 2 开头写的是“五处位置”：

- `docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:441`

但同一节后面又追加：

- `forwardMessagesViaResponses` 也要改：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:579`

Task 3.2 也是同类问题：

- 标题写“五个转发函数”：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:636`
- 但列举时实际上又把 `ForwardMessages` 和 `forwardMessagesViaResponses` 分开写了：`docs/superpowers/plans/2026-04-13-copilot-session-quota-optimization.md:640`

这轮虽然正文已经把额外路径补出来了，但建议把计数和列表彻底对齐，例如直接改成“六个调用点 / 六个返回路径”，并把两个 Messages 分支都单独列在 headline 里，减少执行时遗漏。

---

## 复审结论

### Recommendation
`REQUEST CHANGES`

### 原因
- H1/H2 这轮修订方向是正确的，主链路范围也终于写完整了。
- 但新的测试设计还没有锁住真实 service 调用点，这会让这轮最关键的修复点缺少回归护栏。
- 先把 Task 4 改成真正的 endpoint-level 回归测试后，这份计划才适合进入实现阶段。

### 备注
- 本次仍为静态 review。
- 未修改业务代码。
- 未执行构建、类型检查或测试，因为复审对象仍是计划文档而非已实现代码。
