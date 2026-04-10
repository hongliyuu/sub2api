# Task 1 — 后端 Compare API

> 修改 `model_pricing_handler.go`：新增 `Compare` handler，返回数据库条目列表并附带每条对应的 LiteLLM 价格。

**文件：**
- 修改：`backend/internal/handler/admin/model_pricing_handler.go`
- 修改：`backend/internal/server/routes/admin.go`（第 619-630 行）

---

## 背景知识

- `ModelPricingHandler` 目前只持有 `*service.ModelPricingService`。
- `PricingService.GetModelPricing(modelName string)` 返回 `*LiteLLMModelPricing`（per-token 价格）。
- 我们需要把 `PricingService` 注入到 handler，但要最小改动——不改构造函数签名，用 setter 注入。
- `LiteLLMModelPricing` 在 `backend/internal/service/pricing_service.go` 定义，字段见下：
  ```go
  type LiteLLMModelPricing struct {
      InputCostPerToken          float64
      OutputCostPerToken         float64
      CacheReadInputTokenCost    float64
      CacheCreationInputTokenCost float64
      InputCostPerTokenPriority  float64
      OutputCostPerTokenPriority float64
      CacheReadInputTokenCostPriority float64
      // ...
  }
  ```
- per-token → per-million：乘以 1_000_000。

---

## 步骤

- [ ] **Step 1: 给 handler 添加 PricingService 字段和 setter**

在 `backend/internal/handler/admin/model_pricing_handler.go` 的 `ModelPricingHandler` struct 中添加字段，并提供 setter：

```go
// ModelPricingHandler handles admin model pricing management.
type ModelPricingHandler struct {
	svc        *service.ModelPricingService
	pricingSvc *service.PricingService // 可选，nil 时 LiteLLM 列显示 null
}

func NewModelPricingHandler(svc *service.ModelPricingService) *ModelPricingHandler {
	return &ModelPricingHandler{svc: svc}
}

// SetPricingService 注入 PricingService（启动后调用，非必须）
func (h *ModelPricingHandler) SetPricingService(svc *service.PricingService) {
	h.pricingSvc = svc
}
```

- [ ] **Step 2: 定义 Compare 响应类型**

在同一文件中，在现有 `modelPricingResponse` 之后添加：

```go
// liteLLMPriceSnapshot LiteLLM 动态价格快照（per-million USD）
// nil 表示该模型在 LiteLLM 数据中无记录
type liteLLMPriceSnapshot struct {
	InputPerMillion        float64 `json:"input_per_million"`
	OutputPerMillion       float64 `json:"output_per_million"`
	CacheReadPerMillion    float64 `json:"cache_read_per_million"`
	CacheCreationPerMillion float64 `json:"cache_creation_per_million"`
	InputPriorityPerMillion  float64 `json:"input_priority_per_million"`
	OutputPriorityPerMillion float64 `json:"output_priority_per_million"`
}

// modelPricingCompareItem 对比视图的单行数据
type modelPricingCompareItem struct {
	modelPricingResponse
	LiteLLM *liteLLMPriceSnapshot `json:"litellm"` // nil = 无数据
}
```

- [ ] **Step 3: 实现 Compare handler 方法**

在同一文件末尾添加：

```go
// Compare handles GET /admin/model-pricings/compare
// 返回数据库中所有条目，并附带每条对应的 LiteLLM 动态价格快照。
func (h *ModelPricingHandler) Compare(c *gin.Context) {
	entries, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]modelPricingCompareItem, 0, len(entries))
	for _, e := range entries {
		item := modelPricingCompareItem{
			modelPricingResponse: entryToResponse(e),
			LiteLLM:              h.fetchLiteLLMSnapshot(e.ModelKey),
		}
		out = append(out, item)
	}
	response.Success(c, out)
}

// fetchLiteLLMSnapshot 查询某个 modelKey 的 LiteLLM 价格并转换为 per-million。
// pricingSvc 未注入或查无结果时返回 nil。
func (h *ModelPricingHandler) fetchLiteLLMSnapshot(modelKey string) *liteLLMPriceSnapshot {
	if h.pricingSvc == nil {
		return nil
	}
	p := h.pricingSvc.GetModelPricing(modelKey)
	if p == nil {
		return nil
	}
	const toMillion = 1_000_000.0
	return &liteLLMPriceSnapshot{
		InputPerMillion:         p.InputCostPerToken * toMillion,
		OutputPerMillion:        p.OutputCostPerToken * toMillion,
		CacheReadPerMillion:     p.CacheReadInputTokenCost * toMillion,
		CacheCreationPerMillion: p.CacheCreationInputTokenCost * toMillion,
		InputPriorityPerMillion:  p.InputCostPerTokenPriority * toMillion,
		OutputPriorityPerMillion: p.OutputCostPerTokenPriority * toMillion,
	}
}
```

- [ ] **Step 4: 注册路由**

在 `backend/internal/server/routes/admin.go` 的 `registerModelPricingRoutes` 函数中，在现有路由之后添加：

```go
func registerModelPricingRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if h.Admin.ModelPricing == nil {
		return
	}
	pricings := admin.Group("/model-pricings")
	{
		pricings.GET("", h.Admin.ModelPricing.List)
		pricings.POST("", h.Admin.ModelPricing.Create)
		pricings.PUT("/:id", h.Admin.ModelPricing.Update)
		pricings.DELETE("/:id", h.Admin.ModelPricing.Delete)
		pricings.GET("/compare", h.Admin.ModelPricing.Compare)  // 新增
	}
}
```

> ⚠️ `/compare` 必须在 `/:id` 之前注册，否则 Gin 会把 "compare" 当成 id 参数。当前代码里 `/:id` 只绑定了 PUT/DELETE，GET 没有 `/:id`，所以没有冲突，但仍建议明确放在前面。

- [ ] **Step 5: 注入 PricingService（在 wire.go 中）**

在 `backend/internal/handler/wire.go` 的 `ProvideAdminHandlers` 函数签名中添加 `pricingService *service.PricingService` 参数，并在函数体中调用 setter：

```go
func ProvideAdminHandlers(
    // ... 现有参数不变 ...
    modelPricingHandler *admin.ModelPricingHandler,
    pricingService *service.PricingService,  // 新增
) *AdminHandlers {
    // 注入 PricingService 到 ModelPricingHandler
    if pricingService != nil {
        modelPricingHandler.SetPricingService(pricingService)
    }
    return &AdminHandlers{
        // ... 现有字段不变 ...
        ModelPricing: modelPricingHandler,
    }
}
```

- [ ] **Step 6: 构建验证**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./...
```

预期：无编译错误。如果 wire 报错（`PricingService` 未在 provider set 中），需检查 `PricingService` 是否已在 wire provider set 里。运行：

```bash
grep -r "PricingService" backend/cmd/ backend/internal/app/ 2>/dev/null | head -20
```

若 `PricingService` 已在 wire graph 中，则直接可用。若 wire 未托管，改用在启动代码中手动调用 `handler.Admin.ModelPricing.SetPricingService(pricingSvc)`。

- [ ] **Step 7: 手动验证 API**

启动服务后：
```bash
curl -s -H "Authorization: Bearer <admin_token>" \
  http://localhost:8080/api/admin/model-pricings/compare | jq '.[0]'
```

预期输出示例（有 LiteLLM 数据的条目）：
```json
{
  "id": 1,
  "model_key": "gpt-4o",
  "input_price_per_million": 5.0,
  "output_price_per_million": 15.0,
  "litellm": {
    "input_per_million": 5.0,
    "output_per_million": 15.0,
    "cache_read_per_million": 1.25,
    "cache_creation_per_million": 0,
    "input_priority_per_million": 0,
    "output_priority_per_million": 0
  }
}
```

- [ ] **Step 8: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/handler/admin/model_pricing_handler.go \
        backend/internal/server/routes/admin.go \
        backend/internal/handler/wire.go
git commit -m "Feature: 后端新增 model-pricings/compare API，附带 LiteLLM 价格快照"
```
