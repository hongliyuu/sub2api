# Task 2 — 后端 Lookup API

> 新增 `GET /admin/model-pricings/lookup?model=xxx`，返回指定模型在三层（LiteLLM / 数据库 / Fallback）的价格对比。

**文件：**
- 修改：`backend/internal/handler/admin/model_pricing_handler.go`
- 修改：`backend/internal/server/routes/admin.go`

---

## 背景知识

- `BillingService.fallbackPrices` 是 `map[string]*ModelPricing`，字段是 per-token，存储在私有字段。
- 需要在 `BillingService` 上暴露一个 `GetFallbackPricing(modelKey string) *ModelPricing` 方法（key 已是小写）。
- `ModelPricing` 在 `backend/internal/service/billing_service.go` 定义，per-token 价格需 ×1_000_000 转为 per-million。
- 三层查询逻辑：
  1. LiteLLM：`pricingSvc.GetModelPricing(model)`
  2. 数据库：`modelPricingSvc` 的 repo 查询（`GetByKey`）
  3. Fallback：`billingSvc.GetFallbackPricing(model)`

---

## 步骤

- [ ] **Step 1: 在 BillingService 上暴露 GetFallbackPricing**

在 `backend/internal/service/billing_service.go` 末尾添加：

```go
// GetFallbackPricing 返回指定 modelKey 的硬编码 Fallback 价格，未命中返回 nil。
// modelKey 应为小写。
func (s *BillingService) GetFallbackPricing(modelKey string) *ModelPricing {
	return s.fallbackPrices[strings.ToLower(strings.TrimSpace(modelKey))]
}
```

- [ ] **Step 2: 给 ModelPricingHandler 添加 BillingService 字段和 setter**

在 `backend/internal/handler/admin/model_pricing_handler.go` 的 struct 中添加：

```go
type ModelPricingHandler struct {
	svc        *service.ModelPricingService
	pricingSvc *service.PricingService  // 已在 Task 1 添加
	billingSvc *service.BillingService  // 新增：用于 Fallback 查询
}

// SetBillingService 注入 BillingService（启动后调用，非必须）
func (h *ModelPricingHandler) SetBillingService(svc *service.BillingService) {
	h.billingSvc = svc
}
```

- [ ] **Step 3: 定义 Lookup 响应类型**

在 `model_pricing_handler.go` 中，在 `liteLLMPriceSnapshot` 之后添加：

```go
// priceTier 单层价格数据（per-million USD）
type priceTier struct {
	InputPerMillion         float64 `json:"input_per_million"`
	OutputPerMillion        float64 `json:"output_per_million"`
	CacheReadPerMillion     float64 `json:"cache_read_per_million"`
	CacheCreationPerMillion float64 `json:"cache_creation_per_million"`
	InputPriorityPerMillion  float64 `json:"input_priority_per_million"`
	OutputPriorityPerMillion float64 `json:"output_priority_per_million"`
}

// modelPricingLookupResponse 三层价格对比
type modelPricingLookupResponse struct {
	Model    string     `json:"model"`
	LiteLLM  *priceTier `json:"litellm"`   // nil = 无数据
	Database *priceTier `json:"database"`  // nil = 无数据
	Fallback *priceTier `json:"fallback"`  // nil = 无数据
	// ActiveSource 表示实际生效的层："litellm" | "database" | "fallback" | "none"
	ActiveSource string `json:"active_source"`
}
```

- [ ] **Step 4: 实现 Lookup handler**

在 `model_pricing_handler.go` 末尾添加：

```go
// Lookup handles GET /admin/model-pricings/lookup?model=xxx
// 返回指定模型在三层的价格对比，并标注当前生效层。
func (h *ModelPricingHandler) Lookup(c *gin.Context) {
	model := strings.ToLower(strings.TrimSpace(c.Query("model")))
	if model == "" {
		response.BadRequest(c, "model query parameter is required")
		return
	}

	resp := modelPricingLookupResponse{
		Model: model,
	}

	// 1. LiteLLM 层
	if h.pricingSvc != nil {
		if p := h.pricingSvc.GetModelPricing(model); p != nil {
			resp.LiteLLM = liteLLMToPriceTier(p)
		}
	}

	// 2. 数据库层（精确匹配 model_key）
	dbEntry, err := h.svc.GetByKey(c.Request.Context(), model)
	if err == nil && dbEntry != nil && dbEntry.Enabled {
		resp.Database = dbEntryToPriceTier(dbEntry)
	}

	// 3. Fallback 层
	if h.billingSvc != nil {
		if p := h.billingSvc.GetFallbackPricing(model); p != nil {
			resp.Fallback = modelPricingToPriceTier(p)
		}
	}

	// 确定生效层（优先级顺序）
	switch {
	case resp.LiteLLM != nil:
		resp.ActiveSource = "litellm"
	case resp.Database != nil:
		resp.ActiveSource = "database"
	case resp.Fallback != nil:
		resp.ActiveSource = "fallback"
	default:
		resp.ActiveSource = "none"
	}

	response.Success(c, resp)
}
```

- [ ] **Step 5: 实现转换辅助函数**

在同一文件中添加（放在 `fetchLiteLLMSnapshot` 之后）：

```go
const toMillion = 1_000_000.0

func liteLLMToPriceTier(p *service.LiteLLMModelPricing) *priceTier {
	return &priceTier{
		InputPerMillion:          p.InputCostPerToken * toMillion,
		OutputPerMillion:         p.OutputCostPerToken * toMillion,
		CacheReadPerMillion:      p.CacheReadInputTokenCost * toMillion,
		CacheCreationPerMillion:  p.CacheCreationInputTokenCost * toMillion,
		InputPriorityPerMillion:  p.InputCostPerTokenPriority * toMillion,
		OutputPriorityPerMillion: p.OutputCostPerTokenPriority * toMillion,
	}
}

func dbEntryToPriceTier(e *service.ModelPricingEntry) *priceTier {
	return &priceTier{
		InputPerMillion:          e.InputPricePerMillion,
		OutputPerMillion:         e.OutputPricePerMillion,
		CacheReadPerMillion:      e.CacheReadPricePerMillion,
		CacheCreationPerMillion:  e.CacheCreationPricePerMillion,
		InputPriorityPerMillion:  e.InputPricePerMillionPriority,
		OutputPriorityPerMillion: e.OutputPricePerMillionPriority,
	}
}

func modelPricingToPriceTier(p *service.ModelPricing) *priceTier {
	return &priceTier{
		InputPerMillion:          p.InputPricePerToken * toMillion,
		OutputPerMillion:         p.OutputPricePerToken * toMillion,
		CacheReadPerMillion:      p.CacheReadPricePerToken * toMillion,
		CacheCreationPerMillion:  p.CacheCreationPricePerToken * toMillion,
		InputPriorityPerMillion:  p.InputPricePerTokenPriority * toMillion,
		OutputPriorityPerMillion: p.OutputPricePerTokenPriority * toMillion,
	}
}
```

- [ ] **Step 6: 在 ModelPricingService 暴露 GetByKey**

`ModelPricingRepository` 已有 `GetByKey`，但 `ModelPricingService` 未对外暴露。在 `backend/internal/service/model_pricing_service.go` 末尾添加：

```go
// GetByKey 按 model_key 精确查询价格条目（含 disabled）。
// 未找到返回 nil, nil。
func (s *ModelPricingService) GetByKey(ctx context.Context, modelKey string) (*ModelPricingEntry, error) {
	return s.repo.GetByKey(ctx, modelKey)
}
```

- [ ] **Step 7: 注册路由**

在 `backend/internal/server/routes/admin.go` 的 `registerModelPricingRoutes` 中再添加一条：

```go
pricings.GET("/compare", h.Admin.ModelPricing.Compare)   // Task 1 已添加
pricings.GET("/lookup", h.Admin.ModelPricing.Lookup)     // 新增
```

- [ ] **Step 8: 在 wire.go 注入 BillingService**

在 `backend/internal/handler/wire.go` 的 `ProvideAdminHandlers` 中添加 `billingService *service.BillingService` 参数并注入：

```go
func ProvideAdminHandlers(
    // ... 现有参数 ...
    modelPricingHandler *admin.ModelPricingHandler,
    pricingService *service.PricingService,
    billingService *service.BillingService,  // 新增
) *AdminHandlers {
    if pricingService != nil {
        modelPricingHandler.SetPricingService(pricingService)
    }
    if billingService != nil {
        modelPricingHandler.SetBillingService(billingService)
    }
    return &AdminHandlers{
        // ... 现有字段 ...
        ModelPricing: modelPricingHandler,
    }
}
```

- [ ] **Step 9: 构建验证**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./...
```

预期：无编译错误。

- [ ] **Step 10: 手动验证**

```bash
curl -s -H "Authorization: Bearer <admin_token>" \
  "http://localhost:8080/api/admin/model-pricings/lookup?model=gpt-4o" | jq .
```

预期输出示例：
```json
{
  "model": "gpt-4o",
  "litellm": {
    "input_per_million": 5.0,
    "output_per_million": 15.0,
    "cache_read_per_million": 1.25,
    "cache_creation_per_million": 0,
    "input_priority_per_million": 0,
    "output_priority_per_million": 0
  },
  "database": null,
  "fallback": null,
  "active_source": "litellm"
}
```

- [ ] **Step 11: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/service/billing_service.go \
        backend/internal/service/model_pricing_service.go \
        backend/internal/handler/admin/model_pricing_handler.go \
        backend/internal/server/routes/admin.go \
        backend/internal/handler/wire.go
git commit -m "Feature: 后端新增 model-pricings/lookup API，支持三层价格对比查询"
```
