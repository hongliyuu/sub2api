# Task 3 — 前端 API 层

> 在 `frontend/src/api/admin/modelPricings.ts` 中新增 `compare()` 和 `lookup()` 函数及对应 TypeScript 类型。

**文件：**
- 修改：`frontend/src/api/admin/modelPricings.ts`

---

## 步骤

- [ ] **Step 1: 新增类型定义**

在 `frontend/src/api/admin/modelPricings.ts` 现有内容末尾（`export default modelPricingsAPI` 之前）插入：

```typescript
// ── Compare API ───────────────────────────────────────────────────────────────

/** 单层价格数据（per-million USD），null 表示该层无数据 */
export interface PriceTier {
  input_per_million: number
  output_per_million: number
  cache_read_per_million: number
  cache_creation_per_million: number
  input_priority_per_million: number
  output_priority_per_million: number
}

/** Compare API 返回的单行：数据库条目 + LiteLLM 对比 */
export interface ModelPricingCompareItem extends ModelPricingEntry {
  litellm: PriceTier | null
}

// ── Lookup API ────────────────────────────────────────────────────────────────

export type ActiveSource = 'litellm' | 'database' | 'fallback' | 'none'

/** Lookup API 返回的三层价格对比 */
export interface ModelPricingLookup {
  model: string
  litellm: PriceTier | null
  database: PriceTier | null
  fallback: PriceTier | null
  active_source: ActiveSource
}
```

- [ ] **Step 2: 新增 API 函数**

在类型定义之后、`export const modelPricingsAPI` 之前添加：

```typescript
/** 获取数据库所有条目并附带 LiteLLM 价格快照（用于对比列） */
export async function compare(): Promise<ModelPricingCompareItem[]> {
  const { data } = await apiClient.get<ModelPricingCompareItem[]>('/admin/model-pricings/compare')
  return data
}

/** 查询任意模型在三层的价格对比 */
export async function lookup(model: string): Promise<ModelPricingLookup> {
  const { data } = await apiClient.get<ModelPricingLookup>('/admin/model-pricings/lookup', {
    params: { model }
  })
  return data
}
```

- [ ] **Step 3: 导出新函数**

将文件末尾的导出对象修改为：

```typescript
export const modelPricingsAPI = { list, create, update, remove, compare, lookup }
export default modelPricingsAPI
```

- [ ] **Step 4: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit
```

预期：无类型错误。

- [ ] **Step 5: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/api/admin/modelPricings.ts
git commit -m "Feature: 前端 API 层新增 compare 和 lookup 函数及类型"
```
