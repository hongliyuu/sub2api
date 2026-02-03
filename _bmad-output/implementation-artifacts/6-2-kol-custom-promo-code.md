# Story 6.2: KOL 自定义推广码

Status: ready-for-dev

## Story

As a KOL 用户,
I want 使用我的专属推广码,
so that 建立个人品牌。

## Acceptance Criteria

1. **AC1**: KOL 配置了专属推广码后，返回专属推广码和对应链接 `{domain}/r/{PROMO_CODE}`
2. **AC2**: 使用专属推广码注册时，通过推广码查找对应 KOL 并建立邀请关系
3. **AC3**: 专属推广码支持字母数字组合，长度 3-20 位，全局唯一
4. **AC4**: Redis 缓存 `aff:code:{code}` 支持 KOL 推广码映射
5. **AC5**: KOL 更换推广码时，清除旧推广码的 Redis 缓存 `aff:code:{old_code}`，写入新推广码缓存 `aff:code:{new_code}`

## Dependencies

- **Depends On**: Story 6.1 (KOL 身份标识), Story 1.5 (注册绑定使用推广码查找邀请人)
- **Modifies**: AffiliateRepository.GetUserByCode（扩展查询逻辑支持 KOL 推广码）
- **Depended By**: Story 8.2 (管理员设置 KOL 推广码)

## Tasks / Subtasks

- [ ] Task 1: 推广码查询逻辑扩展 (AC: #1, #2, #4)
  - [ ] 1.1 修改 `AffiliateRepository.GetUserByReferralCode` 方法，先查 `referral_code`，再查 `kol_config.promo_code`
  - [ ] 1.2 Redis 缓存 `aff:code:{code}` 同时覆盖普通邀请码和 KOL 推广码
- [ ] Task 2: API 响应扩展 (AC: #1)
  - [ ] 2.1 `/api/v1/affiliate/info` 中，若 is_kol 且有 promo_code，返回专属链接
- [ ] Task 3: 推广码唯一性校验 (AC: #3)
  - [ ] 3.1 设置 KOL 推广码时校验全局唯一（不与任何 referral_code 或其他 kol promo_code 冲突）

## Dev Notes

推广码查询优先级：
1. 先查 `user_affiliate.referral_code`（精确匹配）
2. 未命中则查 `kol_config->>'promo_code'`（JSON 字段查询）
3. Redis 缓存统一使用 `aff:code:{code}` → `user_id` 映射

```go
func (r *AffiliateRepository) GetUserByCode(ctx context.Context, code string) (*ent.UserAffiliate, error) {
    // 先查普通邀请码
    ua, err := r.client(ctx).UserAffiliate.Query().
        Where(useraffiliate.ReferralCode(code)).
        Only(ctx)
    if err == nil {
        return ua, nil
    }
    // 再查 KOL 推广码（JSON 字段）
    return r.client(ctx).UserAffiliate.Query().
        Where(useraffiliate.IsKol(true)).
        Where(func(s *sql.Selector) {
            s.Where(sql.ExprP("kol_config->>'promo_code' = $1", code))
        }).
        Only(ctx)
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetUserByCode_NormalCode | 普通邀请码 | 通过 referral_code 查找用户 |
| TestGetUserByCode_KolPromoCode | KOL 推广码 | 通过 kol_config.promo_code 查找 KOL |
| TestGetUserByCode_NotFound | 不存在的码 | 返回 nil |
| TestGetUserByCode_CacheHit | Redis 缓存存在 | 直接返回缓存的 user_id |
| TestPromoCodeUnique | 重复推广码 | 返回唯一性冲突错误 |
| TestPromoCodeFormat | 非法格式（特殊字符） | 返回格式校验错误 |
| TestPromoCodeUpdate_CacheSwap | 旧码 OLDC → 新码 NEWC | 删除 aff:code:OLDC，写入 aff:code:NEWC |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| KOL 推广码注册流程 | 使用 KOL 码注册 → 查找 KOL → 建立邀请关系 |
| 推广码更换缓存一致性 | 更换推广码 → 旧码缓存清除 → 新码缓存生效 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.2] - FR3, FR20
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.1] - kol_config 结构

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
