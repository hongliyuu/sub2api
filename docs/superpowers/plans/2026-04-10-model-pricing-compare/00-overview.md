# Model Pricing Compare View — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `/admin/model-pricing` 页面同时展示 LiteLLM 动态价格、数据库配置价格、内置 Fallback 三层对比，并支持查询任意模型名的三层价格。

**Architecture:**
- 后端新增两个 API：`GET /admin/model-pricings/compare`（数据库条目+LiteLLM对比）和 `GET /admin/model-pricings/lookup?model=xxx`（任意模型三层查询）。`PricingService` 已有 `GetModelPricing()`，`BillingService` 已有 `fallbackPrices`，只需在 handler 层组合暴露。
- 前端页面新增「模型查询」卡片区域（C2）和在现有表格的 LiteLLM 对比列（C1），用颜色+符号直观显示价差。
- 不修改现有 CRUD 逻辑，只新增 read-only 接口。

**Tech Stack:** Go/Gin（后端）、Vue 3 + TypeScript（前端）、已有 `PricingService`、`BillingService`、`ModelPricingService`

---

## 文件清单

### 后端（新增/修改）

| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/internal/handler/admin/model_pricing_handler.go` | 修改 | 新增 `Compare` 和 `Lookup` handler 方法 |
| `backend/internal/server/routes/admin.go` | 修改 | 注册两条新路由 |

### 前端（新增/修改）

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/api/admin/modelPricings.ts` | 修改 | 新增 `compare()` 和 `lookup()` API 函数及类型 |
| `frontend/src/views/admin/ModelPricingView.vue` | 修改 | 新增查询卡片 + 表格对比列 |
| `frontend/src/i18n/locales/zh.ts` | 修改 | 新增 i18n key |
| `frontend/src/i18n/locales/en.ts` | 修改 | 新增 i18n key |

---

## 子任务文件索引

- [Task 1 — 后端 Compare API](./01-backend-compare-api.md)
- [Task 2 — 后端 Lookup API](./02-backend-lookup-api.md)
- [Task 3 — 前端 API 层](./03-frontend-api.md)
- [Task 4 — 前端 i18n](./04-frontend-i18n.md)
- [Task 5 — 前端页面：查询卡片（C2）](./05-frontend-lookup-card.md)
- [Task 6 — 前端页面：表格对比列（C1）](./06-frontend-compare-column.md)
