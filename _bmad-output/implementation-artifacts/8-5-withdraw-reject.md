# Story 8.5: 审核拒绝提现

Status: ready-for-dev

## Story

As a 管理员,
I want 拒绝不合规的提现申请,
so that 保护平台利益。

## Acceptance Criteria

1. **AC1**: 填写拒绝原因并提交
2. **AC2**: 提现状态更新为 rejected
3. **AC3**: 记录拒绝原因、审核人、审核时间
4. **AC4**: 将金额退回用户的 withdrawable
5. **AC5**: 记录审计日志 `withdraw_rejected`

## Dependencies

- **Depends On**: Story 8.3 (审核列表页面), Story 5.2 (withdrawal_record 表)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3, #4, #5)
  - [ ] 1.1 `PUT /api/admin/affiliate/withdrawals/:id/reject`
  - [ ] 1.2 事务中：更新状态 + 退回金额

## Dev Notes

```go
func (s *AffiliateService) RejectWithdraw(ctx context.Context, withdrawID int64, adminID int64, reason string) error {
    return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
        withdrawal, err := s.withdrawalRepo.GetByID(txCtx, withdrawID)
        if err != nil {
            return err
        }
        // 退回金额
        s.affiliateRepo.IncrementWithdrawable(txCtx, withdrawal.UserID, withdrawal.Amount)
        // 更新状态
        now := time.Now()
        return s.withdrawalRepo.Update(txCtx, withdrawID, map[string]interface{}{
            "status":        "rejected",
            "reject_reason": reason,
            "reviewed_by":   adminID,
            "reviewed_at":   now,
        })
    })
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestRejectWithdraw_Normal | status=pending, reason="信息不实" | 状态→rejected，withdrawable 恢复 |
| TestRejectWithdraw_NoReason | reason 为空 | 返回错误（原因必填） |
| TestRejectWithdraw_AlreadyReviewed | status=approved | 返回错误 |
| TestRejectWithdraw_BalanceRestore | amount=100 | withdrawable 增加 100 |
| TestRejectWithdraw_AuditLog | 拒绝成功 | 记录 withdraw_rejected 含 reason |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 拒绝退回事务 | 拒绝 → 状态更新 + 余额恢复 → 事务原子性 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.5] - FR63

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
