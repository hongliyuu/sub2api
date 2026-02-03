# Story 4.1: 阶梯佣金比例计算 — 独立测试 + 配置化准备

Status: ready-for-dev

## Story

As a 邀请人,
I want 根据有效邀请数自动获得更高的佣金比例,
so that 激励我邀请更多用户。

> **注意**: `GetCommissionRate`、`TierRule`、`defaultTierRules` 核心代码已在 Story 1.1 中前置实现（因 Story 2.3 和 3.2 在 Sprint 2 就需要使用）。本 Story 聚焦于：(1) 确保阶梯规则独立可测试，(2) 为 Epic 7 配置化做准备。

## Acceptance Criteria

1. **AC1**: 有效邀请 0-10 人使用档位1比例（默认 5%）
2. **AC2**: 有效邀请 11-30 人使用档位2比例（默认 8%）
3. **AC3**: 有效邀请 31+ 人使用档位3比例（默认 12%）
4. **AC4**: KOL 用户优先使用专属比例（不受阶梯限制）
5. **AC5**: 阶梯规则可通过接口替换（为 Epic 7 配置化做准备），定义 `TierRuleProvider` 接口

## Dependencies

- **Depends On**: Story 1.1 (GetCommissionRate 核心实现 + TierRule 定义)
- **Depended By**: Story 4.2 (自动升级依赖阶梯规则), Story 6.3 (KOL 比例覆盖)

## Tasks / Subtasks

- [ ] Task 1: 验证阶梯比例计算的独立测试覆盖 (AC: #1, #2, #3, #4)
  - [ ] 1.1 确保 Story 1.1 的 `GetCommissionRate` 测试全部通过
  - [ ] 1.2 补充边界值测试（tier 边界 10→11, 30→31）
  - [ ] 1.3 补充 KOL 专属比例的边缘场景测试
- [ ] Task 2: 定义 TierRuleProvider 接口，为配置化做准备 (AC: #5)
  - [ ] 2.1 定义 `TierRuleProvider` 接口（`GetTierRules() []TierRule`）
  - [ ] 2.2 实现 `DefaultTierRuleProvider`（返回硬编码规则）
  - [ ] 2.3 修改 `GetCommissionRate` 使用 `TierRuleProvider` 而非直接读 `defaultTierRules`
  - [ ] 2.4 Wire 注入 `TierRuleProvider`（默认使用 `DefaultTierRuleProvider`）

## Dev Notes

### 实现代码

> 核心代码（`TierRule`、`defaultTierRules`、`GetCommissionRate`、`GetTierInfo`）已在 Story 1.1 中实现。本 Story 的增量工作：

```go
// internal/service/affiliate_tier.go

// TierRuleProvider 阶梯规则提供者接口（为 Epic 7 配置化做准备）
type TierRuleProvider interface {
    GetTierRules() []TierRule
}

// DefaultTierRuleProvider 默认硬编码规则提供者
type DefaultTierRuleProvider struct{}

func (p *DefaultTierRuleProvider) GetTierRules() []TierRule {
    return defaultTierRules
}

// 重构 GetCommissionRate 使用 TierRuleProvider
// Epic 7 实现时，注入 ConfigTierRuleProvider 即可替换规则来源
```

已在 Story 2.3 中预定义，此 Story 确保独立可测试。阶梯规则硬编码，Epic 7 配置化。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.1] - FR16

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetCommissionRate_Tier1_Zero | effective_count=0 | 返回 0.05 (5%) |
| TestGetCommissionRate_Tier1_Max | effective_count=10 | 返回 0.05 (5%) |
| TestGetCommissionRate_Tier2_Min | effective_count=11 | 返回 0.08 (8%) |
| TestGetCommissionRate_Tier2_Max | effective_count=30 | 返回 0.08 (8%) |
| TestGetCommissionRate_Tier3_Min | effective_count=31 | 返回 0.12 (12%) |
| TestGetCommissionRate_Tier3_Large | effective_count=1000 | 返回 0.12 (12%) |
| TestGetCommissionRate_KolOverride | is_kol=true, kol_rate=0.15 | 返回 0.15，忽略阶梯 |
| TestGetCommissionRate_KolNoRate | is_kol=true, 无 kol_rate | 回退到阶梯比例 |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
