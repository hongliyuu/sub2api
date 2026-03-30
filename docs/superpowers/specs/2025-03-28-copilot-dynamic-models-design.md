# Copilot Claude Code settings.json 模型字段动态化

**日期：** 2025-03-28
**状态：** 已批准
**文件范围：** `frontend/src/components/keys/UseKeyModal.vue`（唯一改动文件）

---

## 背景与问题

`UseKeyModal.vue` 的 Copilot 平台 Claude Code tab，调用 `generateAnthropicFiles(url, apiKey, richSettings=true)`，在生成的 `~/.claude/settings.json` 中写入以下硬编码模型字段：

```json
"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "claude-haiku-4-5",
"ANTHROPIC_DEFAULT_OPUS_MODEL":   "claude-opus-4-6",
"ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-4-6",
"ANTHROPIC_MODEL":                "claude-sonnet-4-6"
```

这些字段随 GitHub Copilot 上游模型迭代而过时——例如新增 `claude-sonnet-4-7`、废弃旧版本后，用户复制的配置仍指向旧模型名，导致请求失败或绕过新模型。

后端已有 `GET /copilot/v1/models` 接口，使用平台 API Key 鉴权，内置 1 小时缓存与静态降级策略，返回该 group 下实际可用的 Copilot 模型列表。本设计利用此接口实现模型字段的动态生成，零后端改动。

---

## 目标

- `richSettings=true` 路径生成的 settings.json 中，haiku/sonnet/opus/model 四个字段从 `/copilot/v1/models` 动态获取，始终反映当前可用的最新模型
- 接口失败时无缝降级到现有硬编码值，不影响弹窗正常使用
- 不改动后端、i18n、类型定义或其他任何组件

---

## 设计

### 数据流

```
UseKeyModal 打开（watch: show === true）
  │
  └─ platform === 'copilot' ?
       ├─ 是 → loadCopilotModels()
       │         fetch `${copilotV1Base}/models`
       │         Authorization: Bearer ${apiKey}
       │         timeout: 5000ms
       │           ├─ 成功 → resolveModelRoles(ids) → copilotModels.value = { sonnet, opus, haiku }
       │           └─ 失败 → copilotModels.value = null（使用 fallback）
       └─ 否 → 跳过
  │
currentFiles computed（reactive）
  └─ richSettings 路径
       └─ 读取 copilotModels.value ?? HARDCODED_FALLBACK
            → 填充 settings.json 四个模型字段
```

### 新增状态

```typescript
// 动态获取的 Copilot Claude 模型角色映射，null 表示未加载或加载失败（使用 fallback）
const copilotModels = ref<{ sonnet: string; opus: string; haiku: string } | null>(null)
```

`loadingModels` 状态不需要，原因：`currentFiles` 是 computed，弹窗打开时立即使用 fallback 渲染，fetch 完成后自动重算并更新代码块，用户体验上是"配置悄悄更新"而非"等待加载"。

### fetch 实现

使用原生 `fetch` 而非 `apiClient`（后者走 JWT 认证拦截器，而 `/copilot/v1/models` 使用平台 API Key 鉴权）。

```typescript
async function loadCopilotModels(): Promise<void> {
  if (props.platform !== 'copilot' || !props.apiKey) return
  try {
    const baseRoot = (props.baseUrl || window.location.origin)
      .replace(/\/v1\/?$/, '').replace(/\/+$/, '')
    const copilotV1Base = `${baseRoot}/copilot/v1`

    const res = await fetch(`${copilotV1Base}/models`, {
      headers: { Authorization: `Bearer ${props.apiKey}` },
      signal: AbortSignal.timeout(5000),
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const json = await res.json()
    const ids: string[] = (json.data ?? []).map((m: { id: string }) => m.id)
    copilotModels.value = resolveModelRoles(ids)
  } catch {
    copilotModels.value = null
  }
}
```

**URL 构造说明：**
`props.baseUrl` 是外部 API 网关地址（如 `https://xxx.com` 或 `http://localhost:3000`），与 `apiClient` 的内部管理 API 路径 `/api/v1` 完全独立，因此使用原生 `fetch` 而非 `apiClient` 直接调用外部网关是正确做法。URL 构造逻辑与 `currentFiles` computed 中的 `copilotBase` 保持一致：`{baseRoot}/copilot/v1/models`，对应后端路由 `copilotV1.GET("/models", ...)`。

### 模型角色识别

```typescript
function resolveModelRoles(ids: string[]): { sonnet: string; opus: string; haiku: string } {
  const claudeIds = ids.filter(id => id.startsWith('claude-'))
  const pickLatest = (role: string): string | null =>
    claudeIds.filter(id => id.includes(role)).sort().at(-1) ?? null

  return {
    sonnet: pickLatest('sonnet') ?? 'claude-sonnet-4-6',
    opus:   pickLatest('opus')   ?? 'claude-opus-4-6',
    haiku:  pickLatest('haiku')  ?? 'claude-haiku-4-5',
  }
}
```

**版本排序依据：** 后端模型 ID 格式为 `claude-{role}-{major}-{minor}`（如 `claude-sonnet-4-6`），字典序 `sort()` 即可正确排出最新版本（`4-6` > `4-5` > `4`）。此逻辑足够稳定——只要后端不改变命名格式，此函数无需维护。

**各角色 fallback：**

| 角色   | 动态值来源              | fallback（接口失败时）  |
|--------|------------------------|------------------------|
| sonnet | 包含 `sonnet` 的最新 ID | `claude-sonnet-4-6`    |
| opus   | 包含 `opus` 的最新 ID   | `claude-opus-4-6`      |
| haiku  | 包含 `haiku` 的最新 ID  | `claude-haiku-4-5`     |

### watch 触发

```typescript
watch(
  () => props.show,
  (visible) => {
    if (visible) loadCopilotModels()
    // 弹窗关闭时不清空 copilotModels，下次打开前会重新 fetch 覆盖
  }
)
```

每次弹窗打开都重新 fetch，后端 1h 缓存保证成本极低。不需要在组件内做二级缓存（YAGNI）。

### settings.json 生成变更

`generateAnthropicFiles` 中 `richSettings=true` 分支当前硬编码了以下 **6 个模型字段**，全部需要动态化：

```typescript
// 改动前（全部硬编码）
"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "claude-haiku-4-5",
"ANTHROPIC_DEFAULT_OPUS_MODEL":   "claude-opus-4-6",
"ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-4-6",
"ANTHROPIC_MODEL":                "claude-sonnet-4-6",   // env 内
"ANTHROPIC_REASONING_MODEL":      "claude-opus-4-6",     // env 内
// 顶层字段（settings.json 根级）：
"model":                          "claude-sonnet-4-6",
```

```typescript
// 改动后（models 来自 copilotModels.value ?? fallback）
const models = copilotModels.value ?? {
  sonnet: 'claude-sonnet-4-6',
  opus:   'claude-opus-4-6',
  haiku:  'claude-haiku-4-5',
}
// env 内：
"ANTHROPIC_DEFAULT_HAIKU_MODEL":  models.haiku,
"ANTHROPIC_DEFAULT_OPUS_MODEL":   models.opus,
"ANTHROPIC_DEFAULT_SONNET_MODEL": models.sonnet,
"ANTHROPIC_MODEL":                models.sonnet,
"ANTHROPIC_REASONING_MODEL":      models.opus,
// 顶层字段：
"model":                          models.sonnet,
```

### 边界情况

| 情况 | 行为 |
|------|------|
| 接口超时（>5s） | `AbortSignal.timeout(5000)` 触发，catch → `copilotModels = null` → fallback |
| HTTP 非 200 | throw → catch → fallback |
| 返回空 `data[]` | `pickLatest` 全返回 null → 各角色独立 fallback |
| 某角色无对应模型（如无 haiku） | 该角色 fallback，其他角色正常动态值 |
| `apiKey` 为空字符串 | 函数开头 guard 提前返回，不发请求 |
| 弹窗快速多次打开关闭 | 每次打开重新 fetch，无竞态问题（后续 fetch 覆盖 `copilotModels.value`，reactive 更新 computed） |
| 非 copilot 平台 | 函数开头 guard 提前返回，`copilotModels` 保持 null，`richSettings` 不会为 true，无影响 |

---

## 实现步骤

1. 在 `<script setup>` 中添加 `copilotModels` ref
2. 添加纯函数 `resolveModelRoles`
3. 添加异步函数 `loadCopilotModels`
4. 在现有的 `watch(() => props.platform, ...)` 之后添加 `watch(() => props.show, ...)` 触发逻辑
5. 修改 `generateAnthropicFiles` 中 `richSettings=true` 分支，从 `copilotModels.value` 读取模型字段

所有改动均在 `UseKeyModal.vue` 的 `<script setup>` 块内，模板无需改动。

---

## 不在本次范围内

- 在弹窗上展示"正在加载模型"的 spinner（fetch 速度极快，且有 fallback 无感更新，无需视觉反馈）
- 为非 Copilot 平台的模型字段做动态化（不同平台接口结构不同，超出本次目标）
- 前端二级缓存（后端已有 1h 缓存，YAGNI）
- 后端新增任何接口
