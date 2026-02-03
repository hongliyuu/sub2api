# Story 3.5: 佣金明细列表

Status: ready-for-dev

## Story

As a 普通用户,
I want 查看我的佣金明细,
so that 了解每笔收益的来源和状态。

## Acceptance Criteria

1. **AC1**: 显示佣金记录列表（分页，每页 20 条）
2. **AC2**: 每条记录显示：类型（注册/充值）、来源用户（脱敏）、金额、比例、状态、时间
3. **AC3**: 支持 Tab 筛选：全部/待确认/已确认
4. **AC4**: 按时间倒序排列
5. **AC5**: 待确认状态显示「还剩 N 天」

## Dependencies

- **Depends On**: Story 2.3 (commission_record 表数据), Story 3.1 (推广中心入口和路由)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 API (AC: #1, #2, #3, #4, #5)
  - [ ] 1.1 在 `affiliate_handler.go` 添加 `GetCommissionRecords`
  - [ ] 1.2 在 `affiliate_service.go` 添加查询方法
  - [ ] 1.3 注册路由 `GET /api/v1/affiliate/commissions`
- [ ] Task 2: 前端页面 (AC: #1, #2, #3, #5)
  - [ ] 2.1 创建 `src/views/user/affiliate/CommissionRecordsView.vue`
  - [ ] 2.2 创建 `src/components/user/affiliate/CommissionRecordItem.vue`

## Dev Notes

### API

```
GET /api/v1/affiliate/commissions?page=1&size=20&status=all|pending|confirmed
```

Response 见 TDD 3.1.3。待确认记录增加 `days_remaining` 字段（7 - 已过天数）。

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface CommissionRecord {
  id: number
  source_type: 'register' | 'recharge'
  source_user_nickname: string  // 脱敏后
  amount: number
  rate: number
  status: 'pending' | 'confirmed' | 'cancelled'
  days_remaining: number | null  // 仅 pending 状态有值
  created_at: string
}

export interface CommissionRecordsResponse {
  total: number
  list: CommissionRecord[]
}
```

### composable 用法

```typescript
// 使用 useTableLoader composable
const { data, loading, pagination, loadData } = useTableLoader<CommissionRecord>(
  (params) => getCommissionRecords({ ...params, status: activeTab.value })
)
```

### API 客户端

```typescript
// src/api/affiliate.ts
export function getCommissionRecords(params: {
  page: number
  size: number
  status: 'all' | 'pending' | 'confirmed'
}) {
  return request.get<CommissionRecordsResponse>('/api/v1/affiliate/commissions', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_list | 传入佣金记录 | 渲染 CommissionRecordItem 列表 |
| tab_filter | 切换 Tab | 请求对应 status 参数 |
| days_remaining_display | status=pending, days_remaining=3 | 显示 "还剩 3 天" |
| confirmed_no_days | status=confirmed | 不显示剩余天数 |
| amount_format | amount=3.50, rate=0.05 | 显示 "$3.50 (5%)" |
| empty_state | 返回空列表 | 显示空状态提示 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetCommissionRecords_Paginate | 查询第2页 | 返回正确偏移量数据 |
| TestGetCommissionRecords_Filter | status=pending | 只返回 pending 记录 |
| TestDaysRemaining_Calc | 创建于3天前 | days_remaining=4 |

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.1.3] - API 响应结构
- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.5] - FR44

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
