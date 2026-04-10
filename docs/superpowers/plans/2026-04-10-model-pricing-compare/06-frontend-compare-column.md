# Task 6 — 前端页面：表格对比列（C1）

> 在数据库条目表格中新增「LiteLLM 对比」列，调用 compare API 替换原来的 list，并用颜色+符号展示差异。

**文件：**
- 修改：`frontend/src/views/admin/ModelPricingView.vue`

---

## 背景知识

- 现有 `loadEntries()` 调用 `modelPricingsAPI.list()`，返回 `ModelPricingEntry[]`。
- 改为调用 `modelPricingsAPI.compare()`，返回 `ModelPricingCompareItem[]`（继承 `ModelPricingEntry`，多了 `litellm: PriceTier | null`）。
- 差异判断：以输入价格（`input_per_million`）为主判断基准。
  - `|db - litellm| / litellm < 0.01` → 一致（`=`，灰色）
  - `db > litellm * 1.01` → 偏高（`↑`，红色）
  - `db < litellm * 0.99` → 偏低（`↓`，绿色）
  - `litellm === null` → 无数据（`?`，黄色）
- 已有 `formatPrice()` 可复用。
- `columns` 数组目前有 7 列，新增第 8 列 `litellm_compare`（放在 `enabled` 之后、`actions` 之前）。

---

## 步骤

- [ ] **Step 1: 修改 entries 类型和 loadEntries 函数**

在 `<script setup>` 中，将 `entries` 的类型从 `ModelPricingEntry[]` 改为 `ModelPricingCompareItem[]`，并修改 `loadEntries`：

```typescript
import modelPricingsAPI, {
  type ModelPricingEntry,
  type UpsertModelPricingRequest,
  type ModelPricingLookup,
  type ModelPricingCompareItem,   // 新增
} from '@/api/admin/modelPricings'

// 将原来的：
// const entries = ref<ModelPricingEntry[]>([])
// 改为：
const entries = ref<ModelPricingCompareItem[]>([])

// 将原来的 loadEntries 函数改为：
async function loadEntries() {
  loading.value = true
  try {
    entries.value = await modelPricingsAPI.compare()
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || err?.message || 'Failed to load')
  } finally {
    loading.value = false
  }
}
```

- [ ] **Step 2: 在 columns 数组中添加对比列**

找到 `columns` 数组定义，在 `enabled` 和 `actions` 之间插入：

```typescript
const columns: Column[] = [
  { key: 'model_key', label: t('admin.modelPricings.modelKey') },
  { key: 'input_price_per_million', label: t('admin.modelPricings.inputPrice') },
  { key: 'output_price_per_million', label: t('admin.modelPricings.outputPrice') },
  { key: 'cache_read_price_per_million', label: '缓存价格' },
  { key: 'enabled', label: t('admin.modelPricings.enabled') },
  { key: 'litellm_compare', label: t('admin.modelPricings.litellmCompare') },  // 新增
  { key: 'note', label: t('admin.modelPricings.note') },
  { key: 'actions', label: t('common.actions') }
]
```

- [ ] **Step 3: 添加差异计算辅助函数**

在 `formatPrice` 函数之后添加：

```typescript
type DiffLevel = 'higher' | 'lower' | 'equal' | 'nodata'

function calcDiff(dbVal: number, litellmVal: number | null | undefined): DiffLevel {
  if (litellmVal == null || litellmVal === 0) return 'nodata'
  const ratio = dbVal / litellmVal
  if (ratio > 1.01) return 'higher'
  if (ratio < 0.99) return 'lower'
  return 'equal'
}

function diffIcon(level: DiffLevel): string {
  return { higher: '↑', lower: '↓', equal: '=', nodata: '?' }[level]
}

function diffColorClass(level: DiffLevel): string {
  return {
    higher: 'text-red-600 dark:text-red-400',
    lower: 'text-green-600 dark:text-green-400',
    equal: 'text-gray-400 dark:text-dark-400',
    nodata: 'text-amber-500 dark:text-amber-400',
  }[level]
}

function diffTooltip(level: DiffLevel): string {
  return {
    higher: t('admin.modelPricings.priceDiffHigher'),
    lower: t('admin.modelPricings.priceDiffLower'),
    equal: t('admin.modelPricings.priceDiffEqual'),
    nodata: t('admin.modelPricings.priceDiffNoData'),
  }[level]
}
```

- [ ] **Step 4: 在 `<template>` 的 DataTable 中添加对比列 slot**

在现有 `#cell-enabled` slot 之后（`#cell-note` 之前）插入：

```html
<!-- LiteLLM 对比列 -->
<template #cell-litellm_compare="{ row }">
  <div class="flex flex-col gap-0.5 text-xs">
    <!-- 输入价格对比 -->
    <div class="flex items-center gap-1">
      <span class="w-6 text-gray-400">In</span>
      <template v-if="row.litellm">
        <span :class="diffColorClass(calcDiff(row.input_price_per_million, row.litellm.input_per_million))"
              :title="diffTooltip(calcDiff(row.input_price_per_million, row.litellm.input_per_million))">
          {{ diffIcon(calcDiff(row.input_price_per_million, row.litellm.input_per_million)) }}
        </span>
        <span class="font-mono text-gray-500 dark:text-dark-400">
          ${{ formatPrice(row.litellm.input_per_million) }}
        </span>
      </template>
      <span v-else :class="diffColorClass('nodata')" :title="diffTooltip('nodata')">?</span>
    </div>
    <!-- 输出价格对比 -->
    <div class="flex items-center gap-1">
      <span class="w-6 text-gray-400">Out</span>
      <template v-if="row.litellm">
        <span :class="diffColorClass(calcDiff(row.output_price_per_million, row.litellm.output_per_million))"
              :title="diffTooltip(calcDiff(row.output_price_per_million, row.litellm.output_per_million))">
          {{ diffIcon(calcDiff(row.output_price_per_million, row.litellm.output_per_million)) }}
        </span>
        <span class="font-mono text-gray-500 dark:text-dark-400">
          ${{ formatPrice(row.litellm.output_per_million) }}
        </span>
      </template>
      <span v-else :class="diffColorClass('nodata')" :title="diffTooltip('nodata')">?</span>
    </div>
  </div>
</template>
```

列显示效果示例：
```
In  ↑ $5.00    ← 你配的高于 LiteLLM（红色箭头，旁边是 LiteLLM 的价格）
Out = $15.00   ← 一致（灰色等号）
```

- [ ] **Step 5: 本地验证**

访问 `/admin/model-pricing`，验证：
1. 表格新增「LiteLLM 对比」列
2. 有 LiteLLM 数据的行显示 `↑`/`↓`/`=` + LiteLLM 价格
3. 无 LiteLLM 数据的行显示 `?`（黄色）
4. hover 图标显示 tooltip 说明文字
5. 刷新按钮仍可用（调用 `loadEntries` → `compare()`）

- [ ] **Step 6: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit
```

预期：无类型错误。

- [ ] **Step 7: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/views/admin/ModelPricingView.vue
git commit -m "Feature: 模型价格表格新增 LiteLLM 对比列（C1），显示价差符号与方向"
```
