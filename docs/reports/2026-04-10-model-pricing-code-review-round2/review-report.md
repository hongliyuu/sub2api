# 2026-04-10 Model Pricing 复审报告（Round 2）

## 1. 结论

这轮复审里，上一份报告中的 4 个问题（F1-F4）都已经有代码级修复，且我没有再看到原来的高优先级计费错误路径复发。

不过当前改动里仍有 1 个中优先级问题和 1 个低优先级问题：

- [中] `model_key` 没有做规范化，管理页仍可保存“永远不会命中”的价格配置。
- [低] 数据库唯一约束冲突仍会作为通用内部错误返回，管理员会看到 500 而不是可理解的冲突提示。

**建议状态：COMMENT。**

如果团队想把这批修复一次性收干净，我建议在合入前补掉 `model_key` 规范化；其余部分已经明显好于上一轮，可以视发布节奏决定是否继续打磨。

## 2. 新发现

### [中] R1：`model_key` 以原样入库，但运行时按小写精确查缓存，混合大小写/首尾空白配置会静默失效

证据：

- 前端表单直接把用户输入的 `model_key` 原样提交，没有任何 lower-case / trim 处理：`frontend/src/views/admin/ModelPricingView.vue:119`、`frontend/src/views/admin/ModelPricingView.vue:291`
- 后端 handler 也只是把请求原样写入 service entry：`backend/internal/handler/admin/model_pricing_handler.go:41`、`backend/internal/handler/admin/model_pricing_handler.go:119`
- `ModelPricingService` 的缓存 lookup 是精确字符串匹配：`backend/internal/service/model_pricing_service.go:53`
- `BillingService` 在查数据库缓存前会先把请求模型名统一转成小写：`backend/internal/service/billing_service.go:399`

这意味着管理员如果在页面里填了：

- `GPT-5.1`
- `gpt-5.1 `
- ` Claude-Sonnet-4`

这些记录都会创建成功，但运行时查缓存时实际拿的是小写后的模型名，例如 `gpt-5.1`，最终不会命中这条 override，而是继续走 LiteLLM 动态价格或硬编码 fallback。

影响：

- 管理员会以为价格覆盖已经生效，但实际计费仍然使用旧价格。
- 这是“控制面显示成功，数据面未生效”的静默错误，定位成本高。
- 当前唯一索引也是大小写敏感的，所以 `gpt-5.1` 和 `GPT-5.1` 可以同时存在，但运行时只会命中前者。

建议：

- 在 Create / Update 前统一执行 `strings.TrimSpace(strings.ToLower(modelKey))`。
- 如果不想隐式改值，至少也要在后端拒绝非规范 key，并返回 400。
- 如果后续要彻底防止重复，唯一性最好绑定到规范化后的 key，而不是原始输入。

### [低] R2：重复 `model_key` 的数据库约束冲突没有翻译成业务错误，接口会返回 500

证据：

- `model_pricing` repository 的 `Create` / `Update` 直接把底层数据库错误原样返回，没有走项目里已有的 `translatePersistenceError`：`backend/internal/repository/model_pricing_repo.go:59`、`backend/internal/repository/model_pricing_repo.go:80`
- `response.ErrorFrom()` 对非 `ApplicationError` 会走通用内部错误映射：`backend/internal/pkg/response/response.go:82`
- `infraerrors.FromError()` 对未知错误统一退化成 500 internal error：`backend/internal/pkg/errors/errors.go:148`

影响：

- Partial unique index 已经能挡住重复 active key，但管理员在页面上撞到重复 key 时，得到的是 500，而不是 409/400 级别的可理解错误。
- 不会破坏数据正确性，但会放大运维噪音，也不利于前端给出明确提示。

建议：

- 为 `model_pricing` repository 补上冲突错误翻译，和 `user_repo.go` / `group_repo.go` 的处理保持一致。
- 返回一个明确的冲突 reason，例如 `MODEL_PRICING_EXISTS`。

## 3. 上一轮问题核验

### F1：migration seed 与 fallback 价格不一致

已修复。

- `084_model_pricings.sql` 里的关键 seed 值已与 `billing_service.go` 对齐，包括 `claude-opus-4.5`、`claude-3-5-haiku`、`gemini-3.1-pro`、`gpt-5.1`、`gpt-5.4`、`gpt-5.2`、`gpt-5.1-codex`、`gpt-5.2-codex`：`backend/migrations/084_model_pricings.sql:45`、`backend/migrations/084_model_pricings.sql:51`、`backend/migrations/084_model_pricings.sql:61`、`backend/migrations/084_model_pricings.sql:66`、`backend/migrations/084_model_pricings.sql:70`、`backend/migrations/084_model_pricings.sql:77`、`backend/migrations/084_model_pricings.sql:80`、`backend/migrations/084_model_pricings.sql:83`
- 对应 fallback 值见：`backend/internal/service/billing_service.go:167`、`backend/internal/service/billing_service.go:197`、`backend/internal/service/billing_service.go:206`、`backend/internal/service/billing_service.go:217`、`backend/internal/service/billing_service.go:243`、`backend/internal/service/billing_service.go:254`、`backend/internal/service/billing_service.go:264`
- `priority` 和 `cache_creation` 也已补齐：`backend/migrations/084_model_pricings.sql:37`、`backend/migrations/084_model_pricings.sql:39`

### F2：允许保存“全 0 且启用”的价格项

已修复。

- 前端新建表单默认改成 `enabled: false`：`frontend/src/views/admin/ModelPricingView.vue:245`
- 后端 service 在 Create / Update 前统一执行业务校验：`backend/internal/service/model_pricing_service.go:69`、`backend/internal/service/model_pricing_service.go:82`
- 校验逻辑会阻止 `enabled=true` 且没有任何基础价格的 entry：`backend/internal/service/model_pricing_service.go:135`

### F3：`(model_key, deleted_at)` 唯一约束在 PostgreSQL 下无法约束 active 记录

已修复。

- migration 改成 partial unique index：`backend/migrations/084_model_pricings.sql:23`
- Ent schema 去掉了原来的复合唯一索引，只保留注释说明约束由 migration SQL 提供：`backend/ent/schema/model_pricing.go:90`
- `ent/migrate/schema.go` 也同步成非唯一索引定义，没有再保留旧的复合唯一索引：`backend/ent/migrate/schema.go:573`

### F4：更新 `model_key` 后旧缓存不清理

已修复。

- repository 新增 `GetByID()`：`backend/internal/repository/model_pricing_repo.go:47`
- `Update()` 先读取旧记录，再在 key 变化时删除旧缓存项：`backend/internal/service/model_pricing_service.go:87`
- `Delete()` 也改成先按 ID 读取，再精确删除缓存，不再走全量 List：`backend/internal/service/model_pricing_service.go:110`

## 4. 测试与验证

已执行：

```bash
npm --prefix frontend run typecheck
```

结果：

- 通过。

已执行：

```bash
env GOTOOLCHAIN=auto GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org,direct \
  go test ./internal/service -run 'TestGetModelPricing|TestBillingServiceGetModelPricing|TestCalculateCost|TestGetModelPricing_OpenAIGPT54Fallback|TestGetModelPricing_OpenAIGPT51Fallback|TestBillingServiceGetModelPricing_OpenAIFallbackGpt52Variants|TestGetModelPricing_OpenAIGpt52FallbacksExposePriorityPrices'
```

结果：

- 通过。

已执行：

```bash
env GOTOOLCHAIN=auto GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org,direct \
  go test ./... -run '^$'
```

结果：

- backend 全量编译通过，包含新加的 Ent 生成代码、repository、service、handler 和 routes。

## 5. 额外观察

- 当前仓库里仍没有针对 `ModelPricingService` / `ModelPricingHandler` 的专门测试文件；这轮我能确认“编译通”和“既有 BillingService 回归测试通”，但还没有自动化用例直接锁住：
  - `model_key` 改名后的缓存清理
  - “全 0 且启用”返回 400
  - partial unique index 冲突时的接口行为
- 这不是新的功能性缺陷，但也是上一轮 4 个问题能同时漏进来的直接原因之一。
