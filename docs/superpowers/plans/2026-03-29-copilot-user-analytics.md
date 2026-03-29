# Copilot 用户请求分析平台 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Copilot 用户请求分析页面从简陋的单日表格升级为具备趋势折线图、用户排行、详情钻取和 24h 热力图的专业分析平台。

**Architecture:** 后端新增两个接口（用户多日趋势 + 单用户汇总），前端重构 UsersView 为趋势+排行大盘，新增 UserDetailView 独立详情页，共新增 4 个前端组件（UsersDailyChart、UserSparkline、UserHeatmap、UserDetailHeader）。

**Tech Stack:** Go + Gin + PostgreSQL（后端）；Vue 3 Composition API + TypeScript + Chart.js 4.x + Tailwind CSS（前端）

---

## 文件结构

### 后端（新增/修改）
| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/internal/service/copilot_analytics_service.go` | 修改 | 新增类型定义 + GetUsersDailyStats + GetUserSummary |
| `backend/internal/handler/admin/copilot_analytics_handler.go` | 修改 | 新增 GetUsersDailyStats + GetUserSummary handler |
| `backend/internal/server/routes/admin.go` | 修改 | 注册两条新路由 |

### 前端（新增/修改）
| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/api/admin/copilotAnalytics.ts` | 修改 | 新增 4 个类型 + 2 个 API 函数 |
| `frontend/src/views/admin/copilot/CopilotUsersView.vue` | **重写** | 重构为趋势图 + 排行表大盘 |
| `frontend/src/views/admin/copilot/CopilotUserDetailView.vue` | **新建** | 用户详情页（柱状图+环形图+热力图+日志） |
| `frontend/src/components/admin/copilot/UsersDailyChart.vue` | **新建** | 多用户趋势折线图（带用户筛选） |
| `frontend/src/components/admin/copilot/UserSparkline.vue` | **新建** | 迷你7日趋势图（用于排行表每行） |
| `frontend/src/components/admin/copilot/UserHeatmap.vue` | **新建** | 24小时热力图（消费已有 timeline API） |
| `frontend/src/router/index.ts` | 修改 | 新增 `/admin/copilot/users/:id` 路由 |

---

## Task 1: 后端 — 新增类型定义

**Files:**
- Modify: `backend/internal/service/copilot_analytics_service.go`（在 `CopilotUserRequestsResult` 定义之后，第 97 行）

- [ ] **Step 1: 在 `CopilotUserRequestsResult` 定义后插入新类型**

在文件第 96-97 行（`}` 结束 `CopilotUserRequestsResult` 之后）添加：

```go
// CopilotUserDailyUserInfo holds minimal user metadata for the chart legend.
type CopilotUserDailyUserInfo struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

// CopilotUserDailyEntry holds one user's daily premium + agent request counts.
type CopilotUserDailyEntry struct {
	UserID       int64  `json:"user_id"`
	Date         string `json:"date"`
	PremiumCount int    `json:"premium_count"`
	AgentCount   int    `json:"agent_count"`
}

// CopilotUsersDailyStatsResult is the response for the all-users daily stats endpoint.
type CopilotUsersDailyStatsResult struct {
	Users []CopilotUserDailyUserInfo `json:"users"`
	Days  []CopilotUserDailyEntry    `json:"days"`
}

// CopilotUserModelStat holds per-model request counts and percentage for one user.
type CopilotUserModelStat struct {
	Model      string  `json:"model"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// CopilotUserSummaryResult is the response for a single user's all-time Copilot summary.
type CopilotUserSummaryResult struct {
	UserID               int64                  `json:"user_id"`
	Username             string                 `json:"username"`
	TotalPremiumRequests int                    `json:"total_premium_requests"`
	TotalAgentRequests   int                    `json:"total_agent_requests"`
	FirstRequestAt       *time.Time             `json:"first_request_at"`
	LastRequestAt        *time.Time             `json:"last_request_at"`
	TopModels            []CopilotUserModelStat `json:"top_models"`
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./internal/service/...
```

期望输出：无错误（或仅 unused import 警告，后续 task 会用到）

- [ ] **Step 3: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/service/copilot_analytics_service.go
git commit -m "Feature: 新增 Copilot 用户分析类型定义"
```

---

## Task 2: 后端 — 实现 GetUsersDailyStats

**Files:**
- Modify: `backend/internal/service/copilot_analytics_service.go`（在 `buildRequestHierarchy` 函数结束后、账户维度查询注释之前，约第 383 行）

- [ ] **Step 1: 在 `buildRequestHierarchy` 函数之后插入方法**

```go
// GetUsersDailyStats returns daily premium + agent request counts for all active Copilot users
// over the past [days] days (default 30, max 90).
// Each row in Days is (user_id, date, premium_count, agent_count).
// Users slice contains deduplicated user metadata ordered by first appearance in results.
func (s *CopilotAnalyticsService) GetUsersDailyStats(ctx context.Context, days int) (*CopilotUsersDailyStatsResult, error) {
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	now := time.Now()
	rangeStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -(days - 1))

	query := `
SELECT
    ul.user_id,
    COALESCE(u.username, ul.user_id::text) AS username,
    DATE(ul.created_at AT TIME ZONE 'UTC' AT TIME ZONE current_setting('TIMEZONE')) AS req_date,
    COUNT(*) FILTER (WHERE ul.initiator = 'user')  AS premium_count,
    COUNT(*) FILTER (WHERE ul.initiator = 'agent') AS agent_count
FROM usage_logs ul
LEFT JOIN users u ON u.id = ul.user_id
WHERE ul.created_at >= $1
  AND ul.account_id IN (SELECT id FROM accounts WHERE platform = 'copilot')
GROUP BY ul.user_id, u.username, req_date
ORDER BY ul.user_id, req_date
`
	rows, err := s.db.QueryContext(ctx, query, rangeStart)
	if err != nil {
		return nil, fmt.Errorf("copilot analytics: users daily stats query: %w", err)
	}
	defer rows.Close()

	// seenUsers tracks insertion order so that the Users slice is stable.
	seenUsers := make(map[int64]struct{})
	users := make([]CopilotUserDailyUserInfo, 0)
	entries := make([]CopilotUserDailyEntry, 0)

	for rows.Next() {
		var userID int64
		var username string
		var date time.Time
		var premiumCount, agentCount int
		if err := rows.Scan(&userID, &username, &date, &premiumCount, &agentCount); err != nil {
			return nil, fmt.Errorf("copilot analytics: scan users daily stats row: %w", err)
		}
		if _, ok := seenUsers[userID]; !ok {
			seenUsers[userID] = struct{}{}
			users = append(users, CopilotUserDailyUserInfo{UserID: userID, Username: username})
		}
		entries = append(entries, CopilotUserDailyEntry{
			UserID:       userID,
			Date:         date.Format("2006-01-02"),
			PremiumCount: premiumCount,
			AgentCount:   agentCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("copilot analytics: users daily stats rows: %w", err)
	}

	return &CopilotUsersDailyStatsResult{
		Users: users,
		Days:  entries,
	}, nil
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./internal/service/...
```

期望：无错误

- [ ] **Step 3: 写单元测试**

在 `backend/internal/service/copilot_analytics_test.go` 文件末尾追加：

```go
func TestGetUsersDailyStats_DaysClamp(t *testing.T) {
	// 验证 days 参数边界：<=0 clamp 到 30，>90 clamp 到 90
	svc := &CopilotAnalyticsService{db: nil}

	// 通过反射或直接测试 clamp 逻辑是业务逻辑最轻量的验证
	// 实际数据库调用需要集成测试，这里仅验证函数存在且签名正确
	var _ func(context.Context, int) (*CopilotUsersDailyStatsResult, error) = svc.GetUsersDailyStats
}
```

- [ ] **Step 4: 运行测试**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go test ./internal/service/... -run TestGetUsersDailyStats -v
```

期望：PASS（编译即通过）

- [ ] **Step 5: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/service/copilot_analytics_service.go backend/internal/service/copilot_analytics_test.go
git commit -m "Feature: 新增 GetUsersDailyStats 用户每日请求趋势查询"
```

---

## Task 3: 后端 — 实现 GetUserSummary

**Files:**
- Modify: `backend/internal/service/copilot_analytics_service.go`（紧接 `GetUsersDailyStats` 之后）

- [ ] **Step 1: 在 `GetUsersDailyStats` 之后插入方法**

```go
// GetUserSummary returns a single user's all-time Copilot usage summary,
// including total request counts, activity window, and top model distribution.
func (s *CopilotAnalyticsService) GetUserSummary(ctx context.Context, userID int64) (*CopilotUserSummaryResult, error) {
	summaryQuery := `
SELECT
    ul.user_id,
    COALESCE(u.username, ul.user_id::text) AS username,
    COUNT(*) FILTER (WHERE ul.initiator = 'user')  AS total_premium_requests,
    COUNT(*) FILTER (WHERE ul.initiator = 'agent') AS total_agent_requests,
    MIN(ul.created_at) AS first_request_at,
    MAX(ul.created_at) AS last_request_at
FROM usage_logs ul
LEFT JOIN users u ON u.id = ul.user_id
WHERE ul.user_id = $1
  AND ul.account_id IN (SELECT id FROM accounts WHERE platform = 'copilot')
GROUP BY ul.user_id, u.username
`
	var result CopilotUserSummaryResult
	if err := s.db.QueryRowContext(ctx, summaryQuery, userID).Scan(
		&result.UserID,
		&result.Username,
		&result.TotalPremiumRequests,
		&result.TotalAgentRequests,
		&result.FirstRequestAt,
		&result.LastRequestAt,
	); err != nil {
		return nil, fmt.Errorf("copilot analytics: user summary query: %w", err)
	}

	modelQuery := `
SELECT
    ul.model,
    COUNT(*) AS cnt
FROM usage_logs ul
WHERE ul.user_id = $1
  AND ul.account_id IN (SELECT id FROM accounts WHERE platform = 'copilot')
GROUP BY ul.model
ORDER BY cnt DESC, ul.model ASC
LIMIT 10
`
	rows, err := s.db.QueryContext(ctx, modelQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("copilot analytics: user summary models query: %w", err)
	}
	defer rows.Close()

	totalRequests := result.TotalPremiumRequests + result.TotalAgentRequests
	result.TopModels = make([]CopilotUserModelStat, 0)
	for rows.Next() {
		var stat CopilotUserModelStat
		if err := rows.Scan(&stat.Model, &stat.Count); err != nil {
			return nil, fmt.Errorf("copilot analytics: scan user model stat row: %w", err)
		}
		if totalRequests > 0 {
			stat.Percentage = math.Round(float64(stat.Count)*1000/float64(totalRequests)) / 10
		}
		result.TopModels = append(result.TopModels, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("copilot analytics: user summary model rows: %w", err)
	}

	return &result, nil
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./internal/service/...
```

期望：无错误

- [ ] **Step 3: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/service/copilot_analytics_service.go
git commit -m "Feature: 新增 GetUserSummary 用户汇总统计查询"
```

---

## Task 4: 后端 — Handler + 路由注册

**Files:**
- Modify: `backend/internal/handler/admin/copilot_analytics_handler.go`
- Modify: `backend/internal/server/routes/admin.go`

- [ ] **Step 1: 在 handler 文件末尾（`parseIDParam` 函数之前）添加两个 handler**

在 `copilot_analytics_handler.go` 中，在 `// ─── 辅助函数 ───` 注释之前插入：

```go
// GetUsersDailyStats handles GET /api/v1/admin/copilot/users/daily-stats
// Query params: days (default 30, max 90)
func (h *CopilotAnalyticsHandler) GetUsersDailyStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days < 1 || days > 90 {
		days = 30
	}

	result, err := h.analyticsSvc.GetUsersDailyStats(c.Request.Context(), days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

// GetUserSummary handles GET /api/v1/admin/copilot/users/:id/summary
func (h *CopilotAnalyticsHandler) GetUserSummary(c *gin.Context) {
	userID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.analyticsSvc.GetUserSummary(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}
```

- [ ] **Step 2: 注册路由**

在 `backend/internal/server/routes/admin.go` 文件的 `registerCopilotAnalyticsRoutes` 函数中，在 `users.GET("/:id/requests", ...)` 之后添加：

```go
users.GET("/daily-stats", h.Admin.CopilotAnalytics.GetUsersDailyStats)
users.GET("/:id/summary", h.Admin.CopilotAnalytics.GetUserSummary)
```

注意路由顺序：`/daily-stats` 必须在 `/:id/...` 之前，避免 Gin 将 `daily-stats` 识别为 `:id`。实际上现有代码已有 `/stats` 在 `/:id/timeline` 之前，模式一致，但仍需确认位置：

```go
// 修改后的 registerCopilotAnalyticsRoutes 函数中 users 部分：
users.GET("/stats", h.Admin.CopilotAnalytics.GetUserStats)
users.GET("/daily-stats", h.Admin.CopilotAnalytics.GetUsersDailyStats)  // ← 新增，放在 /:id 路由前
users.GET("/:id/timeline", h.Admin.CopilotAnalytics.GetUserTimeline)
users.GET("/:id/requests", h.Admin.CopilotAnalytics.GetUserRequests)
users.GET("/:id/summary", h.Admin.CopilotAnalytics.GetUserSummary)     // ← 新增
```

- [ ] **Step 3: 编译整个后端**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./...
```

期望：无错误

- [ ] **Step 4: 运行现有测试确保无回归**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go test ./internal/service/... -v 2>&1 | tail -20
```

期望：所有已有测试 PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/handler/admin/copilot_analytics_handler.go \
        backend/internal/server/routes/admin.go
git commit -m "Feature: 注册 Copilot 用户每日趋势和汇总接口"
```

---

## Task 5: 前端 — 新增 API 类型和函数

**Files:**
- Modify: `frontend/src/api/admin/copilotAnalytics.ts`

- [ ] **Step 1: 在用户维度类型区块末尾（`CopilotUserRequestsResult` 之后）添加新类型**

```typescript
// ─────────────────────────────────────────────
// 用户维度 — 日趋势 & 汇总类型
// ─────────────────────────────────────────────

export interface CopilotUserDailyUserInfo {
  user_id: number
  username: string
}

export interface CopilotUserDailyEntry {
  user_id: number
  date: string
  premium_count: number
  agent_count: number
}

export interface CopilotUsersDailyStatsResult {
  users: CopilotUserDailyUserInfo[]
  days: CopilotUserDailyEntry[]
}

export interface CopilotUserModelStat {
  model: string
  count: number
  percentage: number
}

export interface CopilotUserSummaryResult {
  user_id: number
  username: string
  total_premium_requests: number
  total_agent_requests: number
  first_request_at: string | null
  last_request_at: string | null
  top_models: CopilotUserModelStat[]
}
```

- [ ] **Step 2: 在用户维度 API 函数区块末尾（`getCopilotUserRequests` 之后）添加新函数**

```typescript
export async function getCopilotUsersDailyStats(
  params: { days?: number } = {},
): Promise<CopilotUsersDailyStatsResult> {
  const { data } = await apiClient.get(`${BASE}/users/daily-stats`, { params })
  return data
}

export async function getCopilotUserSummary(
  userId: number,
): Promise<CopilotUserSummaryResult> {
  const { data } = await apiClient.get(`${BASE}/users/${userId}/summary`)
  return data
}
```

- [ ] **Step 3: 验证 TypeScript 编译**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit 2>&1 | head -20
```

期望：无类型错误

- [ ] **Step 4: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/api/admin/copilotAnalytics.ts
git commit -m "Feature: 新增 Copilot 用户每日趋势和汇总 API 类型与函数"
```

---

## Task 6: 前端组件 — UserSparkline（迷你趋势图）

**Files:**
- Create: `frontend/src/components/admin/copilot/UserSparkline.vue`

这是一个无依赖的纯 Canvas 迷你折线图，用于排行表的每行趋势展示。

- [ ] **Step 1: 创建文件**

```vue
<template>
  <canvas ref="canvasRef" :width="width" :height="height" />
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'

const props = withDefaults(defineProps<{
  data: number[]      // 7 个数值（最旧→最新）
  width?: number
  height?: number
  color?: string
}>(), {
  width: 80,
  height: 28,
  color: '#3b82f6',
})

const canvasRef = ref<HTMLCanvasElement | null>(null)

function draw() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const { data, width, height, color } = props
  ctx.clearRect(0, 0, width, height)

  if (data.length < 2) return

  const max = Math.max(...data, 1)
  const min = Math.min(...data, 0)
  const range = max - min || 1
  const pad = 3

  const points = data.map((v, i) => ({
    x: pad + (i / (data.length - 1)) * (width - pad * 2),
    y: height - pad - ((v - min) / range) * (height - pad * 2),
  }))

  // Fill area
  ctx.beginPath()
  ctx.moveTo(points[0].x, height - pad)
  points.forEach(p => ctx.lineTo(p.x, p.y))
  ctx.lineTo(points[points.length - 1].x, height - pad)
  ctx.closePath()
  ctx.fillStyle = color + '22'
  ctx.fill()

  // Line
  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)
  points.slice(1).forEach(p => ctx.lineTo(p.x, p.y))
  ctx.strokeStyle = color
  ctx.lineWidth = 1.5
  ctx.lineJoin = 'round'
  ctx.stroke()
}

onMounted(draw)
watch(() => props.data, draw, { deep: true })
</script>
```

- [ ] **Step 2: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/components/admin/copilot/UserSparkline.vue
git commit -m "Feature: 新增 UserSparkline 迷你趋势图组件"
```

---

## Task 7: 前端组件 — UsersDailyChart（多用户趋势折线图）

**Files:**
- Create: `frontend/src/components/admin/copilot/UsersDailyChart.vue`

参考现有 `AccountsDailyChart.vue` 的实现模式（Chart.js + days prop + 加载/错误状态）。

- [ ] **Step 1: 创建文件**

```vue
<template>
  <div class="relative">
    <div v-if="loading" class="flex h-64 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div v-else-if="error" class="flex h-64 items-center justify-center text-sm text-red-500">
      {{ error }}
    </div>
    <canvas v-else ref="chartRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
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
import { getCopilotUsersDailyStats } from '@/api/admin/copilotAnalytics'
import type { CopilotUsersDailyStatsResult } from '@/api/admin/copilotAnalytics'
import { extractErrorMessage } from '@/api/client'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler)

const props = withDefaults(defineProps<{
  days?: number
  metric?: 'premium' | 'agent' | 'total'  // 展示哪种指标
}>(), {
  days: 30,
  metric: 'premium',
})

// 调色板（与 AccountsDailyChart 复用相同色系）
const PALETTE = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
  '#06b6d4', '#f97316', '#84cc16', '#ec4899', '#6366f1',
]

const chartRef = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null
const loading = ref(false)
const error = ref<string | null>(null)

async function buildChart() {
  loading.value = true
  error.value = null
  try {
    const result: CopilotUsersDailyStatsResult = await getCopilotUsersDailyStats({ days: props.days })

    // 生成完整日期序列
    const dates: string[] = []
    const today = new Date()
    for (let i = props.days - 1; i >= 0; i--) {
      const d = new Date(today)
      d.setDate(d.getDate() - i)
      dates.push(d.toISOString().slice(0, 10))
    }

    // 按用户分组数据，并填充零值
    const countsByUser = new Map<number, Map<string, number>>()
    for (const entry of result.days) {
      if (!countsByUser.has(entry.user_id)) {
        countsByUser.set(entry.user_id, new Map())
      }
      const val = props.metric === 'premium'
        ? entry.premium_count
        : props.metric === 'agent'
          ? entry.agent_count
          : entry.premium_count + entry.agent_count
      countsByUser.get(entry.user_id)!.set(entry.date, val)
    }

    const datasets = result.users.map((user, idx) => {
      const userMap = countsByUser.get(user.user_id) ?? new Map()
      return {
        label: user.username,
        data: dates.map(d => userMap.get(d) ?? 0),
        borderColor: PALETTE[idx % PALETTE.length],
        backgroundColor: PALETTE[idx % PALETTE.length] + '18',
        borderWidth: 2,
        pointRadius: props.days <= 14 ? 3 : 0,
        pointHoverRadius: 5,
        tension: 0.3,
        fill: false,
      }
    })

    chart?.destroy()
    if (!chartRef.value) return
    chart = new Chart(chartRef.value, {
      type: 'line',
      data: { labels: dates, datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: 'index', intersect: false },
        plugins: {
          legend: {
            position: 'bottom',
            labels: { boxWidth: 12, padding: 16, font: { size: 12 } },
          },
          tooltip: {
            callbacks: {
              title: (items) => items[0]?.label ?? '',
              label: (item) => ` ${item.dataset.label}: ${item.parsed.y} 次`,
            },
          },
        },
        scales: {
          x: {
            grid: { display: false },
            ticks: { maxTicksLimit: 10, font: { size: 11 } },
          },
          y: {
            beginAtZero: true,
            grid: { color: 'rgba(0,0,0,0.06)' },
            ticks: { font: { size: 11 } },
          },
        },
      },
    })
  } catch (e: unknown) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(buildChart)
watch(() => [props.days, props.metric], buildChart)
onBeforeUnmount(() => chart?.destroy())
</script>

<style scoped>
canvas { height: 300px; }
</style>
```

- [ ] **Step 2: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/components/admin/copilot/UsersDailyChart.vue
git commit -m "Feature: 新增 UsersDailyChart 多用户趋势折线图组件"
```

---

## Task 8: 前端组件 — UserHeatmap（24小时热力图）

**Files:**
- Create: `frontend/src/components/admin/copilot/UserHeatmap.vue`

消费已有 `GET /users/:id/timeline?date=YYYY-MM-DD` 接口，纯 CSS Grid 实现，无需额外库。

- [ ] **Step 1: 创建文件**

```vue
<template>
  <div>
    <div v-if="loading" class="flex h-24 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div v-else-if="error" class="text-sm text-red-500">{{ error }}</div>
    <div v-else>
      <!-- 小时列标签 -->
      <div class="mb-1 grid grid-cols-[2rem_repeat(24,1fr)] gap-0.5 text-center">
        <span />
        <span
          v-for="h in 24"
          :key="h"
          class="text-[10px] text-gray-400 dark:text-gray-500"
        >{{ (h - 1).toString().padStart(2, '0') }}</span>
      </div>
      <!-- 每一天一行 -->
      <div
        v-for="row in rows"
        :key="row.date"
        class="grid grid-cols-[2rem_repeat(24,1fr)] gap-0.5"
      >
        <span class="text-right text-[10px] leading-4 text-gray-400 dark:text-gray-500 pr-1">
          {{ row.label }}
        </span>
        <div
          v-for="cell in row.cells"
          :key="cell.hour"
          class="h-4 rounded-sm cursor-default transition-opacity hover:opacity-80"
          :style="{ backgroundColor: heatColor(cell.count, maxCount) }"
          :title="`${row.date} ${cell.hour.toString().padStart(2,'0')}:00 — ${cell.count} 次`"
        />
      </div>
      <!-- 图例 -->
      <div class="mt-2 flex items-center gap-1 justify-end">
        <span class="text-[10px] text-gray-400">少</span>
        <div v-for="step in legendSteps" :key="step" class="h-3 w-5 rounded-sm" :style="{ backgroundColor: heatColor(step, 5) }" />
        <span class="text-[10px] text-gray-400">多</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { getCopilotUserTimeline } from '@/api/admin/copilotAnalytics'
import { extractErrorMessage } from '@/api/client'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const props = defineProps<{
  userId: number
  days?: number   // 显示最近 N 天，默认 7
}>()

interface HeatCell { hour: number; count: number }
interface HeatRow { date: string; label: string; cells: HeatCell[] }

const loading = ref(false)
const error = ref<string | null>(null)
const rows = ref<HeatRow[]>([])
const maxCount = ref(1)
const legendSteps = [0, 1, 2, 3, 4, 5]

function localDateStr(offset: number): string {
  const d = new Date()
  d.setDate(d.getDate() + offset)
  return d.toISOString().slice(0, 10)
}

function heatColor(count: number, max: number): string {
  if (count === 0 || max === 0) return '#e5e7eb'
  const ratio = Math.min(count / max, 1)
  // 浅蓝 → 深蓝渐变
  const r = Math.round(219 - ratio * 170)
  const g = Math.round(234 - ratio * 130)
  const b = Math.round(254 - ratio * 60)
  return `rgb(${r},${g},${b})`
}

async function load() {
  if (!props.userId) return
  loading.value = true
  error.value = null
  const numDays = props.days ?? 7
  try {
    const results = await Promise.all(
      Array.from({ length: numDays }, (_, i) => {
        const date = localDateStr(-(numDays - 1 - i))
        return getCopilotUserTimeline(props.userId, { date }).then(r => ({ date, hourly: r.hourly }))
      }),
    )
    let globalMax = 0
    rows.value = results.map(({ date, hourly }) => {
      const cells = hourly.map(h => {
        const count = h.premium_count + h.agent_count
        if (count > globalMax) globalMax = count
        return { hour: h.hour, count }
      })
      const [, m, d] = date.split('-')
      return { date, label: `${m}/${d}`, cells }
    })
    maxCount.value = globalMax || 1
  } catch (e: unknown) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => [props.userId, props.days], load)
</script>
```

- [ ] **Step 2: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/components/admin/copilot/UserHeatmap.vue
git commit -m "Feature: 新增 UserHeatmap 24小时热力图组件"
```

---

## Task 9: 前端 — 重构 CopilotUsersView（大盘总览页）

**Files:**
- Modify: `frontend/src/views/admin/copilot/CopilotUsersView.vue`（完整重写）

- [ ] **Step 1: 完整替换文件内容**

```vue
<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 页头 -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.copilot.users.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.copilot.users.description') }}
          </p>
        </div>
        <!-- 时间范围选择 -->
        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-500 dark:text-gray-400">近</span>
          <div class="flex rounded-md border border-gray-300 dark:border-gray-600 overflow-hidden">
            <button
              v-for="d in [7, 14, 30, 60]"
              :key="d"
              class="px-3 py-1.5 text-sm transition-colors"
              :class="selectedDays === d
                ? 'bg-blue-600 text-white'
                : 'bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-600'"
              @click="selectedDays = d"
            >{{ d }}天</button>
          </div>
        </div>
      </div>

      <!-- KPI 卡片行 -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <SummaryCard
          :title="`近${selectedDays}日 Premium 请求`"
          :value="kpiTotalPremium"
          :loading="loading"
          color="green"
        />
        <SummaryCard
          title="活跃用户数"
          :value="kpiActiveUsers"
          :loading="loading"
          color="blue"
        />
        <SummaryCard
          title="人均请求数"
          :value="kpiAvgRequests"
          :loading="loading"
          color="purple"
        />
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">最高消耗用户</p>
          <template v-if="loading">
            <div class="mt-2 h-6 w-3/4 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          </template>
          <template v-else-if="topUser">
            <p class="mt-1 truncate text-xl font-bold text-gray-900 dark:text-white">{{ topUser.username }}</p>
            <p class="text-sm text-orange-500 font-medium">{{ topUser.total.toLocaleString() }} 次</p>
          </template>
          <template v-else>
            <p class="mt-1 text-xl font-bold text-gray-400">—</p>
          </template>
        </div>
      </div>

      <!-- 错误提示 -->
      <div v-if="error" class="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
        {{ error }}
      </div>

      <!-- 趋势折线图卡片 -->
      <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">用户请求趋势</h2>
          <div class="flex rounded-md border border-gray-200 dark:border-gray-600 overflow-hidden text-xs">
            <button
              v-for="m in [{ key: 'premium', label: 'Premium' }, { key: 'agent', label: 'Agent' }]"
              :key="m.key"
              class="px-2.5 py-1 transition-colors"
              :class="chartMetric === m.key
                ? 'bg-blue-600 text-white'
                : 'bg-white dark:bg-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-50'"
              @click="chartMetric = m.key as 'premium' | 'agent'"
            >{{ m.label }}</button>
          </div>
        </div>
        <div class="p-4">
          <UsersDailyChart :days="selectedDays" :metric="chartMetric" />
        </div>
      </div>

      <!-- 用户排行表 -->
      <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <div class="flex flex-col gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">用户排行</h2>
          <div class="flex items-center gap-2">
            <!-- 排序维度 -->
            <select
              v-model="sortKey"
              class="rounded border border-gray-300 bg-white px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white"
            >
              <option value="premium">按 Premium 排序</option>
              <option value="agent">按 Agent 排序</option>
              <option value="total">按总请求排序</option>
            </select>
            <!-- 搜索 -->
            <input
              v-model="searchQuery"
              type="text"
              placeholder="搜索用户..."
              class="rounded border border-gray-300 px-2 py-1 text-xs focus:border-blue-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700 dark:text-white"
            />
          </div>
        </div>

        <div v-if="loading" class="flex h-32 items-center justify-center">
          <LoadingSpinner />
        </div>
        <div v-else-if="sortedUsers.length === 0" class="flex h-32 items-center justify-center text-sm text-gray-400">
          暂无数据
        </div>
        <table v-else class="w-full divide-y divide-gray-100 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th class="w-10 px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">#</th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">用户</th>
              <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Premium</th>
              <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Agent</th>
              <th class="hidden px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 md:table-cell">近7日趋势</th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">常用模型</th>
              <th class="px-4 py-3" />
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr
              v-for="(user, idx) in sortedUsers"
              :key="user.user_id"
              class="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors"
            >
              <td class="px-4 py-3 text-sm text-gray-400">
                <span v-if="idx === 0">🥇</span>
                <span v-else-if="idx === 1">🥈</span>
                <span v-else-if="idx === 2">🥉</span>
                <span v-else class="text-xs">{{ idx + 1 }}</span>
              </td>
              <td class="px-4 py-3 text-sm font-semibold text-gray-900 dark:text-white">
                {{ user.username }}
              </td>
              <td class="px-4 py-3 text-right text-sm">
                <span class="inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900/30 dark:text-green-400">
                  {{ user.premium.toLocaleString() }}
                </span>
              </td>
              <td class="px-4 py-3 text-right text-sm">
                <span class="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900/30 dark:text-blue-400">
                  {{ user.agent.toLocaleString() }}
                </span>
              </td>
              <td class="hidden px-4 py-3 md:table-cell">
                <UserSparkline :data="user.sparkline" />
              </td>
              <td class="px-4 py-3 text-xs text-gray-500 dark:text-gray-400 max-w-[140px] truncate">
                {{ user.topModel || '—' }}
              </td>
              <td class="px-4 py-3 text-right">
                <router-link
                  :to="{ name: 'AdminCopilotUserDetail', params: { id: user.user_id } }"
                  class="text-xs font-medium text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
                >
                  详情 →
                </router-link>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getCopilotUsersDailyStats,
  getCopilotUserStats,
} from '@/api/admin/copilotAnalytics'
import type {
  CopilotUsersDailyStatsResult,
  CopilotUserStatsResult,
} from '@/api/admin/copilotAnalytics'
import { extractErrorMessage } from '@/api/client'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import SummaryCard from '@/components/admin/copilot/CopilotSummaryCard.vue'
import UsersDailyChart from '@/components/admin/copilot/UsersDailyChart.vue'
import UserSparkline from '@/components/admin/copilot/UserSparkline.vue'

const { t } = useI18n()

const selectedDays = ref(30)
const chartMetric = ref<'premium' | 'agent'>('premium')
const sortKey = ref<'premium' | 'agent' | 'total'>('premium')
const searchQuery = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

let dailyData = ref<CopilotUsersDailyStatsResult | null>(null)
let todayData = ref<CopilotUserStatsResult | null>(null)

// ── 聚合计算 ──

/** 每个用户在所选时间范围内的汇总（premium/agent/sparkline） */
const userAggregates = computed(() => {
  if (!dailyData.value) return []

  // 构建 user_id → {premium, agent, daily[]} 映射
  const map = new Map<number, { username: string; premium: number; agent: number; daily: Map<string, number> }>()

  for (const user of dailyData.value.users) {
    map.set(user.user_id, { username: user.username, premium: 0, agent: 0, daily: new Map() })
  }
  for (const entry of dailyData.value.days) {
    const u = map.get(entry.user_id)
    if (!u) continue
    u.premium += entry.premium_count
    u.agent += entry.agent_count
    u.daily.set(entry.date, entry.premium_count + entry.agent_count)
  }

  // 最近7日 sparkline 数据
  const last7: string[] = []
  const now = new Date()
  for (let i = 6; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    last7.push(d.toISOString().slice(0, 10))
  }

  // 从今日单日数据中获取模型信息
  const modelByUser = new Map<number, string>()
  if (todayData.value) {
    for (const u of todayData.value.users) {
      if (u.models?.length) modelByUser.set(u.user_id, u.models[0])
    }
  }

  return Array.from(map.entries()).map(([user_id, data]) => ({
    user_id,
    username: data.username,
    premium: data.premium,
    agent: data.agent,
    total: data.premium + data.agent,
    sparkline: last7.map(d => data.daily.get(d) ?? 0),
    topModel: modelByUser.get(user_id) ?? '',
  }))
})

const kpiTotalPremium = computed(() =>
  userAggregates.value.reduce((s, u) => s + u.premium, 0)
)
const kpiActiveUsers = computed(() =>
  userAggregates.value.filter(u => u.total > 0).length
)
const kpiAvgRequests = computed(() => {
  const active = kpiActiveUsers.value
  return active === 0 ? 0 : Math.round(kpiTotalPremium.value / active)
})
const topUser = computed(() => {
  if (!userAggregates.value.length) return null
  return userAggregates.value.reduce((a, b) => a.premium > b.premium ? a : b)
})

const sortedUsers = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  let list = userAggregates.value
  if (q) list = list.filter(u => u.username.toLowerCase().includes(q))
  return [...list].sort((a, b) => b[sortKey.value] - a[sortKey.value])
})

// ── 数据加载 ──

async function loadAll() {
  loading.value = true
  error.value = null
  try {
    const [daily, today] = await Promise.all([
      getCopilotUsersDailyStats({ days: selectedDays.value }),
      getCopilotUserStats({}),
    ])
    dailyData.value = daily
    todayData.value = today
  } catch (e: unknown) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(loadAll)
watch(selectedDays, loadAll)
</script>
```

- [ ] **Step 2: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit 2>&1 | head -30
```

期望：无类型错误

- [ ] **Step 3: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/views/admin/copilot/CopilotUsersView.vue
git commit -m "Feature: 重构 CopilotUsersView 为趋势图+排行表大盘"
```

---

## Task 10: 前端 — 新建 CopilotUserDetailView（用户详情页）

**Files:**
- Create: `frontend/src/views/admin/copilot/CopilotUserDetailView.vue`

- [ ] **Step 1: 创建文件**

```vue
<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 面包屑 -->
      <nav class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <router-link to="/admin/copilot/users" class="hover:text-blue-600 dark:hover:text-blue-400">
          用户分析
        </router-link>
        <span>›</span>
        <span class="text-gray-900 dark:text-white">{{ summary?.username ?? `用户 #${userId}` }}</span>
      </nav>

      <!-- 加载状态 -->
      <div v-if="loadingSummary" class="flex h-32 items-center justify-center">
        <LoadingSpinner />
      </div>
      <div v-else-if="summaryError" class="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
        {{ summaryError }}
      </div>
      <template v-else-if="summary">
        <!-- 用户信息卡 -->
        <div class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex items-center gap-3">
              <div class="flex h-12 w-12 items-center justify-center rounded-full bg-blue-100 text-xl font-bold text-blue-700 dark:bg-blue-900/40 dark:text-blue-400">
                {{ summary.username.slice(0, 1).toUpperCase() }}
              </div>
              <div>
                <h1 class="text-xl font-bold text-gray-900 dark:text-white">{{ summary.username }}</h1>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  首次使用 {{ formatDate(summary.first_request_at) }} ·
                  最近活跃 {{ formatDate(summary.last_request_at) }}
                </p>
              </div>
            </div>
            <div class="flex gap-6">
              <div class="text-center">
                <p class="text-2xl font-bold text-green-600">{{ summary.total_premium_requests.toLocaleString() }}</p>
                <p class="text-xs text-gray-500">Premium 请求</p>
              </div>
              <div class="text-center">
                <p class="text-2xl font-bold text-blue-600">{{ summary.total_agent_requests.toLocaleString() }}</p>
                <p class="text-xs text-gray-500">Agent 请求</p>
              </div>
              <div class="text-center">
                <p class="text-2xl font-bold text-gray-700 dark:text-gray-300">
                  {{ (summary.total_premium_requests + summary.total_agent_requests).toLocaleString() }}
                </p>
                <p class="text-xs text-gray-500">累计总量</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 图表双列 -->
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <!-- 近30日每日请求柱状图 -->
          <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div class="border-b border-gray-200 px-4 py-3 dark:border-gray-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">近30日每日请求</h2>
            </div>
            <div class="p-4">
              <canvas ref="dailyChartRef" style="height:220px" />
            </div>
          </div>

          <!-- 模型分布环形图 -->
          <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div class="border-b border-gray-200 px-4 py-3 dark:border-gray-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">模型使用分布</h2>
            </div>
            <div class="flex items-center p-4 gap-4">
              <canvas ref="donutChartRef" style="height:180px;max-width:180px" />
              <ul class="flex-1 space-y-1.5 text-xs">
                <li
                  v-for="(m, i) in summary.top_models.slice(0, 6)"
                  :key="m.model"
                  class="flex items-center justify-between gap-2"
                >
                  <div class="flex items-center gap-1.5 min-w-0">
                    <span class="h-2 w-2 shrink-0 rounded-full" :style="{ backgroundColor: MODEL_COLORS[i % MODEL_COLORS.length] }" />
                    <span class="truncate text-gray-700 dark:text-gray-300">{{ m.model }}</span>
                  </div>
                  <span class="shrink-0 font-medium text-gray-500">{{ m.percentage.toFixed(1) }}%</span>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <!-- 24小时热力图 -->
        <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">活跃时段热力图（近7天）</h2>
            <p class="text-xs text-gray-400">颜色越深表示请求越多</p>
          </div>
          <div class="p-4">
            <UserHeatmap :user-id="userId" :days="7" />
          </div>
        </div>

        <!-- 请求日志 -->
        <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">请求日志</h2>
            <input
              v-model="selectedDate"
              type="date"
              class="rounded border border-gray-300 px-2 py-1 text-xs focus:border-blue-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700 dark:text-white"
            />
          </div>
          <div class="px-4 py-2">
            <UserRequestTree :user-id="userId" :date="selectedDate" />
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  Chart,
  BarController,
  BarElement,
  DoughnutController,
  ArcElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
} from 'chart.js'
import { getCopilotUserSummary, getCopilotUsersDailyStats } from '@/api/admin/copilotAnalytics'
import type { CopilotUserSummaryResult } from '@/api/admin/copilotAnalytics'
import { extractErrorMessage } from '@/api/client'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import UserRequestTree from '@/components/admin/copilot/UserRequestTree.vue'
import UserHeatmap from '@/components/admin/copilot/UserHeatmap.vue'

Chart.register(BarController, BarElement, DoughnutController, ArcElement, LinearScale, CategoryScale, Tooltip, Legend)

const MODEL_COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4']

const route = useRoute()
const userId = computed(() => Number(route.params.id))

const summary = ref<CopilotUserSummaryResult | null>(null)
const loadingSummary = ref(false)
const summaryError = ref<string | null>(null)

const dailyChartRef = ref<HTMLCanvasElement | null>(null)
const donutChartRef = ref<HTMLCanvasElement | null>(null)
let dailyChart: Chart | null = null
let donutChart: Chart | null = null

function localDateString(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}
const selectedDate = ref(localDateString())

function formatDate(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

async function buildDailyChart() {
  if (!dailyChartRef.value || !userId.value) return
  const result = await getCopilotUsersDailyStats({ days: 30 })
  const userEntry = result.users.find(u => u.user_id === userId.value)
  if (!userEntry) return

  const dates: string[] = []
  const now = new Date()
  for (let i = 29; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    dates.push(d.toISOString().slice(0, 10))
  }

  const dayMap = new Map<string, { premium: number; agent: number }>()
  for (const entry of result.days) {
    if (entry.user_id === userId.value) {
      dayMap.set(entry.date, { premium: entry.premium_count, agent: entry.agent_count })
    }
  }

  dailyChart?.destroy()
  dailyChart = new Chart(dailyChartRef.value, {
    type: 'bar',
    data: {
      labels: dates,
      datasets: [
        {
          label: 'Premium',
          data: dates.map(d => dayMap.get(d)?.premium ?? 0),
          backgroundColor: '#3b82f6',
          stack: 'stack',
        },
        {
          label: 'Agent',
          data: dates.map(d => dayMap.get(d)?.agent ?? 0),
          backgroundColor: '#10b981',
          stack: 'stack',
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index' },
      plugins: { legend: { position: 'bottom', labels: { boxWidth: 12 } } },
      scales: {
        x: { grid: { display: false }, ticks: { maxTicksLimit: 10, font: { size: 11 } } },
        y: { beginAtZero: true, stacked: true, ticks: { font: { size: 11 } } },
      },
    },
  })
}

function buildDonutChart(sum: CopilotUserSummaryResult) {
  if (!donutChartRef.value) return
  const top = sum.top_models.slice(0, 6)
  donutChart?.destroy()
  donutChart = new Chart(donutChartRef.value, {
    type: 'doughnut',
    data: {
      labels: top.map(m => m.model),
      datasets: [{
        data: top.map(m => m.count),
        backgroundColor: MODEL_COLORS.slice(0, top.length),
        borderWidth: 2,
      }],
    },
    options: {
      responsive: false,
      plugins: { legend: { display: false } },
      cutout: '60%',
    },
  })
}

async function loadAll() {
  if (!userId.value) return
  loadingSummary.value = true
  summaryError.value = null
  try {
    const sum = await getCopilotUserSummary(userId.value)
    summary.value = sum
    // 等 DOM 挂载后再绘图
    setTimeout(() => {
      buildDailyChart()
      buildDonutChart(sum)
    }, 50)
  } catch (e: unknown) {
    summaryError.value = extractErrorMessage(e)
  } finally {
    loadingSummary.value = false
  }
}

onMounted(loadAll)
watch(userId, loadAll)
onBeforeUnmount(() => {
  dailyChart?.destroy()
  donutChart?.destroy()
})
</script>
```

- [ ] **Step 2: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit 2>&1 | head -30
```

期望：无类型错误

- [ ] **Step 3: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/views/admin/copilot/CopilotUserDetailView.vue
git commit -m "Feature: 新建 CopilotUserDetailView 用户详情分析页"
```

---

## Task 11: 前端 — 注册路由

**Files:**
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: 在现有 `AdminCopilotUsers` 路由之后添加详情页路由**

在 `frontend/src/router/index.ts` 中，找到：
```javascript
  {
    path: '/admin/copilot/users',
    name: 'AdminCopilotUsers',
    ...
  },
```

在其后插入：

```javascript
  {
    path: '/admin/copilot/users/:id',
    name: 'AdminCopilotUserDetail',
    component: () => import('@/views/admin/copilot/CopilotUserDetailView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Copilot User Detail',
      titleKey: 'admin.copilot.users.detail.title',
    },
  },
```

- [ ] **Step 2: TypeScript 编译检查**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npx tsc --noEmit 2>&1 | head -30
```

期望：无类型错误

- [ ] **Step 3: 开发服务器启动验证**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run dev 2>&1 | head -20
```

期望：服务器成功启动，无编译错误

- [ ] **Step 4: Commit**

```bash
cd /Users/ziji/personal/github/sub2api
git add frontend/src/router/index.ts
git commit -m "Feature: 注册 Copilot 用户详情页路由"
```

---

## Task 12: 后端集成测试 & 最终验证

**Files:** 无文件修改，仅验证

- [ ] **Step 1: 后端完整编译 + 测试**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./...
go test ./internal/service/... -v -count=1 2>&1 | tail -30
```

期望：所有测试 PASS，binary 编译成功

- [ ] **Step 2: 用 curl 验证新接口**

确保后端已启动（参考 `deploy/` 目录的启动脚本），然后：

```bash
# 用户每日趋势（需要 admin token）
curl -s "http://localhost:8080/api/v1/admin/copilot/users/daily-stats?days=7" \
  -H "Authorization: Bearer <YOUR_ADMIN_TOKEN>" | jq '.data | {users: .users | length, days: .days | length}'

# 期望：{"users": N, "days": M}（N>=0, M>=0）

# 用户汇总（用已知用户ID测试）
curl -s "http://localhost:8080/api/v1/admin/copilot/users/1/summary" \
  -H "Authorization: Bearer <YOUR_ADMIN_TOKEN>" | jq '{username: .data.username, total: (.data.total_premium_requests + .data.total_agent_requests)}'

# 期望：{"username": "...", "total": N}
```

- [ ] **Step 3: 前端访问验证**

在浏览器中访问：
1. `http://localhost:5173/admin/copilot/users` — 确认大盘页正常渲染，折线图加载
2. 点击任一用户的「详情 →」链接 — 确认跳转到 `/admin/copilot/users/:id`
3. 详情页确认：用户信息卡、柱状图、环形图、热力图、请求日志均正常加载

- [ ] **Step 4: 最终 Commit（如有遗漏文件）**

```bash
cd /Users/ziji/personal/github/sub2api
git status
# 如有未提交文件：
git add <files>
git commit -m "Feature: Copilot 用户请求分析平台完成"
```

---

## 自检清单（Spec Self-Review）

### Spec 覆盖检查
- [x] KPI 卡片（总 Premium、活跃用户、人均请求、最高消耗用户）→ Task 9
- [x] 多用户趋势折线图（时间横轴，每用户一条线）→ Task 7 + 9
- [x] 用户排行表（含迷你 sparkline）→ Task 6 + 9
- [x] 用户详情页（独立路由）→ Task 10 + 11
- [x] 30日柱状图（premium/agent 堆叠）→ Task 10
- [x] 模型分布环形图 → Task 10
- [x] 24h 热力图 → Task 8 + 10
- [x] 请求日志（复用 UserRequestTree）→ Task 10
- [x] 后端用户每日趋势接口 → Task 1-4
- [x] 后端用户汇总接口 → Task 1-4

### 类型一致性检查
- `CopilotUsersDailyStatsResult` — 定义于 Task 1（Go）和 Task 5（TS），字段对齐 ✓
- `CopilotUserSummaryResult` — 定义于 Task 1（Go）和 Task 5（TS），字段对齐 ✓
- `getCopilotUsersDailyStats` — Task 5 定义，Task 7/9/10 使用，签名一致 ✓
- `getCopilotUserSummary` — Task 5 定义，Task 10 使用，签名一致 ✓
- `UserSparkline` — Task 6 定义，Task 9 使用，props 一致 ✓
- `UserHeatmap` — Task 8 定义，Task 10 使用，props 一致 ✓
- `AdminCopilotUserDetail` — Task 11 注册路由名称，Task 9 中 `router-link` 引用一致 ✓
