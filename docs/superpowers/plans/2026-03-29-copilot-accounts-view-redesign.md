# Copilot Accounts View Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the top bar chart, add a cost sparkbar in the accounts table, rewrite AccountsDailyChart and QuotaTrendChart to use manual Chart.js instances with dark mode support, and optimize per-row quota refresh to patch only the changed row.

**Architecture:** Pure frontend changes across 3 files — the main view and 2 chart components. All chart migrations follow the same pattern: replace vue-chartjs `<Line>`/`<Bar>` wrappers with a `<canvas ref="chartRef">` + manual `Chart` instance managed by `onMounted`/`onBeforeUnmount`. Dark mode is detected via a `MutationObserver` on `document.documentElement` (same pattern as `UserHeatmap.vue` and `UsersDailyChart.vue`).

**Tech Stack:** Vue 3 Composition API, TypeScript, Chart.js 4, Tailwind CSS

---

## Files Affected

| File | Change |
|------|--------|
| `frontend/src/views/admin/copilot/CopilotAccountsView.vue` | Remove Bar chart + costChartData/Options/CHART_COLORS; add sparkbar in cost column; fix per-row refresh; add refresh button dark:hover:border-gray-500 |
| `frontend/src/components/admin/copilot/AccountsDailyChart.vue` | Full rewrite: replace `<Line>` (vue-chartjs) with manual `<canvas>` + Chart.js instance, MutationObserver isDark, `chart.update('active')` on days change |
| `frontend/src/components/admin/copilot/QuotaTrendChart.vue` | Full rewrite: same migration, `h-[180px]` height, dark-mode entitlement line color |

---

## Reference: Existing Pattern (UsersDailyChart.vue)

Before implementing, review `frontend/src/components/admin/copilot/UsersDailyChart.vue` — it is the canonical example of the manual Chart.js pattern used in this codebase:
- `chart` is a non-reactive module-level `let` variable (not `ref`)
- `loading.value = false; await nextTick()` before `chart = new Chart(chartRef.value, {...})`
- `chart?.destroy()` at the top of rebuild to prevent memory leaks
- Dark mode colors read from `document.documentElement.classList.contains('dark')` at chart creation time
- `onBeforeUnmount(() => chart?.destroy())`

---

## Task 1: Remove Top Bar Chart from CopilotAccountsView

**Files:**
- Modify: `frontend/src/views/admin/copilot/CopilotAccountsView.vue`

This task removes the `<!-- Cost Comparison Chart -->` section and all supporting code from the view. After this task, the page goes directly from KPI cards to the daily trend chart section.

- [ ] **Step 1: Remove the `<!-- Cost Comparison Chart -->` template block**

In `CopilotAccountsView.vue`, delete lines 118–127 (the entire card block):

```html
<!-- REMOVE this entire block -->
<!-- Cost Comparison Chart -->
<div v-if="!loading && overview && overview.accounts.length > 0" class="card p-5">
  <div class="mb-4 flex items-center justify-between">
    <div>
      <h2 class="text-sm font-semibold text-gray-900 dark:text-white">账户成本对比</h2>
      <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">各账户月估算成本 · 基于套餐定价</p>
    </div>
  </div>
  <Bar :data="costChartData" :options="costChartOptions" class="max-h-56" />
</div>
```

- [ ] **Step 2: Remove vue-chartjs import and ChartJS registrations**

In the `<script setup>` section, remove the following import and registration:

```ts
// REMOVE these lines:
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Tooltip,
  Legend,
} from 'chart.js'
// ...
ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend)
```

- [ ] **Step 3: Remove costChartData, costChartOptions, and CHART_COLORS**

Delete the following constants and computed from the script:

```ts
// REMOVE the entire CHART_COLORS constant:
const CHART_COLORS = [
  '#10b981', '#6366f1', '#f59e0b', '#3b82f6',
  '#f43f5e', '#06b6d4', '#ec4899', '#8b5cf6',
]

// REMOVE the entire costChartData computed:
const costChartData = computed(() => { ... })

// REMOVE the entire costChartOptions constant:
const costChartOptions = { ... }
```

- [ ] **Step 4: Verify the page still renders without errors**

Open the browser dev console (or check terminal for TypeScript errors). The page should show: KPI cards → daily trend chart → accounts table. No `Bar is not defined` or `ChartJS is not defined` errors.

Run TypeScript check:
```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep -i "CopilotAccountsView"
```
Expected: no errors for this file.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/admin/copilot/CopilotAccountsView.vue
git commit -m "Refactor: 移除账户成本对比柱状图"
```

---

## Task 2: Add Cost Sparkbar to Monthly Cost Column

**Files:**
- Modify: `frontend/src/views/admin/copilot/CopilotAccountsView.vue`

This task adds a sparkbar (small horizontal progress bar) inside the monthly cost `<td>` cell. The bar is normalized so the account with the highest monthly cost fills 100% and others are proportional.

- [ ] **Step 1: Add `maxMonthlyCost` computed property**

In the `// ── Chart ─────────` section (after `accountAvatarColor`), add:

```ts
const maxMonthlyCost = computed(() =>
  Math.max(1, ...(overview.value?.accounts ?? []).map(a => a.monthly_cost))
)
```

The `Math.max(1, ...)` prevents division by zero when all accounts have 0 cost.

- [ ] **Step 2: Replace the Monthly Cost `<td>` with sparkbar version**

Find this existing `<td>` (around line 303–312 after Task 1 line adjustments):

```html
<!-- Monthly Cost — plan-aware display -->
<td class="px-4 py-4 text-right">
  <span class="text-sm font-bold text-gray-900 dark:text-white">${{ formatCost(account.monthly_cost) }}</span>
  <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
    {{ planCostBreakdown(account.plan_type, account.seat_count) }}
  </p>
  <p v-if="account.budget_alert?.enabled && account.budget_alert.monthly_budget > 0" class="text-xs text-gray-400">
    / ${{ account.budget_alert.monthly_budget }} 预算
  </p>
</td>
```

Replace with:

```html
<!-- Monthly Cost — plan-aware display with sparkbar -->
<td class="px-4 py-4 text-right">
  <span class="text-sm font-bold text-gray-900 dark:text-white">
    ${{ formatCost(account.monthly_cost) }}
  </span>
  <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
    {{ planCostBreakdown(account.plan_type, account.seat_count) }}
  </p>
  <!-- Sparkbar -->
  <div class="mt-1.5 h-1 w-24 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700 ml-auto">
    <div
      class="h-full rounded-full bg-emerald-500 transition-all duration-500"
      :style="{ width: `${Math.round((account.monthly_cost / maxMonthlyCost) * 100)}%` }"
    />
  </div>
  <p v-if="account.budget_alert?.enabled && account.budget_alert.monthly_budget > 0"
     class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
    / ${{ account.budget_alert.monthly_budget }} 预算
  </p>
</td>
```

The sparkbar is `w-24` (96px wide), `h-1` (4px tall), right-aligned via `ml-auto`. The emerald color matches the "预估月费用" KPI card.

- [ ] **Step 3: Add dark:hover:border-gray-500 to the top-right refresh button**

Find the page-level refresh button (in the `<!-- Page Header -->` section):

```html
<button
  class="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
  :disabled="loading"
  @click="loadOverview"
>
```

Add `dark:hover:border-gray-500` to the class string:

```html
<button
  class="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:border-gray-500"
  :disabled="loading"
  @click="loadOverview"
>
```

- [ ] **Step 4: Verify sparkbar renders correctly**

Check that accounts with higher cost show a wider sparkbar. The highest-cost account should show a bar near 100% width.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/admin/copilot/CopilotAccountsView.vue
git commit -m "Feature: 月费用列添加 sparkbar 可视化"
```

---

## Task 3: Fix Per-Row Quota Refresh (Patch Single Row)

**Files:**
- Modify: `frontend/src/views/admin/copilot/CopilotAccountsView.vue`
- Modify: `frontend/src/api/admin/copilotAnalytics.ts`

Currently `doRefresh()` calls `await loadOverview()` after quota refresh, which re-fetches all data and causes a full table re-render. The `quota-refresh` endpoint returns a `CopilotCachedQuota` object. We'll update the API return type and patch only the changed row.

**Background:** The backend `QuotaRefresh` handler returns a `CopilotCachedQuota` which has:
- `AccountID` (int64)
- `QuotaInfo` (with `Entitlement`, `Remaining`, `Used`, `Overage`, `Unlimited`, `ExternalUsed`, `CachedAt`)
- `CachedAt` (time.Time)

The frontend API function currently types this as `Promise<unknown>`. We will type it as `Promise<RefreshedQuotaResult>` (a minimal interface matching what the backend returns as JSON).

- [ ] **Step 1: Add `RefreshedQuotaResult` interface to copilotAnalytics.ts**

In `frontend/src/api/admin/copilotAnalytics.ts`, after the `CopilotAccountQuotaSnapshot` interface (around line 114), add:

```ts
// Returned by POST /accounts/:id/quota-refresh
export interface RefreshedQuotaResult {
  account_id: number
  quota_info: {
    entitlement: number
    remaining: number
    used: number
    overage: number
    unlimited: boolean
    external_used: number
    cached_at: string | null
  } | null
  cached_at: string
}
```

- [ ] **Step 2: Update `refreshCopilotAccountQuota` return type**

In `frontend/src/api/admin/copilotAnalytics.ts`, update the function signature:

```ts
// BEFORE:
export async function refreshCopilotAccountQuota(accountId: number): Promise<unknown> {
  const { data } = await apiClient.post(`${BASE}/accounts/${accountId}/quota-refresh`)
  return data
}

// AFTER:
export async function refreshCopilotAccountQuota(accountId: number): Promise<RefreshedQuotaResult> {
  const { data } = await apiClient.post(`${BASE}/accounts/${accountId}/quota-refresh`)
  return data
}
```

- [ ] **Step 3: Check the actual JSON response shape**

Before writing the patch logic, verify the exact JSON keys the backend returns. The Go struct serializes via `json` tags. Check `CopilotCachedQuota` and `CopilotQuotaInfo`:

```bash
cd /Users/ziji/personal/github/sub2api
grep -A 20 "type CopilotCachedQuota struct" backend/internal/service/copilot_quota_cache_service.go
grep -A 30 "type CopilotQuotaInfo struct" backend/internal/pkg/copilot/types.go
```

Look at the json tags (e.g. `json:"account_id"`, `json:"quota_info"`). The `RefreshedQuotaResult` interface fields must match these tags exactly.

**Expected output** (from existing code review):
- `CopilotCachedQuota.AccountID` → `"account_id"`
- `CopilotCachedQuota.QuotaInfo` → the `CopilotQuotaInfo` struct
- `CopilotCachedQuota.CachedAt` → `"cached_at"`

Inside `CopilotQuotaInfo`, the fields relevant for `quota_snapshot` are:
- `Entitlement` → `"entitlement"` or similar
- `Remaining` → `"remaining"`
- etc.

**If the JSON keys differ from `RefreshedQuotaResult`, update the interface accordingly before proceeding.**

- [ ] **Step 4: Update `doRefresh` to patch the row instead of full reload**

In `CopilotAccountsView.vue`, replace the existing `doRefresh` function:

```ts
// BEFORE:
async function doRefresh(account: CopilotAccountOverviewEntry) {
  refreshing.value.add(account.account_id)
  try {
    await refreshCopilotAccountQuota(account.account_id)
    showToast('success', t('admin.copilot.accounts.refreshSuccess'))
    await loadOverview()
  } catch {
    showToast('error', t('admin.copilot.accounts.refreshFailed'))
  } finally {
    refreshing.value.delete(account.account_id)
  }
}
```

```ts
// AFTER:
async function doRefresh(account: CopilotAccountOverviewEntry) {
  refreshing.value.add(account.account_id)
  try {
    const updated = await refreshCopilotAccountQuota(account.account_id)
    // Patch only this row — no full reload
    if (overview.value) {
      const idx = overview.value.accounts.findIndex(a => a.account_id === account.account_id)
      if (idx !== -1) {
        const qi = updated.quota_info
        const newSnapshot: CopilotAccountQuotaSnapshot | null = qi
          ? {
              entitlement: qi.entitlement,
              remaining: qi.remaining,
              github_total_used: qi.used,
              overage: qi.overage,
              unlimited: qi.unlimited,
              external_used: qi.external_used,
              cached_at: updated.cached_at,
            }
          : null
        overview.value.accounts[idx] = {
          ...overview.value.accounts[idx],
          quota_snapshot: newSnapshot,
        }
      }
    }
    showToast('success', t('admin.copilot.accounts.refreshSuccess'))
  } catch {
    showToast('error', t('admin.copilot.accounts.refreshFailed'))
  } finally {
    refreshing.value.delete(account.account_id)
  }
}
```

**Important:** After checking the actual JSON shape in Step 3, adjust the field mapping in `newSnapshot` if needed (e.g., `qi.used` may need to be `qi.github_total_used` depending on the actual JSON key).

- [ ] **Step 5: Add `CopilotAccountQuotaSnapshot` to the import in CopilotAccountsView.vue**

The `CopilotAccountQuotaSnapshot` type is already exported from `copilotAnalytics.ts`. Add it to the existing import in `CopilotAccountsView.vue`:

```ts
// Find this import block and add CopilotAccountQuotaSnapshot:
import type {
  CopilotAccountsOverviewResult,
  CopilotAccountOverviewEntry,
  CopilotAccountBudgetAlertInfo,
  CopilotAlertStatus,
  CopilotAccountQuotaSnapshot,  // ADD THIS
} from '@/api/admin/copilotAnalytics'
```

- [ ] **Step 6: TypeScript check and verify behavior**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep -iE "(CopilotAccountsView|copilotAnalytics)"
```
Expected: no type errors.

Manual verification: click the "刷新配额" button on a row. The quota bar in that row should update immediately without the full table disappearing and reloading.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/admin/copilot/CopilotAccountsView.vue \
        frontend/src/api/admin/copilotAnalytics.ts
git commit -m "Optimize: 配额刷新改为单行更新，不再触发全表重载"
```

---

## Task 4: Rewrite AccountsDailyChart.vue — Manual Chart.js with Dark Mode

**Files:**
- Modify: `frontend/src/components/admin/copilot/AccountsDailyChart.vue`

**Reference files to read before implementing:**
- `frontend/src/components/admin/copilot/UsersDailyChart.vue` — the canonical pattern
- `frontend/src/components/admin/copilot/UserHeatmap.vue` — shows the MutationObserver isDark pattern

The current component uses vue-chartjs `<Line>` which doesn't support dark mode and requires component remounting when data changes. We rewrite it to use a manual `<canvas>` + Chart.js instance with `chart.update('active')` for smooth transitions.

Key behaviors to preserve:
- `props.days` prop (number, no default — parent always passes it)
- `getCopilotAccountsDailyStats({ days })` API call
- `LINE_COLORS` palette cycling by account_id sort order
- `dateLabels` logic (unique dates from API, sorted)
- `pointRadius: labels.length <= 14 ? 3 : 0`

New behaviors:
- Dark mode: MutationObserver watches `document.documentElement` for `class` attribute changes
- When `isDark` changes: rebuild chart (destroy + recreate with new colors)
- When `days` prop changes: re-fetch data and rebuild chart
- Loading/error/empty states with `h-[256px]` height (spec says 256px)

- [ ] **Step 1: Read the reference implementation**

Before writing any code, read `UsersDailyChart.vue` to understand the exact pattern:

```bash
cat frontend/src/components/admin/copilot/UsersDailyChart.vue
```

Note: this file loads the chart in `buildChart()`, which is called from `onMounted` and the `watch`. It sets `loading.value = false` then `await nextTick()` before creating the chart.

- [ ] **Step 2: Replace the entire AccountsDailyChart.vue content**

Replace the full file with:

```vue
<template>
  <div>
    <div v-if="loading" class="flex h-[256px] items-center justify-center text-xs text-gray-400 dark:text-gray-500">
      <svg class="mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
      加载中…
    </div>
    <div v-else-if="loadError" class="flex h-[256px] items-center justify-center text-xs text-red-500">
      加载失败：{{ loadError }}
    </div>
    <div v-else-if="!data || data.days.length === 0"
         class="flex h-[256px] items-center justify-center text-xs text-gray-400 dark:text-gray-500">
      暂无数据
    </div>
    <div v-else class="h-[256px]">
      <canvas ref="chartRef" class="!h-full !w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { getCopilotAccountsDailyStats } from '@/api/admin/copilotAnalytics'
import type { CopilotAccountsDailyStatsResult } from '@/api/admin/copilotAnalytics'
import { extractErrorMessage } from '@/api/client'

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler)

const props = defineProps<{ days: number }>()

// Palette — cycles for many accounts
const LINE_COLORS = [
  '#10b981', '#6366f1', '#f59e0b', '#3b82f6',
  '#f43f5e', '#06b6d4', '#ec4899', '#8b5cf6',
  '#84cc16', '#fb923c',
]

const chartRef = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

const loading = ref(false)
const loadError = ref<string | null>(null)
const data = ref<CopilotAccountsDailyStatsResult | null>(null)

// Dark mode reactive state
const isDark = ref(document.documentElement.classList.contains('dark'))
let darkObserver: MutationObserver | null = null

function buildChartData() {
  if (!data.value) return { labels: [], datasets: [] }

  const dates = Array.from(new Set(data.value.days.map(d => d.date))).sort()

  const countMap = new Map<number, Map<string, number>>()
  for (const entry of data.value.days) {
    if (!countMap.has(entry.account_id)) countMap.set(entry.account_id, new Map())
    countMap.get(entry.account_id)!.set(entry.date, entry.count)
  }

  const accountsSorted = [...data.value.accounts].sort((a, b) => a.account_id - b.account_id)

  const datasets = accountsSorted.map((acc, idx) => {
    const color = LINE_COLORS[idx % LINE_COLORS.length]
    const dayMap = countMap.get(acc.account_id) ?? new Map()
    return {
      label: acc.name,
      data: dates.map(d => dayMap.get(d) ?? 0),
      borderColor: color,
      backgroundColor: color + '18',
      borderWidth: 2,
      pointRadius: dates.length <= 14 ? 3 : 0,
      pointHoverRadius: 5,
      tension: 0.3,
      fill: false,
    }
  })

  return { labels: dates, datasets }
}

function buildChartOptions() {
  const dark = isDark.value
  const tickColor  = dark ? '#9ca3af' : '#6b7280'
  const gridColor  = dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)'
  const legendColor = dark ? '#d1d5db' : '#374151'

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: {
        display: true,
        position: 'bottom' as const,
        labels: { boxWidth: 12, font: { size: 11 }, padding: 16, color: legendColor },
      },
      tooltip: {
        callbacks: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          title: (items: any[]) => items[0]?.label ?? '',
        },
      },
    },
    scales: {
      x: {
        grid: { display: false },
        ticks: { font: { size: 10 }, maxTicksLimit: 14, color: tickColor },
      },
      y: {
        beginAtZero: true,
        grid: { color: gridColor },
        ticks: { font: { size: 10 }, precision: 0, color: tickColor },
      },
    },
  }
}

async function renderChart() {
  if (!data.value || data.value.days.length === 0) return

  // Show canvas before creating chart so chartRef.value is valid
  loading.value = false
  await nextTick()

  if (!chartRef.value) return

  if (chart) {
    // Smooth update without full re-create
    chart.data = buildChartData()
    // Update options in-place for dark mode changes
    const opts = buildChartOptions()
    chart.options = opts as typeof chart.options
    chart.update('active')
  } else {
    chart = new Chart(chartRef.value, {
      type: 'line',
      data: buildChartData(),
      options: buildChartOptions() as any, // eslint-disable-line @typescript-eslint/no-explicit-any
    })
  }
}

async function load() {
  loading.value = true
  loadError.value = null
  chart?.destroy()
  chart = null
  try {
    data.value = await getCopilotAccountsDailyStats({ days: props.days })
    if (!data.value || data.value.days.length === 0) {
      // Empty state — show the "暂无数据" branch
      loading.value = false
      return
    }
    await renderChart()
  } catch (e: unknown) {
    loadError.value = extractErrorMessage(e)
    loading.value = false
  }
}

onMounted(() => {
  darkObserver = new MutationObserver(() => {
    const newDark = document.documentElement.classList.contains('dark')
    if (newDark !== isDark.value) {
      isDark.value = newDark
      // Rebuild chart with new colors
      chart?.destroy()
      chart = null
      renderChart()
    }
  })
  darkObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  load()
})

watch(() => props.days, load)

onBeforeUnmount(() => {
  chart?.destroy()
  chart = null
  darkObserver?.disconnect()
  darkObserver = null
})
</script>
```

- [ ] **Step 3: TypeScript check**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep -i "AccountsDailyChart"
```
Expected: no errors.

- [ ] **Step 4: Verify in browser**

1. Open the accounts page — daily chart should load with colored lines per account
2. Toggle dark mode — chart colors (tick labels, grid, legend text) should update immediately
3. Click "7天" / "30天" / "90天" buttons — chart should transition smoothly (not flash blank)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/admin/copilot/AccountsDailyChart.vue
git commit -m "Refactor: AccountsDailyChart 迁移至手动 Chart.js 实例，支持深色模式"
```

---

## Task 5: Rewrite QuotaTrendChart.vue — Manual Chart.js with Dark Mode

**Files:**
- Modify: `frontend/src/components/admin/copilot/QuotaTrendChart.vue`

The current component uses vue-chartjs `<Line>` with hardcoded light-mode colors (`rgb(209, 213, 219)` for the entitlement dashed line). This migration follows the exact same pattern as Task 4.

Key behaviors to preserve:
- `props.accountId` and `props.days` props
- `getCopilotAccountQuotaTrend(accountId, { days })` API call
- Two datasets: "已用配额" (blue fill area) + "配额上限" (dashed line, no fill)
- Reload when either `accountId` or `days` changes

New behaviors:
- `h-[180px]` container height (spec requirement; current is `max-h-48` on the component element)
- Dark-mode entitlement line: `#4b5563` (dark) / `#d1d5db` (light)
- MutationObserver for `isDark` reactive dark mode

- [ ] **Step 1: Replace the entire QuotaTrendChart.vue content**

Replace the full file with:

```vue
<template>
  <div>
    <div v-if="loading" class="flex h-[180px] items-center justify-center text-xs text-gray-400 dark:text-gray-500">
      加载中…
    </div>
    <div v-else-if="loadError" class="flex h-[180px] items-center justify-center text-xs text-red-500">
      加载失败：{{ loadError }}
    </div>
    <div v-else-if="!data || data.trend.length === 0"
         class="flex h-[180px] items-center justify-center text-xs text-gray-400 dark:text-gray-500">
      暂无趋势数据
    </div>
    <div v-else class="h-[180px]">
      <canvas ref="chartRef" class="!h-full !w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { getCopilotAccountQuotaTrend } from '@/api/admin/copilotAnalytics'
import type { CopilotAccountQuotaTrendResult } from '@/api/admin/copilotAnalytics'
import { extractErrorMessage } from '@/api/client'

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler)

const props = defineProps<{ accountId: number; days?: number }>()

const chartRef = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

const loading = ref(false)
const loadError = ref<string | null>(null)
const data = ref<CopilotAccountQuotaTrendResult | null>(null)

// Dark mode reactive state
const isDark = ref(document.documentElement.classList.contains('dark'))
let darkObserver: MutationObserver | null = null

function buildChartData() {
  const trend = data.value?.trend ?? []
  const dark = isDark.value
  const limitColor = dark ? '#4b5563' : '#d1d5db'

  return {
    labels: trend.map(t => t.snapshot_date),
    datasets: [
      {
        label: '已用配额',
        data: trend.map(t => t.premium_used),
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        fill: true,
        tension: 0.3,
        pointRadius: 3,
        pointHoverRadius: 5,
      },
      {
        label: '配额上限',
        data: trend.map(t => t.premium_entitlement),
        borderColor: limitColor,
        borderDash: [4, 4],
        fill: false,
        tension: 0.3,
        pointRadius: 0,
        pointHoverRadius: 0,
      },
    ],
  }
}

function buildChartOptions() {
  const dark = isDark.value
  const tickColor  = dark ? '#9ca3af' : '#6b7280'
  const gridColor  = dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)'
  const legendColor = dark ? '#d1d5db' : '#374151'

  return {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: true,
        position: 'bottom' as const,
        labels: { boxWidth: 12, font: { size: 11 }, color: legendColor },
      },
      tooltip: { mode: 'index' as const, intersect: false },
    },
    scales: {
      x: {
        grid: { display: false },
        ticks: { font: { size: 10 }, maxTicksLimit: 10, color: tickColor },
      },
      y: {
        beginAtZero: true,
        grid: { color: gridColor },
        ticks: { font: { size: 10 }, color: tickColor },
      },
    },
  }
}

async function renderChart() {
  if (!data.value || data.value.trend.length === 0) return

  loading.value = false
  await nextTick()

  if (!chartRef.value) return

  if (chart) {
    chart.data = buildChartData()
    chart.options = buildChartOptions() as typeof chart.options
    chart.update('active')
  } else {
    chart = new Chart(chartRef.value, {
      type: 'line',
      data: buildChartData(),
      options: buildChartOptions() as any, // eslint-disable-line @typescript-eslint/no-explicit-any
    })
  }
}

async function load() {
  loading.value = true
  loadError.value = null
  chart?.destroy()
  chart = null
  try {
    data.value = await getCopilotAccountQuotaTrend(props.accountId, { days: props.days ?? 30 })
    if (!data.value || data.value.trend.length === 0) {
      loading.value = false
      return
    }
    await renderChart()
  } catch (e: unknown) {
    loadError.value = extractErrorMessage(e)
    loading.value = false
  }
}

onMounted(() => {
  darkObserver = new MutationObserver(() => {
    const newDark = document.documentElement.classList.contains('dark')
    if (newDark !== isDark.value) {
      isDark.value = newDark
      chart?.destroy()
      chart = null
      renderChart()
    }
  })
  darkObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  load()
})

watch(() => [props.accountId, props.days], load)

onBeforeUnmount(() => {
  chart?.destroy()
  chart = null
  darkObserver?.disconnect()
  darkObserver = null
})
</script>
```

- [ ] **Step 2: TypeScript check**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep -i "QuotaTrendChart"
```
Expected: no errors.

- [ ] **Step 3: Verify in browser**

1. Expand an account row in the accounts table
2. The quota trend chart (near the right) should render inside the `h-[180px]` container
3. Toggle dark mode — the dashed entitlement line should change from `#d1d5db` (light gray) to `#4b5563` (dark gray), and axes should update

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/admin/copilot/QuotaTrendChart.vue
git commit -m "Refactor: QuotaTrendChart 迁移至手动 Chart.js 实例，支持深色模式"
```

---

## Task 6: Final Type Check and E2E Smoke Test

**Files:**
- Read-only verification across all modified files

This task confirms the full build passes and the key user flows work end-to-end.

- [ ] **Step 1: Full TypeScript check**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1
```
Expected: zero errors. If there are errors, fix them before proceeding.

- [ ] **Step 2: Lint check**

```bash
cd frontend && npx eslint src/views/admin/copilot/CopilotAccountsView.vue \
  src/components/admin/copilot/AccountsDailyChart.vue \
  src/components/admin/copilot/QuotaTrendChart.vue \
  src/api/admin/copilotAnalytics.ts 2>&1
```
Expected: no errors (warnings about `// eslint-disable-line` are acceptable since they're already in the codebase).

- [ ] **Step 3: Smoke test checklist (manual)**

Open the accounts page (e.g. `http://localhost:3000/admin/copilot/accounts`):

1. **Bar chart gone** — no bar chart between KPI cards and the daily trend line chart
2. **Sparkbar visible** — in the monthly cost column, each row has a small green bar proportional to cost; the highest-cost account has the widest bar
3. **Daily chart dark mode** — toggle dark mode; chart axes, grid lines, and legend text update to light-on-dark colors
4. **Quota refresh (single row)** — click "刷新配额" on any row; the quota bar updates without the full table disappearing; success toast appears
5. **Quota trend chart** — expand a row; the trend chart renders at 180px height; toggle dark mode; the dashed line changes color
6. **Refresh button** — in dark mode, hovering the top-right "刷新" button shows a slightly lighter border

- [ ] **Step 4: Final commit (if any cleanup needed)**

If there were any minor fixes during smoke test:
```bash
git add -p  # stage only the actual changes
git commit -m "Fix: 账户视图重设计收尾修复"
```
