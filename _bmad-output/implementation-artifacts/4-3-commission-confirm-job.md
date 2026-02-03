# Story 4.3: 佣金自动确认定时任务

Status: ready-for-dev

## Story

As a 系统,
I want 定时确认超过确认期的待确认佣金,
so that 佣金可以提现。

## Acceptance Criteria

1. **AC1**: 每日凌晨执行定时任务
2. **AC2**: 查询 status=pending 且 created_at 超过 7 天的佣金记录
3. **AC3**: 关联订单未退款 → 状态更新为 confirmed，增加 withdrawable
4. **AC4**: 关联订单已退款 → 状态更新为 cancelled
5. **AC5**: 记录审计日志
6. **AC6**: 批量处理，每次最多 1000 条

## Dependencies

- **Depends On**: Story 2.3 (commission_record 表数据)
- **Modifies**: commission_record.status (pending → confirmed/cancelled)
- **Depended By**: Story 5.1 (提现资格依赖 confirmed 佣金)

## Tasks / Subtasks

- [ ] Task 1: 定时任务实现 (AC: #1, #2, #6)
  - [ ] 1.1 创建 `internal/service/commission_confirm_scheduler.go`
  - [ ] 1.2 实现 Start/Stop 方法
  - [ ] 1.3 cron: `0 2 * * *`（每日凌晨2点）
- [ ] Task 2: 确认逻辑 (AC: #3, #4, #5)
  - [ ] 2.1 批量查询待确认记录
  - [ ] 2.2 检查关联订单退款状态
  - [ ] 2.3 确认或取消佣金
  - [ ] 2.4 更新 withdrawable
- [ ] Task 3: Wire 注册 (AC: #1)
  - [ ] 3.1 注册 Scheduler，在 cleanup 中 Stop

## Dev Notes

### Scheduler 结构

```go
type CommissionConfirmScheduler struct {
    affiliateService *AffiliateService
    ticker          *time.Ticker
    done            chan struct{}
}

func (s *CommissionConfirmScheduler) Start() {
    // 每日凌晨2点执行
    go s.run()
}

func (s *CommissionConfirmScheduler) Stop() {
    close(s.done)
}
```

### 确认逻辑

```go
func (s *AffiliateService) ConfirmPendingCommissions(ctx context.Context) error {
    confirmDays := 7
    cutoff := time.Now().AddDate(0, 0, -confirmDays)

    records, err := s.commissionRepo.GetPendingBefore(ctx, cutoff, 1000)
    if err != nil {
        return err
    }

    for _, record := range records {
        if record.SourceOrderID != "" {
            // 检查订单是否退款
            refunded, _ := s.paymentService.IsOrderRefunded(ctx, record.SourceOrderID)
            if refunded {
                s.commissionRepo.UpdateStatus(ctx, record.ID, "cancelled")
                continue
            }
        }
        // 确认佣金
        now := time.Now()
        s.commissionRepo.Confirm(ctx, record.ID, &now)
        s.affiliateRepo.IncrementWithdrawable(ctx, record.UserID, record.Amount)
    }
    return nil
}
```

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.6] - 佣金确认流程
- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.3] - FR31, FR32

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestConfirmPending_Normal | pending 且超过7天 | status → confirmed, withdrawable 增加 |
| TestConfirmPending_NotYet | pending 但仅3天 | 不处理 |
| TestConfirmPending_Refunded | 关联订单已退款 | status → cancelled |
| TestConfirmPending_BatchLimit | 超过1000条 | 只处理前1000条 |
| TestConfirmPending_Empty | 无待确认记录 | 正常返回不报错 |
| TestConfirmPending_AuditLog | 确认/取消佣金 | 记录审计日志 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整确认流程 | 创建 pending 佣金 → 等待7天 → 定时任务确认 → withdrawable 更新 |
| 退款取消流程 | pending 佣金 + 退款标记 → 定时任务取消 → withdrawable 不变 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
