package service

import (
	"context"
	"time"
)

// 余额批次状态常量
const (
	BalanceLotStatusActive   = "active"   // 活跃（有余额可用）
	BalanceLotStatusDepleted = "depleted" // 已耗尽
	BalanceLotStatusExpired  = "expired"  // 已过期
)

// 余额批次来源类型常量
const (
	BalanceLotSourceRecharge  = "recharge"  // 充值
	BalanceLotSourceRedeem    = "redeem"    // 兑换码
	BalanceLotSourcePromo     = "promo"     // 优惠码
	BalanceLotSourceAdjust    = "adjust"    // 管理员调整
	BalanceLotSourceMigration = "migration" // 历史余额迁移
)

// BalanceLot 余额批次领域模型
type BalanceLot struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	SourceType      string     `json:"source_type"`
	SourceRef       *string    `json:"source_ref,omitempty"`
	OriginalAmount  float64    `json:"original_amount"`
	RemainingAmount float64    `json:"remaining_amount"`
	Status          string     `json:"status"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ExpiredAt       *time.Time `json:"expired_at,omitempty"`
	Description     string     `json:"description"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// BalanceLotSummary 余额批次概览
type BalanceLotSummary struct {
	TotalBalance float64             `json:"total_balance"`
	ExpiringSoon *ExpiringSoonDetail `json:"expiring_soon"`
}

// ExpiringSoonDetail 即将过期的批次汇总
type ExpiringSoonDetail struct {
	Amount         float64   `json:"amount"`
	EarliestExpiry time.Time `json:"earliest_expiry"`
}

// BalanceLotRepository 余额批次仓储接口
type BalanceLotRepository interface {
	// Create 创建余额批次
	Create(ctx context.Context, lot *BalanceLot) error

	// ListActiveByUserForUpdate 获取用户所有活跃批次并加行锁，按过期时间升序排列
	ListActiveByUserForUpdate(ctx context.Context, userID int64) ([]*BalanceLot, error)

	// UpdateRemainingAndStatus 更新批次的剩余金额和状态
	UpdateRemainingAndStatus(ctx context.Context, lotID int64, remaining float64, status string) error

	// BatchExpire 批量过期处理：查询 status=active 且 expires_at < now 的批次
	// 返回过期的批次列表（已将 status 更新为 expired, expired_at 设置为 now）
	BatchExpire(ctx context.Context, batchSize int) ([]*BalanceLot, error)

	// SumActiveRemaining 计算用户所有活跃批次的剩余金额总和
	SumActiveRemaining(ctx context.Context, userID int64) (float64, error)

	// ListByUser 分页查询用户的余额批次，status 为空时查询所有状态
	ListByUser(ctx context.Context, userID int64, status string, page, pageSize int) ([]*BalanceLot, int64, error)

	// ListExpiringSoon 查询即将过期的活跃批次（提前 advanceDays 天，所有用户）
	ListExpiringSoon(ctx context.Context, advanceDays int) ([]*BalanceLot, error)

	// ListExpiringSoonByUser 查询指定用户即将过期的活跃批次
	ListExpiringSoonByUser(ctx context.Context, userID int64, advanceDays int) ([]*BalanceLot, error)

	// CountByUser 统计用户的批次数量（用于迁移检查）
	CountByUser(ctx context.Context, userID int64) (int64, error)
}
