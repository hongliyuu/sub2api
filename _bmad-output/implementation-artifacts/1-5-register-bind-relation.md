# Story 1.5: 注册时绑定邀请关系

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 新用户,
I want 通过邀请链接注册后自动绑定邀请关系,
so that 邀请人和我都能获得奖励。

## Acceptance Criteria

1. **AC1**: 注册 API 请求中携带 `referral_code` 字段时，系统在用户创建后创建 `referral_relation` 记录
2. **AC2**: `referral_relation` 记录包含 `inviter_id`（邀请码对应用户）、`invitee_id`（新注册用户）、`referral_code`、`binding_type=first_charge`、`invitee_status=registered`
3. **AC3**: `invitee_id` 有唯一约束，每个用户只能被邀请一次；重复绑定时静默处理，不报错
4. **AC4**: 邀请码对应的用户不存在时，忽略邀请关系绑定，正常完成注册
5. **AC5**: 自邀自充检查：`inviter_id != invitee_id`，自己邀请自己时忽略绑定
6. **AC6**: KOL 用户的被邀请人 `binding_type` 由 `DetermineBindingType(inviter)` 决定（KOL 可设为 `lifetime`）
7. **AC7**: 邀请关系绑定失败不影响注册本身，失败时记录 warning 日志
8. **AC8**: 记录审计日志 `relation_created`

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表和 ExistsByCode 方法), Story 1.4 (前端传递 referral_code)
- **Modifies**: AuthService.RegisterWithVerification（引入 RegisterOptions DTO，添加 ReferralCode 字段）
- **Depended By**: Story 1.6 (注册奖励), Story 2.1 (查询邀请关系), Story 2.3 (首充佣金计算), Story 6.4 (KOL 终身绑定)

## Tasks / Subtasks

- [ ] Task 1: 创建 referral_relation Ent Schema (AC: #1, #2, #3)
  - [ ] 1.1 创建 `backend/ent/schema/referral_relation.go`
  - [ ] 1.2 定义字段：id, inviter_id, invitee_id, referral_code, binding_type, invitee_status, first_charge_at
  - [ ] 1.3 定义 invitee_id UNIQUE 索引
  - [ ] 1.4 定义 inviter_id、invitee_status 复合索引
  - [ ] 1.5 在 `ent/schema/user.go` 添加反向 edge 定义
  - [ ] 1.6 运行 `go generate ./ent` 生成代码
- [ ] Task 2: 创建 SQL 迁移文件 (AC: #1, #3)
  - [ ] 2.1 创建 `migrations/XXX_create_referral_relation.sql`
  - [ ] 2.2 包含表结构、索引、外键约束
- [ ] Task 3: 实现 Repository 层 (AC: #1, #2, #3)
  - [ ] 3.1 创建 `internal/repository/referral_relation_repo.go`
  - [ ] 3.2 实现 `Create(ctx, relation)` 方法，支持事务（使用 `clientFromContext`）
  - [ ] 3.3 实现 `GetByInviteeID(ctx, inviteeID)` 方法
  - [ ] 3.4 实现 `ExistsByInviteeID(ctx, inviteeID)` 方法
- [ ] Task 4: 扩展 AffiliateRepository (AC: #4)
  - [ ] 4.1 在 `affiliate_repo.go` 添加 `GetByCode(ctx, code)` 方法
  - [ ] 4.2 更新 `AffiliateRepository` 接口定义
- [ ] Task 5: 实现 Service 层 (AC: #1, #2, #4, #5, #6, #7, #8)
  - [ ] 5.1 在 `affiliate_service.go` 中定义 `ReferralRelationRepository` 接口
  - [ ] 5.2 实现 `BindReferralRelation(ctx, inviteeID, referralCode)` 方法
  - [ ] 5.3 实现 `DetermineBindingType(inviter)` 辅助方法
  - [ ] 5.4 自邀检测逻辑
  - [ ] 5.5 审计日志记录
- [ ] Task 6: 集成到注册流程 (AC: #1, #7)
  - [ ] 6.1 修改 `AuthService` 结构体，注入 `AffiliateService`
  - [ ] 6.2 定义 `RegisterOptions` DTO 结构体（封装 Email, Password, VerifyCode, PromoCode, ReferralCode 等可选参数），替代逐个添加参数的方式
  - [ ] 6.3 修改 `RegisterWithVerification` 方法签名为 `RegisterWithVerification(ctx, opts RegisterOptions)`
  - [ ] 6.4 在用户创建成功后调用 `AffiliateService.BindReferralRelation()`
  - [ ] 6.5 失败时仅记录 warning 日志，不影响注册返回
  - [ ] 6.6 同步修改 `Register` 方法和其他调用方（OAuth 回调等）
- [ ] Task 7: 修改注册 Handler 和 DTO (AC: #1)
  - [ ] 7.1 在注册请求 DTO 中添加 `referral_code` 字段
  - [ ] 7.2 Handler 中将 `referral_code` 传递给 Service
- [ ] Task 8: Wire 注册 (AC: #1)
  - [ ] 8.1 在 `repository/wire.go` 注册 `NewReferralRelationRepository`
  - [ ] 8.2 更新 `AffiliateService` 构造函数，注入 `ReferralRelationRepository`
  - [ ] 8.3 更新 `AuthService` 构造函数，注入 `AffiliateService`
  - [ ] 8.4 运行 `go generate ./cmd/server` 重新生成 Wire 代码

## Dev Notes

### 前置依赖

- Story 1.1：`user_affiliate` 表和 `AffiliateService` 已存在，`ExistsByCode` 方法可用
- Story 1.4：前端已能在注册请求中传递 `referral_code`

### 数据库 Schema

```sql
-- 迁移文件: migrations/XXX_create_referral_relation.sql

CREATE TABLE referral_relation (
    id              BIGSERIAL PRIMARY KEY,
    inviter_id      BIGINT NOT NULL,
    invitee_id      BIGINT NOT NULL,
    referral_code   VARCHAR(16) NOT NULL,
    binding_type    VARCHAR(20) NOT NULL DEFAULT 'first_charge',
    invitee_status  VARCHAR(20) NOT NULL DEFAULT 'registered',
    first_charge_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (inviter_id) REFERENCES users(id),
    FOREIGN KEY (invitee_id) REFERENCES users(id)
);

-- 唯一约束：每个用户只能被邀请一次
CREATE UNIQUE INDEX idx_referral_relation_invitee ON referral_relation(invitee_id);

-- 索引：按邀请人查找其邀请列表
CREATE INDEX idx_referral_relation_inviter ON referral_relation(inviter_id);

-- 复合索引：按邀请人和被邀请人状态查找（用于统计有效邀请数）
CREATE INDEX idx_referral_relation_inviter_status ON referral_relation(inviter_id, invitee_status);
```

### Ent Schema 定义

```go
// backend/ent/schema/referral_relation.go
package schema

import (
    "time"

    "github.com/Wei-Shaw/sub2api/ent/schema/mixins"
    "entgo.io/ent"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type ReferralRelation struct {
    ent.Schema
}

func (ReferralRelation) Annotations() []schema.Annotation {
    return []schema.Annotation{
        entsql.Annotation{Table: "referral_relation"},
    }
}

func (ReferralRelation) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("inviter_id").
            Comment("邀请人用户 ID"),
        field.Int64("invitee_id").
            Comment("被邀请人用户 ID"),
        field.String("referral_code").
            MaxLen(16).
            NotEmpty().
            Comment("使用的邀请码"),
        field.String("binding_type").
            Default("first_charge").
            Comment("绑定类型: first_charge | lifetime"),
        field.String("invitee_status").
            Default("registered").
            Comment("被邀请人状态: registered | first_charged | qualified"),
        field.Time("first_charge_at").
            Optional().
            Nillable().
            Comment("被邀请人首充时间"),
        field.Time("created_at").
            Default(time.Now).
            Immutable(),
    }
}

func (ReferralRelation) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("inviter", User.Type).
            Ref("referral_invites").
            Field("inviter_id").
            Unique().
            Required(),
        edge.From("invitee", User.Type).
            Ref("referral_invited_by").
            Field("invitee_id").
            Unique().
            Required(),
    }
}

func (ReferralRelation) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("invitee_id").Unique(),
        index.Fields("inviter_id"),
        index.Fields("inviter_id", "invitee_status"),
    }
}
```

**注意**：需在 `ent/schema/user.go` 的 `Edges()` 中添加对应的反向 edge：

```go
// backend/ent/schema/user.go - Edges() 方法中添加
edge.To("referral_invites", ReferralRelation.Type),      // 用户作为邀请人的邀请列表
edge.To("referral_invited_by", ReferralRelation.Type),    // 用户作为被邀请人的关系（最多一条）
```

如果 `user.go` 的 Edge 命名与 `ReferralRelation` 的 `edge.From` 不匹配会导致 `go generate ./ent` 失败。务必保证 Ref 名称一致。

### Repository 实现

```go
// internal/repository/referral_relation_repo.go
package repository

import (
    "context"
    "fmt"

    "github.com/Wei-Shaw/sub2api/ent"
    "github.com/Wei-Shaw/sub2api/ent/referralrelation"
)

type ReferralRelationRepository struct {
    client *ent.Client
}

func NewReferralRelationRepository(client *ent.Client) *ReferralRelationRepository {
    return &ReferralRelationRepository{client: client}
}

// Create 创建邀请关系记录，支持事务上下文
func (r *ReferralRelationRepository) Create(ctx context.Context, relation *ReferralRelation) (*ent.ReferralRelation, error) {
    client := clientFromContext(ctx, r.client)
    return client.ReferralRelation.Create().
        SetInviterID(relation.InviterID).
        SetInviteeID(relation.InviteeID).
        SetReferralCode(relation.ReferralCode).
        SetBindingType(relation.BindingType).
        SetInviteeStatus(relation.InviteeStatus).
        Save(ctx)
}

// GetByInviteeID 根据被邀请人 ID 查找邀请关系
func (r *ReferralRelationRepository) GetByInviteeID(ctx context.Context, inviteeID int64) (*ent.ReferralRelation, error) {
    return clientFromContext(ctx, r.client).ReferralRelation.
        Query().
        Where(referralrelation.InviteeID(inviteeID)).
        Only(ctx)
}

// ExistsByInviteeID 检查被邀请人是否已有邀请关系
func (r *ReferralRelationRepository) ExistsByInviteeID(ctx context.Context, inviteeID int64) (bool, error) {
    return clientFromContext(ctx, r.client).ReferralRelation.
        Query().
        Where(referralrelation.InviteeID(inviteeID)).
        Exist(ctx)
}
```

### AffiliateRepository 扩展 - GetByCode

在 `internal/repository/affiliate_repo.go` 中新增方法：

```go
// GetByCode 通过邀请码查找用户分销信息
func (r *affiliateRepository) GetByCode(ctx context.Context, code string) (*ent.UserAffiliate, error) {
    return clientFromContext(ctx, r.client).UserAffiliate.
        Query().
        Where(useraffiliate.ReferralCode(code)).
        Only(ctx)
}
```

同时更新 `AffiliateRepository` 接口（在 `affiliate_service.go` 中定义）：

```go
type AffiliateRepository interface {
    Create(ctx context.Context, affiliate *UserAffiliate) error
    GetByUserID(ctx context.Context, userID int64) (*UserAffiliate, error)
    ExistsByCode(ctx context.Context, code string) (bool, error)
    GetByCode(ctx context.Context, code string) (*ent.UserAffiliate, error)  // 新增
}
```

### Service 层实现

```go
// internal/service/affiliate_service.go

// ReferralRelationRepository 接口定义
type ReferralRelationRepository interface {
    Create(ctx context.Context, relation *ReferralRelation) (*ent.ReferralRelation, error)
    GetByInviteeID(ctx context.Context, inviteeID int64) (*ent.ReferralRelation, error)
    ExistsByInviteeID(ctx context.Context, inviteeID int64) (bool, error)
}

// 更新 AffiliateService 构造函数
type AffiliateService struct {
    affiliateRepo AffiliateRepository
    relationRepo  ReferralRelationRepository
    cfg           *config.Config
}

func NewAffiliateService(
    affiliateRepo AffiliateRepository,
    relationRepo ReferralRelationRepository,
    cfg *config.Config,
) *AffiliateService {
    return &AffiliateService{
        affiliateRepo: affiliateRepo,
        relationRepo:  relationRepo,
        cfg:           cfg,
    }
}

// BindReferralRelation 绑定邀请关系
// 在注册流程中调用，失败不影响注册本身
func (s *AffiliateService) BindReferralRelation(ctx context.Context, inviteeID int64, referralCode string) error {
    if referralCode == "" {
        return nil // 无邀请码，跳过
    }

    // 1. 通过邀请码查找邀请人
    inviterAffiliate, err := s.affiliateRepo.GetByCode(ctx, referralCode)
    if err != nil {
        // 邀请码不存在或查询失败，静默返回
        log.Printf("[Affiliate] Referral code lookup failed: code=%s, err=%v", referralCode, err)
        return nil
    }

    inviterID := inviterAffiliate.UserID

    // 2. 自邀自充检查
    if inviterID == inviteeID {
        log.Printf("[Affiliate] Self-referral detected, skipping: user_id=%d", inviteeID)
        return nil
    }

    // 3. 检查是否已有邀请关系（幂等保护）
    exists, err := s.relationRepo.ExistsByInviteeID(ctx, inviteeID)
    if err != nil {
        return fmt.Errorf("check existing relation: %w", err)
    }
    if exists {
        log.Printf("[Affiliate] Invitee %d already has referral relation, skipping", inviteeID)
        return nil
    }

    // 4. 确定绑定类型
    bindingType := s.DetermineBindingType(inviterAffiliate)

    // 5. 创建邀请关系
    relation := &ReferralRelation{
        InviterID:     inviterID,
        InviteeID:     inviteeID,
        ReferralCode:  referralCode,
        BindingType:   bindingType,
        InviteeStatus: "registered",
    }

    created, err := s.relationRepo.Create(ctx, relation)
    if err != nil {
        // UNIQUE 约束冲突时也静默处理
        if isUniqueConstraintError(err) {
            log.Printf("[Affiliate] Duplicate relation for invitee %d, skipping", inviteeID)
            return nil
        }
        return fmt.Errorf("create referral relation: %w", err)
    }

    log.Printf("[Affiliate] Referral relation created: id=%d, inviter=%d, invitee=%d, code=%s, type=%s",
        created.ID, inviterID, inviteeID, referralCode, bindingType)

    // 6. Story 1.6 在此处调用 GrantRegisterBonus
    // if err := s.GrantRegisterBonus(ctx, inviterID, inviteeID, created.ID); err != nil {
    //     log.Printf("[Affiliate] Warning: grant register bonus failed: %v", err)
    // }

    return nil
}

// DetermineBindingType 根据邀请人身份确定绑定类型
// KOL 用户可能设置 lifetime 绑定类型
func (s *AffiliateService) DetermineBindingType(inviter *ent.UserAffiliate) string {
    if inviter.IsKol && inviter.KolConfig != nil {
        if defaultType, ok := inviter.KolConfig["default_binding_type"].(string); ok {
            if defaultType == "lifetime" {
                return "lifetime"
            }
        }
    }
    return "first_charge" // 普通用户默认首充绑定
}
```

### 错误定义

```go
// internal/service/affiliate_errors.go 新增

var (
    ErrRelationAlreadyExists = infraerrors.Conflict("RELATION_ALREADY_EXISTS", "referral relation already exists for this user")
    ErrSelfReferral          = infraerrors.BadRequest("SELF_REFERRAL", "cannot refer yourself")
    ErrInvalidReferralCode   = infraerrors.NotFound("INVALID_REFERRAL_CODE", "referral code not found")
)
```

### 集成到注册流程

```go
// internal/service/auth_service.go

// RegisterOptions 注册参数 DTO（避免 RegisterWithVerification 签名频繁变更）
// 后续 Story（如 1.7 添加 ClientIP）只需在此结构体中增加字段，无需修改方法签名
type RegisterOptions struct {
    Email        string
    Password     string
    VerifyCode   string
    PromoCode    string
    ReferralCode string // Story 1.5 新增
    // ClientIP  string // Story 1.7 将在此添加
}

// AuthService 结构体添加 affiliateService
type AuthService struct {
    userRepo          UserRepository
    cfg               *config.Config
    settingService    *SettingService
    emailService      *EmailService
    turnstileService  *TurnstileService
    emailQueueService *EmailQueueService
    promoService      *PromoService
    affiliateService  *AffiliateService  // 新增
}

// NewAuthService 添加 affiliateService 参数
func NewAuthService(
    userRepo UserRepository,
    cfg *config.Config,
    settingService *SettingService,
    emailService *EmailService,
    turnstileService *TurnstileService,
    emailQueueService *EmailQueueService,
    promoService *PromoService,
    affiliateService *AffiliateService,  // 新增
) *AuthService {
    return &AuthService{
        // ... 现有字段 ...
        affiliateService: affiliateService,
    }
}

// RegisterWithVerification 修改签名为 Options 模式
// 原签名: RegisterWithVerification(ctx, email, password, verifyCode, promoCode string)
// 新签名: RegisterWithVerification(ctx context.Context, opts RegisterOptions)
func (s *AuthService) RegisterWithVerification(
    ctx context.Context,
    opts RegisterOptions,
) (string, *User, error) {
    // ... 现有注册逻辑（邮箱检查、密码哈希、用户创建，使用 opts.Email, opts.Password 等）...

    // 用户创建成功后，绑定邀请关系
    if opts.ReferralCode != "" && s.affiliateService != nil {
        if err := s.affiliateService.BindReferralRelation(ctx, user.ID, opts.ReferralCode); err != nil {
            // 绑定失败不影响注册，仅记录 warning 日志
            log.Printf("[Auth] Warning: failed to bind referral relation for user %d with code %s: %v",
                user.ID, opts.ReferralCode, err)
        }
    }

    // 应用优惠码（现有逻辑）...
    // 生成 token（现有逻辑）...
}

// Register 方法也需要同步修改
func (s *AuthService) Register(ctx context.Context, email, password string) (string, *User, error) {
    return s.RegisterWithVerification(ctx, RegisterOptions{
        Email:    email,
        Password: password,
    })
}
```

### 注册 Handler/DTO 修改

```go
// 查找项目中注册请求的 DTO 定义，添加 referral_code 字段
// 通常在 internal/handler/dto/ 或 handler 文件内部

type RegisterRequest struct {
    Email        string `json:"email" binding:"required,email"`
    Password     string `json:"password" binding:"required,min=6"`
    VerifyCode   string `json:"verify_code"`
    PromoCode    string `json:"promo_code"`
    ReferralCode string `json:"referral_code"`  // 新增
}
```

Handler 中传递 `referralCode`：

```go
// 在注册 Handler 中，构造 RegisterOptions 传递给 RegisterWithVerification
token, user, err := h.authService.RegisterWithVerification(
    c.Request.Context(),
    service.RegisterOptions{
        Email:        req.Email,
        Password:     req.Password,
        VerifyCode:   req.VerifyCode,
        PromoCode:    req.PromoCode,
        ReferralCode: req.ReferralCode,
    },
)
```

### Wire 注册更新

```go
// service/wire.go 更新 AffiliateService 的 provider
// NewAffiliateService 现在需要 ReferralRelationRepository 参数

// repository/wire.go 添加
wire.Struct(new(ReferralRelationRepository), "*"),
// 或
NewReferralRelationRepository,

// AuthService 的 wire provider 现在需要 *AffiliateService 参数
```

### Project Structure Notes

**新增文件：**
```
backend/
├── ent/schema/
│   └── referral_relation.go           # Ent Schema 定义
├── migrations/
│   └── XXX_create_referral_relation.sql  # 数据库迁移
├── internal/
│   └── repository/
│       └── referral_relation_repo.go  # Repository 实现
```

**修改文件：**
```
backend/
├── ent/schema/
│   └── user.go                        # 添加 referral_invites/referral_invited_by 反向 edge
├── internal/
│   ├── repository/
│   │   ├── affiliate_repo.go          # 添加 GetByCode 方法
│   │   └── wire.go                    # 注册 NewReferralRelationRepository
│   ├── service/
│   │   ├── affiliate_service.go       # 添加 ReferralRelationRepository 接口、BindReferralRelation、DetermineBindingType
│   │   ├── affiliate_errors.go        # 添加 ErrRelationAlreadyExists、ErrSelfReferral、ErrInvalidReferralCode
│   │   ├── auth_service.go            # 注入 AffiliateService、RegisterWithVerification 添加 referralCode 参数
│   │   └── wire.go                    # 更新 AffiliateService ProviderSet
│   ├── handler/
│   │   └── auth_handler.go            # 注册 Handler 传递 referral_code（或对应的 DTO 文件）
│   └── cmd/server/
│       ├── wire.go                    # 更新依赖注入配置
│       └── wire_gen.go                # go generate 重新生成
```

**命名约定（遵循项目模式）：**
- Schema 文件：`referral_relation.go`（snake_case 表名）
- Repository 文件：`referral_relation_repo.go`，类型 `ReferralRelationRepository`
- 接口定义：在 `affiliate_service.go` 中，类型 `ReferralRelationRepository`（interface）
- 绑定类型值：`"first_charge"` / `"lifetime"`（字符串常量，Epic 7 可改为 enum）
- 状态值：`"registered"` / `"first_charged"` / `"qualified"`（字符串常量）

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.2] - referral_relation 表完整 Schema 定义
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.1] - 邀请注册流程和绑定类型逻辑
- [Source: _bmad-output/planning-artifacts/epics.md#FR6] - 新用户通过邀请链接注册时自动绑定邀请关系
- [Source: _bmad-output/planning-artifacts/epics.md#FR7] - 邀请关系绑定后不可更改
- [Source: backend/internal/service/auth_service.go] - 现有注册流程（RegisterWithVerification 方法，第 85-176 行）
- [Source: backend/ent/schema/user.go] - User Schema 模式参考（Edge 定义）
- [Source: backend/ent/schema/user_affiliate.go] - 参考 Story 1.1 的 Ent Schema 定义模式
- [Source: _bmad-output/implementation-artifacts/1-1-user-affiliate-init.md] - Story 1.1 AffiliateRepository 接口定义
- [Source: _bmad-output/implementation-artifacts/1-4-invite-link-redirect.md] - Story 1.4 前端传递 referral_code

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestBindRelation_Normal | 有效邀请码 | 创建 relation 记录，返回 nil |
| TestBindRelation_EmptyCode | referralCode="" | 直接返回 nil，不调用任何 repo |
| TestBindRelation_CodeNotFound | 邀请码不存在 | 静默返回 nil |
| TestBindRelation_SelfReferral | inviter == invitee | 静默返回 nil |
| TestBindRelation_DuplicateInvitee | 已有邀请关系 | 静默返回 nil |
| TestBindRelation_UniqueConflict | UNIQUE 约束冲突 | 静默返回 nil |
| TestBindRelation_KolLifetime | KOL + default_binding_type=lifetime | binding_type="lifetime" |
| TestBindRelation_DefaultFirstCharge | 普通用户 | binding_type="first_charge" |
| TestDetermineBindingType_KolLifetime | KOL config 有 lifetime | 返回 "lifetime" |
| TestDetermineBindingType_KolNoConfig | KOL config 为 nil | 返回 "first_charge" |
| TestDetermineBindingType_NonKol | 普通用户 | 返回 "first_charge" |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整注册绑定 | 注册带邀请码 → 用户创建 + relation 创建 |
| 无邀请码注册 | 正常注册，无 relation 记录 |
| 自邀场景 | 自己邀请码注册 → 正常注册，无 relation |
| UNIQUE 约束 | 同一用户重复绑定 → 静默处理 |

### 注意事项

1. **事务边界**：`BindReferralRelation` 在用户创建成功后调用。当前项目中 `userRepo.Create` 直接提交（非显式事务），所以 relation 创建使用独立的数据库操作。如果注册流程改为事务模式，需要使用 `clientFromContext` 传递事务上下文
2. **幂等性**：双重保护 -- 先通过 `ExistsByInviteeID` 检查，再由 UNIQUE 约束兜底
3. **性能**：`GetByCode` 查询走 `idx_user_affiliate_referral_code` 唯一索引；`ExistsByInviteeID` 走 `idx_referral_relation_invitee` 唯一索引
4. **向后兼容**：`RegisterWithVerification` 方法签名改为 `RegisterOptions` DTO 模式。后续 Story（如 1.7 添加 ClientIP）只需在 DTO 中增加字段，无需再次修改方法签名。需要更新的调用方：(1) `Register` 方法、(2) OAuth 注册回调中的调用、(3) 所有测试中的调用。建议先全局搜索 `RegisterWithVerification` 的所有调用点
5. **Wire 循环依赖风险**：`AuthService` 依赖 `AffiliateService`，如果后续 `AffiliateService` 依赖 `AuthService`（如用户信息获取），会导致循环依赖。此时需要通过接口解耦或延迟注入解决
6. **Story 1.6 联动**：本 Story 在 `BindReferralRelation` 末尾预留了 `GrantRegisterBonus` 调用入口（注释状态），Story 1.6 实现时取消注释即可
7. **缓存写入**：创建关系后可选写入 Redis `aff:relation:{invitee_id}` 缓存（TTL 1h），减少后续查询压力。当前 Sprint 1 暂不实现缓存

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
