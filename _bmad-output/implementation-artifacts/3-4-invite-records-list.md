# Story 3.4: 邀请记录列表

Status: ready-for-dev

## Story

As a 普通用户,
I want 查看我邀请的用户列表,
so that 了解邀请详情和贡献情况。

## Acceptance Criteria

1. **AC1**: 显示邀请用户列表（分页，每页 20 条）
2. **AC2**: 每条记录显示：脱敏昵称、注册时间、状态（有效/待激活）、首充时间、累计贡献
3. **AC3**: 支持 Tab 筛选：全部 / 有效 / 待激活
4. **AC4**: 按注册时间倒序排列

## Dependencies

- **Depends On**: Story 1.5 (referral_relation 表数据), Story 3.1 (推广中心入口和路由)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 API 实现 (AC: #1, #2, #3, #4)
  - [ ] 1.1 在 `affiliate_handler.go` 添加 `GetInviteRecords` 方法
  - [ ] 1.2 在 `affiliate_service.go` 添加 `GetInviteRecords(ctx, userID, status, page, size)`
  - [ ] 1.3 JOIN referral_relation + users 表查询
  - [ ] 1.4 注册路由 `GET /api/v1/affiliate/invites`
- [ ] Task 2: 前端列表页面 (AC: #1, #2, #3)
  - [ ] 2.1 创建 `src/views/user/affiliate/InviteRecordsView.vue`
  - [ ] 2.2 创建 `src/components/user/affiliate/InviteRecordItem.vue`
  - [ ] 2.3 Tab 筛选组件

## Dev Notes

### API 设计

```
GET /api/v1/affiliate/invites?page=1&size=20&status=all|effective|pending
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "total": 23,
    "list": [
      {
        "user_id": 10086,
        "nickname": "用户***86",
        "registered_at": "2026-01-15T10:00:00Z",
        "status": "first_charged",
        "first_charge_at": "2026-01-16T08:00:00Z",
        "contributed": 3.50
      }
    ]
  }
}
```

### 昵称脱敏

```go
func maskNickname(nickname string) string {
    runes := []rune(nickname)
    if len(runes) <= 2 {
        return string(runes[:1]) + "***"
    }
    return string(runes[:1]) + "***" + string(runes[len(runes)-2:])
}
```

### Repository 查询

```go
// ReferralRelationRepo
func (r *repo) GetByInviterID(ctx context.Context, inviterID int64, status string, params PaginationParams) ([]InviteRecord, *PaginationResult, error) {
    query := r.client.ReferralRelation.Query().
        Where(referralrelation.InviterIDEQ(inviterID)).
        Order(ent.Desc(referralrelation.FieldCreatedAt))

    if status == "effective" {
        query = query.Where(referralrelation.InviteeStatusEQ("first_charged"))
    } else if status == "pending" {
        query = query.Where(referralrelation.InviteeStatusEQ("registered"))
    }

    // 分页处理...
}
```

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.1.2] - API 响应结构
- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.4] - FR43

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface InviteRecord {
  user_id: number
  nickname: string       // 脱敏后
  registered_at: string  // ISO 8601
  status: 'registered' | 'first_charged'
  first_charge_at: string | null
  contributed: number
}

export interface InviteRecordsResponse {
  total: number
  list: InviteRecord[]
}
```

### composable 用法

```typescript
// 使用 useTableLoader composable
const { data, loading, pagination, loadData } = useTableLoader<InviteRecord>(
  (params) => getInviteRecords({ ...params, status: activeTab.value })
)
```

### API 客户端

```typescript
// src/api/affiliate.ts
export function getInviteRecords(params: {
  page: number
  size: number
  status: 'all' | 'effective' | 'pending'
}) {
  return request.get<InviteRecordsResponse>('/api/v1/affiliate/invites', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_list | 传入 3 条记录 | 渲染 3 个 InviteRecordItem |
| tab_filter_all | 点击"全部" Tab | 请求 status=all |
| tab_filter_effective | 点击"有效" Tab | 请求 status=effective |
| pagination_next | 点击下一页 | 请求 page=2 |
| nickname_masked | nickname="用户***86" | 正确显示脱敏昵称 |
| empty_state | 返回空列表 | 显示空状态提示 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetInviteRecords_Normal | 有邀请记录 | 返回分页列表 |
| TestGetInviteRecords_Filter | status=effective | 只返回 first_charged 记录 |
| TestMaskNickname_Short | 2字符昵称 | "张***" |
| TestMaskNickname_Long | 5字符昵称 | "张***五六" |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
