package service

import (
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrLotteryActivityNotFound    = infraerrors.NotFound("LOTTERY_ACTIVITY_NOT_FOUND", "lottery activity not found")
	ErrLotteryParticipantNotFound = infraerrors.NotFound("LOTTERY_PARTICIPANT_NOT_FOUND", "lottery participant not found")
	ErrLotteryCouponNotFound      = infraerrors.NotFound("LOTTERY_COUPON_NOT_FOUND", "lottery coupon not found")
)

// LotteryActivity 抽奖活动 Domain Model
type LotteryActivity struct {
	ID                     int64
	Title                  string
	Description            string
	ShareCode              string
	Status                 string
	DrawAt                 time.Time
	ActivityStartAt        time.Time
	ActivityEndAt          time.Time
	MinParticipants        int
	BaseWinRate            float64
	WeightConfig           string // JSON string
	WinnerDiscountPercent  int
	LoserCouponAmount      float64
	TempGroupID            *int64
	ParticipantCount       int
	WinnerCount            int
	CreatedBy              int64
	CreatedAt              time.Time
	UpdatedAt              time.Time

	// 关联
	Participants []LotteryParticipant
	Coupons      []LotteryCoupon
}

// WeightConfigMap 解析权重配置为 map
func (a *LotteryActivity) WeightConfigMap() map[string]float64 {
	m := map[string]float64{
		LotteryUserCategoryNewUser:    3.0,
		LotteryUserCategoryRegular:    1.0,
		LotteryUserCategoryPaid:       0.3,
		LotteryUserCategorySubscriber: 0.1,
	}
	if a.WeightConfig != "" {
		_ = json.Unmarshal([]byte(a.WeightConfig), &m)
	}
	return m
}

// IsActive 活动是否正在进行中
func (a *LotteryActivity) IsActive() bool {
	return a.Status == LotteryStatusActive
}

// LotteryParticipant 参与记录 Domain Model
type LotteryParticipant struct {
	ID               int64
	ActivityID       int64
	UserID           int64
	UserCategory     string
	WeightMultiplier float64
	IsWinner         *bool
	CouponID         *int64
	ParticipatedAt   time.Time

	// 关联（可选展开）
	User *User
}

// LotteryCoupon 优惠券 Domain Model
type LotteryCoupon struct {
	ID              int64
	ActivityID      int64
	UserID          int64
	CouponType      string
	DiscountPercent *int
	ReductionAmount *float64
	ApplicableScope string
	Status          string
	UsedAt          *time.Time
	UsedOrderID     *string
	ExpiresAt       time.Time
	Reminded        bool
	CreatedAt       time.Time
}

// IsActive 优惠券是否可用
func (c *LotteryCoupon) IsActive() bool {
	return c.Status == LotteryCouponStatusActive && time.Now().Before(c.ExpiresAt)
}

// CreateLotteryActivityInput 创建活动输入
type CreateLotteryActivityInput struct {
	Title                 string
	Description           string
	DrawAt                *time.Time // nil 则默认次日 10:00
	ValidityDays          int        // 开奖后有效期天数（默认 3 天）
	MinParticipants       int
	BaseWinRate           float64
	WinnerDiscountPercent int
	LoserCouponAmount     float64
	AccountIDs            []int64 // 绑定的过期账号 ID
	DailyLimitUSD         float64 // 每日限额（美元），默认 20
}

// LotteryDrawResult 开奖结果
type LotteryDrawResult struct {
	ActivityID          int64
	TotalParticipants   int
	WinnerCount         int
	BelowMinParticipant bool // 参与人数是否低于最低要求
}
