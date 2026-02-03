# Story 1.1: 用户分销信息初始化

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 系统,
I want 在用户注册时自动创建分销信息并生成唯一邀请码,
so that 每个用户都拥有专属的推广身份。

## Acceptance Criteria

1. **AC1**: 新用户完成注册流程后，系统自动在 `user_affiliate` 表中创建对应记录
2. **AC2**: 生成 6-8 位唯一邀请码（字母数字组合，排除易混淆字符 0/O/1/I/l）
3. **AC3**: 邀请码在数据库中有唯一索引约束，冲突时自动重新生成
4. **AC4**: 初始化字段值：tier_level=1, effective_count=0, total_earnings=0, withdrawable=0, is_kol=false
5. **AC5**: 创建分销记录的操作在同一事务中完成，确保与用户创建的原子性
6. **AC6**: 实现阶梯佣金比例计算方法 `GetCommissionRate`，根据 effective_count 返回对应档位的佣金比例（默认 3 档：0-10 人 5%，11-30 人 8%，31+ 人 12%），KOL 用户优先使用专属比例

## Dependencies

- **Depends On**: 无（基础设施，分销系统第一个故事）
- **Depended By**: Story 1.2 (推广信息API), Story 1.4 (邀请链接跳转), Story 1.5 (注册绑定), Story 2.1~2.4, Story 3.2 (战绩卡片阶梯数据), Story 4.1 (阶梯规则独立测试+配置化), Story 6.1, Story 8.1

## Tasks / Subtasks

- [ ] Task 1: 创建 Ent Schema 和数据库迁移 (AC: #1, #3)
  - [ ] 1.1 创建 `user_affiliate` Ent schema 文件
  - [ ] 1.2 创建 SQL 迁移文件，定义表结构和索引
  - [ ] 1.3 运行 `go generate ./ent` 生成 Ent 代码
- [ ] Task 2: 实现邀请码生成算法 (AC: #2, #3)
  - [ ] 2.1 在 `internal/service/affiliate_service.go` 中实现 `GenerateReferralCode()` 函数
  - [ ] 2.2 排除易混淆字符（0, O, 1, I, l）
  - [ ] 2.3 实现冲突检测和重试逻辑（最多 5 次重试）
- [ ] Task 3: 实现 Repository 层 (AC: #1, #5)
  - [ ] 3.1 创建 `internal/repository/affiliate_repo.go`
  - [ ] 3.2 实现 `Create()` 方法，支持事务
  - [ ] 3.3 实现 `GetByUserID()` 方法
  - [ ] 3.4 实现 `ExistsByCode()` 方法（用于邀请码冲突检测）
- [ ] Task 4: 实现 Service 层 (AC: #4, #5)
  - [ ] 4.1 创建 `internal/service/affiliate_service.go`
  - [ ] 4.2 实现 `CreateUserAffiliate()` 方法
  - [ ] 4.3 定义 `AffiliateRepository` 接口
- [ ] Task 5: 集成到用户注册流程 (AC: #5)
  - [ ] 5.1 修改 `auth_service.go` 的 `Register()` 方法
  - [ ] 5.2 在用户创建成功后调用 `AffiliateService.CreateUserAffiliate()`
  - [ ] 5.3 确保在同一事务中执行
- [ ] Task 6: 实现阶梯佣金比例计算 (AC: #6)
  - [ ] 6.1 定义 `TierRule` 结构体（Level, Name, MinCount, MaxCount, Rate）
  - [ ] 6.2 定义 `defaultTierRules` 硬编码阶梯规则（3 档：青铜 5%、白银 8%、黄金 12%），Epic 7 配置化
  - [ ] 6.3 在 `affiliate_service.go` 实现 `GetCommissionRate(inviter *UserAffiliate) float64`
  - [ ] 6.4 KOL 优先使用 `kol_config["commission_rate"]`，无则回退到阶梯规则

## Dev Notes

### 技术栈要求

- **ORM**: Ent (entgo.io/ent)
- **数据库**: PostgreSQL
- **迁移策略**: SQL 文件版本管理（非自动迁移）

### 数据库 Schema

```sql
-- 迁移文件: migrations/XXX_create_user_affiliate.sql

CREATE TABLE user_affiliate (
    user_id         BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    referral_code   VARCHAR(16) NOT NULL,
    tier_level      SMALLINT NOT NULL DEFAULT 1,
    effective_count INT NOT NULL DEFAULT 0,
    is_kol          BOOLEAN NOT NULL DEFAULT FALSE,
    kol_config      JSONB,
    total_earnings  DECIMAL(12,2) NOT NULL DEFAULT 0,
    withdrawable    DECIMAL(12,2) NOT NULL DEFAULT 0,
    version         INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 唯一索引：确保邀请码全局唯一
CREATE UNIQUE INDEX idx_user_affiliate_referral_code ON user_affiliate(referral_code);

-- 索引：支持按邀请码快速查找
CREATE INDEX idx_user_affiliate_tier_level ON user_affiliate(tier_level);
CREATE INDEX idx_user_affiliate_is_kol ON user_affiliate(is_kol) WHERE is_kol = TRUE;
```

### Ent Schema 定义

```go
// backend/ent/schema/user_affiliate.go
package schema

import (
    "github.com/Wei-Shaw/sub2api/ent/schema/mixins"
    "entgo.io/ent"
    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type UserAffiliate struct {
    ent.Schema
}

func (UserAffiliate) Annotations() []schema.Annotation {
    return []schema.Annotation{
        entsql.Annotation{Table: "user_affiliate"},
    }
}

func (UserAffiliate) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixins.TimeMixin{},
    }
}

func (UserAffiliate) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("user_id").
            Unique(),
        field.String("referral_code").
            MaxLen(16).
            NotEmpty(),
        field.Int("tier_level").
            Default(1),
        field.Int("effective_count").
            Default(0),
        field.Bool("is_kol").
            Default(false),
        field.JSON("kol_config", map[string]interface{}{}).
            Optional(),
        field.Float("total_earnings").
            SchemaType(map[string]string{dialect.Postgres: "decimal(12,2)"}).
            Default(0),
        field.Float("withdrawable").
            SchemaType(map[string]string{dialect.Postgres: "decimal(12,2)"}).
            Default(0),
        field.Int("version").
            Default(0),
    }
}

func (UserAffiliate) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("user", User.Type).
            Ref("affiliate").
            Field("user_id").
            Unique().
            Required(),
    }
}

func (UserAffiliate) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("referral_code").Unique(),
        index.Fields("tier_level"),
    }
}
```

### 邀请码生成算法

```go
// 字符集：排除易混淆字符 (0, O, 1, I, l)
const referralCodeCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func GenerateReferralCode(length int) string {
    // 使用 crypto/rand 生成安全随机数
    b := make([]byte, length)
    _, err := rand.Read(b)
    if err != nil {
        // fallback to time-based seed
        r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
        for i := range b {
            b[i] = referralCodeCharset[r.Intn(len(referralCodeCharset))]
        }
        return string(b)
    }
    for i := range b {
        b[i] = referralCodeCharset[int(b[i])%len(referralCodeCharset)]
    }
    return string(b)
}
```

### 错误定义

```go
// internal/service/affiliate_errors.go
var (
    ErrAffiliateNotFound     = infraerrors.NotFound("AFFILIATE_NOT_FOUND", "affiliate info not found")
    ErrAffiliateAlreadyExists = infraerrors.Conflict("AFFILIATE_ALREADY_EXISTS", "affiliate info already exists for this user")
    ErrReferralCodeConflict  = infraerrors.InternalServer("REFERRAL_CODE_CONFLICT", "failed to generate unique referral code")
)
```

### 服务接口定义

```go
// internal/service/affiliate_service.go

type AffiliateRepository interface {
    Create(ctx context.Context, affiliate *UserAffiliate) error
    GetByUserID(ctx context.Context, userID int64) (*UserAffiliate, error)
    ExistsByCode(ctx context.Context, code string) (bool, error)
}

type AffiliateService struct {
    affiliateRepo AffiliateRepository
}

func NewAffiliateService(repo AffiliateRepository) *AffiliateService {
    return &AffiliateService{affiliateRepo: repo}
}

// CreateUserAffiliate 创建用户分销信息
// 应在用户注册成功后的同一事务中调用
func (s *AffiliateService) CreateUserAffiliate(ctx context.Context, userID int64) (*UserAffiliate, error) {
    // 1. 生成唯一邀请码（最多重试 5 次）
    var code string
    for i := 0; i < 5; i++ {
        code = GenerateReferralCode(8) // 使用 8 位
        exists, err := s.affiliateRepo.ExistsByCode(ctx, code)
        if err != nil {
            return nil, err
        }
        if !exists {
            break
        }
        if i == 4 {
            return nil, ErrReferralCodeConflict
        }
    }

    // 2. 创建分销记录
    affiliate := &UserAffiliate{
        UserID:         userID,
        ReferralCode:   code,
        TierLevel:      1,
        EffectiveCount: 0,
        IsKol:          false,
        TotalEarnings:  0,
        Withdrawable:   0,
        Version:        0,
    }

    if err := s.affiliateRepo.Create(ctx, affiliate); err != nil {
        return nil, err
    }

    return affiliate, nil
}
```

### 阶梯佣金比例计算（前置实现）

> **设计决策**: `GetCommissionRate` 和阶梯规则在 Story 1.1 中实现，因为 Story 2.3（首充佣金, Sprint 2）和 Story 3.2（战绩卡片, Sprint 2）都依赖此方法。Story 4.1 将在 Sprint 3 中补充独立测试用例并为 Epic 7 配置化做准备。

```go
// internal/service/affiliate_tier.go（或直接写在 affiliate_service.go 中）

// TierRule 定义阶梯佣金规则
type TierRule struct {
    Level    int
    Name     string
    MinCount int
    MaxCount int     // 0 表示无上限
    Rate     float64
}

// defaultTierRules 默认阶梯规则（硬编码，Epic 7 通过 affiliate_config 表配置化）
var defaultTierRules = []TierRule{
    {Level: 1, Name: "青铜", MinCount: 0, MaxCount: 10, Rate: 0.05},
    {Level: 2, Name: "白银", MinCount: 11, MaxCount: 30, Rate: 0.08},
    {Level: 3, Name: "黄金", MinCount: 31, MaxCount: 0, Rate: 0.12},
}

// GetCommissionRate 获取用户的佣金比例
// KOL 优先使用专属比例，普通用户按阶梯规则匹配
func (s *AffiliateService) GetCommissionRate(inviter *UserAffiliate) float64 {
    // KOL 优先使用专属比例
    if inviter.IsKol {
        if rate, ok := inviter.KolConfig["commission_rate"].(float64); ok && rate > 0 {
            return rate
        }
    }
    // 阶梯规则匹配
    for _, tier := range defaultTierRules {
        if inviter.EffectiveCount >= tier.MinCount &&
           (tier.MaxCount == 0 || inviter.EffectiveCount <= tier.MaxCount) {
            return tier.Rate
        }
    }
    return 0.05 // 默认 5%
}

// GetTierInfo 获取用户当前阶梯信息（供 API 返回）
func (s *AffiliateService) GetTierInfo(inviter *UserAffiliate) (tierName string, rate float64, nextThreshold int) {
    for i, tier := range defaultTierRules {
        if inviter.EffectiveCount >= tier.MinCount &&
           (tier.MaxCount == 0 || inviter.EffectiveCount <= tier.MaxCount) {
            tierName = tier.Name
            rate = tier.Rate
            if i+1 < len(defaultTierRules) {
                nextThreshold = defaultTierRules[i+1].MinCount
            } else {
                nextThreshold = 0 // 已达最高档位
            }
            return
        }
    }
    return defaultTierRules[0].Name, defaultTierRules[0].Rate, defaultTierRules[1].MinCount
}
```

### 集成到注册流程

```go
// internal/service/auth_service.go
// 修改 Register 方法

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*User, error) {
    // ... 现有的用户创建逻辑 ...

    // 在事务中创建用户后，立即创建分销信息
    tx, err := s.userRepo.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    user, err := s.userRepo.Create(ctx, userInput)
    if err != nil {
        return nil, err
    }

    // 创建分销信息
    _, err = s.affiliateService.CreateUserAffiliate(ctx, user.ID)
    if err != nil {
        return nil, err
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return user, nil
}
```

### Project Structure Notes

**文件位置遵循现有项目结构：**

```
backend/
├── ent/schema/
│   └── user_affiliate.go          # Ent Schema 定义
├── migrations/
│   └── XXX_create_user_affiliate.sql  # 数据库迁移
├── internal/
│   ├── repository/
│   │   └── affiliate_repo.go      # Repository 实现
│   └── service/
│       ├── affiliate_service.go   # Service 实现
│       └── affiliate_errors.go    # 错误定义
```

**命名约定遵循项目现有模式：**

- Schema 文件：`user_affiliate.go`（snake_case 表名）
- Repository：`affiliate_repo.go`，类型 `AffiliateRepository`
- Service：`affiliate_service.go`，类型 `AffiliateService`
- 错误变量：`Err` 前缀，如 `ErrAffiliateNotFound`

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#数据库设计] - user_affiliate 表完整 Schema
- [Source: backend/ent/schema/user.go] - 现有 User Schema 模式参考
- [Source: backend/ent/schema/mixins/time.go] - TimeMixin 使用方式
- [Source: backend/internal/service/auth_service.go] - 用户注册流程
- [Source: _bmad-output/planning-artifacts/epics.md#Story-1.1] - Story 需求定义

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGenerateReferralCode_Length | 生成邀请码 | 长度为 8 位 |
| TestGenerateReferralCode_Charset | 生成邀请码 | 不含 0/O/1/I/l |
| TestGenerateReferralCode_Unique | 生成多次 | 每次不同 |
| TestCreateUserAffiliate_Normal | 新用户 | 正确创建记录，tier=1, count=0 |
| TestCreateUserAffiliate_CodeConflict | 邀请码冲突 | 重试最多5次 |
| TestCreateUserAffiliate_MaxRetries | 连续5次冲突 | 返回 ErrReferralCodeConflict |
| TestCreateUserAffiliate_AlreadyExists | 用户已有记录 | 返回 ErrAffiliateAlreadyExists |
| TestGetCommissionRate_Tier1_Zero | effective_count=0 | 返回 0.05 (5%) |
| TestGetCommissionRate_Tier1_Max | effective_count=10 | 返回 0.05 (5%) |
| TestGetCommissionRate_Tier2_Min | effective_count=11 | 返回 0.08 (8%) |
| TestGetCommissionRate_Tier2_Max | effective_count=30 | 返回 0.08 (8%) |
| TestGetCommissionRate_Tier3_Min | effective_count=31 | 返回 0.12 (12%) |
| TestGetCommissionRate_Tier3_Large | effective_count=1000 | 返回 0.12 (12%) |
| TestGetCommissionRate_KolOverride | is_kol=true, kol_rate=0.15 | 返回 0.15，忽略阶梯 |
| TestGetCommissionRate_KolNoRate | is_kol=true, 无 kol_rate | 回退到阶梯比例 |
| TestGetTierInfo_Tier1 | effective_count=5 | 返回 "青铜", 0.05, nextThreshold=11 |
| TestGetTierInfo_MaxTier | effective_count=50 | 返回 "黄金", 0.12, nextThreshold=0 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整注册流程 | 用户创建 + 分销信息创建在同一事务中 |
| 邀请码唯一索引 | 插入重复邀请码触发 UNIQUE 约束 |
| 事务回滚 | 分销信息创建失败 → 用户创建也回滚 |

### 注意事项

1. **邀请码生成**：使用 `crypto/rand` 而非 `math/rand` 以确保安全性
2. **事务处理**：分销信息创建必须在用户创建的同一事务中
3. **幂等性**：如果用户已有分销信息，不应重复创建
4. **性能**：邀请码查询应使用索引，避免全表扫描

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
