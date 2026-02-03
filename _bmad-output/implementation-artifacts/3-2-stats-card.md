# Story 3.2: 推广中心首页 - 战绩卡片

Status: ready-for-dev

## Story

As a 普通用户,
I want 查看我的邀请数据和阶梯进度,
so that 了解当前档位和升级目标。

## Acceptance Criteria

1. **AC1**: 显示邀请用户总数和有效用户数（已首充）
2. **AC2**: 显示当前阶梯档位名称和佣金比例
3. **AC3**: 显示阶梯升级进度条（再邀请 N 人升级）
4. **AC4**: 已达最高档位时显示「已达最高档位」

## Dependencies

- **Depends On**: Story 3.1 (AffiliateHomeView 页面和 AffiliateInfo 类型), Story 1.1 (阶梯佣金比例数据: GetCommissionRate + GetTierInfo + defaultTierRules)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 创建战绩卡片组件 (AC: #1, #2, #3, #4)
  - [ ] 1.1 创建 `src/components/user/affiliate/StatsCard.vue`
  - [ ] 1.2 创建 `src/components/user/affiliate/TierProgressBar.vue`
- [ ] Task 2: 后端扩展推广信息 API (AC: #1)
  - [ ] 2.1 在 `/api/v1/affiliate/info` 响应中添加 total_invite_count 字段
  - [ ] 2.2 查询 referral_relation 表统计总邀请数

## Dev Notes

### 数据来源

从 `GET /api/v1/affiliate/info` 获取：
- effective_count：有效邀请数（user_affiliate 表）
- total_invite_count：需新增，查询 referral_relation 表 COUNT
- tier_level, tier_name, commission_rate：阶梯信息
- next_tier_threshold：下一档位门槛

### 进度条计算

```typescript
const progress = computed(() => {
  if (info.tier_level >= 3) return 100 // 最高档位
  const current = info.effective_count
  const target = info.next_tier_threshold
  return Math.min(100, (current / target) * 100)
})
```

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.2] - FR37, FR38, FR39
- [Source: _bmad-output/affiliate/UX-affiliate-system.md] - 战绩卡片设计

### 组件框架

```vue
<!-- src/components/user/affiliate/StatsCard.vue -->
<script setup lang="ts">
import type { AffiliateInfo } from '@/types/affiliate'

const props = defineProps<{
  info: AffiliateInfo
}>()

const isMaxTier = computed(() => props.info.tier_level >= 3)
const tierProgress = computed(() => {
  if (isMaxTier.value) return 100
  const current = props.info.effective_count
  const target = props.info.next_tier_threshold
  return Math.min(100, (current / target) * 100)
})
const remainingToUpgrade = computed(() =>
  Math.max(0, props.info.next_tier_threshold - props.info.effective_count)
)
</script>
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_invite_counts | total=23, effective=10 | 显示总邀请 23，有效 10 |
| renders_tier_info | tier_name="银牌", rate=0.08 | 显示 "银牌 8%" |
| tier_progress_normal | effective=8, next_threshold=10 | 进度条 80%，显示 "再邀请 2 人升级" |
| tier_progress_max | tier_level=3 | 进度条 100%，显示 "已达最高档位" |
| tier_progress_zero | effective=0, next_threshold=10 | 进度条 0% |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
