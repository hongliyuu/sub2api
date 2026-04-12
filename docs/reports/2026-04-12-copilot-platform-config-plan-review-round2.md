# Copilot Platform Config 实施计划复审报告（Round 2）

## 基本信息
- 复审日期：2026-04-12
- 复审对象：
  - `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md`
  - `docs/superpowers/plans/2026-04-12-copilot-platform-config/`
- 对应上一轮报告：`docs/reports/2026-04-12-copilot-platform-config-plan-review.md`
- 复审类型：计划修订后的二次 review
- 复审方式：静态审阅更新后的 spec 与 batch 文档，并对照当前代码结构核对可实施性

## 复审结论（摘要）
- 上一轮指出的 8 个问题里，大部分已经在计划层修正，特别是：
  - 前端 API 解包方式
  - `max_body_kb` 运行路径接入
  - Copilot 模型集来源
  - GET 固定顺序
  - 数值输入空值 normalize
  - 错误 i18n key
- 本轮未再发现前一版那种“做完即失效”的大面积结构性问题。
- 但仍有 1 个高风险问题和 1 个中风险问题，建议继续修计划后再实施。

---

## Findings

### 1) HIGH: `CopilotAccountListView` 的“跳板式 replace”会破坏 Copilot 分组的路由语义和侧边栏激活状态

#### 问题
设计文档 Section 1 仍将“账户列表”定义为 Copilot 分组下的独立路由 `/admin/copilot/accounts`：

- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:38`
- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:45`

但更新后的 spec / plan 实际实现改成了“路由跳板”：

- `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md:158`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:28`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:76`

也就是用户进入 `/admin/copilot/accounts` 后，会立刻 `router.replace` 到 `/admin/accounts?platform=copilot`。

当前侧边栏高亮逻辑基于 `route.path` 与菜单项 path 的精确/前缀匹配：

- `frontend/src/components/layout/AppSidebar.vue:640`
- `frontend/src/components/layout/AppSidebar.vue:646`
- `frontend/src/components/layout/AppSidebar.vue:709`

因此一旦 replace 到 `/admin/accounts`，激活的将是通用“账号管理”菜单，而不是 Copilot 下的“账户列表”菜单。

#### 风险
- “所有 Copilot 相关页面重组到 Copilot 分组”这一目标只在入口层成立，落地页实际仍回到全局 `/admin/accounts`。
- 侧边栏激活状态会跳到通用账号管理，不再停留在 Copilot 分组。
- 路由语义、页面 meta、用户可分享链接都会变成通用账号页，而不是 Copilot 子域页面。

#### 建议
- 更稳妥的做法仍是保留 `/admin/copilot/accounts` 作为真实承载页，而不是跳板页。
- 可选方案：
  1. 抽出 `AccountsView` 的可复用表格壳层，在 `/admin/copilot/accounts` 中真正嵌入并初始化 `platform='copilot'`。
  2. 或让 `AccountsView` 支持一个“嵌入/别名模式”，在不改路由 path 的前提下复用列表逻辑。
- 如果坚持使用 replace，至少还需要在计划中补上：
  - 侧边栏 Copilot 项的 active 判定修正
  - 页面标题/描述与 breadcrumb 的语义修正

---

### 2) MEDIUM: `max_body_kb` 新增测试没有验证“平台配置层”本身，保护力度仍不足

#### 问题
这轮新增了 `Task 11b` 来把 `max_body_kb` 接入运行路径，这个方向是对的：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-4-inheritance-whitelist.md:487`

但它附带的测试代码并没有真正验证“平台配置命中”这条新分支：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-4-inheritance-whitelist.md:588`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-4-inheritance-whitelist.md:603`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-4-inheritance-whitelist.md:611`

所谓 `TestCopilotGatewayHandler_platformBodyLimit_ReturnsPlatformConfig` 实际上把 `platformConfigSvc` 设成了 `nil`，断言的也是 fallback 到系统默认，并没有覆盖：
- 平台配置命中时返回 `max_body_kb * 1024`
- 账号级 `GetMaxBodyBytes()` 优先于平台配置

另外，`CopilotPlatformConfigService` 的依赖仓储字段是未导出的：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-2-repo-service.md:379`

这意味着按当前文档写法，`handler` 包测试里并没有现成的正向 stub seam。

#### 风险
- `max_body_kb` 虽然补进了实现步骤，但测试仍可能只验证“默认值 fallback 没坏”。
- 真正新增的“平台配置层”行为如果实现错了，当前计划里的测试不一定能抓到。

#### 建议
- 至少补 1 个真正命中平台配置的测试用例。
- 可选做法：
  1. 在 `handler` 侧引入一个窄接口（例如仅 `GetByPlanType(ctx, planType)`），便于 stub。
  2. 或把正向行为测试放到更容易注入 stub 的位置，再在 handler 侧保留 fallback smoke test。

---

## 已关闭的上一轮问题

以下问题在本轮计划中已得到有效处理，不再作为阻断项重复提出：
- 前端 API 双层解包问题
- `max_body_kb` 完全未接入运行路径
- `ModelWhitelistSelector` 不支持 `platform="copilot"`
- GET 返回字母序而非固定顺序
- `v-model.number` 空值发出 `""`
- `admin.accounts.copilot.addMapping` 错误 key

关于 `wire_gen.go`：
- 当前计划已明确说明这是仓库现有的混合 Wire 维护策略，且与现有代码风格一致。
- 我不再把它作为本轮阻断问题，但它仍然是一个流程性维护风险，后续若团队重新统一 DI 生成策略，应一并收敛。

---

## 复审结论

### Recommendation
`REQUEST CHANGES`

### 原因
- 当前计划已明显优于上一版，但“Copilot 账户列表页”的最终落地形态仍没有真正满足 Copilot 分组目标。
- `max_body_kb` 的新增测试也还没有覆盖到真正新增的行为分支。

### 建议的下一步
1. 先决定 `/admin/copilot/accounts` 是否要保留为真实承载页，而不是跳板页。
2. 再补 `max_body_kb` 的正向平台配置测试。
3. 这两处修完后，再开始按 batch 执行实现。

### 备注
- 本次仍为静态 review。
- 未修改任何业务代码。
- 未执行构建、类型检查或测试，因为复审对象仍是计划文档而非已实现代码。
