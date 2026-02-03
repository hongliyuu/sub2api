# Story 5.2: 提现申请提交

Status: ready-for-dev

## Story

As a 普通用户,
I want 提交提现申请,
so that 将佣金转入我的账户。

## Acceptance Criteria

1. **AC1**: 满足提现条件时可提交申请
2. **AC2**: 创建 withdrawal_record（status=pending）
3. **AC3**: 使用行锁（SELECT FOR UPDATE）+ 乐观锁（version）扣减 withdrawable
4. **AC4**: 提现金额 > 可提现金额时拒绝
5. **AC5**: 提现金额 < 门槛时拒绝
6. **AC6**: 记录审计日志 `withdraw_requested`

## Dependencies

- **Depends On**: Story 5.1 (提现资格检查), Story 4.3 (confirmed 佣金产生 withdrawable)
- **Depended By**: Story 8.3 (管理员审核列表), Story 8.4 (审核通过), Story 8.5 (审核拒绝)

## Tasks / Subtasks

- [ ] Task 1: 创建 withdrawal_record Ent Schema + 迁移 (AC: #2)
  - [ ] 1.1 创建 `backend/ent/schema/withdrawal_record.go`
  - [ ] 1.2 SQL 迁移文件
- [ ] Task 2: Repository (AC: #2)
  - [ ] 2.1 创建 `internal/repository/withdrawal_repo.go`
- [ ] Task 3: Service 层提现逻辑 (AC: #1, #3, #4, #5, #6)
  - [ ] 3.1 在 `affiliate_service.go` 添加 `RequestWithdraw(ctx, userID, amount, method, account)` 方法
  - [ ] 3.2 并发控制：SELECT FOR UPDATE + 乐观锁
- [ ] Task 4: API 端点 (AC: #1)
  - [ ] 4.1 在 `affiliate_handler.go` 添加 `RequestWithdraw`
  - [ ] 4.2 注册路由 `POST /api/v1/affiliate/withdraw`

## Dev Notes

### 并发控制实现

```go
func (s *AffiliateService) RequestWithdraw(ctx context.Context, userID int64, req WithdrawRequest) error {
    return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
        // 行锁
        affiliate, err := s.affiliateRepo.GetForUpdate(txCtx, userID)
        if err != nil {
            return err
        }

        if affiliate.Withdrawable < req.Amount {
            return ErrInsufficientBalance
        }
        if req.Amount < withdrawThreshold {
            return ErrBelowThreshold
        }

        // 乐观锁更新
        oldVersion := affiliate.Version
        affiliate.Withdrawable -= req.Amount
        affiliate.Version++
        rows, err := s.affiliateRepo.UpdateWithVersion(txCtx, affiliate, oldVersion)
        if rows == 0 {
            return ErrConcurrentUpdate
        }

        // 创建提现记录
        withdrawal := &WithdrawalRecord{
            UserID:         userID,
            Amount:         req.Amount,
            Status:         "pending",
            PaymentMethod:  req.PaymentMethod,
            PaymentAccount: encrypt(req.PaymentAccount), // AES-256 加密
        }
        return s.withdrawalRepo.Create(txCtx, withdrawal)
    })
}
```

### withdrawal_record 表

```sql
CREATE TABLE withdrawal_record (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    amount          DECIMAL(10,2) NOT NULL,
    status          VARCHAR(20) DEFAULT 'pending',
    payment_method  VARCHAR(20),
    payment_account VARCHAR(128),
    reject_reason   VARCHAR(256),
    reviewed_by     BIGINT,
    reviewed_at     TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### API

```
POST /api/v1/affiliate/withdraw
Body: { "amount": 100.00, "payment_method": "alipay", "payment_account": "138****8888" }
```

### Ent Schema 框架

```go
// backend/ent/schema/withdrawal_record.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type WithdrawalRecord struct {
    ent.Schema
}

func (WithdrawalRecord) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("user_id"),
        field.Float("amount"),
        field.String("status").Default("pending"),
        field.String("payment_method").Optional(),
        field.String("payment_account").Optional(),
        field.String("reject_reason").Optional(),
        field.Int64("reviewed_by").Optional().Nillable(),
        field.Time("reviewed_at").Optional().Nillable(),
        field.Time("completed_at").Optional().Nillable(),
    }
}

func (WithdrawalRecord) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("user_id", "status"),
    }
}
```

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestRequestWithdraw_Normal | withdrawable=150, amount=100 | 创建 pending 记录，withdrawable 减少100 |
| TestRequestWithdraw_InsufficientBalance | amount > withdrawable | 返回 ErrInsufficientBalance |
| TestRequestWithdraw_BelowThreshold | amount=50 | 返回 ErrBelowThreshold |
| TestRequestWithdraw_ConcurrentUpdate | 乐观锁冲突 | 返回 ErrConcurrentUpdate |
| TestRequestWithdraw_AuditLog | 提交成功 | 记录 withdraw_requested 审计日志 |
| TestRequestWithdraw_AccountEncrypted | 账户信息 | payment_account 以 AES-256 加密存储 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整提现流程 | 资格检查 → 行锁 → 扣减 → 创建记录 → 事务提交 |
| 并发提现测试 | 两个 goroutine 同时提现 → 只有一个成功 |

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.4] - withdrawal_record 表
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#4.5] - 提现并发控制
- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.2] - FR26, FR28, NFR10

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
