# Story 8.6: 分销数据报表

Status: ready-for-dev

## Story

As a 管理员,
I want 查看分销数据概览,
so that 了解分销业务整体情况。

## Acceptance Criteria

1. **AC1**: 概览数据：总邀请数、有效邀请数、佣金总额、提现总额
2. **AC2**: 日趋势图（最近 30 天）
3. **AC3**: 支持按日期范围查询日报表
4. **AC4**: 日报表包含：新增邀请、新增有效邀请、新增佣金、新增提现

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表), Story 2.3 (commission_record 表), Story 5.2 (withdrawal_record 表)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3, #4)
  - [ ] 1.1 `GET /api/admin/affiliate/stats/overview` 概览数据
  - [ ] 1.2 `GET /api/admin/affiliate/stats/daily?start=&end=` 日报表数据
  - [ ] 1.3 概览数据聚合查询（SUM/COUNT）
- [ ] Task 2: 前端页面 (AC: #1, #2, #3)
  - [ ] 2.1 创建 `src/views/admin/affiliate/AffiliateStatsView.vue`
  - [ ] 2.2 使用 Chart.js 绘制趋势图

## Dev Notes

### API

概览接口：
```
GET /api/admin/affiliate/stats/overview
```

响应：
```json
{
  "total_invites": 1234,
  "effective_invites": 567,
  "total_commission": 8900.50,
  "total_withdrawn": 5000.00,
  "pending_withdrawals": 3,
  "active_kol_count": 12
}
```

日报表接口：
```
GET /api/admin/affiliate/stats/daily?start=2026-01-01&end=2026-01-31
```

响应：
```json
{
  "items": [
    {
      "date": "2026-01-01",
      "new_invites": 15,
      "new_effective": 5,
      "new_commission": 123.45,
      "new_withdrawals": 2
    }
  ]
}
```

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface AffiliateStatsOverview {
  total_invites: number
  effective_invites: number
  total_commission: number
  total_withdrawn: number
  pending_withdrawals: number
  active_kol_count: number
}

export interface DailyStatsItem {
  date: string
  new_invites: number
  new_effective: number
  new_commission: number
  new_withdrawals: number
}

export interface DailyStatsResponse {
  items: DailyStatsItem[]
}
```

### 组件框架

```vue
<!-- src/views/admin/affiliate/AffiliateStatsView.vue -->
<script setup lang="ts">
import type { AffiliateStatsOverview, DailyStatsItem } from '@/types/affiliate'
import { ref, onMounted } from 'vue'
import { Chart } from 'chart.js/auto'
import { getStatsOverview, getDailyStats } from '@/api/admin/affiliate'

const overview = ref<AffiliateStatsOverview>()
const dailyData = ref<DailyStatsItem[]>([])
const chartRef = ref<HTMLCanvasElement>()

onMounted(async () => {
  const [overviewRes, dailyRes] = await Promise.all([
    getStatsOverview(),
    getDailyStats({ start: thirtyDaysAgo(), end: today() }),
  ])
  overview.value = overviewRes.data
  dailyData.value = dailyRes.data.items
  renderChart()
})
</script>
```

### API 客户端

```typescript
// src/api/admin/affiliate.ts
export function getStatsOverview() {
  return request.get<AffiliateStatsOverview>('/api/admin/affiliate/stats/overview')
}

export function getDailyStats(params: { start: string; end: string }) {
  return request.get<DailyStatsResponse>('/api/admin/affiliate/stats/daily', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_overview | 概览数据加载 | 显示 4 个统计卡片 |
| renders_chart | 日数据加载 | Chart.js 渲染趋势图 |
| date_range_filter | 切换日期范围 | 重新请求日报表数据 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetOverview | 有数据 | 返回正确聚合值 |
| TestGetDailyStats | 30天数据 | 按日分组返回正确统计 |
| TestGetDailyStats_EmptyRange | 无数据日期范围 | 返回空列表不报错 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.6] - FR59
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.2.3] - 统计报表 API

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
