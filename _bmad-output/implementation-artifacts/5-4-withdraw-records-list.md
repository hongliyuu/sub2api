# Story 5.4: 提现记录列表

Status: ready-for-dev

## Story

As a 普通用户,
I want 查看我的提现记录,
so that 了解提现进度和历史。

## Acceptance Criteria

1. **AC1**: 显示提现记录列表（分页）
2. **AC2**: 每条记录显示：金额、方式、脱敏账户、申请时间、状态、完成时间
3. **AC3**: 状态颜色区分（审核中=黄色，已完成=绿色，已拒绝=红色）

## Dependencies

- **Depends On**: Story 5.2 (withdrawal_record 表数据), Story 3.1 (推广中心路由)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 API (AC: #1, #2)
  - [ ] 1.1 在 `affiliate_handler.go` 添加 `GetWithdrawRecords`
  - [ ] 1.2 注册路由 `GET /api/v1/affiliate/withdrawals`
- [ ] Task 2: 前端页面 (AC: #1, #2, #3)
  - [ ] 2.1 创建 `src/views/user/affiliate/WithdrawRecordsView.vue`

## Dev Notes

### API

```
GET /api/v1/affiliate/withdrawals?page=1&size=20
```

查询当前用户的提现记录，按创建时间倒序。

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface WithdrawRecord {
  id: number
  amount: number
  payment_method: 'wechat' | 'alipay'
  payment_account: string  // 脱敏后
  status: 'pending' | 'approved' | 'rejected' | 'completed'
  reject_reason: string | null
  created_at: string
  completed_at: string | null
}

export interface WithdrawRecordsResponse {
  total: number
  list: WithdrawRecord[]
}
```

### composable 用法

```typescript
// 使用 useTableLoader composable
const { data, loading, pagination, loadData } = useTableLoader<WithdrawRecord>(
  (params) => getWithdrawRecords(params)
)
```

### API 客户端

```typescript
// src/api/affiliate.ts
export function getWithdrawRecords(params: { page: number; size: number }) {
  return request.get<WithdrawRecordsResponse>('/api/v1/affiliate/withdrawals', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_list | 传入提现记录 | 渲染列表 |
| status_color_pending | status=pending | 显示黄色标签 |
| status_color_completed | status=completed | 显示绿色标签 |
| status_color_rejected | status=rejected | 显示红色标签 |
| rejected_reason | status=rejected | 显示拒绝原因 |
| pagination | 点击下一页 | 请求 page=2 |
| empty_state | 返回空列表 | 显示空状态提示 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetWithdrawRecords_Paginate | 查询分页 | 正确返回分页数据 |
| TestGetWithdrawRecords_AccountMask | 原始账户 | 返回脱敏后的账户信息 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.4] - FR45

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
