# GitHub Copilot 1M Context 可行性研究报告

**日期**：2026-04-07  
**主题**：`claude-opus-4.6-1m` 模型在 GitHub Copilot 上的可行性分析  
**结论**：个人 Pro/Pro+ 账号当前不支持，1M context 为 Enterprise/特定计划专属功能

---

## 背景

有用户反映可以通过 GitHub Copilot 使用 `claude-opus-4.6-1m`（1M token 上下文窗口）模型。本文档记录对该说法的系统性验证过程和最终结论。

---

## 一、模型 ID 真实性

### 结论：`claude-opus-4.6-1m` 是真实存在的 Copilot model ID

**证据来源**：

- [anomalyco/models.dev#1292](https://github.com/anomalyco/models.dev/issues/1292)（2026-03-29）：VS Code 的 GitHub Copilot 模型选择器中确实显示 "Claude Opus 4.6 (1M context)" 作为独立可选项
- [anomalyco/opencode#12338](https://github.com/anomalyco/opencode/issues/12338)：有用户成功通过自定义配置使用该 model ID 发起请求
- [openclaw/openclaw#60174](https://github.com/openclaw/openclaw/issues/60174)：实测数据显示 `claude-opus-4.6-1m` 在 Copilot Enterprise API 上可正常调用

### 它的本质

`claude-opus-4.6-1m` **不是独立的后端模型**，而是 `claude-opus-4.6` 在 GitHub 服务端配置了更高 `max_prompt_tokens` 配额的变体：

- 普通变体（`claude-opus-4.6`）：`max_prompt_tokens` = 128K–200K（取决于账号 plan）
- 1M 变体（`claude-opus-4.6-1m`）：`max_prompt_tokens` ≈ 900K+

---

## 二、直接测试结果

### 测试环境

- **账号类型**：GitHub Copilot Pro+（$39/月个人订阅）
- **Token SKU**：`plus_monthly_subscriber_quota`（token 内明文字段）
- **测试端点**：`https://api.githubcopilot.com/chat/completions`
- **测试地区**：美国（排除地区限制因素）

### 测试过程

**步骤一**：获取 Copilot token

```bash
COPILOT_TOKEN=$(curl -s \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "x-github-api-version: 2025-04-01" \
  https://api.github.com/copilot_internal/v2/token | jq -r .token)
```

**步骤二**：测试 `claude-opus-4.6-1m` 请求（用 `copilot-developer-cli` integration-id）

```bash
curl -s \
  -H "Authorization: Bearer $COPILOT_TOKEN" \
  -H "copilot-integration-id: copilot-developer-cli" \
  -H "Content-Type: application/json" \
  https://api.githubcopilot.com/chat/completions \
  -d '{"model": "claude-opus-4.6-1m", "messages": [...]}'
```

**结果**：
```json
{"error": {"message": "The requested model is not supported.", "code": "model_not_supported"}}
```

**步骤三**：查询 `/models` 接口实际返回的模型列表

Pro+ 个人账号返回的 Claude Opus 相关模型：
- `claude-opus-4.6-fast`
- `claude-opus-4.6`
- `claude-opus-4.5`

`claude-opus-4.6-1m` **未出现在列表中**。

**步骤四**：查询 `claude-opus-4.6` 的完整 capabilities

```bash
curl -s "https://api.githubcopilot.com/models" \
  -H "Authorization: Bearer $COPILOT_TOKEN" \
  ... | jq '.data[] | select(.id == "claude-opus-4.6")'
```

关键字段返回：

```json
{
  "capabilities": {
    "limits": {
      "max_context_window_tokens": 200000,
      "max_prompt_tokens": 128000,
      "max_output_tokens": 64000,
      "max_non_streaming_output_tokens": 16000
    }
  }
}
```

---

## 三、根本原因分析

### Copilot API 的 context 控制机制

GitHub Copilot 的 `/models` 接口在 `capabilities.limits` 字段中暴露真实的 token 限制，包含三个关键值：

| 字段 | 含义 |
|------|------|
| `max_context_window_tokens` | 模型支持的总上下文上限（输入+输出） |
| `max_prompt_tokens` | **Copilot 实际允许的最大输入** |
| `max_output_tokens` | 最大输出 token 数 |

**`max_prompt_tokens` 是 GitHub 服务端按订阅计划硬编码的**，客户端无法通过任何 header 或参数修改。

### 各 Plan 的实测 max_prompt_tokens（`claude-opus-4.6`）

| 订阅计划 | max_context_window_tokens | max_prompt_tokens | 来源 |
|---------|--------------------------|-------------------|------|
| Trial/免费试用 | 200K | 144K | opencode#20317 社区数据 |
| Copilot Pro ($10/月) | 200K | < 128K | copilot-cli#2401 社区数据 |
| **Copilot Pro+ ($39/月)** | **200K** | **128K** | **本次实测** |
| Copilot Enterprise Pro | 200K+ | 200K+ | opencode#20317 keith-pw 截图 |
| 1M 变体专用 plan | 1M | ~900K | 仅特定高级计划，未公开 |

---

## 四、为什么有人说"可以用 1m"

| 来源说法 | 实际情况 |
|---------|---------|
| "VS Code 里能看到 1M 模型" | 显示在 UI 里但标记为"需要升级" |
| "Pro+ 可以用" | Pro+ 用户测试也返回 `model_not_supported` |
| "用 `copilot-developer-cli` integration-id 就行" | 改 integration-id 无法绕过 plan 限制 |
| opencode 里有人配置成功 | 通过自定义 provider 配置 + plugin hook，把 `claude-opus-4.6-1m` 当本地别名，实际底层走的是不同路径 |
| 1M context 通过 `context-1m-2025-08-07` beta header 触发 | 这是 **Anthropic 直连 API** 的机制，Copilot 渠道不适用 |

---

## 五、关键区分：两种"1M context"机制

### 机制 A：Anthropic 直连 API（有效）

```
模型: claude-opus-4-6
Header: anthropic-beta: context-1m-2025-08-07
```

直接调用 Anthropic API 时，发送这个 beta header 可以激活 1M context 窗口。该机制对 API Key 账号有效，对 OAuth/subscription token 有限制（[senara-solutions/mika#322](https://github.com/senara-solutions/mika/issues/322)）。

### 机制 B：GitHub Copilot 渠道（受限）

```
模型: claude-opus-4.6-1m（独立 model ID）
```

Copilot 自己维护的 model ID 体系，`max_prompt_tokens` 由 GitHub 服务端根据你的订阅 plan 决定，**无法通过任何客户端参数修改**。当前只对 Enterprise 或特定高级 plan 开放。

---

## 六、对 sub2api 集成的影响与建议

### 当前状态

- **Copilot 个人账号（Pro/Pro+）**：`claude-opus-4.6` 的 `max_prompt_tokens` 最高 128K–200K，`claude-opus-4.6-1m` 不可用
- **Copilot Enterprise 账号**：可能支持 200K+，1M 变体取决于具体配置

### 实现建议

**1. 动态读取 capabilities.limits，不要硬编码**

每次获取账号可用模型列表时，解析 `capabilities.limits.max_prompt_tokens` 字段，用实际值告知客户端，而非依赖 model ID 判断：

```go
// 从 /models 响应中读取实际限制
type ModelCapabilities struct {
    Limits *ModelLimits `json:"limits,omitempty"`
}

type ModelLimits struct {
    MaxContextWindowTokens int `json:"max_context_window_tokens"`
    MaxPromptTokens        int `json:"max_prompt_tokens"`
    MaxOutputTokens        int `json:"max_output_tokens"`
}
```

**2. 按 `/models` 实际返回列表决定暴露的模型**

若某账号的 `/models` 返回了 `claude-opus-4.6-1m`，则暴露；否则不暴露。无需静态配置。

**3. 1M context 的可行替代路径**

| 渠道 | 方式 | 成本 |
|------|------|------|
| Anthropic 直连 API Key | `anthropic-beta: context-1m-2025-08-07` header | 按量计费 |
| AWS Bedrock | `anthropic_beta: ["context-1m-2025-08-07"]` | 按量计费 |
| Copilot Business/Enterprise | 使用 `claude-opus-4.6-1m` model ID | 订阅费用更高 |

---

## 七、参考资料

| 来源 | 内容 |
|------|------|
| [anomalyco/opencode#12338](https://github.com/anomalyco/opencode/issues/12338) | 1M context 在 opencode 中的实现讨论，beta header 机制 |
| [anomalyco/models.dev#1292](https://github.com/anomalyco/models.dev/issues/1292) | VS Code 显示 1M 变体，模型 ID 确认 |
| [github/copilot-cli#2401](https://github.com/github/copilot-cli/issues/2401) | Pro+ 用户确认无法使用 1M 模型 |
| [openclaw/openclaw#60174](https://github.com/openclaw/openclaw/issues/60174) | Enterprise 账号实测数据，api 格式对比 |
| [anomalyco/opencode#20317](https://github.com/anomalyco/opencode/issues/20317) | 不同 plan context window 差异，动态读取建议 |
| [senara-solutions/mika#322](https://github.com/senara-solutions/mika/issues/322) | OAuth token + 1M context 不兼容分析 |
| [orgs/community/discussions/186340](https://github.com/orgs/community/discussions/186340) | Copilot API limits 字段（context_window vs max_prompt）解析 |
| [Anthropic 官方文档](https://docs.anthropic.com/en/docs/build-with-claude/context-windows) | context-1m-2025-08-07 beta header 说明 |
