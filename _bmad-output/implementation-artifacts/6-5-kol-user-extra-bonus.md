# Story 6.5: KOL 用户专属奖励

Status: ready-for-dev

## Story

As a 通过 KOL 推广码注册的用户,
I want 获得额外的注册奖励,
so that 享受 KOL 专属福利。

## Acceptance Criteria

1. **AC1**: 通过 KOL 推广码注册时，被邀请人获得普通奖励 + KOL 额外奖励
2. **AC2**: KOL 未配置 user_bonus 时，只发放普通注册奖励
3. **AC3**: 额外奖励生成单独的 commission_record（source_type=kol_bonus）

## Dependencies

- **Depends On**: Story 6.1 (KOL 身份标识), Story 1.6 (注册奖励逻辑)
- **Modifies**: AffiliateService.GrantRegisterBonus（扩展 KOL 额外奖励发放）
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 注册奖励逻辑扩展 (AC: #1, #2, #3)
  - [ ] 1.1 修改 Story 1.6 的注册奖励逻辑，检查邀请人 KOL 配置
  - [ ] 1.2 若 kol_config.user_bonus > 0，额外发放奖励给被邀请人

## Dev Notes

在 Story 1.6 的注册奖励流程中增加 KOL 额外奖励：

```go
func (s *AffiliateService) GrantRegisterBonus(ctx context.Context, inviterID, inviteeID int64) error {
    return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
        // 1. 普通注册奖励（邀请人 + 被邀请人）
        inviterBonus := s.configService.GetFloat(txCtx, "register_bonus_inviter", 1.0)
        inviteeBonus := s.configService.GetFloat(txCtx, "register_bonus_invitee", 1.0)
        // ... 发放普通奖励 ...

        // 2. KOL 额外奖励
        inviter, _ := s.affiliateRepo.GetByUserID(txCtx, inviterID)
        if inviter.IsKol && inviter.KolConfig != nil {
            if extraBonus, ok := inviter.KolConfig["user_bonus"].(float64); ok && extraBonus > 0 {
                // 发放额外奖励给被邀请人
                s.userRepo.IncrementBalance(txCtx, inviteeID, extraBonus)
                s.commissionRepo.Create(txCtx, &CommissionRecord{
                    UserID:     inviteeID,
                    SourceType: "kol_bonus",
                    Amount:     extraBonus,
                    Status:     "confirmed",
                })
            }
        }
        return nil
    })
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGrantRegisterBonus_KolExtraBonus | KOL user_bonus=2.00 | 被邀请人获得普通奖励+额外2元 |
| TestGrantRegisterBonus_KolNoBonus | KOL user_bonus 未配置 | 只发放普通注册奖励 |
| TestGrantRegisterBonus_KolBonusZero | KOL user_bonus=0 | 只发放普通注册奖励 |
| TestGrantRegisterBonus_NonKol | 普通邀请人 | 只发放普通注册奖励 |
| TestGrantRegisterBonus_BonusRecord | KOL 额外奖励 | 创建 source_type=kol_bonus 的佣金记录 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.5] - FR24
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.1] - kol_config.user_bonus

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
