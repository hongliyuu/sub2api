# Task 5 — 前端页面：查询卡片（C2）

> 在 `ModelPricingView.vue` 顶部新增一个「查询任意模型价格」卡片，输入模型名后展示三层价格对比。

**文件：**
- 修改：`frontend/src/views/admin/ModelPricingView.vue`

---

## 背景知识

- `modelPricingsAPI.lookup(model)` 返回 `ModelPricingLookup`：
  ```typescript
  {
    model: string
    litellm: PriceTier | null
    database: PriceTier | null
    fallback: PriceTier | null
    active_source: 'litellm' | 'database' | 'fallback' | 'none'
  }
  ```
- `PriceTier` 字段均为 per-million USD。
- 已有 `formatPrice(v: number): string` 辅助函数可复用。
- 页面使用 `useI18n()` 的 `t()` 函数，i18n key 均以 `admin.modelPricings.` 开头。
- 样式跟随项目已有的 Tailwind + dark mode 约定（`dark:` 前缀）。

---

## 步骤

- [ ] **Step 1: 在 `<script setup>` 中添加 Lookup 状态和逻辑**

在 `ModelPricingView.vue` 的 `<script setup>` 块中，在 `onMounted(loadEntries)` 之前添加：

```typescript
import type { ModelPricingLookup } from '@/api/admin/modelPricings'

// ── Lookup（C2） ─────────────────────────────────────────────────────────────
const lookupModel = ref('')
const lookupResult = ref<ModelPricingLookup | null>(null)
const lookupLoading = ref(false)
const lookupError = ref('')

async function doLookup() {
  const model = lookupModel.value.trim()
  if (!model) return
  lookupLoading.value = true
  lookupError.value = ''
  lookupResult.value = null
  try {
    lookupResult.value = await modelPricingsAPI.lookup(model)
  } catch (err: any) {
    lookupError.value = err?.response?.data?.message || err?.message || 'Failed'
  } finally {
    lookupLoading.value = false
  }
}

function lookupSourceLabel(source: string): string {
  const map: Record<string, string> = {
    litellm: t('admin.modelPricings.activeSourceLitellm'),
    database: t('admin.modelPricings.activeSourceDatabase'),
    fallback: t('admin.modelPricings.activeSourceFallback'),
    none: t('admin.modelPricings.activeSourceNone'),
  }
  return map[source] ?? source
}

function lookupSourceColor(source: string): string {
  const map: Record<string, string> = {
    litellm: 'badge-primary',
    database: 'badge-success',
    fallback: 'badge-warning',
    none: 'badge-gray',
  }
  return map[source] ?? 'badge-gray'
}
```

- [ ] **Step 2: 在 `<template>` 中插入查询卡片**

在 `<AppLayout>` 内、`<TablePageLayout>` 之前，插入以下完整卡片块：

```html
<!-- ── Lookup Card（C2）────────────────────────────────────────── -->
<div class="mb-4 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
  <h3 class="mb-3 text-sm font-semibold text-gray-800 dark:text-white">
    {{ t('admin.modelPricings.lookupTitle') }}
  </h3>
  <div class="flex items-center gap-2">
    <input
      v-model="lookupModel"
      type="text"
      class="input flex-1"
      :placeholder="t('admin.modelPricings.lookupPlaceholder')"
      @keydown.enter="doLookup"
    />
    <button
      @click="doLookup"
      :disabled="lookupLoading || !lookupModel.trim()"
      class="btn btn-primary whitespace-nowrap"
    >
      {{ lookupLoading ? t('admin.modelPricings.lookupLoading') : t('admin.modelPricings.lookupButton') }}
    </button>
  </div>

  <!-- 错误提示 -->
  <p v-if="lookupError" class="mt-2 text-sm text-red-500">{{ lookupError }}</p>

  <!-- 结果表格 -->
  <div v-if="lookupResult" class="mt-4">
    <div class="mb-2 flex items-center gap-2">
      <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
        {{ lookupResult.model }}
      </span>
      <span :class="['badge text-xs', lookupSourceColor(lookupResult.active_source)]">
        {{ lookupSourceLabel(lookupResult.active_source) }}
      </span>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-200 dark:border-dark-600">
            <th class="py-2 pr-4 text-left text-xs font-medium text-gray-500 dark:text-dark-400 w-32">
              {{ t('admin.modelPricings.tierLabel') }}
            </th>
            <th class="py-2 pr-4 text-left text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('admin.modelPricings.inputOutput') }}
            </th>
            <th class="py-2 pr-4 text-left text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('admin.modelPricings.cacheReadCreation') }}
            </th>
            <th class="py-2 text-left text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('admin.modelPricings.priorityPrice') }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <!-- LiteLLM 行 -->
          <tr :class="lookupResult.active_source === 'litellm' ? 'bg-blue-50 dark:bg-blue-900/20' : ''">
            <td class="py-2 pr-4">
              <div class="flex items-center gap-1">
                <span class="font-medium text-blue-700 dark:text-blue-300">LiteLLM</span>
                <span v-if="lookupResult.active_source === 'litellm'" class="badge badge-primary text-xs">
                  {{ t('admin.modelPricings.activeTag') }}
                </span>
              </div>
            </td>
            <td class="py-2 pr-4 font-mono text-xs">
              <template v-if="lookupResult.litellm">
                ${{ formatPrice(lookupResult.litellm.input_per_million) }} /
                ${{ formatPrice(lookupResult.litellm.output_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="py-2 pr-4 font-mono text-xs">
              <template v-if="lookupResult.litellm">
                ${{ formatPrice(lookupResult.litellm.cache_read_per_million) }} /
                ${{ formatPrice(lookupResult.litellm.cache_creation_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="py-2 font-mono text-xs">
              <template v-if="lookupResult.litellm && lookupResult.litellm.input_priority_per_million > 0">
                ${{ formatPrice(lookupResult.litellm.input_priority_per_million) }} /
                ${{ formatPrice(lookupResult.litellm.output_priority_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
          </tr>

          <!-- 数据库行 -->
          <tr :class="lookupResult.active_source === 'database' ? 'bg-green-50 dark:bg-green-900/20' : ''">
            <td class="py-2 pr-4">
              <div class="flex items-center gap-1">
                <span class="font-medium text-green-700 dark:text-green-300">{{ t('admin.modelPricings.dbPrice') }}</span>
                <span v-if="lookupResult.active_source === 'database'" class="badge badge-success text-xs">
                  {{ t('admin.modelPricings.activeTag') }}
                </span>
              </div>
            </td>
            <td class="py-2 pr-4 font-mono text-xs">
              <template v-if="lookupResult.database">
                ${{ formatPrice(lookupResult.database.input_per_million) }} /
                ${{ formatPrice(lookupResult.database.output_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="py-2 pr-4 font-mono text-xs">
              <template v-if="lookupResult.database">
                ${{ formatPrice(lookupResult.database.cache_read_per_million) }} /
                ${{ formatPrice(lookupResult.database.cache_creation_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="py-2 font-mono text-xs">
              <template v-if="lookupResult.database && lookupResult.database.input_priority_per_million > 0">
                ${{ formatPrice(lookupResult.database.input_priority_per_million) }} /
                ${{ formatPrice(lookupResult.database.output_priority_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
          </tr>

          <!-- Fallback 行 -->
          <tr :class="lookupResult.active_source === 'fallback' ? 'bg-amber-50 dark:bg-amber-900/20' : ''">
            <td class="py-2 pr-4">
              <div class="flex items-center gap-1">
                <span class="font-medium text-amber-700 dark:text-amber-300">{{ t('admin.modelPricings.fallbackPrice') }}</span>
                <span v-if="lookupResult.active_source === 'fallback'" class="badge badge-warning text-xs">
                  {{ t('admin.modelPricings.activeTag') }}
                </span>
              </div>
            </td>
            <td class="py-2 pr-4 font-mono text-xs">
              <template v-if="lookupResult.fallback">
                ${{ formatPrice(lookupResult.fallback.input_per_million) }} /
                ${{ formatPrice(lookupResult.fallback.output_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="py-2 pr-4 font-mono text-xs">
              <template v-if="lookupResult.fallback">
                ${{ formatPrice(lookupResult.fallback.cache_read_per_million) }} /
                ${{ formatPrice(lookupResult.fallback.cache_creation_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="py-2 font-mono text-xs">
              <template v-if="lookupResult.fallback && lookupResult.fallback.input_priority_per_million > 0">
                ${{ formatPrice(lookupResult.fallback.input_priority_per_million) }} /
                ${{ formatPrice(lookupResult.fallback.output_priority_per_million) }}
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 无任何数据提示 -->
    <p v-if="lookupResult.active_source === 'none'" class="mt-2 text-sm text-gray-500 dark:text-dark-400">
      {{ t('admin.modelPricings.lookupNoResult') }}
    </p>
  </div>
</div>
```

- [ ] **Step 3: 检查 import**

确保 `<script setup>` 顶部已从 API 模块导入 `ModelPricingLookup` 类型（Task 3 已定义）：

```typescript
import modelPricingsAPI, {
  type ModelPricingEntry,
  type UpsertModelPricingRequest,
  type ModelPricingLookup,   // 新增
} from '@/api/admin/modelPricings'
```

- [ ] **Step 4: 本地验证**

启动开发服务器并访问 `/admin/model-pricing`：
```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run dev
```

1. 在查询框输入 `gpt-4o`，点击「查询」
2. 验证出现三行（LiteLLM / 数据库 / Fallback），有数据的行显示价格，生效行有背景色高亮和「生效中」徽章
3. 输入一个不存在的模型名（如 `fake-model-xyz`），验证显示「无价格数据」提示

- [ ] **Step 5: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit
```

预期：无类型错误。

- [ ] **Step 6: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/views/admin/ModelPricingView.vue
git commit -m "Feature: 模型价格页新增任意模型查询卡片（三层价格对比 C2）"
```
