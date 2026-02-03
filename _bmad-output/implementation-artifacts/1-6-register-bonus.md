# Story 1.6: 发放注册奖励

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 邀请人和被邀请人,
I want 邀请关系建立后双方都获得余额奖励,
so that 激励推广行为。

## Acceptance Criteria

1. **AC1**: 邀请关系建立后，邀请人余额增加配置金额（默认 $1）
2. **AC2**: 被邀请人余额增加配置金额（默认 $1）
3. **AC3**: 双方各生成一条 `commission_record`（source_type=register, status=confirmed）
4. **AC4**: 更新邀请人的 `user_affiliate.total_earnings` 和 `user_affiliate.withdrawable`
5. **AC5**: 注册奖励直接确认（status=confirmed, confirmed_at=now），无需 7 天确认期
6. **AC6**: 所有操作在同一事务中完成，任一步骤失败全部回滚
7. **AC7**: 记录审计日志 `commission_created`
8. **AC8**: 奖励发放失败不影响邀请关系绑定（BindReferralRelation 仍然成功）

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表和 AffiliateService), Story 1.5 (referral_relation 表和 BindReferralRelation 方法)
- **Modifies**: Story 1.5 (BindReferralRelation 方法末尾取消 GrantRegisterBonus 注释调用)
- **Depended By**: Story 2.3 (commission_record 表复用), Story 4.3 (佣金确认 Job), Story 4.4 (退款取消佣金), Story 6.5 (KOL 额外奖励)

## Tasks / Subtasks

- [ ] Task 1: 创建 commission_record Ent Schema (AC: #3)
  - [ ] 1.1 创建 `backend/ent/schema/commission_record.go`
  - [ ] 1.2 定义字段：id, user_id, source_user_id, relation_id, source_type, source_order_id, amount, rate, status, confirmed_at, created_at
  - [ ] 1.3 定义索引：(user_id, status) 复合索引、source_user_id 索引、created_at 索引
  - [ ] 1.4 定义 source_order_id 条件唯一索引（WHERE source_order_id IS NOT NULL）
  - [ ] 1.5 在 `ent/schema/user.go` 添加 edge 关联
  - [ ] 1.6 运行 `go generate ./ent` 生成代码
- [ ] Task 2: 创建 SQL 迁移文件 (AC: #3)
  - [ ] 2.1 创建 `migrations/XXX_create_commission_record.sql`
  - [ ] 2.2 包含表结构、索引、外键约束
- [ ] Task 3: 实现 CommissionRecord Repository (AC: #3, #6)
  - [ ] 3.1 创建 `internal/repository/commission_record_repo.go`
  - [ ] 3.2 实现 `Create(ctx, record)` 方法，支持事务（使用 `clientFromContext`）
- [ ] Task 4: 扩展 AffiliateRepository (AC: #4)
  - [ ] 4.1 在 `affiliate_repo.go` 添加 `IncrementEarnings(ctx, userID, amount)` 方法
  - [ ] 4.2 使用 Ent 的 `AddTotalEarnings` 和 `AddWithdrawable` 原子更新
- [ ] Task 5: 实现注册奖励 Service 逻辑 (AC: #1, #2, #3, #4, #5, #6, #7)
  - [ ] 5.1 在 `affiliate_service.go` 添加 `GrantRegisterBonus(ctx, inviterID, inviteeID, relationID)` 方法
  - [ ] 5.2 使用事务包装所有数据库操作
  - [ ] 5.3 创建邀请人 commission_record（source_type=register, status=confirmed）
  - [ ] 5.4 创建被邀请人 commission_record（source_type=register, status=confirmed）
  - [ ] 5.5 调用 `userRepo.UpdateBalance` 增加双方余额
  - [ ] 5.6 调用 `affiliateRepo.IncrementEarnings` 更新邀请人分销收益
  - [ ] 5.7 审计日志记录
- [ ] Task 6: 在 BindReferralRelation 中调用 (AC: #8)
  - [ ] 6.1 取消 Story 1.5 中 GrantRegisterBonus 的注释调用
  - [ ] 6.2 奖励发放失败时记录 warning 日志，不影响 BindReferralRelation 返回值
- [ ] Task 7: AffiliateService 依赖更新 (AC: #1, #2)
  - [ ] 7.1 AffiliateService 注入 `UserRepository`（用于 UpdateBalance）
  - [ ] 7.2 AffiliateService 注入 `CommissionRecordRepository`
  - [ ] 7.3 更新 Wire ProviderSet
  - [ ] 7.4 运行 `go generate ./cmd/server` 重新生成 Wire 代码

## Dev Notes

### 前置依赖

- Story 1.1：`user_affiliate` 表和 `AffiliateService` 已存在
- Story 1.5：`referral_relation` 表已创建，`BindReferralRelation` 方法已实现
- 项目中已存在 `userRepo.UpdateBalance(ctx, userID, amount)` 方法（参见 `backend/internal/service/user_service.go:185` 和 `backend/internal/repository/user_repo.go:322`）

### 数据库 Schema

```sql
-- 迁移文件: migrations/XXX_create_commission_record.sql

CREATE TABLE commission_record (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,             -- 获得佣金的用户
    source_user_id  BIGINT NOT NULL,             -- 贡献佣金的用户
    relation_id     BIGINT,                      -- 关联邀请关系
    source_type     VARCHAR(20) NOT NULL,        -- register | recharge
    source_order_id VARCHAR(64),                 -- 关联订单号（充值场景）
    amount          DECIMAL(10,2) NOT NULL,      -- 佣金金额
    rate            DECIMAL(5,4),                -- 佣金比例（充值场景使用）
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | confirmed | withdrawn | cancelled
    confirmed_at    TIMESTAMPTZ,                 -- 确认时间
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (source_user_id) REFERENCES users(id),
    FOREIGN KEY (relation_id) REFERENCES referral_relation(id)
);

-- 索引：按用户和状态查询佣金记录
CREATE INDEX idx_commission_user ON commission_record(user_id, status);

-- 索引：按来源用户查询
CREATE INDEX idx_commission_source ON commission_record(source_user_id);

-- 索引：按创建时间排序
CREATE INDEX idx_commission_created ON commission_record(created_at);

-- 条件唯一索引：同一订单只能有一条佣金记录（充值场景防重复）
CREATE UNIQUE INDEX idx_commission_order ON commission_record(source_order_id) WHERE source_order_id IS NOT NULL;
```

### Ent Schema 定义

```go
// backend/ent/schema/commission_record.go
package schema

import (
    "time"

    "entgo.io/ent"
    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type CommissionRecord struct {
    ent.Schema
}

func (CommissionRecord) Annotations() []schema.Annotation {
    return []schema.Annotation{
        entsql.Annotation{Table: "commission_record"},
    }
}

func (CommissionRecord) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("user_id").
            Comment("获得佣金的用户 ID"),
        field.Int64("source_user_id").
            Comment("贡献佣金的用户 ID"),
        field.Int64("relation_id").
            Optional().
            Nillable().
            Comment("关联邀请关系 ID"),
        field.String("source_type").
            NotEmpty().
            Comment("来源类型: register | recharge"),
        field.String("source_order_id").
            Optional().
            Nillable().
            MaxLen(64).
            Comment("关联订单号（充值场景）"),
        field.Float("amount").
            SchemaType(map[string]string{dialect.Postgres: "decimal(10,2)"}).
            Comment("佣金金额"),
        field.Float("rate").
            Optional().
            Nillable().
            SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}).
            Comment("佣金比例（充值场景）"),
        field.String("status").
            Default("pending").
            Comment("状态: pending | confirmed | withdrawn | cancelled"),
        field.Time("confirmed_at").
            Optional().
            Nillable().
            Comment("确认时间"),
        field.Time("created_at").
            Default(time.Now).
            Immutable(),
    }
}

func (CommissionRecord) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("user", User.Type).
            Ref("commission_records").
            Field("user_id").
            Unique().
            Required(),
        edge.From("source_user", User.Type).
            Ref("contributed_commissions").
            Field("source_user_id").
            Unique().
            Required(),
        edge.From("relation", ReferralRelation.Type).
            Ref("commissions").
            Field("relation_id").
            Unique(),
    }
}

func (CommissionRecord) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("user_id", "status"),
        index.Fields("source_user_id"),
        index.Fields("created_at"),
        // 注意：Ent 不直接支持条件唯一索引（WHERE clause），
        // source_order_id 的条件唯一索引需要通过 SQL 迁移文件手动创建
    }
}
```

**需在相关 Schema 添加反向 edge：**

```go
// backend/ent/schema/user.go - Edges() 方法中添加
edge.To("commission_records", CommissionRecord.Type),        // 用户获得的佣金记录
edge.To("contributed_commissions", CommissionRecord.Type),   // 用户贡献的佣金记录

// backend/ent/schema/referral_relation.go - Edges() 方法中添加
edge.To("commissions", CommissionRecord.Type),               // 该邀请关系产生的佣金记录
```

### Repository 实现

```go
// internal/repository/commission_record_repo.go
package repository

import (
    "context"
    "time"

    "github.com/Wei-Shaw/sub2api/ent"
    "github.com/Wei-Shaw/sub2api/ent/commissionrecord"
)

type CommissionRecordRepository struct {
    client *ent.Client
}

func NewCommissionRecordRepository(client *ent.Client) *CommissionRecordRepository {
    return &CommissionRecordRepository{client: client}
}

// Create 创建佣金记录，支持事务上下文
func (r *CommissionRecordRepository) Create(ctx context.Context, record *CommissionRecord) (*ent.CommissionRecord, error) {
    client := clientFromContext(ctx, r.client)
    builder := client.CommissionRecord.Create().
        SetUserID(record.UserID).
        SetSourceUserID(record.SourceUserID).
        SetSourceType(record.SourceType).
        SetAmount(record.Amount).
        SetStatus(record.Status)

    // 可选字段
    if record.RelationID != nil {
        builder.SetRelationID(*record.RelationID)
    }
    if record.SourceOrderID != nil {
        builder.SetSourceOrderID(*record.SourceOrderID)
    }
    if record.Rate != nil {
        builder.SetRate(*record.Rate)
    }
    if record.ConfirmedAt != nil {
        builder.SetConfirmedAt(*record.ConfirmedAt)
    }

    return builder.Save(ctx)
}
```

### AffiliateRepository 扩展 - IncrementEarnings

在 `internal/repository/affiliate_repo.go` 中新增方法：

```go
// IncrementEarnings 原子更新用户分销收益和可提现金额
// 使用 Ent 的 Add* 方法确保原子性
func (r *affiliateRepository) IncrementEarnings(ctx context.Context, userID int64, amount float64) error {
    client := clientFromContext(ctx, r.client)
    n, err := client.UserAffiliate.Update().
        Where(useraffiliate.UserIDEQ(userID)).
        AddTotalEarnings(amount).
        AddWithdrawable(amount).
        Save(ctx)
    if err != nil {
        return err
    }
    if n == 0 {
        return fmt.Errorf("user affiliate not found: user_id=%d", userID)
    }
    return nil
}
```

更新 `AffiliateRepository` 接口：

```go
type AffiliateRepository interface {
    Create(ctx context.Context, affiliate *UserAffiliate) error
    GetByUserID(ctx context.Context, userID int64) (*UserAffiliate, error)
    ExistsByCode(ctx context.Context, code string) (bool, error)
    GetByCode(ctx context.Context, code string) (*ent.UserAffiliate, error)
    IncrementEarnings(ctx context.Context, userID int64, amount float64) error  // 新增
}
```

### Service 层实现

```go
// internal/service/affiliate_service.go

// 注册奖励常量（硬编码，Epic 7 配置化）
const (
    defaultInviterRegisterBonus = 1.00 // 邀请人注册奖励 $1
    defaultInviteeRegisterBonus = 1.00 // 被邀请人注册奖励 $1
)

// CommissionRecordRepository 接口定义
type CommissionRecordRepository interface {
    Create(ctx context.Context, record *CommissionRecord) (*ent.CommissionRecord, error)
}

// 更新 AffiliateService 结构体
type AffiliateService struct {
    affiliateRepo  AffiliateRepository
    relationRepo   ReferralRelationRepository
    commissionRepo CommissionRecordRepository
    userRepo       UserRepository              // 用于 UpdateBalance
    cfg            *config.Config
}

func NewAffiliateService(
    affiliateRepo AffiliateRepository,
    relationRepo ReferralRelationRepository,
    commissionRepo CommissionRecordRepository,
    userRepo UserRepository,
    cfg *config.Config,
) *AffiliateService {
    return &AffiliateService{
        affiliateRepo:  affiliateRepo,
        relationRepo:   relationRepo,
        commissionRepo: commissionRepo,
        userRepo:       userRepo,
        cfg:            cfg,
    }
}

// GrantRegisterBonus 发放注册奖励
// 在邀请关系建立后调用，在同一事务中完成：
// 1. 创建邀请人 commission_record
// 2. 创建被邀请人 commission_record
// 3. 增加双方用户余额
// 4. 更新邀请人 user_affiliate 的 total_earnings 和 withdrawable
func (s *AffiliateService) GrantRegisterBonus(ctx context.Context, inviterID, inviteeID, relationID int64) error {
    now := time.Now()

    // 使用项目现有的事务管理模式
    // 参考 promo_service.go 和 payment_callback_service.go 的事务用法
    tx, err := s.client.Tx(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    txCtx := context.WithValue(ctx, txKey, tx)

    // 1. 创建邀请人佣金记录
    inviterCommission := &CommissionRecord{
        UserID:       inviterID,
        SourceUserID: inviteeID,
        RelationID:   &relationID,
        SourceType:   "register",
        Amount:       defaultInviterRegisterBonus,
        Status:       "confirmed",  // 注册奖励直接确认
        ConfirmedAt:  &now,
    }
    if _, err = s.commissionRepo.Create(txCtx, inviterCommission); err != nil {
        return fmt.Errorf("create inviter commission: %w", err)
    }

    // 2. 创建被邀请人佣金记录
    inviteeCommission := &CommissionRecord{
        UserID:       inviteeID,
        SourceUserID: inviterID,
        RelationID:   &relationID,
        SourceType:   "register",
        Amount:       defaultInviteeRegisterBonus,
        Status:       "confirmed",  // 注册奖励直接确认
        ConfirmedAt:  &now,
    }
    if _, err = s.commissionRepo.Create(txCtx, inviteeCommission); err != nil {
        return fmt.Errorf("create invitee commission: %w", err)
    }

    // 3. 增加邀请人余额
    if err = s.userRepo.UpdateBalance(txCtx, inviterID, defaultInviterRegisterBonus); err != nil {
        return fmt.Errorf("update inviter balance: %w", err)
    }

    // 4. 增加被邀请人余额
    if err = s.userRepo.UpdateBalance(txCtx, inviteeID, defaultInviteeRegisterBonus); err != nil {
        return fmt.Errorf("update invitee balance: %w", err)
    }

    // 5. 更新邀请人分销收益
    if err = s.affiliateRepo.IncrementEarnings(txCtx, inviterID, defaultInviterRegisterBonus); err != nil {
        return fmt.Errorf("update inviter earnings: %w", err)
    }

    // 6. 提交事务
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    log.Printf("[Affiliate] Register bonus granted: inviter=%d (+$%.2f), invitee=%d (+$%.2f), relation=%d",
        inviterID, defaultInviterRegisterBonus, inviteeID, defaultInviteeRegisterBonus, relationID)

    return nil
}
```

**关于事务管理模式的说明：**

项目中存在两种事务模式，需要根据实际代码选择：

1. **clientFromContext 模式**：大部分 repository 使用 `clientFromContext(ctx, r.client)` 从 context 中提取事务客户端
2. **显式 Tx 模式**：部分 service（如 `payment_callback_service.go`）使用 `s.client.Tx(ctx)` 显式创建事务

建议先查看项目中已有的事务管理 helper（如 `txKey`、`WithTx` 等），遵循一致的模式。如果项目使用 `clientFromContext` 模式，则 `GrantRegisterBonus` 的实现需要调整为：

```go
// 替代方案：使用 WithTx helper（如项目提供）
func (s *AffiliateService) GrantRegisterBonus(ctx context.Context, inviterID, inviteeID, relationID int64) error {
    return WithTx(ctx, s.client, func(txCtx context.Context) error {
        // ... 所有操作使用 txCtx ...
    })
}
```

### 在 BindReferralRelation 中集成

取消 Story 1.5 中预留的注释：

```go
// internal/service/affiliate_service.go - BindReferralRelation 方法末尾

// 创建关系成功后发放注册奖励
if err := s.GrantRegisterBonus(ctx, inviterID, inviteeID, created.ID); err != nil {
    // 奖励发放失败不影响关系绑定
    // 关系已经创建成功，奖励可以后续补发
    log.Printf("[Affiliate] Warning: grant register bonus failed for relation %d: %v", created.ID, err)
}
```

### 关于 UserRepository 接口

`AffiliateService` 需要调用 `userRepo.UpdateBalance`。该方法已在 `UserRepository` 接口中定义（`backend/internal/service/user_service.go:36`）：

```go
type UserRepository interface {
    // ... 其他方法 ...
    UpdateBalance(ctx context.Context, id int64, amount float64) error
}
```

实现位于 `backend/internal/repository/user_repo.go:322`，支持正数（增加余额）和负数（扣减余额）。`UpdateBalance` 已经支持事务上下文（通过 `clientFromContext`），所以在事务中调用是安全的。

### Project Structure Notes

**新增文件：**
```
backend/
├── ent/schema/
│   └── commission_record.go            # Ent Schema 定义
├── migrations/
│   └── XXX_create_commission_record.sql # 数据库迁移
├── internal/
│   └── repository/
│       └── commission_record_repo.go   # Repository 实现
```

**修改文件：**
```
backend/
├── ent/schema/
│   ├── user.go                         # 添加 commission_records/contributed_commissions edge
│   └── referral_relation.go            # 添加 commissions edge
├── internal/
│   ├── repository/
│   │   ├── affiliate_repo.go           # 添加 IncrementEarnings 方法
│   │   └── wire.go                     # 注册 NewCommissionRecordRepository
│   ├── service/
│   │   ├── affiliate_service.go        # 添加 GrantRegisterBonus 方法、更新构造函数注入 CommissionRecordRepository 和 UserRepository
│   │   └── wire.go                     # 更新 AffiliateService ProviderSet
│   └── cmd/server/
│       ├── wire.go                     # 更新依赖注入配置
│       └── wire_gen.go                 # go generate 重新生成
```

**命名约定（遵循项目模式）：**
- Schema 文件：`commission_record.go`（snake_case 表名）
- Repository 文件：`commission_record_repo.go`，类型 `CommissionRecordRepository`
- 接口定义：在 `affiliate_service.go` 中，类型 `CommissionRecordRepository`（interface）
- 来源类型值：`"register"` / `"recharge"`（字符串常量）
- 状态值：`"pending"` / `"confirmed"` / `"withdrawn"` / `"cancelled"`（字符串常量）

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.3] - commission_record 表完整 Schema（第 197-220 行）
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.5] - affiliate_config 配置项：register_bonus 默认值 inviter=1.00, invitee=1.00
- [Source: _bmad-output/planning-artifacts/epics.md#FR9] - 被邀请人完成注册时，邀请人获得配置金额的余额奖励
- [Source: _bmad-output/planning-artifacts/epics.md#FR10] - 被邀请人完成注册时，自己获得配置金额的余额奖励
- [Source: _bmad-output/planning-artifacts/epics.md#FR11] - 注册奖励金额可在后台配置（邀请人/被邀请人分别配置）
- [Source: backend/internal/service/user_service.go#L36] - UserRepository 接口中 UpdateBalance 定义
- [Source: backend/internal/repository/user_repo.go#L322] - UpdateBalance 具体实现
- [Source: backend/internal/service/promo_service.go#L127] - 参考 PromoService 使用 UpdateBalance 的模式
- [Source: backend/internal/service/payment_callback_service.go#L378] - 参考事务中使用 UpdateBalance 的模式
- [Source: _bmad-output/implementation-artifacts/1-5-register-bind-relation.md] - Story 1.5 BindReferralRelation 方法和 GrantRegisterBonus 调用入口

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGrantRegisterBonus_Normal | 正常发放奖励 | 双方 commission_record 正确创建（source_type=register, status=confirmed, amount=1.00） |
| TestGrantRegisterBonus_InviterBalance | 邀请人余额更新 | mock UserRepo.UpdateBalance 被调用 (inviterID, 1.00) |
| TestGrantRegisterBonus_InviteeBalance | 被邀请人余额更新 | mock UserRepo.UpdateBalance 被调用 (inviteeID, 1.00) |
| TestGrantRegisterBonus_Earnings | 邀请人收益更新 | mock AffiliateRepo.IncrementEarnings 被调用 (inviterID, 1.00) |
| TestGrantRegisterBonus_ConfirmedAt | 直接确认 | confirmed_at 字段非空，status=confirmed |
| TestGrantRegisterBonus_CommissionFail | CreateCommission 失败 | 事务回滚，UpdateBalance 不执行 |
| TestGrantRegisterBonus_BalanceFail | UpdateBalance 失败 | 事务回滚，IncrementEarnings 不执行，已创建 commission 回滚 |
| TestGrantRegisterBonus_CalledFromBind | BindReferralRelation 调用 | 奖励失败不影响关系绑定（warning 日志） |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整注册奖励流程 | 注册(带邀请码) → 绑定关系 → 发放奖励 → inviter/invitee 余额各增 $1.00，commission_record 表有 2 条记录，user_affiliate.total_earnings/withdrawable 增加 $1.00 |
| 事务完整性 | 模拟 IncrementEarnings 失败 → 双方余额未变化、commission_record 未创建 |
| 幂等性 | 同一 invitee 不因重复调用获得双份奖励（依赖 referral_relation UNIQUE 约束） |

### 注意事项

1. **Wire 循环依赖**：`AffiliateService` 需要注入 `UserRepository`。由于 `AuthService` 也依赖 `UserRepository` 和 `AffiliateService`，需要确保 Wire 的依赖图是 DAG（无环）。当前设计中 `AffiliateService` 不依赖 `AuthService`，所以不会有循环依赖
2. **奖励金额硬编码**：`defaultInviterRegisterBonus = 1.00`、`defaultInviteeRegisterBonus = 1.00`，Epic 7 通过 `affiliate_config` 表（config_key=register_bonus）实现配置化
3. **事务管理**：务必查看项目现有的事务模式再实现。不同项目可能使用不同的 `WithTx`/`Tx`/`clientFromContext` 组合方式
4. **decimal 精度**：使用 PostgreSQL 的 `decimal(10,2)` 类型存储金额，Go 侧使用 `float64`。对于分销场景 float64 精度足够，但如果项目使用了 `shopspring/decimal` 等库则应保持一致
5. **并发安全**：`IncrementEarnings` 使用 SQL `UPDATE ... SET total_earnings = total_earnings + amount`，是原子操作。但如果 `user_affiliate` 使用了乐观锁（version 字段），需要在 IncrementEarnings 中同时处理 version 更新
6. **奖励仅发一次**：幂等性依赖 `referral_relation.invitee_id` UNIQUE 约束保证。如果关系已存在，`BindReferralRelation` 会跳过，自然不会调用 `GrantRegisterBonus`

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
