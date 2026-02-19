package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/redis/go-redis/v9"
)

// LotteryDrawService 抽奖开奖服务
type LotteryDrawService struct {
	activityRepo      LotteryActivityRepository
	participantRepo   LotteryParticipantRepository
	couponRepo        LotteryCouponRepository
	userRepo          UserRepository
	rechargeOrderRepo RechargeOrderRepository
	userSubRepo       UserSubscriptionRepository
	emailService      *EmailService
	settingService    *SettingService
	redisClient       *redis.Client
	entClient         *dbent.Client
}

// NewLotteryDrawService 创建抽奖开奖服务实例
func NewLotteryDrawService(
	activityRepo LotteryActivityRepository,
	participantRepo LotteryParticipantRepository,
	couponRepo LotteryCouponRepository,
	userRepo UserRepository,
	rechargeOrderRepo RechargeOrderRepository,
	userSubRepo UserSubscriptionRepository,
	emailService *EmailService,
	settingService *SettingService,
	redisClient *redis.Client,
	entClient *dbent.Client,
) *LotteryDrawService {
	return &LotteryDrawService{
		activityRepo:      activityRepo,
		participantRepo:   participantRepo,
		couponRepo:        couponRepo,
		userRepo:          userRepo,
		rechargeOrderRepo: rechargeOrderRepo,
		userSubRepo:       userSubRepo,
		emailService:      emailService,
		settingService:    settingService,
		redisClient:       redisClient,
		entClient:         entClient,
	}
}

// Participate 用户参与抽奖活动
func (s *LotteryDrawService) Participate(ctx context.Context, activityID, userID int64) (*LotteryParticipant, error) {
	// 1. 获取活动并验证状态（仅允许 active 状态参与，drawing/completed/cancelled 均拒绝）
	activity, err := s.activityRepo.GetByID(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if activity.Status != LotteryStatusActive {
		return nil, ErrLotteryActivityNotActive
	}

	// 2. 获取用户并验证邮箱
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Email == "" ||
		strings.HasSuffix(user.Email, LinuxDoConnectSyntheticEmailDomain) ||
		strings.HasSuffix(user.Email, WeChatSyntheticEmailDomain) {
		return nil, ErrLotteryEmailRequired
	}

	// 3. 检查是否已经参与过
	existing, err := s.participantRepo.GetByActivityAndUser(ctx, activityID, userID)
	if err != nil && err != ErrLotteryParticipantNotFound {
		return nil, fmt.Errorf("check participation: %w", err)
	}
	if existing != nil {
		return nil, ErrLotteryAlreadyParticipated
	}

	// 4. 分类用户
	category := s.classifyUser(ctx, user, activity.ActivityStartAt)

	// 5. 获取权重倍率
	weightMap := activity.WeightConfigMap()
	weightMultiplier := weightMap[category]
	if weightMultiplier == 0 {
		weightMultiplier = 1.0
	}

	// 6. 创建参与记录
	participant := &LotteryParticipant{
		ActivityID:       activityID,
		UserID:           userID,
		UserCategory:     category,
		WeightMultiplier: weightMultiplier,
		ParticipatedAt:   time.Now(),
	}

	if err := s.participantRepo.Create(ctx, participant); err != nil {
		return nil, fmt.Errorf("create participant: %w", err)
	}

	// 7. 增加活动参与人数
	if err := s.activityRepo.IncrementParticipantCount(ctx, activityID); err != nil {
		log.Printf("[LotteryDraw] Failed to increment participant count for activity %d: %v", activityID, err)
	}

	log.Printf("[LotteryDraw] User %d participated in activity %d (category=%s, weight=%.2f)",
		userID, activityID, category, weightMultiplier)

	return participant, nil
}

// classifyUser 根据用户特征分类
func (s *LotteryDrawService) classifyUser(ctx context.Context, user *User, activityStartAt time.Time) string {
	// 活动开始后注册的新用户
	if user.CreatedAt.After(activityStartAt) {
		return LotteryUserCategoryNewUser
	}

	// 有活跃订阅的用户
	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, user.ID)
	if err == nil && len(activeSubs) > 0 {
		return LotteryUserCategorySubscriber
	}

	// 有付费充值订单的用户
	result, err := s.rechargeOrderRepo.ListByUserID(ctx, user.ID, &ListRechargeOrdersRequest{
		PaginationParams: pagination.PaginationParams{Page: 1, PageSize: 1},
		Status:           "paid",
	})
	if err == nil && result != nil && len(result.Orders) > 0 {
		return LotteryUserCategoryPaid
	}

	// 普通用户
	return LotteryUserCategoryRegular
}

// ExecuteDraw 执行开奖
func (s *LotteryDrawService) ExecuteDraw(ctx context.Context, activityID int64) (*LotteryDrawResult, error) {
	// 1. 获取活动并验证状态
	activity, err := s.activityRepo.GetByID(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if activity.Status != LotteryStatusActive {
		return nil, ErrLotteryActivityNotActive
	}

	// 2. 获取 Redis 分布式锁
	lockKey := fmt.Sprintf("lottery_draw:%d", activityID)
	locked, err := s.redisClient.SetNX(ctx, lockKey, "1", 5*time.Minute).Result()
	if err != nil {
		log.Printf("[LotteryDraw] Redis SetNX error for activity %d: %v", activityID, err)
		return nil, fmt.Errorf("acquire draw lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("lottery draw is already in progress for activity %d", activityID)
	}
	defer func() {
		if err := s.redisClient.Del(ctx, lockKey).Err(); err != nil {
			log.Printf("[LotteryDraw] Failed to release lock for activity %d: %v", activityID, err)
		}
	}()

	// 3. 更新活动状态为 drawing
	activity.Status = LotteryStatusDrawing
	if err := s.activityRepo.Update(ctx, activity); err != nil {
		return nil, fmt.Errorf("update activity status to drawing: %w", err)
	}

	// 若后续出错，回滚状态为 active（成功路径会覆盖为 completed/cancelled）
	drawSuccess := false
	defer func() {
		if !drawSuccess {
			log.Printf("[LotteryDraw] Rolling back activity %d from drawing to active", activityID)
			activity.Status = LotteryStatusActive
			if rollbackErr := s.activityRepo.Update(ctx, activity); rollbackErr != nil {
				log.Printf("[LotteryDraw] Failed to rollback activity %d: %v", activityID, rollbackErr)
			}
		}
	}()

	// 4. 获取所有参与者
	participants, err := s.participantRepo.ListAllByActivity(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}

	// 5. 检查是否低于最低参与人数
	belowMin := len(participants) < activity.MinParticipants

	// 人数不足：取消活动，不开奖不发券
	if belowMin {
		activity.Status = LotteryStatusCancelled
		if err := s.activityRepo.Update(ctx, activity); err != nil {
			log.Printf("[LotteryDraw] Failed to cancel under-min activity %d: %v", activityID, err)
		}
		drawSuccess = true // 状态已更新为 cancelled，无需回滚

		log.Printf("[LotteryDraw] Activity %d cancelled: participants=%d < min=%d",
			activityID, len(participants), activity.MinParticipants)

		return &LotteryDrawResult{
			ActivityID:          activityID,
			TotalParticipants:   len(participants),
			WinnerCount:         0,
			BelowMinParticipant: true,
		}, nil
	}

	// 6. 开奖：为每个参与者计算是否中奖
	drawAt := time.Now()
	couponExpiresAt := drawAt.Add(3 * 24 * time.Hour) // 统一过期时间：实际开奖时间 + 3 天
	winnerCount := 0

	for i := range participants {
		p := &participants[i]

		// 计算有效中奖率 = 基础中奖率 * 权重倍率
		effectiveRate := activity.BaseWinRate * p.WeightMultiplier

		// 使用 crypto/rand 生成随机数
		randomVal := cryptoRandFloat64()

		isWinner := randomVal < effectiveRate
		if isWinner {
			winnerCount++
		}

		// 7. 创建优惠券
		var coupon *LotteryCoupon
		if isWinner {
			discountPercent := activity.WinnerDiscountPercent
			coupon = &LotteryCoupon{
				ActivityID:      activityID,
				UserID:          p.UserID,
				CouponType:      LotteryCouponTypeWinnerDiscount,
				DiscountPercent: &discountPercent,
				ApplicableScope: LotteryCouponScopeRecharge,
				Status:          LotteryCouponStatusActive,
				ExpiresAt:       couponExpiresAt,
			}
		} else {
			reductionAmount := activity.LoserCouponAmount
			coupon = &LotteryCoupon{
				ActivityID:      activityID,
				UserID:          p.UserID,
				CouponType:      LotteryCouponTypeLoserReduction,
				ReductionAmount: &reductionAmount,
				ApplicableScope: LotteryCouponScopeMonthlySubscription,
				Status:          LotteryCouponStatusActive,
				ExpiresAt:       couponExpiresAt,
			}
		}

		if err := s.couponRepo.Create(ctx, coupon); err != nil {
			log.Printf("[LotteryDraw] Failed to create coupon for user %d in activity %d: %v",
				p.UserID, activityID, err)
			continue
		}

		// 8. 更新参与者的中奖状态和优惠券 ID
		if err := s.participantRepo.UpdateWinnerStatus(ctx, p.ID, isWinner, &coupon.ID); err != nil {
			log.Printf("[LotteryDraw] Failed to update winner status for participant %d: %v", p.ID, err)
		}
	}

	// 9. 更新活动开奖结果（统一使用实际开奖时间 + 3 天作为 activity_end_at）
	if err := s.activityRepo.UpdateDrawResult(ctx, activityID, winnerCount, couponExpiresAt); err != nil {
		log.Printf("[LotteryDraw] Failed to update draw result for activity %d: %v", activityID, err)
		return nil, fmt.Errorf("update draw result: %w", err)
	}
	drawSuccess = true // 状态已更新为 completed，无需回滚

	log.Printf("[LotteryDraw] Draw completed for activity %d: total=%d, winners=%d, below_min=%v",
		activityID, len(participants), winnerCount, belowMin)

	// 10. 异步发送通知邮件
	go s.sendDrawResultEmails(activityID)

	return &LotteryDrawResult{
		ActivityID:          activityID,
		TotalParticipants:   len(participants),
		WinnerCount:         winnerCount,
		BelowMinParticipant: belowMin,
	}, nil
}

// sendDrawResultEmails 异步发送开奖结果通知邮件
func (s *LotteryDrawService) sendDrawResultEmails(activityID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	activity, err := s.activityRepo.GetByID(ctx, activityID)
	if err != nil {
		log.Printf("[LotteryDraw] sendDrawResultEmails: failed to get activity %d: %v", activityID, err)
		return
	}

	participants, err := s.participantRepo.ListAllByActivity(ctx, activityID)
	if err != nil {
		log.Printf("[LotteryDraw] sendDrawResultEmails: failed to list participants for activity %d: %v", activityID, err)
		return
	}

	for _, p := range participants {
		user, err := s.userRepo.GetByID(ctx, p.UserID)
		if err != nil {
			log.Printf("[LotteryDraw] sendDrawResultEmails: failed to get user %d: %v", p.UserID, err)
			continue
		}

		if user.Email == "" ||
			strings.HasSuffix(user.Email, LinuxDoConnectSyntheticEmailDomain) ||
			strings.HasSuffix(user.Email, WeChatSyntheticEmailDomain) {
			continue
		}

		var subject, body string
		if p.IsWinner != nil && *p.IsWinner {
			subject = fmt.Sprintf("恭喜中奖 - %s", activity.Title)
			body = fmt.Sprintf(
				"<h2>恭喜您在「%s」抽奖活动中中奖！</h2>"+
					"<p>您获得了 <strong>%d%% 折扣优惠券</strong>，可用于充值订单。</p>"+
					"<p>优惠券有效期至：%s</p>"+
					"<p>请尽快使用，过期将失效。</p>",
				activity.Title,
				activity.WinnerDiscountPercent,
				activity.ActivityEndAt.Format("2006-01-02 15:04"),
			)
		} else {
			subject = fmt.Sprintf("感谢参与 - %s", activity.Title)
			body = fmt.Sprintf(
				"<h2>感谢您参与「%s」抽奖活动</h2>"+
					"<p>虽然本次未中奖，但我们为您准备了 <strong>%.0f 元减免优惠券</strong>，可用于月度订阅。</p>"+
					"<p>优惠券有效期至：%s</p>"+
					"<p>请尽快使用，过期将失效。</p>",
				activity.Title,
				activity.LoserCouponAmount,
				activity.ActivityEndAt.Format("2006-01-02 15:04"),
			)
		}

		if err := s.emailService.SendEmail(ctx, user.Email, subject, body); err != nil {
			log.Printf("[LotteryDraw] sendDrawResultEmails: failed to send email to %s: %v", user.Email, err)
		}
	}

	log.Printf("[LotteryDraw] sendDrawResultEmails: completed for activity %d, processed %d participants",
		activityID, len(participants))
}

// cryptoRandFloat64 使用 crypto/rand 生成 [0, 1) 范围内的随机浮点数
func cryptoRandFloat64() float64 {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	n := binary.BigEndian.Uint64(buf[:])
	return float64(n) / float64(^uint64(0))
}
