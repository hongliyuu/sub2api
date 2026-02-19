package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrLotteryActivityNotActive   = infraerrors.BadRequest("LOTTERY_NOT_ACTIVE", "lottery activity is not active")
	ErrLotteryAlreadyDrawn        = infraerrors.BadRequest("LOTTERY_ALREADY_DRAWN", "lottery has already been drawn")
	ErrLotteryAlreadyParticipated = infraerrors.Conflict("LOTTERY_ALREADY_PARTICIPATED", "you have already participated in this lottery")
	ErrLotteryEmailRequired       = infraerrors.BadRequest("LOTTERY_EMAIL_REQUIRED", "email binding is required to participate")
	ErrLotteryCancelled           = infraerrors.BadRequest("LOTTERY_CANCELLED", "lottery activity has been cancelled")
)

// LotteryService 抽奖服务
type LotteryService struct {
	activityRepo    LotteryActivityRepository
	participantRepo LotteryParticipantRepository
	couponRepo      LotteryCouponRepository
	groupService    *GroupService
	accountRepo     AccountRepository
	entClient       *dbent.Client
}

// NewLotteryService 创建抽奖服务实例
func NewLotteryService(
	activityRepo LotteryActivityRepository,
	participantRepo LotteryParticipantRepository,
	couponRepo LotteryCouponRepository,
	groupService *GroupService,
	accountRepo AccountRepository,
	entClient *dbent.Client,
) *LotteryService {
	return &LotteryService{
		activityRepo:    activityRepo,
		participantRepo: participantRepo,
		couponRepo:      couponRepo,
		groupService:    groupService,
		accountRepo:     accountRepo,
		entClient:       entClient,
	}
}

// CreateActivity 创建抽奖活动
func (s *LotteryService) CreateActivity(ctx context.Context, input CreateLotteryActivityInput, adminID int64) (*LotteryActivity, error) {
	shareCode := generateShareCode()

	now := time.Now()

	// 设置开奖时间：如果未指定，默认次日 10:00 AM (UTC+8)
	var drawAt time.Time
	if input.DrawAt != nil {
		drawAt = *input.DrawAt
	} else {
		cst := time.FixedZone("CST", 8*3600)
		tomorrow := now.In(cst).AddDate(0, 0, 1)
		drawAt = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, cst)
	}

	activityStartAt := now
	activityEndAt := drawAt.Add(3 * 24 * time.Hour)

	// 设置默认值
	baseWinRate := input.BaseWinRate
	if baseWinRate == 0 {
		baseWinRate = 0.1
	}
	winnerDiscountPercent := input.WinnerDiscountPercent
	if winnerDiscountPercent == 0 {
		winnerDiscountPercent = 95
	}
	loserCouponAmount := input.LoserCouponAmount
	if loserCouponAmount == 0 {
		loserCouponAmount = 30
	}
	minParticipants := input.MinParticipants
	if minParticipants == 0 {
		minParticipants = 10
	}

	// 创建临时分组
	groupName := fmt.Sprintf("lottery-%s", shareCode)
	groupDesc := fmt.Sprintf("抽奖活动临时分组: %s", input.Title)
	group, err := s.groupService.Create(ctx, CreateGroupRequest{
		Name:           groupName,
		Description:    groupDesc,
		IsExclusive:    false,
		RateMultiplier: 1.0,
	})
	if err != nil {
		return nil, fmt.Errorf("create temp group: %w", err)
	}

	log.Printf("[Lottery] Created temp group %d (%s) for activity %q", group.ID, groupName, input.Title)

	// 默认权重配置
	weightConfig := `{"new_user":3.0,"regular":1.0,"paid":0.3,"subscriber":0.1}`

	activity := &LotteryActivity{
		Title:                 input.Title,
		Description:           input.Description,
		ShareCode:             shareCode,
		Status:                LotteryStatusActive,
		DrawAt:                drawAt,
		ActivityStartAt:       activityStartAt,
		ActivityEndAt:         activityEndAt,
		MinParticipants:       minParticipants,
		BaseWinRate:           baseWinRate,
		WeightConfig:          weightConfig,
		WinnerDiscountPercent: winnerDiscountPercent,
		LoserCouponAmount:     loserCouponAmount,
		TempGroupID:           &group.ID,
		ParticipantCount:      0,
		WinnerCount:           0,
		CreatedBy:             adminID,
	}

	if err := s.activityRepo.Create(ctx, activity); err != nil {
		return nil, fmt.Errorf("create lottery activity: %w", err)
	}

	log.Printf("[Lottery] Created activity %d (share_code=%s, draw_at=%s)", activity.ID, shareCode, drawAt.Format(time.RFC3339))

	// 将指定的过期账号绑定到临时分组
	if len(input.AccountIDs) > 0 {
		boundCount := 0
		for _, accountID := range input.AccountIDs {
			if err := s.accountRepo.BindGroups(ctx, accountID, []int64{group.ID}); err != nil {
				log.Printf("[Lottery] Warning: failed to bind account %d to temp group %d: %v", accountID, group.ID, err)
				continue
			}
			boundCount++
		}
		log.Printf("[Lottery] Bound %d/%d accounts to temp group %d for activity %d",
			boundCount, len(input.AccountIDs), group.ID, activity.ID)
	}

	return activity, nil
}

// GetByID 根据ID获取抽奖活动
func (s *LotteryService) GetByID(ctx context.Context, id int64) (*LotteryActivity, error) {
	activity, err := s.activityRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return activity, nil
}

// GetByShareCode 根据分享码获取抽奖活动
func (s *LotteryService) GetByShareCode(ctx context.Context, code string) (*LotteryActivity, error) {
	activity, err := s.activityRepo.GetByShareCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return activity, nil
}

// ListActivities 获取抽奖活动列表
func (s *LotteryService) ListActivities(ctx context.Context, params pagination.PaginationParams, status string) ([]LotteryActivity, *pagination.PaginationResult, error) {
	return s.activityRepo.List(ctx, params, status)
}

// CancelActivity 取消抽奖活动
func (s *LotteryService) CancelActivity(ctx context.Context, id int64) error {
	activity, err := s.activityRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 只有 pending 或 active 状态的活动可以取消
	if activity.Status != LotteryStatusPending && activity.Status != LotteryStatusActive {
		return ErrLotteryCancelled
	}

	activity.Status = LotteryStatusCancelled
	if err := s.activityRepo.Update(ctx, activity); err != nil {
		return fmt.Errorf("update activity status: %w", err)
	}

	// 过期该活动下所有优惠券
	if err := s.couponRepo.ExpireByActivity(ctx, id); err != nil {
		log.Printf("[Lottery] Failed to expire coupons for activity %d: %v", id, err)
	}

	log.Printf("[Lottery] Cancelled activity %d", id)

	return nil
}

// GetActiveActivity 获取当前活跃的抽奖活动（返回第一个）
func (s *LotteryService) GetActiveActivity(ctx context.Context) (*LotteryActivity, error) {
	activities, err := s.activityRepo.ListByStatus(ctx, LotteryStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list active activities: %w", err)
	}
	if len(activities) == 0 {
		return nil, nil
	}
	return &activities[0], nil
}

// ListParticipants 获取活动参与者列表
func (s *LotteryService) ListParticipants(ctx context.Context, activityID int64, params pagination.PaginationParams) ([]LotteryParticipant, *pagination.PaginationResult, error) {
	return s.participantRepo.ListByActivity(ctx, activityID, params)
}

// ListUserParticipations 获取用户参与记录
func (s *LotteryService) ListUserParticipations(ctx context.Context, userID int64, params pagination.PaginationParams) ([]LotteryParticipant, *pagination.PaginationResult, error) {
	return s.participantRepo.ListByUser(ctx, userID, params)
}

// ListUserCoupons 获取用户优惠券列表
func (s *LotteryService) ListUserCoupons(ctx context.Context, userID int64, status string, params pagination.PaginationParams) ([]LotteryCoupon, *pagination.PaginationResult, error) {
	return s.couponRepo.ListByUser(ctx, userID, status, params)
}

// generateShareCode 生成8字节随机十六进制分享码（16字符）
func generateShareCode() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
