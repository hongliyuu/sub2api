# Story 8.4: 审核通过提现

Status: ready-for-dev

## Story

As a 管理员,
I want 审核通过提现申请,
so that 用户可以收到款项。

## Acceptance Criteria

1. **AC1**: 点击「通过」后提现状态更新为 approved
2. **AC2**: 记录审核人和审核时间
3. **AC3**: 记录审计日志 `withdraw_approved`
4. **AC4**: 打款完成后状态更新为 completed

## Dependencies

- **Depends On**: Story 8.3 (审核列表页面), Story 5.2 (withdrawal_record 表)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3)
  - [ ] 1.1 `PUT /api/admin/affiliate/withdrawals/:id/approve`
  - [ ] 1.2 更新 status, reviewed_by, reviewed_at
- [ ] Task 2: 完成打款接口 (AC: #4)
  - [ ] 2.1 `PUT /api/admin/affiliate/withdrawals/:id/complete`（手动标记完成）

## Dev Notes

```go
func (s *AffiliateService) ApproveWithdraw(ctx context.Context, withdrawID int64, adminID int64) error {
    now := time.Now()
    return s.withdrawalRepo.Update(ctx, withdrawID, map[string]interface{}{
        "status":      "approved",
        "reviewed_by": adminID,
        "reviewed_at": now,
    })
}
```

实际打款为线下操作，管理员确认后手动标记 completed。

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestApproveWithdraw_Normal | status=pending | 更新为 approved，记录 reviewed_by 和 reviewed_at |
| TestApproveWithdraw_AlreadyReviewed | status=approved | 返回错误（不允许重复审核） |
| TestApproveWithdraw_Rejected | status=rejected | 返回错误 |
| TestApproveWithdraw_AuditLog | 审核通过 | 记录 withdraw_approved 审计日志 |
| TestCompleteWithdraw | status=approved | 更新为 completed，记录 completed_at |
| TestCompleteWithdraw_NotApproved | status=pending | 返回错误（必须先审核通过） |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.4] - FR29, FR30, FR62

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
