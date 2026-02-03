# Story 8.1: 分销用户列表

Status: ready-for-dev

## Story

As a 管理员,
I want 查看分销用户列表,
so that 了解整体分销情况。

## Acceptance Criteria

1. **AC1**: 分页显示用户列表
2. **AC2**: 每条记录：用户ID、昵称、邀请码、档位、有效邀请数、累计收益、是否KOL
3. **AC3**: 支持搜索（用户ID或昵称）
4. **AC4**: 支持筛选 KOL

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表)
- **Depended By**: Story 8.2 (设置 KOL 操作从此列表发起)

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3, #4)
  - [ ] 1.1 创建 `internal/handler/admin/affiliate_user_handler.go`
  - [ ] 1.2 `GET /api/admin/affiliate/users`
  - [ ] 1.3 JOIN user_affiliate + users 表查询
- [ ] Task 2: 前端管理页面 (AC: #1, #2, #3, #4)
  - [ ] 2.1 创建 `src/views/admin/affiliate/AffiliateUsersView.vue`

## Dev Notes

### API

```
GET /api/admin/affiliate/users?keyword=&is_kol=true&page=1&size=20
```

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface AffiliateUser {
  user_id: number
  nickname: string
  referral_code: string
  tier_level: number
  tier_name: string
  effective_count: number
  total_earnings: number
  is_kol: boolean
  created_at: string
}

export interface AffiliateUsersResponse {
  total: number
  list: AffiliateUser[]
}
```

### composable 用法

```typescript
// 使用 useTableLoader composable
const { data, loading, pagination, loadData } = useTableLoader<AffiliateUser>(
  (params) => getAffiliateUsers({
    ...params,
    keyword: searchKeyword.value,
    is_kol: filterKol.value,
  })
)
```

### API 客户端

```typescript
// src/api/admin/affiliate.ts
export function getAffiliateUsers(params: {
  page: number
  size: number
  keyword?: string
  is_kol?: boolean
}) {
  return request.get<AffiliateUsersResponse>('/api/admin/affiliate/users', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_user_list | 传入用户数据 | 渲染列表 |
| search_by_keyword | 输入搜索关键词 | 请求包含 keyword 参数 |
| filter_kol | 勾选 KOL 筛选 | 请求 is_kol=true |
| kol_badge | is_kol=true | 显示 KOL 标识 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetAffiliateUsers_Paginate | 查询分页 | 正确返回 |
| TestGetAffiliateUsers_SearchByID | keyword=10086 | 按用户 ID 搜索 |
| TestGetAffiliateUsers_FilterKol | is_kol=true | 只返回 KOL 用户 |

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.1] - FR53
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.2.2] - Admin 用户管理 API

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
