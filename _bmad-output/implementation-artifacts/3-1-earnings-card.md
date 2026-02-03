# Story 3.1: 推广中心首页 - 收益卡片

Status: ready-for-dev

## Story

As a 普通用户,
I want 在推广中心看到我的收益概览,
so that 快速了解推广效果。

## Acceptance Criteria

1. **AC1**: 推广中心首页显示累计收益金额
2. **AC2**: 显示可提现金额
3. **AC3**: 显示提现进度条（距离 $100 门槛的百分比）
4. **AC4**: 可提现金额 < $100 时显示「还差 $XX」

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表提供收益数据), Story 2.3 (佣金计算产生收益)
- **Depended By**: Story 3.2 (战绩卡片在同一页面), Story 3.3 (复制链接在同一页面)

## Tasks / Subtasks

- [ ] Task 1: 创建推广中心首页 (AC: #1, #2, #3, #4)
  - [ ] 1.1 创建 `src/views/user/affiliate/AffiliateHomeView.vue`
  - [ ] 1.2 创建 `src/components/user/affiliate/EarningsCard.vue`
  - [ ] 1.3 创建 `src/components/user/affiliate/WithdrawProgressBar.vue`
- [ ] Task 2: 前端路由和导航 (AC: #1)
  - [ ] 2.1 在 `src/router/index.ts` 添加 `/user/affiliate` 路由
  - [ ] 2.2 在用户菜单中添加「推广中心」入口
- [ ] Task 3: API 调用 (AC: #1, #2)
  - [ ] 3.1 创建 `src/api/affiliate.ts`
  - [ ] 3.2 调用 `GET /api/v1/affiliate/info` 获取数据

## Dev Notes

### 组件结构

```
AffiliateHomeView.vue
├── EarningsCard.vue        // 收益卡片
│   ├── 累计收益金额
│   ├── 可提现金额
│   └── WithdrawProgressBar.vue  // 提现进度条
└── StatsCard.vue           // 战绩卡片（Story 3.2）
```

### EarningsCard 样式参考

- 使用 TailwindCSS
- 主色 Claude Orange (`#d97757`)
- 进度条：已有金额/门槛金额的百分比
- 支持暗色模式

### API 客户端

```typescript
// src/api/affiliate.ts
import request from '@/utils/request'

export function getAffiliateInfo() {
  return request.get('/api/v1/affiliate/info')
}
```

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.1] - FR34, FR35, FR36
- [Source: _bmad-output/affiliate/UX-affiliate-system.md] - 推广中心 UI 设计
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.1.1] - API 响应结构

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface AffiliateInfo {
  user_id: number
  total_earnings: number
  withdrawable_amount: number
  pending_amount: number
  effective_count: number
  total_invite_count: number
  tier_level: number
  tier_name: string
  commission_rate: number
  next_tier_threshold: number
  referral_link: string
  referral_code: string
  is_kol: boolean
}
```

### 组件框架

```vue
<!-- src/components/user/affiliate/EarningsCard.vue -->
<script setup lang="ts">
import type { AffiliateInfo } from '@/types/affiliate'

const props = defineProps<{
  info: AffiliateInfo
}>()

const withdrawThreshold = 100
const progress = computed(() =>
  Math.min(100, (props.info.withdrawable_amount / withdrawThreshold) * 100)
)
const remaining = computed(() =>
  Math.max(0, withdrawThreshold - props.info.withdrawable_amount)
)
</script>
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_earnings_amount | 传入 total_earnings=150.50 | 显示 "$150.50" |
| renders_withdrawable | 传入 withdrawable_amount=80 | 显示 "$80.00" |
| progress_bar_percentage | withdrawable=60 | 进度条宽度 60% |
| shows_remaining_hint | withdrawable=60 | 显示 "还差 $40" |
| progress_bar_full | withdrawable=120 | 进度条 100%，不显示还差提示 |
| formats_zero_amount | total_earnings=0 | 显示 "$0.00" |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
