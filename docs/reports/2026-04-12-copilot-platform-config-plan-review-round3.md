# Copilot Platform Config 实施计划复审报告（Round 3）

## 基本信息
- 复审日期：2026-04-12
- 复审对象：
  - `docs/superpowers/specs/2026-04-12-copilot-platform-config-design.md`
  - `docs/superpowers/plans/2026-04-12-copilot-platform-config/`
- 对应上一轮报告：`docs/reports/2026-04-12-copilot-platform-config-plan-review-round2.md`
- 复审类型：计划修订后的三次 review
- 复审方式：静态审阅更新后的 spec 与 batch 文档，并对照当前代码结构核对可实施性

## 复审结论（摘要）
- Round 2 提出的 `max_body_kb` 测试可测性问题，这一轮在计划层已经基本修正。
- 但新的 `/admin/copilot/accounts -> AccountsView + route.meta.defaultPlatform` 方案，仍然存在一个前端路由复用问题。
- 结论仍为：`REQUEST CHANGES`。

---

## Findings

### 1) HIGH: `AccountsView` 只在初始化时读取 `route.meta.defaultPlatform`，同组件路由切换时预筛会失效

#### 问题
当前修订后的计划把 `/admin/copilot/accounts` 改成直接复用 `AccountsView`，并通过 `route.meta.defaultPlatform = 'copilot'` 作为平台预筛来源：

- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-5-frontend-routes-sidebar-api.md:226`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-5-frontend-routes-sidebar-api.md:261`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:34`
- `docs/superpowers/plans/2026-04-12-copilot-platform-config/batch-6-views-modal.md:59`

但计划中的实现方式仍然只是把 `defaultPlatform` 塞进 `useTableLoader(... initialParams ...)` 的初始值里。

而当前代码结构显示：
- `RouterView` 没有设置 `:key`，组件实例默认会被复用：
  - `frontend/src/App.vue:116`
- `useTableLoader` 只在创建时用 `initialParams` 初始化一次 `params`：
  - `frontend/src/composables/useTableLoader.ts:29`
- 当前 `AccountsView` 也没有任何 `watch(route...)` / `onBeforeRouteUpdate` 之类的更新逻辑：
  - `frontend/src/views/admin/AccountsView.vue:628`

这意味着如果用户在 SPA 内部从：
- `/admin/accounts` 切到 `/admin/copilot/accounts`
或反过来切换，

同一个 `AccountsView` 实例大概率会被复用，`setup` 不会重跑，`params.platform` 也不会按新 route meta 重置。

#### 风险
- 从通用账号页进入 Copilot 账户列表时，仍可能看到旧筛选状态，而不是 `platform='copilot'`。
- 从 Copilot 账户列表切回通用账号页时，也可能残留 `copilot` 过滤。
- 这会导致“路由 path 是对的、侧边栏高亮也是对的，但列表内容不对”的隐蔽错误。

#### 建议
- 至少补一个 route 变化同步方案，而不只是“初始化读取 meta”。
- 可选做法：
  1. 在 `AccountsView` 中 `watch(() => route.meta.defaultPlatform)`，切换时同步 `params.platform` 并触发 reload。
  2. 监听 `route.path` / `route.name`，在 `/admin/accounts` 与 `/admin/copilot/accounts` 之间切换时重置筛选。
  3. 或使用一个真正独立的承载组件/包装组件，让 Copilot 账户列表页不依赖同实例复用。

#### 建议补充验证
- 手动验证必须新增以下场景：
  1. 先进入 `/admin/accounts`，再点击侧边栏进入 `/admin/copilot/accounts`
  2. 再从 `/admin/copilot/accounts` 返回 `/admin/accounts`
  3. 确认两个方向切换后列表过滤都正确

---

## 已关闭的问题

以下上一轮问题在当前计划中已基本关闭，不再重复作为阻断项提出：
- `max_body_kb` 运行路径未接入
- `max_body_kb` 测试无法 stub 平台配置服务
- 前端 API 双层解包
- Copilot 模型集缺失
- GET 顺序错误
- 数值输入空值 normalize
- 错误 i18n key

---

## 复审结论

### Recommendation
`REQUEST CHANGES`

### 原因
- 当前计划还差最后一个关键前端行为修正：
  - 不能只在 `AccountsView` 初始化时读取 `route.meta.defaultPlatform`
  - 还必须处理同组件路由切换时的状态同步

### 建议的下一步
1. 先补上 `AccountsView` 对 route 变化的同步策略。
2. 再把该策略写进计划的实现步骤和手动验证步骤。
3. 这一步修完后，计划基本可以进入实现阶段。

### 备注
- 本次仍为静态 review。
- 未修改任何业务代码。
- 未执行构建、类型检查或测试，因为复审对象仍是计划文档而非已实现代码。
