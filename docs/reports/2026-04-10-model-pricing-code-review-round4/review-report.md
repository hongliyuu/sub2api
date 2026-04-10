# 2026-04-10 Model Pricing 复审报告（Round 4）

## 1. 结论

本轮复审没有发现新的功能性问题。

cc 这次补的 R3 修复是正确的：

- `Create` / `Update` 仍然先执行 `normalizeModelKey()`：`backend/internal/service/model_pricing_service.go:73`、`backend/internal/service/model_pricing_service.go:87`
- 规范化后的空 key 会在 `validatePricingEntry()` 里直接以 `MODEL_PRICING_EMPTY_KEY` 拒绝：`backend/internal/service/model_pricing_service.go:144`
- 因此纯空白输入不会再落到 Ent `ValidationError`，也不会再冒成 500

结合上一轮已经确认的 R1 / R2 修复，本次 feature 在我审查到的范围内已经没有剩余 code-review finding。

**建议状态：APPROVE。**

## 2. 本轮核验点

### R3：纯空白 `model_key` 不应再触发 500

已修复，逻辑闭环如下：

1. handler 收到请求后把 `model_key` 原样映射到 entry：`backend/internal/handler/admin/model_pricing_handler.go:119`
2. service 在 Create / Update 一进入就先规范化 key：`backend/internal/service/model_pricing_service.go:73`、`backend/internal/service/model_pricing_service.go:87`
3. `normalizeModelKey()` 会执行 `strings.TrimSpace(strings.ToLower(...))`：`backend/internal/service/model_pricing_service.go:160`
4. `validatePricingEntry()` 现在首先检查规范化后的 `entry.ModelKey == ""`，并返回 `infraerrors.BadRequest("MODEL_PRICING_EMPTY_KEY", ...)`：`backend/internal/service/model_pricing_service.go:145`

这意味着输入 `"   "` 的执行路径已经变成：

- `"   "` -> `""`
- service 返回 HTTP 400
- 请求不会继续进入 repository / Ent 持久化层

### Round 2 / R1：`model_key` 规范化

仍然有效，没有回归：

- 服务端统一 lower-case + trim：`backend/internal/service/model_pricing_service.go:160`
- 与 `BillingService` 查缓存前的 lower-case 逻辑保持一致：`backend/internal/service/billing_service.go:399`

### Round 2 / R2：唯一约束冲突返回 409

仍然有效，没有回归：

- `ErrModelPricingNotFound` / `ErrModelPricingExists` 现在都是 `ApplicationError`：`backend/internal/service/model_pricing_service.go:11`、`backend/internal/service/model_pricing_service.go:14`
- repository `Create` / `Update` 继续通过 `translatePersistenceError(...)` 做错误翻译：`backend/internal/repository/model_pricing_repo.go:59`、`backend/internal/repository/model_pricing_repo.go:80`

### Round 1 / F1-F4：计费基线、零价启用、唯一约束、缓存失效

本轮未发现这些路径有回归：

- migration seed / fallback 对齐仍在：`backend/migrations/084_model_pricings.sql:31`
- 零价启用校验仍在：`backend/internal/service/model_pricing_service.go:148`
- partial unique index 仍在：`backend/migrations/084_model_pricings.sql:25`
- 改 key 清旧缓存仍在：`backend/internal/service/model_pricing_service.go:93`

## 3. 测试与验证

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

## 4. 剩余风险 / 测试缺口

虽然这轮没有新的 code-review finding，但仍然建议后续补 dedicated tests，直接锁住这些行为：

- 纯空白 `model_key` 返回 400
- `model_key` 写入前会被规范化
- 唯一约束冲突返回 409
- 改 key 时旧缓存会清理

这些更像质量护栏缺口，不是当前阻断问题。
