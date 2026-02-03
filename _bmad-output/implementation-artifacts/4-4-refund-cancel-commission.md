# Story 4.4: 退款取消佣金

Status: ready-for-dev

## Story

As a 系统,
I want 充值订单退款时自动取消对应佣金,
so that 防止虚假充值套利。

## Acceptance Criteria

1. **AC1**: 退款回调触发时查询关联佣金记录
2. **AC2**: 佣金状态更新为 cancelled
3. **AC3**: 如佣金已 confirmed，从 withdrawable 中扣减
4. **AC4**: 佣金已提现时记录警告日志（需人工处理）
5. **AC5**: 记录审计日志 `commission_cancelled`

## Dependencies

- **Depends On**: Story 2.3 (commission_record 表数据)
- **Modifies**: commission_record.status, user_affiliate.withdrawable
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 退款取消逻辑 (AC: #1, #2, #3, #4, #5)
  - [ ] 1.1 在 `affiliate_service.go` 添加 `CancelCommission(ctx, orderID)` 方法
  - [ ] 1.2 集成到退款回调流程

## Dev Notes

```go
func (s *AffiliateService) CancelCommission(ctx context.Context, orderID string) error {
    record, err := s.commissionRepo.GetByOrderID(ctx, orderID)
    if err != nil || record == nil {
        return nil // 无关联佣金
    }

    if record.Status == "withdrawn" {
        log.Warn("commission already withdrawn, manual handling needed", "order_id", orderID)
        return nil
    }

    return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
        if record.Status == "confirmed" {
            s.affiliateRepo.DecrementWithdrawable(txCtx, record.UserID, record.Amount)
        }
        return s.commissionRepo.UpdateStatus(txCtx, record.ID, "cancelled")
    })
}
```

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.4] - FR33
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.2] - 佣金计算流程

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestCancelCommission_Pending | status=pending | 直接取消，withdrawable 不变 |
| TestCancelCommission_Confirmed | status=confirmed | 取消并从 withdrawable 扣减 |
| TestCancelCommission_Withdrawn | status=withdrawn | 记录警告日志，不修改 |
| TestCancelCommission_NotFound | 无关联佣金记录 | 返回 nil，不报错 |
| TestCancelCommission_AuditLog | 取消佣金 | 记录 commission_cancelled 审计日志 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 退款取消事务完整性 | confirmed 佣金取消 → withdrawable 扣减 → 事务原子性 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
