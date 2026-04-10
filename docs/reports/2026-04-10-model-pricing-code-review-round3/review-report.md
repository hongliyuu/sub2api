# 2026-04-10 Model Pricing 复审报告（Round 3）

## 1. 结论

这轮复审里，cc 针对上一份 Round 2 报告中的两个问题都已经修到位：

- R1：`model_key` 现在会在服务层统一做 `strings.TrimSpace(strings.ToLower(...))` 规范化。
- R2：数据库唯一约束冲突现在会翻译成 `MODEL_PRICING_EXISTS`，接口语义从通用 500 改成了 409。

我没有看到这两处修复本身有回归，也没有再看到 Round 1 里 F1-F4 那类会直接影响计费结果的阻断问题复发。

不过当前代码里还有 1 个新的低优先级输入校验问题：

- [低] `model_key` 如果传入纯空白字符，会在规范化后变成空字符串，最终从 Ent 校验层冒成 500，而不是返回 400。

**建议状态：COMMENT。**

## 2. 新发现

### [低] R3：纯空白 `model_key` 会在规范化后触发 Ent 校验错误，接口返回 500 而不是 400

证据：

- handler 对 `model_key` 只做了 `binding:"required"`，没有 trim 后判空：`backend/internal/handler/admin/model_pricing_handler.go:41`
- 服务层在 Create / Update 最前面做规范化：`backend/internal/service/model_pricing_service.go:73`、`backend/internal/service/model_pricing_service.go:87`
- 规范化逻辑会把纯空白输入变成空字符串：`backend/internal/service/model_pricing_service.go:155`
- Ent schema 对 `model_key` 配了 `NotEmpty()`：`backend/ent/schema/model_pricing.go:38`
- Create / Update builder 校验失败时会返回 `ValidationError`：`backend/ent/modelpricing_create.go:310`、`backend/ent/modelpricing_update.go:743`
- repository 没有把这种校验错误翻译成业务错误；`response.ErrorFrom()` 对非 `ApplicationError` 会退化成通用 500：`backend/internal/repository/model_pricing_repo.go:73`、`backend/internal/pkg/response/response.go:82`、`backend/internal/pkg/errors/errors.go:148`

一个具体例子：

- 请求体里传 `"model_key": "   "`
- `binding:"required"` 会放行这个值
- `normalizeModelKey()` 把它变成 `""`
- Ent `NotEmpty()` 校验失败
- 接口最终返回 500，而不是“`model_key` 不能为空”的 400

影响：

- 不会写入脏数据，但管理员会看到误导性的内部错误。
- 这是低优先级问题，因为只影响明显非法输入，不影响正常计费路径。

建议：

- 在 `normalizeModelKey()` 后补一层显式校验：空字符串直接返回 `infraerrors.BadRequest(...)`
- 或者在 handler 层增加 trim 后非空校验，避免把格式错误推迟到持久化层

## 3. 已确认修复

### Round 2 / R1：`model_key` 未规范化

已修复。

- 服务层 Create / Update 统一调用 `normalizeModelKey()`：`backend/internal/service/model_pricing_service.go:73`、`backend/internal/service/model_pricing_service.go:87`
- 规范化逻辑为 `strings.TrimSpace(strings.ToLower(entry.ModelKey))`：`backend/internal/service/model_pricing_service.go:157`
- 这与 `BillingService` 查询前统一 lower-case 的行为已经对齐：`backend/internal/service/billing_service.go:399`

### Round 2 / R2：唯一约束冲突返回 500

已修复。

- `ErrModelPricingNotFound` 已改成 `ApplicationError` 风格的 404：`backend/internal/service/model_pricing_service.go:11`
- 新增 `ErrModelPricingExists`，用于冲突映射：`backend/internal/service/model_pricing_service.go:14`
- repository `Create` / `Update` 现在都走 `translatePersistenceError(...)`：`backend/internal/repository/model_pricing_repo.go:59`、`backend/internal/repository/model_pricing_repo.go:80`

### Round 1 / F1-F4：计费基线、零价启用、唯一约束、改 key 缓存失效

本轮未发现回归，之前确认过的修复仍然成立：

- migration seed 与 fallback 对齐：`backend/migrations/084_model_pricings.sql:31`
- 全 0 且启用被服务层拦截：`backend/internal/service/model_pricing_service.go:141`
- partial unique index 仍在：`backend/migrations/084_model_pricings.sql:25`
- 改 `model_key` 时旧缓存会删除：`backend/internal/service/model_pricing_service.go:93`

## 4. 测试与验证

已执行：

```bash
env GOTOOLCHAIN=auto GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org,direct \
  go test ./internal/service -run 'TestGetModelPricing|TestBillingServiceGetModelPricing|TestGetModelPricing_CaseInsensitive|TestGetModelPricing_OpenAIGPT51Fallback|TestGetModelPricing_OpenAIGPT54Fallback'
```

结果：

- 通过。

已执行：

```bash
env GOTOOLCHAIN=auto GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org,direct \
  go test ./... -run '^$'
```

结果：

- backend 全量编译通过。

## 5. 测试缺口

- 仍然没有针对 `ModelPricingService` / `ModelPricingHandler` 的专门测试文件。
- 当前没有自动化用例直接覆盖：
  - `model_key` 规范化写入
  - 纯空白 `model_key` 的 400 校验
  - 唯一约束冲突返回 409
  - 改 key 时旧缓存清理

