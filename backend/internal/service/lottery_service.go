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
	"github.com/redis/go-redis/v9"
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
	settingService  *SettingService
	accountRepo     AccountRepository
	entClient       *dbent.Client
	redisClient     *redis.Client
}

// NewLotteryService 创建抽奖服务实例
func NewLotteryService(
	activityRepo LotteryActivityRepository,
	participantRepo LotteryParticipantRepository,
	couponRepo LotteryCouponRepository,
	groupService *GroupService,
	settingService *SettingService,
	accountRepo AccountRepository,
	entClient *dbent.Client,
	redisClient *redis.Client,
) *LotteryService {
	return &LotteryService{
		activityRepo:    activityRepo,
		participantRepo: participantRepo,
		couponRepo:      couponRepo,
		groupService:    groupService,
		settingService:  settingService,
		accountRepo:     accountRepo,
		entClient:       entClient,
		redisClient:     redisClient,
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
	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 3 // 默认 3 天
	}
	activityEndAt := drawAt.Add(time.Duration(validityDays) * 24 * time.Hour)

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

	// 每日限额默认 20 美元
	dailyLimit := input.DailyLimitUSD
	if dailyLimit <= 0 {
		dailyLimit = 20
	}

	// 创建订阅类型临时分组（名称含活动标题避免重复）
	groupName := fmt.Sprintf("ClaudeMax%d天体验-%s", validityDays, shareCode[:8])
	groupDesc := fmt.Sprintf("抽奖活动临时分组: %s", input.Title)
	group, err := s.groupService.Create(ctx, CreateGroupRequest{
		Name:             groupName,
		Description:      groupDesc,
		IsExclusive:      false,
		RateMultiplier:   1.0,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &dailyLimit,
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
		// 活动入库失败，清理已创建的临时分组
		deleteTempGroup(ctx, s.groupService, &group.ID, 0)
		return nil, fmt.Errorf("create lottery activity: %w", err)
	}

	// 开启强制绑定邮箱（活动期间全局生效）。
	// 使用同一 Redis 锁防止与 syncForceEmailBindOff 竞态。
	// 即使获锁失败，由于活动已入库，syncForceEmailBindOff 会查到此活动而不会关闭开关，
	// 故仍可安全设置 ForceEmailBind(true)。
	acquired := acquireForceEmailBindLock(ctx, s.redisClient)
	if err := s.settingService.SetForceEmailBind(ctx, true); err != nil {
		log.Printf("[Lottery] Failed to enable force email bind for activity %d: %v", activity.ID, err)
	}
	if acquired {
		releaseForceEmailBindLock(ctx, s.redisClient)
	}

	log.Printf("[Lottery] Created activity %d (share_code=%s, draw_at=%s)", activity.ID, shareCode, drawAt.Format(time.RFC3339))

	// 将指定的过期账号追加到临时分组（不影响账号已有的其他分组）
	if len(input.AccountIDs) > 0 {
		boundCount := 0
		for _, accountID := range input.AccountIDs {
			if err := s.accountRepo.AddToGroup(ctx, accountID, group.ID, 0); err != nil {
				log.Printf("[Lottery] Warning: failed to add account %d to temp group %d: %v", accountID, group.ID, err)
				continue
			}
			boundCount++
		}
		log.Printf("[Lottery] Added %d/%d accounts to temp group %d for activity %d",
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

	// 禁用临时分组
	deleteTempGroup(ctx, s.groupService, activity.TempGroupID, id)

	// 若无其他活跃活动，关闭强制绑定邮箱开关
	syncForceEmailBindOff(ctx, s.settingService, s.activityRepo, s.redisClient)

	log.Printf("[Lottery] Cancelled activity %d", id)

	return nil
}

// GetActiveActivity 获取当前活跃的抽奖活动（返回第一个，内部使用）
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

// ListActiveActivities 获取所有活跃的抽奖活动
func (s *LotteryService) ListActiveActivities(ctx context.Context) ([]LotteryActivity, error) {
	activities, err := s.activityRepo.ListByStatus(ctx, LotteryStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list active activities: %w", err)
	}
	return activities, nil
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

const forceEmailBindLockKey = "lottery_force_email_bind_sync"

// acquireForceEmailBindLock 获取强制绑定邮箱开关的分布式锁（自旋等待，最多 5 秒）。
// 返回 true 表示成功获取锁，false 表示超时或 Redis 错误。
// 无 Redis 时始终返回 true（单机模式，跳过互斥）。
func acquireForceEmailBindLock(ctx context.Context, redisClient *redis.Client) bool {
	if redisClient == nil {
		return true
	}
	for i := 0; i < 10; i++ {
		locked, err := redisClient.SetNX(ctx, forceEmailBindLockKey, "1", 10*time.Second).Result()
		if err != nil {
			log.Printf("[Lottery] Redis lock error for email bind sync: %v", err)
			return false
		}
		if locked {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Printf("[Lottery] Failed to acquire email bind lock after retries")
	return false
}

// releaseForceEmailBindLock 释放强制绑定邮箱开关的分布式锁
func releaseForceEmailBindLock(ctx context.Context, redisClient *redis.Client) {
	if redisClient == nil {
		return
	}
	redisClient.Del(ctx, forceEmailBindLockKey)
}

// syncForceEmailBindOff 原子检查是否还有活跃活动，若无则关闭强制绑定邮箱开关。
// 使用 Redis 分布式锁防止与 CreateActivity 的 SetForceEmailBind(true) 产生 TOCTOU 竞态。
// 用于活动取消、开奖完成、活动过期等场景。
func syncForceEmailBindOff(ctx context.Context, settingService *SettingService, activityRepo LotteryActivityRepository, redisClient *redis.Client) {
	if !acquireForceEmailBindLock(ctx, redisClient) {
		log.Printf("[Lottery] Skipping email bind sync: could not acquire lock")
		return
	}
	defer releaseForceEmailBindLock(ctx, redisClient)

	syncForceEmailBindOffNoLock(ctx, settingService, activityRepo)
}

func syncForceEmailBindOffNoLock(ctx context.Context, settingService *SettingService, activityRepo LotteryActivityRepository) {
	// 同时检查 active 和 drawing 两个状态：开奖过程中活动处于 drawing，
	// 若仅查 active 会误判为"无活跃活动"而错误关闭强制绑定邮箱开关。
	for _, status := range []string{LotteryStatusActive, LotteryStatusDrawing} {
		activities, err := activityRepo.ListByStatus(ctx, status)
		if err != nil {
			log.Printf("[Lottery] Failed to check %s activities for email bind switch: %v", status, err)
			return
		}
		if len(activities) > 0 {
			return // 仍有进行中的活动，不关闭开关
		}
	}
	if err := settingService.SetForceEmailBind(ctx, false); err != nil {
		log.Printf("[Lottery] Failed to disable force email bind: %v", err)
	} else {
		log.Printf("[Lottery] Disabled force email bind (no active/drawing activities)")
	}
}

// deleteTempGroup 删除活动关联的临时分组（先解除账号绑定，再删除分组）
func deleteTempGroup(ctx context.Context, groupService *GroupService, tempGroupID *int64, activityID int64) {
	if tempGroupID == nil || groupService == nil {
		return
	}
	n, err := groupService.DeleteWithBindings(ctx, *tempGroupID)
	if err != nil {
		log.Printf("[Lottery] Failed to delete temp group %d for activity %d: %v", *tempGroupID, activityID, err)
		return
	}
	if n > 0 {
		log.Printf("[Lottery] Cleared %d account binding(s) from temp group %d (activity %d)", n, *tempGroupID, activityID)
	}
	log.Printf("[Lottery] Deleted temp group %d for activity %d", *tempGroupID, activityID)
}

// generateShareCode 生成8字节随机十六进制分享码（16字符）
func generateShareCode() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
