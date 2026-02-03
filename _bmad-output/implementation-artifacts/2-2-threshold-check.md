# Story 2.2: 分成门槛检查

Status: ready-for-dev

## Story

As a 系统,
I want 检查邀请人是否满足分成门槛条件,
so that 只有符合条件的邀请人才能获得佣金。

## Acceptance Criteria

1. **AC1**: 检查条件1：邀请人是否购买过包月套餐
2. **AC2**: 检查条件2：邀请人历史消费是否累计满 $10
3. **AC3**: 两个条件满足其一即可获得分成资格
4. **AC4**: 不满足门槛时不产生佣金（邀请关系保留）

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表和分销身份已存在)
- **Depended By**: Story 2.3 (首充佣金计算需要门槛检查)

## Tasks / Subtasks

- [ ] Task 1: 实现门槛检查逻辑 (AC: #1, #2, #3)
  - [ ] 1.1 在 `affiliate_service.go` 添加 `CheckThreshold(ctx, inviterID)` 方法
  - [ ] 1.2 查询用户订阅状态（是否有活跃包月）
  - [ ] 1.3 查询用户累计消费金额
  - [ ] 1.4 两者满足其一返回 true

## Dev Notes

### Service 实现

```go
const defaultMinSpend = 10.00 // 最低消费门槛

func (s *AffiliateService) CheckThreshold(ctx context.Context, inviterID int64) (bool, error) {
    // 条件1: 是否有包月套餐
    hasSubscription, err := s.subscriptionRepo.HasActiveSubscription(ctx, inviterID)
    if err != nil {
        return false, err
    }
    if hasSubscription {
        return true, nil
    }

    // 条件2: 累计消费 >= $10
    totalSpent, err := s.usageService.GetTotalSpent(ctx, inviterID)
    if err != nil {
        return false, err
    }
    return totalSpent >= defaultMinSpend, nil
}
```

### 依赖说明

需要查询现有的 SubscriptionRepository 和 UsageService：
- 查看 `user_subscriptions` 表获取订阅状态
- 查看用户累计消费数据

门槛值硬编码 $10，Epic 7 配置化。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-2.2] - FR14, FR15
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.2] - 充值佣金流程

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestCheckThreshold_HasSubscription | 有活跃包月套餐 | 返回 true |
| TestCheckThreshold_SpentAbove10 | 无包月，累计消费 $15 | 返回 true |
| TestCheckThreshold_SpentExact10 | 无包月，累计消费恰好 $10.00 | 返回 true（边界值） |
| TestCheckThreshold_SpentBelow10 | 无包月，累计消费 $9.99 | 返回 false |
| TestCheckThreshold_BothMet | 有包月且消费满 $10 | 返回 true（短路不查消费） |
| TestCheckThreshold_NeitherMet | 无包月，消费 $0 | 返回 false |
| TestCheckThreshold_SubscriptionError | 查询订阅失败 | 返回 error |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 真实数据库查询 | 通过 testcontainers 验证 subscription + usage 联合查询 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
