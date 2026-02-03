# Story 4.2: 阶梯自动升级

Status: ready-for-dev

## Story

As a 邀请人,
I want 有效邀请数达标后自动升级档位,
so that 后续佣金使用更高比例计算。

## Acceptance Criteria

1. **AC1**: effective_count 达到下一档位门槛时自动更新 tier_level
2. **AC2**: 记录审计日志 `tier_upgraded`
3. **AC3**: 前端检测到档位变化时显示升级成就弹窗

## Dependencies

- **Depends On**: Story 2.3 (effective_count 更新后调用升级检查), Story 4.1 (阶梯规则定义)
- **Modifies**: Story 2.3 (在 CalculateCommission 中 effective_count+1 后调用 UpdateTierLevel)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端自动升级逻辑 (AC: #1, #2)
  - [ ] 1.1 在 `affiliate_service.go` 实现 `UpdateTierLevel(ctx, userID)` 方法
  - [ ] 1.2 在 effective_count 更新后调用（Story 2.3 中）
  - [ ] 1.3 记录审计日志
- [ ] Task 2: 前端升级弹窗 (AC: #3)
  - [ ] 2.1 前端缓存上次 tier_level，检测变化显示弹窗

## Dev Notes

### 升级逻辑

```go
func (s *AffiliateService) UpdateTierLevel(ctx context.Context, userID int64) error {
    affiliate, err := s.affiliateRepo.GetByUserID(ctx, userID)
    if err != nil {
        return err
    }

    oldTier := affiliate.TierLevel
    newTier := 1
    for _, tier := range defaultTierRules {
        if affiliate.EffectiveCount >= tier.MinCount {
            newTier = tier.Level
        }
    }

    if newTier != oldTier {
        affiliate.TierLevel = newTier
        if err := s.affiliateRepo.UpdateTierLevel(ctx, userID, newTier); err != nil {
            return err
        }
        // 审计日志
        s.auditLog(ctx, userID, "tier_upgraded",
            map[string]int{"tier": oldTier},
            map[string]int{"tier": newTier})
    }
    return nil
}
```

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.4] - 阶梯升级流程
- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.2] - FR16(升级逻辑)

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestUpdateTierLevel_Upgrade_1to2 | effective_count=11, old_tier=1 | tier_level 更新为 2 |
| TestUpdateTierLevel_Upgrade_2to3 | effective_count=31, old_tier=2 | tier_level 更新为 3 |
| TestUpdateTierLevel_NoChange | effective_count=5, old_tier=1 | 不更新，不记录审计日志 |
| TestUpdateTierLevel_AlreadyMax | effective_count=50, old_tier=3 | 不更新 |
| TestUpdateTierLevel_AuditLog | 升级发生 | 记录 tier_upgraded 日志含 old/new tier |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整升级流程 | effective_count+1 → 触发 UpdateTierLevel → tier_level 更新 → 审计日志写入 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
