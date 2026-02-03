# Story 8.8: 审计日志查询

Status: ready-for-dev

## Story

As a 管理员,
I want 查询分销相关的操作日志,
so that 追溯问题和审计。

## Acceptance Criteria

1. **AC1**: 支持按用户 ID 查询日志
2. **AC2**: 支持按操作类型筛选（relation_created, commission_created, commission_confirmed, commission_cancelled, tier_upgraded, withdraw_requested, withdraw_approved, withdraw_rejected, kol_granted, kol_revoked）
3. **AC3**: 每条记录：操作类型、用户、变更前后值、操作人、IP、时间
4. **AC4**: 分页显示，按时间倒序

## Dependencies

- **Depends On**: 所有写入审计日志的 Story（2.3, 4.2, 4.3, 4.4, 5.2, 8.2, 8.4, 8.5）
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3, #4)
  - [ ] 1.1 `GET /api/admin/affiliate/audit-logs?user_id=&action=&page=1&size=20`
  - [ ] 1.2 创建 `AffiliateAuditLogRepository` 查询方法
- [ ] Task 2: 前端页面 (AC: #1, #2, #3, #4)
  - [ ] 2.1 创建 `src/views/admin/affiliate/AuditLogView.vue`
  - [ ] 2.2 操作类型下拉筛选
  - [ ] 2.3 变更前后值 JSON 格式化展示

## Dev Notes

### API

```
GET /api/admin/affiliate/audit-logs?user_id=123&action=kol_granted&page=1&size=20
```

响应：
```json
{
  "items": [
    {
      "id": 1,
      "user_id": 123,
      "action": "kol_granted",
      "entity_type": "user_affiliate",
      "entity_id": 123,
      "before_value": {"is_kol": false},
      "after_value": {"is_kol": true, "kol_config": {"promo_code": "DAXIN"}},
      "operator_id": 1,
      "ip": "192.168.1.1",
      "created_at": "2026-01-15T10:30:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "size": 20
}
```

### 审计日志操作类型

| action | 说明 | 触发场景 |
|--------|------|---------|
| relation_created | 邀请关系建立 | Story 1.5 |
| commission_created | 佣金创建 | Story 2.3 |
| commission_confirmed | 佣金确认 | Story 4.3 |
| commission_cancelled | 佣金取消 | Story 4.4 |
| tier_upgraded | 阶梯升级 | Story 4.2 |
| withdraw_requested | 提现申请 | Story 5.2 |
| withdraw_approved | 提现通过 | Story 8.4 |
| withdraw_rejected | 提现拒绝 | Story 8.5 |
| kol_granted | 设置 KOL | Story 8.2 |
| kol_revoked | 取消 KOL | Story 8.2 |

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface AuditLogItem {
  id: number
  user_id: number
  action: string
  entity_type: string
  entity_id: number
  before_value: Record<string, unknown> | null
  after_value: Record<string, unknown> | null
  operator_id: number | null
  ip: string
  created_at: string
}

export interface AuditLogResponse {
  items: AuditLogItem[]
  total: number
  page: number
  size: number
}
```

### 组件框架

```vue
<!-- src/views/admin/affiliate/AuditLogView.vue -->
<script setup lang="ts">
import type { AuditLogItem } from '@/types/affiliate'
import { useTableLoader } from '@/composables/useTableLoader'
import { getAuditLogs } from '@/api/admin/affiliate'

const userIdFilter = ref<number | undefined>()
const actionFilter = ref<string>('')

const { data, loading, pagination, loadData } = useTableLoader<AuditLogItem>(
  (params) => getAuditLogs({
    ...params,
    user_id: userIdFilter.value,
    action: actionFilter.value || undefined,
  })
)

// JSON diff 展示
function formatJsonDiff(before: Record<string, unknown> | null, after: Record<string, unknown> | null) {
  return { before: JSON.stringify(before, null, 2), after: JSON.stringify(after, null, 2) }
}
</script>
```

### API 客户端

```typescript
// src/api/admin/affiliate.ts
export function getAuditLogs(params: {
  page: number
  size: number
  user_id?: number
  action?: string
}) {
  return request.get<AuditLogResponse>('/api/admin/affiliate/audit-logs', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_log_list | 传入日志数据 | 渲染列表 |
| filter_by_user | 输入用户 ID | 请求 user_id 参数 |
| filter_by_action | 选择操作类型 | 请求 action 参数 |
| json_diff_display | 点击查看详情 | 格式化展示 before/after JSON |
| empty_state | 无日志 | 显示空状态 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetAuditLogs_ByUser | user_id=123 | 只返回该用户日志 |
| TestGetAuditLogs_ByAction | action=kol_granted | 只返回 KOL 相关日志 |
| TestGetAuditLogs_Paginate | page=2, size=20 | 返回正确偏移量 |
| TestGetAuditLogs_OrderByTime | 查询结果 | 按 created_at 倒序 |

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.8] - FR64, NFR6
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.7] - affiliate_audit_log 表

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
