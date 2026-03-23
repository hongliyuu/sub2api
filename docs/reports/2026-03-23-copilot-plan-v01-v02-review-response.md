# Codex Review 回应报告：v0.1 → v0.2 修订说明

- 回应对象: `docs/reports/2026-03-23-copilot-implementation-plan-review.md`
- 修订文档: `docs/copilot-improvements/v0.2-implementation-plan.md`
- 日期: 2026-03-23
- 结论: Codex 的所有发现均确认属实，5 条问题全部已在 v0.2 中修订

---

## 核实结果与修订情况

### H1：P1 改动范围被低估（已确认，已修订）

**核实结论：Codex 正确。**

v0.1 描述有误："`hasImage` 在 `translateAnthropicToOpenAI` 内计算"。实际情况：

- `translateAnthropicToOpenAI`（第 267 行）调用 `buildOpenAIMessages`
- `buildOpenAIMessages`（第 293 行）调用 `handleAnthropicUserMessage`
- **`hasImage` 是 `handleAnthropicUserMessage` 的局部变量**（第 478 行），不向调用方返回
- `translateAnthropicToOpenAI` 只返回 `([]byte, error)`，完全不持有 `hasImage`

要将 `hasImage` 冒泡到 `ForwardMessages`，方案 A（修改返回签名）需要穿透三层：
`handleAnthropicUserMessage → buildOpenAIMessages → translateAnthropicToOpenAI → ForwardMessages`，改动量远超"2处"。

**v0.2 修订**：
- 废弃方案 A，改为方案 B——在 `ForwardMessages` 中独立扫描原始 `anthropicBody`（`containsImageBlock` 函数），不修改翻译层签名
- P1 工作量从 S 上调为 M（拆分为 P1-A + P1-B 两个子任务）

---

### H2：含图消息可能在 merge 逻辑中丢失 image part（已确认，已修订）

**核实结论：Codex 正确，且此 bug 独立于 Vision Header 问题，必须单独修复。**

完整链路分析：

1. `handleAnthropicUserMessage`（第 461 行）：检测到图片时将 `content` 设为 `[]openAIContentPart`（含 image_url），`hasImage = true`
2. `buildOpenAIMessages`（第 308-317 行）：将翻译后的消息传给 `sanitizeOpenAIMessages`
3. `sanitizeOpenAIMessages` Step 3（第 381-414 行）：`canMerge` 条件只检查 `ToolCalls`，不检查 content 是否为图片格式
4. `canMerge = true` 时调用 `contentToString`（第 418 行）：对 `[]openAIContentPart` 只提取 text，**image_url 被丢弃**
5. 丢弃后 `prev.Content` 被覆盖为纯字符串，图片永久丢失

实际触发条件：连续两条 user 消息（例如 Claude Code 发送 `<available-deferred-tools>` 注入 + 含图的实际用户消息），第二条含图，merge 时图片被字符串化。

**v0.2 修订**：
- P1 新增子任务 **P1-A**：在 `sanitizeOpenAIMessages` 的 `canMerge` 中增加 `!hasImageContentPart(cur.Content) && !hasImageContentPart(prev.Content)` 条件，阻止含图片消息参与字符串化合并
- 新增 `hasImageContentPart` 辅助函数（纯函数，检查 `[]openAIContentPart` 中是否含 `image_url` type）
- **此 bug 应优先修复**，它独立于 Vision Header 问题，且直接导致图片内容丢失

---

### M1：P2 使用单一全局缓存桶，可能跨 group 污染模型列表（已确认，已修订）

**核实结论：Codex 正确。**

`Models()` handler（第 304 行）：
```go
account, err := h.gatewayService.SelectAccountForModelWithExclusions(ctx, apiKey.GroupID, "", "", nil)
```
账号选择是按 `apiKey.GroupID` 的。不同 group 账号套餐不同，模型列表可能不同（例如 enterprise group 有更多模型）。v0.1 的单桶缓存会把 A group 的模型列表返回给 B group。

**v0.2 修订**：
- 缓存改为 `map[int64]*copilotModelCacheEntry`，以 `groupID` 为键
- 新增 `copilotModelCacheEntry` 类型（含 `data`、`cachedAt`、`fromUpstream` 字段）
- 缓存读写方法均接受 `groupID` 参数

---

### M2：P2 降级策略把失败语义改成 200，存在可观测性回归（已确认，已修订）

**核实结论：Codex 正确。**

v0.1 方案在账号不可用时直接返回 HTTP 200（静态默认列表），完全掩盖了服务故障。原有代码（`copilot_gateway_handler.go:307/316`）在账号不可用时返回 503，上游失败时返回 502，这些信号对监控是有价值的。

**v0.2 修订**：采用分级降级策略：

| 场景 | 响应 | 原则 |
|------|------|------|
| 有 stale 缓存时（任意故障场景） | 200 + stale 数据 + warn 日志 | 模型列表基本不变，降级合理 |
| 无任何缓存 + 无可用账号 | 503（保留原语义） | 真正不可服务 |
| 无任何缓存 + 有账号但上游失败 | 200 + 静态默认列表 + warn 日志 | 兜底，记录 warn 供监控感知 |

关键修订：只有在**完全没有任何缓存**时才可能返回错误或静态列表。有缓存（含过期）时，优先降级而不是报错，同时保留 warn 日志使故障可被监控感知。

---

### M3：P3a 准备用推断值写 SupportedEndpoints（已确认，已修订）

**核实结论：Codex 正确。**

v0.1 写道"上面的值是基于 OpenAI o-series 的已知特征推断的，不保证准确，需要实测后填写"——但仍然把推断值作为"步骤一"放进了实施方案，有引导"先写再测"的风险。

Codex 指出的具体危害：一旦 P3b 依赖该字段做请求前校验，错误标注（例如把支持 `/chat/completions` 的模型标记为只支持 `/responses`）会导致合法请求被误拦截。

**v0.2 修订**：
- P3a 前置步骤改为**先采集真实数据**（curl 调用 `GET /models`，记录每个模型的 `supported_endpoints` 实际值）
- 将采集结果记录到 `docs/copilot-improvements/supported-endpoints-data.md`
- 只有在数据确认后才写入 `DefaultModels`
- 代码文件中**不出现任何推断值**

---

## v0.1 中确认正确的部分（保持不变）

以下 v0.1 内容经核实无误，v0.2 保留：

1. `ForwardMessages()` 当前对 `CopilotHeaders` 传 `false`，Vision Header 未触发（`service.go:791`）✓
2. `ForwardResponses()` 已固定使用 `copilot.CopilotAPIBase`，不受 `plan_type`/`base_url` 影响（`service.go:647`）✓
3. `Models()` 每次实时调用上游，无缓存（`handler:311`）✓
4. `Model` struct 的 `SupportedEndpoints` 字段存在但 `DefaultModels` 未配置（`types.go:147/156`）✓
5. `/responses` 的 URL 路由问题已在代码中正确处理，P3 的核心无需修改 ✓

---

## 各问题处置汇总

| 问题 | Codex 评级 | 核实结果 | v0.2 处置 |
|------|-----------|---------|----------|
| H1：hasImage 冒泡链路比预期长 | 高 | 确认 | 改用方案 B，不修改翻译层签名，工作量 S→M |
| H2：图片在 merge 中丢失 | 高 | 确认，**独立 bug** | 新增 P1-A 子任务，修复 `sanitizeOpenAIMessages` |
| M1：单桶缓存跨 group 污染 | 中 | 确认 | 改为 `map[groupID]` 分组缓存 |
| M2：降级 200 掩盖故障 | 中 | 确认 | 有 stale 缓存时降级 + warn 日志；无缓存时保留 5xx |
| M3：推断值进代码 | 中 | 确认 | P3a 前置改为先采集真实数据，禁止推断值入库 |

---

*本报告对应 v0.2 实施方案，如有进一步审阅意见请基于 v0.2 提出*
