# Copilot Accounts View Redesign

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the large top bar chart, replace it with a cost sparkbar inside the accounts table, add full dark-mode support to all charts, and improve per-row refresh UX.

**Architecture:** Pure frontend changes across 1 view + 2 chart components. No backend changes required. All chart instances migrate from vue-chartjs wrappers to hand-managed Chart.js instances for precise dark-mode control and smooth data updates.

**Tech Stack:** Vue 3 Composition API, TypeScript, Chart.js 4, Tailwind CSS

---

## Files Affected

| File | Change |
|------|--------|
| `frontend/src/views/admin/copilot/CopilotAccountsView.vue` | Remove Bar chart + costChartData/Options; add sparkbar in cost column; fix per-row refresh; dark-mode button fixes |
| `frontend/src/components/admin/copilot/AccountsDailyChart.vue` | Migrate from `<Line>` (vue-chartjs) to manual Chart.js instance; dark-mode computed options; smooth data update on days change |
| `frontend/src/components/admin/copilot/QuotaTrendChart.vue` | Migrate from `<Line>` (vue-chartjs) to manual Chart.js instance; dark-mode computed options; fix height |

---

## Section 1: Remove Top Bar Chart

**Scope:** `CopilotAccountsView.vue` only.

Remove:
- `import { Bar } from 'vue-chartjs'` and related ChartJS registrations (`BarElement`)
- The entire `<!-- Cost Comparison Chart -->` block in the template
- `costChartData` computed
- `costChartOptions` constant
- `CHART_COLORS` constant (only used by removed chart)
- `ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend)` — keep only what's still needed (nothing after this removal since charts moved to components)

Result: The view no longer imports vue-chartjs at all. The `Bar` section is replaced by nothing — the page goes straight from KPI cards to the daily trend chart.

---

## Section 2: Cost Sparkbar in Table

**Scope:** `CopilotAccountsView.vue` — the monthly cost `<td>` cell.

**New computed: `maxMonthlyCost`**
```ts
const maxMonthlyCost = computed(() =>
  Math.max(1, ...(overview.value?.accounts ?? []).map(a => a.monthly_cost))
)
```

**Updated cost cell template** (replaces the existing `<td>` for monthly cost):
```html
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

The sparkbar is right-aligned (consistent with the column), 96px wide, 4px tall. Normalization uses `maxMonthlyCost` so the account with the highest cost fills 100% and others are proportional.

---

## Section 3: AccountsDailyChart.vue — Full Rewrite

**Goal:** Replace vue-chartjs `<Line>` with a manual `<canvas>` + Chart.js instance. This enables:
1. Smooth `chart.update('active')` transitions when `days` prop changes (no component re-mount)
2. Precise dark-mode color control via `isDark` MutationObserver

**Template:**
```html
<template>
  <div>
    <div v-if="loading" class="flex h-[256px] items-center justify-center text-xs text-gray-400 dark:text-gray-500">
      <svg class="mr-2 h-4 w-4 animate-spin" ...>...</svg>
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
```

**Script structure:**
- `isDark` ref + MutationObserver (same pattern as UserHeatmap.vue)
- `chartRef` ref, `chart` Chart.js instance (not reactive)
- `load()`: fetches data, sets `loading`, calls `renderChart()`
- `renderChart()`:
  - If chart exists AND data changed: `chart.data = buildChartData(); chart.options = buildChartOptions(); chart.update('active')`
  - If chart doesn't exist: destroy any old instance, `loading.value = false; await nextTick(); chart = new Chart(chartRef.value, {...})`
- `buildChartData()`: builds labels + datasets from `data.value` (same logic as current `chartData` computed)
- `buildChartOptions()`: returns options with colors from `isDark.value`
- `watch(() => props.days, load)` — reload data when days changes
- `onMounted(load)`, `onBeforeUnmount(() => chart?.destroy())`

**Dark mode colors in `buildChartOptions()`:**
```ts
const dark = isDark.value
const tickColor  = dark ? '#9ca3af' : '#6b7280'
const gridColor  = dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)'
const legendColor = dark ? '#d1d5db' : '#374151'
```

---

## Section 4: QuotaTrendChart.vue — Full Rewrite

**Goal:** Same migration as AccountsDailyChart. Replace `<Line>` with `<canvas>`.

**Template:** Same pattern — `h-[180px]` container with `v-if/v-else` states.

**Script:**
- `isDark` + MutationObserver
- `chartRef`, `chart`
- `load()` → `renderChart()`
- `renderChart()`: create chart after `loading.value = false; await nextTick()`
- Two datasets: "已用配额" (blue fill) + "配额上限" (dashed line, light gray / dark gray)

**Dark mode colors:**
```ts
const dark = isDark.value
const limitColor = dark ? '#4b5563' : '#d1d5db'  // dashed entitlement line
const tickColor  = dark ? '#9ca3af' : '#6b7280'
const gridColor  = dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)'
```

---

## Section 5: Per-Row Quota Refresh UX

**Current:** `doRefresh()` calls `await loadOverview()` which re-fetches all data — full table re-render.

**New:** After `refreshCopilotAccountQuota(id)` returns the updated quota snapshot, patch only that row in `overview.value.accounts`:

```ts
async function doRefresh(account: CopilotAccountOverviewEntry) {
  refreshing.value.add(account.account_id)
  try {
    const updated = await refreshCopilotAccountQuota(account.account_id)
    // Patch single row — no full reload
    if (overview.value) {
      const idx = overview.value.accounts.findIndex(a => a.account_id === account.account_id)
      if (idx !== -1) {
        overview.value.accounts[idx] = {
          ...overview.value.accounts[idx],
          quota_snapshot: updated,
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

**Prerequisite:** Check what `refreshCopilotAccountQuota` returns in `@/api/admin/copilotAnalytics` — if it returns a full `CopilotAccountOverviewEntry`, patch the full row; if it only returns a quota snapshot, patch `quota_snapshot` field only. Adjust accordingly.

---

## Section 6: Minor Dark-Mode Fixes in CopilotAccountsView

- Refresh button: add `dark:hover:border-gray-500` to the top-right page refresh button
- Row hover: `dark:hover:bg-gray-700/30` already exists ✅
- Today badge: `dark:bg-blue-900/30 dark:text-blue-400` already exists ✅

---

## Constraints

- Do NOT change any backend files
- Do NOT add new dependencies (no new npm packages)
- Do NOT change the `BudgetAlertDialog` component
- Keep `AccountCostCard.vue` as-is (it's not used in the main view, only potentially elsewhere)
- All chart instances must be properly destroyed in `onBeforeUnmount`
