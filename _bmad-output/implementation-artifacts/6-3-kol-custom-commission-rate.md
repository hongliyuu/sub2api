# Story 6.3: KOL 专属佣金比例

Status: ready-for-dev

## Story

As a KOL 用户,
I want 使用我的专属佣金比例,
so that 获得更高收益。

## Acceptance Criteria

1. **AC1**: KOL 配置了专属佣金比例时，使用专属比例计算佣金（不受阶梯限制）
2. **AC2**: KOL 未配置专属比例时，使用阶梯佣金比例
3. **AC3**: 专属佣金比例上限不超过系统配置的 commission_rate_cap

## Dependencies

- **Depends On**: Story 6.1 (KOL 身份标识), Story 4.1 (阶梯佣金比例计算)
- **Modifies**: AffiliateService.GetCommissionRate（优先 KOL 比例 → 上限截断 → 回退阶梯）
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 佣金比例计算逻辑修改 (AC: #1, #2, #3)
  - [ ] 1.1 修改 `AffiliateService.GetCommissionRate` 方法，优先使用 KOL 专属比例
  - [ ] 1.2 添加佣金比例上限校验

## Dev Notes

修改 Story 4.1 中的 `GetCommissionRate` 方法：

```go
func (s *AffiliateService) GetCommissionRate(ctx context.Context, inviter *ent.UserAffiliate) (float64, error) {
    // KOL 优先使用专属佣金比例
    if inviter.IsKol && inviter.KolConfig != nil {
        if rate, ok := inviter.KolConfig["commission_rate"].(float64); ok && rate > 0 {
            cap := s.configService.GetFloat(ctx, "commission_rate_cap", 0.20)
            if rate > cap {
                rate = cap
            }
            return rate, nil
        }
    }
    // 普通用户使用阶梯比例
    return s.getTierCommissionRate(ctx, inviter.TierLevel)
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetCommissionRate_KolCustomRate | KOL rate=0.10 | 返回 0.10 |
| TestGetCommissionRate_KolRateExceedsCap | KOL rate=0.25, cap=0.20 | 截断为 0.20 |
| TestGetCommissionRate_KolNoRate | KOL 未配置比例 | 回退到阶梯比例 |
| TestGetCommissionRate_KolRateZero | KOL rate=0 | 回退到阶梯比例 |
| TestGetCommissionRate_NonKol | 普通用户 | 使用阶梯比例 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.3] - FR21
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.3.3] - 佣金计算流程

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
