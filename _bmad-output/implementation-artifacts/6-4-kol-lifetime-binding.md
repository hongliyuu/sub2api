# Story 6.4: KOL 终身绑定分成

Status: ready-for-dev

## Story

As a KOL 用户,
I want 被邀请人每次充值我都能获得佣金,
so that 持续获得收益。

## Acceptance Criteria

1. **AC1**: KOL 开启终身绑定后，被邀请人的 binding_type 设为 `lifetime`
2. **AC2**: binding_type=lifetime 时，被邀请人每次充值 KOL 均获得佣金
3. **AC3**: 普通用户（binding_type=first_charge）仅首充时产生佣金
4. **AC4**: 终身绑定佣金记录 source_type=recharge（区分首充 source_type=first_charge）
5. **AC5**: KOL 关闭终身绑定后，已有 lifetime 邀请关系保持不变（不回溯修改 binding_type）；新注册用户使用新的 binding_type

## Dependencies

- **Depends On**: Story 6.1 (KOL 身份标识), Story 1.5 (注册绑定邀请关系), Story 2.3 (佣金计算)
- **Modifies**: AffiliateService.CalculateCommission（扩展 lifetime 模式每次充值计算佣金）
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 绑定类型处理 (AC: #1)
  - [ ] 1.1 注册绑定邀请关系时，若邀请人是 KOL 且 kol_config.default_binding_type=lifetime，设置 binding_type=lifetime
- [ ] Task 2: 佣金计算逻辑扩展 (AC: #2, #3, #4)
  - [ ] 2.1 修改 `CalculateCommission` 方法，binding_type=lifetime 时每次充值都计算佣金
  - [ ] 2.2 lifetime 模式的佣金 source_type 标记为 `recharge`

## Dev Notes

修改 Story 2.3 的佣金计算逻辑：

```go
func (s *AffiliateService) CalculateCommission(ctx context.Context, order *OrderInfo) error {
    relation, err := s.GetRelationByInvitee(ctx, order.UserID)
    if err != nil || relation == nil {
        return nil // 无邀请关系，跳过
    }

    sourceType := "first_charge"
    if relation.BindingType == "lifetime" {
        sourceType = "recharge"
    } else {
        // first_charge 模式：检查是否已有首充佣金
        exists, _ := s.commissionRepo.ExistsByInviteeAndType(ctx, order.UserID, "first_charge")
        if exists {
            return nil // 首充模式已产生过佣金，跳过
        }
    }

    // 继续佣金计算...
}
```

`referral_relation.binding_type` 取值：
- `first_charge`：默认，仅首充分成
- `lifetime`：终身绑定，每次充值分成（KOL 专属）

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestCalculateCommission_Lifetime | binding_type=lifetime, 第2次充值 | 计算佣金，source_type=recharge |
| TestCalculateCommission_LifetimeMultiple | lifetime 连续3次充值 | 每次都产生佣金记录 |
| TestCalculateCommission_FirstChargeOnly | binding_type=first_charge, 第2次充值 | 跳过 |
| TestBindingType_KolLifetime | 邀请人是 KOL, default_binding_type=lifetime | 新关系 binding_type=lifetime |
| TestBindingType_KolFirstCharge | 邀请人是 KOL, default_binding_type=first_charge | 新关系 binding_type=first_charge |
| TestBindingType_NonKol | 邀请人非 KOL | binding_type=first_charge |
| TestLifetime_DisableNoRetroactive | KOL 关闭 lifetime 后 | 已有 lifetime 关系不变，新注册用户用 first_charge |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 终身绑定完整流程 | KOL 注册邀请 → lifetime 绑定 → 多次充值 → 每次产生佣金 |
| 关闭后不回溯 | 关闭 lifetime → 已有用户继续每次充值分成 → 新用户仅首充分成 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.4] - FR13, FR22
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.2] - referral_relation 表

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
