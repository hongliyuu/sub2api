# Story 8.2: 设置 KOL

Status: ready-for-dev

## Story

As a 管理员,
I want 将用户设置为 KOL,
so that 开展 KOL 合作推广。

## Acceptance Criteria

1. **AC1**: 管理员可设置用户为 KOL，填写 KOL 配置（推广码、佣金比例、绑定类型、额外奖励）
2. **AC2**: 设置后 is_kol=true，kol_config 存储完整配置
3. **AC3**: 管理员可取消 KOL 身份（is_kol=false，保留 kol_config 便于恢复）
4. **AC4**: 记录审计日志 `kol_granted` / `kol_revoked`
5. **AC5**: 专属推广码全局唯一校验
6. **AC6**: 取消 KOL 后，已有 lifetime 绑定关系保持不变（不回溯修改已有邀请关系的 binding_type）
7. **AC7**: 取消 KOL 后，佣金比例回退到阶梯规则计算（不再使用 KOL 专属比例）

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表), Story 6.1 (KOL 身份标识)
- **Depended By**: 无

## Tasks / Subtasks

- [ ] Task 1: 后端 Admin API (AC: #1, #2, #3, #4, #5)
  - [ ] 1.1 `PUT /api/admin/affiliate/users/:id/kol`
  - [ ] 1.2 请求体包含 is_kol + kol_config
  - [ ] 1.3 推广码唯一性校验
  - [ ] 1.4 记录审计日志
- [ ] Task 2: 前端管理页面 (AC: #1, #3)
  - [ ] 2.1 在用户详情中添加 KOL 配置表单
  - [ ] 2.2 支持设置/取消 KOL 操作

## Dev Notes

### API

```
PUT /api/admin/affiliate/users/:id/kol
```

请求体：
```json
{
  "is_kol": true,
  "kol_config": {
    "promo_code": "DAXIN",
    "commission_rate": 0.10,
    "user_bonus": 2.00,
    "coupon_template_id": null,
    "default_binding_type": "lifetime"
  }
}
```

### 实现

```go
func (s *AffiliateService) UpdateKolStatus(ctx context.Context, userID int64, isKol bool, kolConfig map[string]interface{}, adminID int64) error {
    // 1. 推广码唯一性校验
    if isKol && kolConfig != nil {
        if promoCode, ok := kolConfig["promo_code"].(string); ok && promoCode != "" {
            exists, _ := s.affiliateRepo.PromoCodeExists(ctx, promoCode, userID)
            if exists {
                return errors.New("推广码已被占用")
            }
        }
    }

    // 2. 更新 KOL 状态
    err := s.affiliateRepo.UpdateKolStatus(ctx, userID, isKol, kolConfig)
    if err != nil {
        return err
    }

    // 3. 清除缓存
    s.cache.Delete(ctx, fmt.Sprintf("aff:user:%d", userID))
    if promoCode, ok := kolConfig["promo_code"].(string); ok {
        s.cache.Delete(ctx, fmt.Sprintf("aff:code:%s", promoCode))
    }

    // 4. 审计日志
    action := "kol_granted"
    if !isKol {
        action = "kol_revoked"
    }
    s.auditRepo.Create(ctx, &AffiliateAuditLog{
        UserID:     userID,
        Action:     action,
        OperatorID: adminID,
    })

    return nil
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestSetKol_Normal | 设置用户为 KOL | is_kol=true, kol_config 存储 |
| TestSetKol_PromoCodeUnique | 推广码已存在 | 返回唯一性冲突错误 |
| TestSetKol_PromoCodeSameUser | 用户自己的旧推广码 | 允许（排除自身） |
| TestRevokeKol_Normal | 取消 KOL | is_kol=false, kol_config 保留 |
| TestRevokeKol_LifetimePreserved | 取消 KOL | 已有 lifetime 关系不修改 binding_type |
| TestRevokeKol_CommissionRateFallback | 取消 KOL 后查佣金比例 | 使用阶梯规则而非 KOL 比例 |
| TestSetKol_AuditLog | 设置/取消 | 记录 kol_granted/kol_revoked 审计日志 |
| TestSetKol_CacheInvalidated | 设置 KOL 推广码 | 清除 aff:user:{id} 和 aff:code:{code} 缓存 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整 KOL 设置流程 | 设置 KOL → 推广码唯一检查 → 配置存储 → 缓存清除 → 审计日志 |
| 取消后回退验证 | 取消 KOL → lifetime 关系不变 → 新用户首充模式 → 佣金用阶梯比例 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.2] - FR54, FR55, FR56, FR57, FR58
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.2.2] - Admin KOL API

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
