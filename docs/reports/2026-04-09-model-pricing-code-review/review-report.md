# 2026-04-09 Model Pricing 代码审查报告

## 1. 审查范围
本次审查聚焦当前未提交的 model pricing / admin 相关改动：

- 后端
  - `backend/internal/service/billing_service.go`
  - `backend/internal/service/model_pricing.go`
  - `backend/internal/service/model_pricing_service.go`
  - `backend/internal/repository/model_pricing_repo.go`
  - `backend/internal/handler/admin/model_pricing_handler.go`
  - `backend/internal/server/routes/admin.go`
  - `backend/cmd/server/wire_gen.go`
  - `backend/ent/schema/model_pricing.go`
  - `backend/migrations/084_model_pricings.sql`
  - 以及对应 Ent 生成代码
- 前端
  - `frontend/src/api/admin/modelPricings.ts`
  - `frontend/src/views/admin/ModelPricingView.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/components/layout/AppSidebar.vue`
  - `frontend/src/i18n/locales/en.ts`
  - `frontend/src/i18n/locales/zh.ts`
- 关联变更
  - `backend/internal/handler/copilot_gateway_handler.go`

## 2. 结论
**当前改动不建议直接合入。**

主要原因不是样式或实现细节，而是存在会直接影响计费正确性的风险：
- 上线迁移后，数据库种子价格会覆盖现有 fallback 价格，而且多组关键数值与当前计费逻辑不一致。
- 管理页允许以“全 0 且启用”的配置创建价格项，会把对应模型直接变成 0 成本计费。
- 当前唯一性设计无法真正阻止同一个 `model_key` 出现多条有效记录，缓存层会静默选中其中一条，结果不可预测。
- 修改 `model_key` 时，内存缓存不会清掉旧 key，旧模型会继续命中陈旧价格直到服务重启。

## 3. 关键发现（按严重级别排序）

### [高] F1：迁移种子价格与当前 `BillingService` fallback 定价不一致，部署后会直接改变线上计费结果

证据：`BillingService` 已明确把数据库价格放在 fallback 之前使用（`backend/internal/service/billing_service.go:430`-`backend/internal/service/billing_service.go:443`）。但迁移里的 seed 数据与当前 fallback 常量并不一致：

- `gpt-5.1`
  - migration: `2.0 / 8.0 / 0.5`（`backend/migrations/084_model_pricings.sql:38`）
  - code fallback: `1.25 / 10 / 0.125`（`backend/internal/service/billing_service.go:206`-`backend/internal/service/billing_service.go:214`）
- `gpt-5.4`
  - migration: `2.5 / 10 / 1.25`（`backend/migrations/084_model_pricings.sql:39`）
  - code fallback: `2.5 / 15 / 0.25`，且还带 `272k` 长上下文策略（`backend/internal/service/billing_service.go:217`-`backend/internal/service/billing_service.go:228`）
- `gpt-5.2-codex`
  - migration: `3.0 / 15.0 / 0.75`（`backend/migrations/084_model_pricings.sql:44`）
  - code fallback: `1.75 / 14 / 0.175`（`backend/internal/service/billing_service.go:264`-`backend/internal/service/billing_service.go:273`）

这不是“展示文案和代码没同步”的问题，而是**迁移会把数据库价格缓存加载进来，并优先于 fallback 生效**。也就是说，只要跑了 `084_model_pricings.sql`，实际计费就会和现在的测试/逻辑基线偏离。

现有测试也明确把当前 fallback 当成正确基线，例如：
- `gpt-5.1` 输入价应为 `1.25e-6`（`backend/internal/service/billing_service_test.go:154`-`backend/internal/service/billing_service_test.go:160`）
- `gpt-5.4` 输出价应为 `15e-6`，cache read 应为 `0.25e-6`（`backend/internal/service/billing_service_test.go:163`-`backend/internal/service/billing_service_test.go:174`）

影响：
- 部署迁移后，不需要任何管理员操作，就会改变现有模型的计费金额。
- 风险既包括**少收费**，也包括**多收费**，属于直接影响账单正确性的上线阻断项。

建议：
- 在合入前统一一份“单一真相来源”：要么迁移 seed 严格复刻 `billing_service.go` 当前值，要么反过来先改代码和测试，再一起调整迁移。
- 这部分至少补一组回归测试，覆盖“加载 DB cache 后的价格 == 预期价格”。

### [高] F2：后端允许创建“全 0 且启用”的价格项，前端默认表单正好就是这个危险组合

证据：
- 前端新建表单默认所有价格字段都是 `0`，且 `enabled: true`（`frontend/src/views/admin/ModelPricingView.vue:245`-`frontend/src/views/admin/ModelPricingView.vue:256`）。
- 后端请求校验只要求 `model_key` 必填，价格字段仅限制 `min=0`，没有要求“至少一个价格 > 0”（`backend/internal/handler/admin/model_pricing_handler.go:40`-`backend/internal/handler/admin/model_pricing_handler.go:52`）。
- `Create` / `Update` 直接透传到服务层，没有任何业务校验（`backend/internal/handler/admin/model_pricing_handler.go:68`-`backend/internal/handler/admin/model_pricing_handler.go:102`，`backend/internal/service/model_pricing_service.go:63`-`backend/internal/service/model_pricing_service.go:80`）。
- `ToModelPricing()` 会把这些 `0` 原样转换成 per-token 价格（`backend/internal/service/model_pricing.go:27`-`backend/internal/service/model_pricing.go:38`）。

结果是：管理员只要点开“新增价格”，填一个 `model_key` 然后直接保存，就能创建一条“启用中、但所有费用为 0”的配置。由于数据库价格优先级高于 fallback（`backend/internal/service/billing_service.go:430`-`backend/internal/service/billing_service.go:435`），该模型随后会被按 0 成本计费。

影响：
- 这是一个非常容易误触发的免费计费漏洞。
- 不是极端输入；当前 UI 默认值本身就在引导这个危险路径。

建议：
- 后端服务层加硬校验：`enabled=true` 时至少要求一类价格大于 0。
- 前端新建表单默认应为 `enabled=false`，或者在无有效价格前禁用提交。
- 最好补一条接口测试：全 0 + 启用时返回 400，而不是保存成功。

### [高] F3：`(model_key, deleted_at)` 这个唯一约束在 PostgreSQL 下不能阻止多条有效记录，缓存会静默选一条，计费结果不稳定

证据：
- migration 使用 `UNIQUE (model_key, deleted_at)`（`backend/migrations/084_model_pricings.sql:21`）。
- Ent schema 也用了同样的唯一索引定义（`backend/ent/schema/model_pricing.go:90`-`backend/ent/schema/model_pricing.go:94`）。

在 PostgreSQL 里，`NULL` 不参与普通唯一比较，因此多条 `deleted_at IS NULL` 的记录可以同时存在。也就是说，当前设计**无法真正保证 active 记录里的 `model_key` 唯一**。

后续影响进一步被缓存放大：
- `List()` 只是按 `model_key` 排序拿全量记录（`backend/internal/repository/model_pricing_repo.go:19`-`backend/internal/repository/model_pricing_repo.go:30`）。
- `LoadCache()` 再把结果塞进 `map[string]*ModelPricingEntry`，同 key 后写覆盖前写（`backend/internal/service/model_pricing_service.go:27`-`backend/internal/service/model_pricing_service.go:43`）。

如果数据库里出现两条未删除且同 `model_key` 的记录：
- 创建接口不会被数据库唯一约束挡住；
- 启动加载缓存时只会保留其中一条；
- 由于查询只按 `model_key` 排序，没有稳定的次序约束，最终哪条生效并不可靠。

影响：
- 同一个模型可能在不同实例/不同重启后命中不同价格。
- 管理员从 UI 上看到的是两条记录，但运行时只会偷偷用其中一条，定位会非常困难。

建议：
- 改成 partial unique index：`UNIQUE (model_key) WHERE deleted_at IS NULL`。
- 在服务层也补一层重复 key 检查，避免脏数据进入表里。

### [中] F4：更新时如果修改了 `model_key`，旧 key 的缓存不会被清掉，旧模型会继续命中陈旧价格直到重启

证据：
- `Update()` 在仓储更新完成后，只调用 `updateCacheEntry(updated)`（`backend/internal/service/model_pricing_service.go:73`-`backend/internal/service/model_pricing_service.go:80`）。
- `updateCacheEntry()` 只会写入/删除“更新后的 key”，不会处理“更新前的 key”（`backend/internal/service/model_pricing_service.go:108`-`backend/internal/service/model_pricing_service.go:115`）。
- 前端编辑页允许直接修改 `model_key`（`frontend/src/views/admin/ModelPricingView.vue:118`-`frontend/src/views/admin/ModelPricingView.vue:120`，`frontend/src/views/admin/ModelPricingView.vue:268`-`frontend/src/views/admin/ModelPricingView.vue:283`）。

一个具体场景：
1. 现有条目 `gpt-5.1` 在缓存里；
2. 管理员把它改名成 `gpt-5.1-2026`；
3. 服务会把新 key 写入缓存，但不会把旧的 `gpt-5.1` 删除；
4. 之后 `gpt-5.1` 仍然会继续命中旧价格，直到进程重启或缓存全量重载。

影响：
- UI 看起来改成功了，但线上计费并没有完全切换。
- 这是典型的“控制面已变更、数据面仍旧值”的隐性错误。

建议：
- 更新前先读取旧记录，若 `old.ModelKey != updated.ModelKey`，显式删除旧 cache key。
- 给“改 key”这条路径补一个单元测试。

## 4. 其他观察 / 测试缺口
- 没看到新的后端单元测试覆盖 `ModelPricingService` / `ModelPricingRepository` 的缓存刷新、重复 key、删除、改 key 等关键路径。
- 前端页面能通过 typecheck，但没有看到对应的交互测试（新增、编辑、删除、危险表单校验、国际化回归）。
- `frontend/src/views/admin/ModelPricingView.vue` 中仍有硬编码中文列名和说明文案（如 `缓存价格`、说明卡片正文），英文界面下会混出中英文，但这属于次要问题，不是当前阻断项。

## 5. 验证记录
已执行：

```bash
cd frontend && npm run typecheck
```

结果：
- 通过。

尝试执行：

```bash
cd backend && go test ./internal/service/... ./internal/repository/...
```

结果：
- 未实际运行测试；本机 Go 版本是 `1.26.1`，而 `backend/go.mod` 要求 `1.26.2`。
- 进一步尝试 `GOTOOLCHAIN=auto` 也被环境里的 `GOSUMDB=off` 阻断，无法自动拉取工具链。

## 6. 审查结论
**建议状态：REQUEST CHANGES。**

至少应先修完以下阻断项再重新提审：
1. 校正 migration seed 与当前计费基线的不一致；
2. 禁止保存“全 0 且启用”的价格配置；
3. 修正 active 记录的唯一性约束；
4. 修复更新 `model_key` 时的缓存失效问题；
5. 补最基本的后端缓存/计费回归测试。
