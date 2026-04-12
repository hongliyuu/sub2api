# Copilot Platform Config 实施计划复审报告

## 基本信息
- 复审日期：2026-04-12
- 复审对象：
  - `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md`
  - `docs/superpowers/plans/2026-04-12-copilot-platform-config/`
- 复审类型：实施前方案 review
- 复审方式：静态审阅计划文档，并对照当前仓库代码结构核对可实施性

## 复审结论（摘要）
- 方案方向正确，功能边界也比较清晰。
- 但当前计划中存在若干会直接影响落地的实现问题，其中包含多项阻断级问题。
- 结论建议为：`REQUEST CHANGES`，先修计划，再实施。

---

## 阻断级问题

### 1) 前端 API 层按错误的响应结构取值，页面会直接拿到 `undefined`

#### 问题
计划中的 `listCopilotPlatformConfigs` / `updateCopilotPlatformConfig` 使用了 `res.data.data` 读取响应：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-5-frontend-routes-sidebar-api.md:169`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-5-frontend-routes-sidebar-api.md:185`

但当前前端 `apiClient` 已经在响应拦截器里把标准包裹响应 `{ code, message, data }` 解包成 `response.data = apiResponse.data`：

- `frontend/src/api/client.ts:87`

#### 风险
- 平台配置页首次加载时会读不到数组数据。
- 单卡保存后会拿不到更新后的返回值。
- 这不是类型层面的“小问题”，而是运行时直接失效。

#### 建议
- 新 API 文件应和现有 `frontend/src/api/admin/*.ts` 保持一致，直接返回 `res.data`。

---

### 2) 计划要求手改 `wire_gen.go`，与仓库当前生成链冲突

#### 问题
计划多处要求手动编辑 `backend/cmd/server/wire_gen.go`：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-3-handler-routes-wire.md:244`

但当前文件明确标记为 Wire 生成文件：

- `backend/cmd/server/wire_gen.go:1`

并且仓库本身已经通过 `//go:generate` 维护生成链。

#### 风险
- 当前改完能过，后续一旦重新执行 `go generate` / `wire` 就会被覆盖。
- 手工维护生成文件容易造成 provider graph 与源码声明不一致。

#### 建议
- 计划应改成：更新 `handler/wire.go`、`repository/wire.go`、必要的 provider 构造后，重新生成 `wire_gen.go`。
- 不建议把“手动 patch 生成文件”作为正式实施步骤。

---

### 3) `max_body_kb` 虽然进入了表和 API，但计划没有把继承逻辑真正接到运行路径

#### 问题
设计文档把 `max_body_kb` 定义为平台默认值的一部分：

- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:61`
- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:86`

但 Batch 4 实际只详细实现了：
- `max_output_tokens` fallback
- `model_whitelist` fallback

对应文档：
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-4-inheritance-whitelist.md:74`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-4-inheritance-whitelist.md:286`

当前真实的 body size 校验发生在 handler：

- `backend/internal/handler/copilot_gateway_handler.go:885`

它只读取账号级 `account.GetMaxBodyBytes()`，账号未设置时直接回退系统默认：

- `backend/internal/service/account.go:599`
- `backend/internal/handler/copilot_gateway_handler.go:886`

#### 风险
- 平台配置页能保存 `max_body_kb`，但运行时完全不生效。
- 会出现“配置成功但行为没变”的假功能。

#### 建议
- 计划中必须新增一个独立任务，明确把 `max_body_kb` 平台 fallback 接到 `CopilotGatewayHandler.checkCopilotBodySize` 所在链路。
- 同时补对应测试，而不是只测 `max_output_tokens`。

---

### 4) `CopilotAccountListView` 的计划实现并不是“账户列表页”

#### 问题
spec 里写的是：

- `/admin/copilot/accounts` 复用现有账户列表页并预设 `platform=copilot`
- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:39`

但计划给出的实际实现是一个说明页，里面只有跳转 `/admin/accounts?platform=copilot` 的链接：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:25`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:31`

更关键的是，当前 `AccountsView` 并不会从 route query 初始化过滤器。它的初始参数是固定空值：

- `frontend/src/views/admin/AccountsView.vue:639`

所以就算跳过去，也还是全量账户页，不会自动筛成 Copilot。

#### 风险
- 菜单“账户列表”与实际页面含义不一致。
- 用户路径会断层，达不到 spec 目标。

#### 建议
- 二选一：
  - 给 `AccountsView` 增加基于 route query 的初始化筛选能力。
  - 或抽出真正可复用的账户列表内容，让 `CopilotAccountListView` 直接承载 Copilot 过滤后的列表。

---

### 5) `ModelWhitelistSelector` 当前不支持 `platform=\"copilot\"`

#### 问题
计划在两个地方都直接传了 `platform=\"copilot\"`：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:227`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:464`

但当前模型白名单来源函数 `getModelsByPlatform` 并没有 `copilot` 分支：

- `frontend/src/composables/useModelWhitelist.ts:382`

默认分支会回退到 `claudeModels`：

- `frontend/src/composables/useModelWhitelist.ts:405`

#### 风险
- Copilot 白名单组件里看到的是 Claude 模型集，不是 Copilot 实际可路由模型。
- 用户会错误配置白名单，进而影响调度。

#### 建议
- 先定义 Copilot 的白名单候选模型来源，再复用 `ModelWhitelistSelector`。
- 如果 Copilot 需要“跨上游模型集”，要先确定 UI 的来源策略，而不是直接沿用 Anthropic 默认值。

---

## 中风险问题

### 6) GET 返回顺序与设计文档不一致

#### 问题
设计文档强调固定 5 条并按 plan_type 展示顺序返回：

- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:105`

但计划里的 repo/handler 都采用按字符串字母序返回：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-2-repo-service.md:144`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-3-handler-routes-wire.md:66`

#### 风险
- 页面卡片顺序会变成 `business -> enterprise -> individual_*`。
- 和设计文档、运营预期不一致。

#### 建议
- 返回顺序应以 `service.AllCopilotPlanTypes` 为准，而不是数据库字母序。

---

### 7) 平台配置页的数值输入清空逻辑与 API contract 不匹配

#### 问题
计划在页面里使用 `v-model.number`：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:151`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:166`

保存时再通过 `?? null` 转空值：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:367`

但空输入在这里很容易变成空字符串，不一定会被转换成 `null`。

#### 风险
- 前端发出的值可能是 `\"\"`，不是 API 约定的 `null`。
- 后端如果严格绑定数值类型，会出现校验失败或语义漂移。

#### 建议
- 保存前显式做一次 normalize：
  - 空字符串 -> `null`
  - 非空字符串 -> 数值

---

## 低风险问题

### 8) 示例代码里用了不存在的 i18n key

#### 问题
平台配置页示例里使用：

- `t('admin.accounts.copilot.addMapping')`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:217`

但当前已有词条是：

- `frontend/src/i18n/locales/zh.ts:2456`
- `frontend/src/i18n/locales/en.ts:2307`

对应 key 为 `admin.accounts.addMapping`。

#### 风险
- 实现时直接照抄会在 UI 上显示原始 key。

#### 建议
- 修正计划中的 i18n key，或补齐实际词条。

---

## 建议修订后的执行顺序

1. 先修计划中的基础前提问题：
   - 前端 API 解包方式
   - `wire` 生成策略
   - Copilot 模型来源
   - 账户列表页复用策略
2. 再补全运行时继承逻辑：
   - `max_output_tokens`
   - `max_body_kb`
   - `model_whitelist`
3. 最后再做前端页面与菜单重组。

---

## 复审结论

### Recommendation
`REQUEST CHANGES`

### 原因
- 当前计划不是不能做，而是“直接按文档开工会出现多处落地偏差”。
- 其中至少 5 个问题会导致功能不生效、页面不正确或后续维护不稳定。

### 备注
- 本次仅做静态 review。
- 未实际修改代码。
- 未执行构建、类型检查或测试，因为当前复审对象是方案文档，不是已落地实现。
