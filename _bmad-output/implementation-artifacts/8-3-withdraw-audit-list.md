# Story 8.3: 提现审核列表

Status: ready-for-dev

## Story

As a 管理员,
I want 查看待审核的提现申请,
so that 处理用户提现。

## Acceptance Criteria

1. **AC1**: 显示提现申请列表（默认显示待审核）
2. **AC2**: 每条记录：用户、金额、方式、账户、申请时间
3. **AC3**: 支持状态筛选（pending/approved/rejected/completed）
4. **AC4**: 点击查看详情显示用户分销数据

## Dependencies

- **Depends On**: Story 5.2 (withdrawal_record 表数据)
- **Depended By**: Story 8.4 (审核通过), Story 8.5 (审核拒绝)

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3)
  - [ ] 1.1 `GET /api/admin/affiliate/withdrawals?status=pending&page=1`
- [ ] Task 2: 前端页面 (AC: #1, #2, #3, #4)
  - [ ] 2.1 创建 `src/views/admin/affiliate/WithdrawAuditView.vue`

## Dev Notes

### API

```
GET /api/admin/affiliate/withdrawals?status=pending&page=1&size=20
```

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface WithdrawAuditItem {
  id: number
  user_id: number
  user_nickname: string
  amount: number
  payment_method: string
  payment_account: string  // 脱敏
  status: 'pending' | 'approved' | 'rejected' | 'completed'
  created_at: string
  reviewed_by: number | null
  reviewed_at: string | null
}

export interface WithdrawAuditListResponse {
  total: number
  list: WithdrawAuditItem[]
}
```

### 组件框架

```vue
<!-- src/views/admin/affiliate/WithdrawAuditView.vue -->
<script setup lang="ts">
import type { WithdrawAuditItem } from '@/types/affiliate'
import { useTableLoader } from '@/composables/useTableLoader'
import { getAdminWithdrawals } from '@/api/admin/affiliate'

const statusFilter = ref<string>('pending')

const { data, loading, pagination, loadData } = useTableLoader<WithdrawAuditItem>(
  (params) => getAdminWithdrawals({ ...params, status: statusFilter.value })
)
</script>
```

### API 客户端

```typescript
// src/api/admin/affiliate.ts
export function getAdminWithdrawals(params: {
  page: number
  size: number
  status?: string
}) {
  return request.get<WithdrawAuditListResponse>('/api/admin/affiliate/withdrawals', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_list | 传入审核数据 | 渲染列表 |
| default_pending | 初始加载 | 默认 status=pending |
| filter_status | 切换状态筛选 | 重新请求对应状态 |
| detail_dialog | 点击查看详情 | 弹窗显示用户分销数据 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetAdminWithdrawals_Default | 无 status 参数 | 返回 pending 记录 |
| TestGetAdminWithdrawals_FilterStatus | status=approved | 只返回已审核 |
| TestGetAdminWithdrawals_JoinUser | 查询结果 | 包含用户昵称 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.3] - FR61
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.2.3] - 提现管理 API

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
