# Story 8.7: KOL 排行榜

Status: ready-for-dev

## Story

As a 管理员,
I want 查看 KOL 业绩排行,
so that 评估 KOL 合作效果。

## Acceptance Criteria

1. **AC1**: 显示 KOL 排行榜，支持按有效邀请数或佣金总额排序
2. **AC2**: 每条记录：KOL 昵称、推广码、有效邀请数、佣金总额
3. **AC3**: 支持分页

## Dependencies

- **Depends On**: Story 6.1 (KOL 身份标识), Story 1.1 (user_affiliate 表)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3)
  - [ ] 1.1 `GET /api/admin/affiliate/stats/kol-ranking?sort_by=effective_count&page=1&size=20`
  - [ ] 1.2 查询 is_kol=true 的用户，JOIN users 表获取昵称
- [ ] Task 2: 前端页面 (AC: #1, #2, #3)
  - [ ] 2.1 在 `AffiliateStatsView.vue` 中添加 KOL 排行 Tab

## Dev Notes

### API

```
GET /api/admin/affiliate/stats/kol-ranking?sort_by=effective_count&order=desc&page=1&size=20
```

`sort_by` 可选值：`effective_count`（默认）、`total_earnings`

响应：
```json
{
  "items": [
    {
      "user_id": 123,
      "nickname": "大鑫",
      "promo_code": "DAXIN",
      "effective_count": 156,
      "total_earnings": 2345.67,
      "withdrawable": 1200.00
    }
  ],
  "total": 12,
  "page": 1,
  "size": 20
}
```

### 实现

```go
func (r *AffiliateRepository) GetKolRanking(ctx context.Context, sortBy string, page, size int) ([]*KolRankingItem, int, error) {
    query := r.client(ctx).UserAffiliate.Query().
        Where(useraffiliate.IsKol(true)).
        WithUser() // JOIN users

    switch sortBy {
    case "total_earnings":
        query = query.Order(ent.Desc(useraffiliate.FieldTotalEarnings))
    default:
        query = query.Order(ent.Desc(useraffiliate.FieldEffectiveCount))
    }

    total, _ := query.Clone().Count(ctx)
    items, err := query.Offset((page - 1) * size).Limit(size).All(ctx)
    return items, total, err
}
```

### TypeScript 类型定义

```typescript
// src/types/affiliate.ts
export interface KolRankingItem {
  user_id: number
  nickname: string
  promo_code: string
  effective_count: number
  total_earnings: number
  withdrawable: number
}

export interface KolRankingResponse {
  items: KolRankingItem[]
  total: number
  page: number
  size: number
}
```

### composable 用法

```typescript
// 使用 useTableLoader composable
const sortBy = ref<'effective_count' | 'total_earnings'>('effective_count')

const { data, loading, pagination, loadData } = useTableLoader<KolRankingItem>(
  (params) => getKolRanking({ ...params, sort_by: sortBy.value })
)
```

### API 客户端

```typescript
// src/api/admin/affiliate.ts
export function getKolRanking(params: {
  page: number
  size: number
  sort_by: string
}) {
  return request.get<KolRankingResponse>('/api/admin/affiliate/stats/kol-ranking', { params })
}
```

### Testing Requirements

#### 组件测试 (Vitest + @vue/test-utils)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| renders_ranking_list | 传入排行数据 | 渲染 KOL 列表 |
| sort_by_effective | 选择按邀请数排序 | sort_by=effective_count |
| sort_by_earnings | 选择按收益排序 | sort_by=total_earnings |
| pagination | 翻页 | 正确请求分页 |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetKolRanking_ByEffective | sort_by=effective_count | 按有效邀请数降序 |
| TestGetKolRanking_ByEarnings | sort_by=total_earnings | 按收益降序 |
| TestGetKolRanking_Empty | 无 KOL 用户 | 返回空列表 |

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.7] - FR60
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.2.3] - KOL 排行 API

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
