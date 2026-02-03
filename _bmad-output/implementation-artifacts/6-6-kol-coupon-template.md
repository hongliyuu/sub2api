# Story 6.6: KOL 优惠券关联（预留）

Status: ready-for-dev

## Story

As a 通过 KOL 推广码注册的用户,
I want 获得首充优惠券,
so that 享受充值优惠。

## Acceptance Criteria

1. **AC1**: KOL 关联了优惠券模板后，通过 KOL 推广码注册的用户自动获得优惠券
2. **AC2**: kol_config.coupon_template_id 存储关联的优惠券模板 ID
3. **AC3**: 若系统无优惠券模块，本 Story 标记为预留，仅实现数据字段

## Dependencies

- **Depends On**: Story 6.1 (KOL 身份标识)
- **Depended By**: 无（预留功能，待优惠券模块就绪）

## Tasks / Subtasks

- [ ] Task 1: 数据字段支持 (AC: #2, #3)
  - [ ] 1.1 确认 kol_config.coupon_template_id 字段已在 Story 1.1 中定义
  - [ ] 1.2 管理员设置 KOL 时可填写 coupon_template_id（Story 8.2）
- [ ] Task 2: 优惠券发放集成（预留）(AC: #1)
  - [ ] 2.1 在注册绑定流程中检查 coupon_template_id
  - [ ] 2.2 调用优惠券系统发放优惠券（待优惠券模块就绪后实现）

## Dev Notes

当前系统未实现优惠券模块，本 Story 仅做数据层预留：

1. `kol_config.coupon_template_id` 字段已在 kol_config JSON 结构中定义
2. 管理员可通过 Story 8.2 的 KOL 配置表单填写该值
3. 注册流程中预留检查逻辑，暂时跳过实际发放

```go
// 预留代码结构
if inviter.IsKol && inviter.KolConfig != nil {
    if templateID, ok := inviter.KolConfig["coupon_template_id"].(float64); ok && templateID > 0 {
        // TODO: 优惠券系统就绪后集成
        // s.couponService.GrantCoupon(ctx, inviteeID, int64(templateID))
        log.Info("coupon template configured but coupon system not available", "template_id", templateID)
    }
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestCouponCheck_HasTemplate | coupon_template_id=123 | 记录占位日志，不报错 |
| TestCouponCheck_NoTemplate | coupon_template_id=null | 跳过，不报错 |
| TestCouponCheck_NoKol | 普通邀请人 | 跳过 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.6] - FR23, FR25
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.1] - kol_config.coupon_template_id

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
