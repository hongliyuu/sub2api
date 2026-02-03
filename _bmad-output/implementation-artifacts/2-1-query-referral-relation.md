# Story 2.1: 查询邀请关系

Status: ready-for-dev

## Story

As a 系统,
I want 在被邀请人充值时查询其邀请关系,
so that 确定是否需要计算佣金。

## Acceptance Criteria

1. **AC1**: 通过 invitee_id 查询 referral_relation 返回完整邀请关系
2. **AC2**: 无邀请关系时返回 nil（跳过佣金计算）
3. **AC3**: 查询结果缓存到 Redis `aff:relation:{invitee_id}`，TTL 1h
4. **AC4**: 缓存未命中时查询数据库并回填

## Dependencies

- **Depends On**: Story 1.5 (referral_relation 表和绑定逻辑已实现)
- **Depended By**: Story 2.3 (首充佣金计算需要查询邀请关系)

## Tasks / Subtasks

- [ ] Task 1: Repository 层 (AC: #1, #2)
  - [ ] 1.1 确保 `referral_relation_repo.go` 中 `GetByInviteeID` 方法正确实现
- [ ] Task 2: Service 层缓存逻辑 (AC: #3, #4)
  - [ ] 2.1 在 `affiliate_service.go` 添加 `GetRelationByInvitee(ctx, inviteeID)` 方法
  - [ ] 2.2 先查 Redis 缓存，未命中查 DB 并回填

## Dev Notes

### Service 实现

```go
func (s *AffiliateService) GetRelationByInvitee(ctx context.Context, inviteeID int64) (*ReferralRelation, error) {
    cacheKey := fmt.Sprintf("aff:relation:%d", inviteeID)

    // 1. 查缓存
    cached, err := s.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var relation ReferralRelation
        if json.Unmarshal([]byte(cached), &relation) == nil {
            return &relation, nil
        }
    }

    // 2. 查数据库
    relation, err := s.relationRepo.GetByInviteeID(ctx, inviteeID)
    if err != nil {
        return nil, err
    }
    if relation == nil {
        return nil, nil
    }

    // 3. 回填缓存
    data, _ := json.Marshal(relation)
    s.redis.Set(ctx, cacheKey, string(data), time.Hour)

    return relation, nil
}
```

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#5.1] - 缓存策略 aff:relation:{invitee_id} 1h
- [Source: _bmad-output/planning-artifacts/epics.md#Story-2.1] - FR12前置

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetRelationByInvitee_CacheHit | Redis 缓存命中 | 直接返回缓存数据，不查 DB |
| TestGetRelationByInvitee_CacheMiss | Redis 缓存未命中 | 查 DB 并回填缓存，TTL=1h |
| TestGetRelationByInvitee_NoRelation | 无邀请关系 | 返回 nil, nil |
| TestGetRelationByInvitee_CacheCorrupt | 缓存 JSON 损坏 | 降级查 DB，不 panic |
| TestGetRelationByInvitee_DBError | DB 查询失败 | 返回 error，不写缓存 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整缓存流程 | 第一次查 DB 回填缓存 → 第二次直接从 Redis 读取 |
| 缓存过期后重新查询 | TTL 过期后重新查 DB 并更新缓存 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
