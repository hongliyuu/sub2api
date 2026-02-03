# Story 6.1: KOL 身份标识

Status: ready-for-dev

## Story

As a KOL 用户,
I want 系统识别我的 KOL 身份,
so that 享受专属功能。

## Acceptance Criteria

1. **AC1**: 被管理员设置为 KOL 后 is_kol=true
2. **AC2**: kol_config JSONB 字段存储专属配置
3. **AC3**: 查询推广信息时返回 KOL 标识和专属配置

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表含 is_kol 和 kol_config 字段)
- **Depended By**: Story 6.2 (KOL 推广码), Story 6.3 (KOL 佣金比例), Story 6.4 (终身绑定), Story 6.5 (额外奖励), Story 6.6 (优惠券), Story 8.2 (设置 KOL)

## Tasks / Subtasks

- [ ] Task 1: 数据层支持 (AC: #1, #2)
  - [ ] 1.1 user_affiliate 表已有 is_kol 和 kol_config 字段（Story 1.1）
  - [ ] 1.2 确保 AffiliateRepository 有 UpdateKolStatus 方法
- [ ] Task 2: API 扩展 (AC: #3)
  - [ ] 2.1 在 `/api/v1/affiliate/info` 响应中包含 is_kol 和 kol_config

## Dev Notes

kol_config JSON 结构：
```json
{
  "promo_code": "KOLDAXIN",
  "commission_rate": 0.10,
  "user_bonus": 2.00,
  "coupon_template_id": null,
  "default_binding_type": "lifetime"
}
```

字段已在 Story 1.1 的 user_affiliate 表中定义。本 Story 确保读取和返回逻辑完整。

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetAffiliateInfo_KOL | is_kol=true, 有 kol_config | 返回 KOL 标识和专属配置 |
| TestGetAffiliateInfo_NonKOL | is_kol=false | 不返回 KOL 配置字段 |
| TestGetAffiliateInfo_KolNoConfig | is_kol=true, kol_config=nil | 返回 is_kol=true，配置为空 |

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.1] - FR19
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.1] - kol_config 结构

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
