# Story 2.3: 首充佣金计算

Status: ready-for-dev

## Story

As a 邀请人,
I want 在被邀请人首次充值时获得佣金,
so that 从推广中获得收益。

## Acceptance Criteria

1. **AC1**: 被邀请人首次充值成功时，系统计算佣金 = 充值金额 × 邀请人当前佣金比例
2. **AC2**: 创建 commission_record（status=pending），含 source_order_id
3. **AC3**: 更新被邀请人的 invitee_status 为 `first_charged`
4. **AC4**: 更新邀请人的 effective_count +1（原子操作）
5. **AC5**: 首充模式（binding_type=first_charge）只计算一次
6. **AC6**: 订单号唯一索引保证幂等性
7. **AC7**: 佣金计算在充值回调中异步调用（goroutine），失败不影响充值
8. **AC8**: 记录审计日志 `commission_created`

## Dependencies

- **Depends On**: Story 1.5 (邀请关系绑定), Story 2.1 (查询邀请关系), Story 2.2 (门槛检查)
- **Modifies**: 充值回调逻辑（在充值成功后异步调用 CalculateCommission）
- **Depended By**: Story 4.1~4.4 (阶梯佣金/升级/确认/退款), Story 6.4 (KOL 终身绑定), Story 2.4 (失败重试)

## Tasks / Subtasks

- [ ] Task 1: 实现佣金计算核心逻辑 (AC: #1, #2, #5, #6)
  - [ ] 1.1 在 `affiliate_service.go` 添加 `CalculateCommission(ctx, order)` 方法
  - [ ] 1.2 查询邀请关系 → 检查 binding_type → 检查门槛 → 计算佣金 → 创建记录
- [ ] Task 2: 更新被邀请人状态和邀请人计数 (AC: #3, #4)
  - [ ] 2.1 更新 referral_relation.invitee_status = "first_charged"
  - [ ] 2.2 原子更新 user_affiliate.effective_count +1
- [ ] Task 3: 集成到充值回调 (AC: #7)
  - [ ] 3.1 在充值成功回调中 goroutine 调用 CalculateCommission
  - [ ] 3.2 失败记录到 commission_retry 表
- [ ] Task 4: Repository 扩展 (AC: #2, #3, #4)
  - [ ] 4.1 CommissionRecordRepo: 添加 ExistsByOrderID 方法
  - [ ] 4.2 ReferralRelationRepo: 添加 UpdateInviteeStatus 方法
  - [ ] 4.3 AffiliateRepo: 添加 IncrementEffectiveCount 方法

## Dev Notes

### 核心逻辑

```go
func (s *AffiliateService) CalculateCommission(ctx context.Context, order Order) error {
    // 1. 查询邀请关系
    relation, err := s.GetRelationByInvitee(ctx, order.UserID)
    if relation == nil {
        return nil // 无邀请关系
    }

    // 2. 首充模式检查
    if relation.BindingType == "first_charge" {
        exists, _ := s.commissionRepo.ExistsBySourceUser(ctx, relation.InviterID, order.UserID, "recharge")
        if exists {
            return nil // 已有充值佣金
        }
    }

    // 3. 门槛检查
    passed, _ := s.CheckThreshold(ctx, relation.InviterID)
    if !passed {
        return nil
    }

    // 4. 获取佣金比例
    inviter, _ := s.affiliateRepo.GetByUserID(ctx, relation.InviterID)
    rate := s.GetCommissionRate(inviter)
    amount := order.Amount * rate

    // 5. 事务操作
    return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
        commission := &CommissionRecord{
            UserID:        relation.InviterID,
            SourceUserID:  order.UserID,
            RelationID:    relation.ID,
            SourceType:    "recharge",
            SourceOrderID: order.OrderID,
            Amount:        amount,
            Rate:          rate,
            Status:        "pending",
        }
        if err := s.commissionRepo.Create(txCtx, commission); err != nil {
            return err
        }

        // 更新被邀请人状态
        if relation.InviteeStatus == "registered" {
            s.relationRepo.UpdateInviteeStatus(txCtx, relation.ID, "first_charged")
            s.affiliateRepo.IncrementEffectiveCount(txCtx, relation.InviterID)
        }

        return nil
    })
}
```

### 充值回调集成

```go
// 在充值成功回调中
go func() {
    if err := affiliateService.CalculateCommission(ctx, order); err != nil {
        log.Error("commission calculation failed", "error", err, "order_id", order.OrderID)
        saveRetryRecord(order, err)
    }
}()
```

### 佣金比例获取

> `GetCommissionRate` 已在 Story 1.1 中实现（`affiliate_service.go`），此处直接调用。

```go
func (s *AffiliateService) GetCommissionRate(inviter *UserAffiliate) float64 {
    // KOL 优先使用专属比例
    if inviter.IsKol {
        if rate, ok := inviter.KolConfig["commission_rate"].(float64); ok && rate > 0 {
            return rate
        }
    }
    // 阶梯规则
    for _, tier := range defaultTierRules {
        if inviter.EffectiveCount >= tier.MinCount &&
           (tier.MaxCount == 0 || inviter.EffectiveCount <= tier.MaxCount) {
            return tier.Rate
        }
    }
    return 0.05 // 默认5%
}
```

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.2] - 充值佣金流程完整伪代码
- [Source: _bmad-output/planning-artifacts/epics.md#Story-2.3] - FR12, FR17, FR18
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.3] - 原子操作 SQL

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestCalculateCommission_Normal | 正常首充计算 | 创建 pending 佣金记录，金额=充值额×比例 |
| TestCalculateCommission_NoRelation | 无邀请关系 | 跳过，返回 nil |
| TestCalculateCommission_FirstChargeOnly | 首充模式已有佣金 | 跳过重复计算 |
| TestCalculateCommission_ThresholdNotMet | 门槛不满足 | 跳过，不创建佣金 |
| TestCalculateCommission_Idempotent | 同一订单号重复调用 | 唯一索引保证只创建一次 |
| TestCalculateCommission_EffectiveCountAtomic | 并发调用 | effective_count 原子递增不丢失 |
| TestCalculateCommission_InviteeStatusUpdate | 首次充值 | invitee_status 从 registered 变为 first_charged |
| TestCalculateCommission_AlreadyCharged | invitee_status=first_charged | 不重复递增 effective_count |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整首充佣金流程 | 充值→查关系→检查门槛→计算佣金→创建记录→更新状态 |
| 事务回滚 | 创建佣金记录后更新状态失败，整个事务回滚 |
| 并发幂等测试 | 多个 goroutine 同时处理同一订单，只创建一条佣金记录 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
