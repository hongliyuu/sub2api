# Story 5.1: 提现资格检查

Status: ready-for-dev

## Story

As a 普通用户,
I want 系统检查我是否满足提现条件,
so that 了解何时可以提现。

## Acceptance Criteria

1. **AC1**: 可提现金额 >= $100 时 can_withdraw=true
2. **AC2**: 可提现金额 < $100 时返回距离门槛还差多少
3. **AC3**: 提现资格信息包含在 `/api/v1/affiliate/info` 响应中

## Dependencies

- **Depends On**: Story 4.3 (佣金确认后 withdrawable 增加), Story 1.1 (user_affiliate.withdrawable 字段)
- **Depended By**: Story 5.2 (提现提交前需资格检查)

## Tasks / Subtasks

- [ ] Task 1: 扩展 GetAffiliateInfo (AC: #1, #2, #3)
  - [ ] 1.1 在响应中增加 can_withdraw 和 withdraw_gap 字段
  - [ ] 1.2 提现门槛硬编码 $100

## Dev Notes

已在 Story 1.2 的 `/api/v1/affiliate/info` 中包含 `can_withdraw` 和 `withdraw_threshold` 字段。本 Story 确保逻辑完整：

```go
canWithdraw := affiliate.Withdrawable >= withdrawThreshold
withdrawGap := max(0, withdrawThreshold - affiliate.Withdrawable)
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestWithdrawEligibility_Eligible | withdrawable=150 | can_withdraw=true, withdraw_gap=0 |
| TestWithdrawEligibility_Exact | withdrawable=100 | can_withdraw=true |
| TestWithdrawEligibility_NotEnough | withdrawable=60 | can_withdraw=false, withdraw_gap=40 |
| TestWithdrawEligibility_Zero | withdrawable=0 | can_withdraw=false, withdraw_gap=100 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.1] - FR26, FR27

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
