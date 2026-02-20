package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// LotteryActivityRepository 抽奖活动仓储接口
type LotteryActivityRepository interface {
	Create(ctx context.Context, activity *LotteryActivity) error
	GetByID(ctx context.Context, id int64) (*LotteryActivity, error)
	GetByShareCode(ctx context.Context, code string) (*LotteryActivity, error)
	Update(ctx context.Context, activity *LotteryActivity) error
	List(ctx context.Context, params pagination.PaginationParams, status string) ([]LotteryActivity, *pagination.PaginationResult, error)
	ListByStatus(ctx context.Context, status string) ([]LotteryActivity, error)
	ListActiveForDraw(ctx context.Context) ([]LotteryActivity, error) // draw_at <= now && status = active
	ListExpired(ctx context.Context) ([]LotteryActivity, error)      // activity_end_at <= now && status in (completed, cancelled)
	IncrementParticipantCount(ctx context.Context, id int64) error
	UpdateDrawResult(ctx context.Context, id int64, winnerCount int, activityEndAt time.Time) error
}

// LotteryParticipantRepository 参与记录仓储接口
type LotteryParticipantRepository interface {
	Create(ctx context.Context, p *LotteryParticipant) error
	GetByActivityAndUser(ctx context.Context, activityID, userID int64) (*LotteryParticipant, error)
	ListByActivity(ctx context.Context, activityID int64, params pagination.PaginationParams) ([]LotteryParticipant, *pagination.PaginationResult, error)
	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]LotteryParticipant, *pagination.PaginationResult, error)
	ListAllByActivity(ctx context.Context, activityID int64) ([]LotteryParticipant, error)
	UpdateWinnerStatus(ctx context.Context, id int64, isWinner bool, couponID *int64) error
}

// LotteryCouponRepository 优惠券仓储接口
type LotteryCouponRepository interface {
	Create(ctx context.Context, coupon *LotteryCoupon) error
	GetByID(ctx context.Context, id int64) (*LotteryCoupon, error)
	ListByUser(ctx context.Context, userID int64, status string, params pagination.PaginationParams) ([]LotteryCoupon, *pagination.PaginationResult, error)
	ListByActivity(ctx context.Context, activityID int64) ([]LotteryCoupon, error)
	MarkUsed(ctx context.Context, id int64, orderID string) error
	ReleaseByOrderID(ctx context.Context, orderID string) error        // 释放被订单占用的优惠券（订单取消/过期时），未过期券恢复为 active，已过期券标记为 expired
	ReOccupyByOrderID(ctx context.Context, orderID string) (int, error) // 重新占用被释放的券（延迟支付成功时），返回 affected 行数
	ExpireByActivity(ctx context.Context, activityID int64) error
	ListUnremindedExpiringSoon(ctx context.Context, withinHours int) ([]LotteryCoupon, error)
	MarkReminded(ctx context.Context, id int64) error
	FindActiveForUser(ctx context.Context, userID int64, scope string) (*LotteryCoupon, error) // 查找用户可用的优惠券
}
