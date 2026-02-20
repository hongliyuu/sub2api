package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LotteryScheduler 抽奖定时任务调度器
// 每小时检查：
// 0. 卡在 drawing 状态超过 10 分钟的活动 → 重置为 active 以便重试
// 1. 到达 draw_at 的活动 → 触发自动开奖
// 2. 到达 activity_end_at 的活动 → 过期优惠券 + 禁用临时分组 + 同步强制邮箱绑定开关
// 3. 距 activity_end_at ≤ 24 小时 → 提醒持有未使用优惠券的中奖用户
type LotteryScheduler struct {
	drawService    *LotteryDrawService
	activityRepo   LotteryActivityRepository
	couponRepo     LotteryCouponRepository
	groupService   *GroupService
	emailService   *EmailService
	settingService *SettingService
	userRepo       UserRepository
	redisClient    *redis.Client

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewLotteryScheduler creates a new scheduler instance.
func NewLotteryScheduler(
	drawService *LotteryDrawService,
	activityRepo LotteryActivityRepository,
	couponRepo LotteryCouponRepository,
	groupService *GroupService,
	emailService *EmailService,
	settingService *SettingService,
	userRepo UserRepository,
	redisClient *redis.Client,
) *LotteryScheduler {
	return &LotteryScheduler{
		drawService:    drawService,
		activityRepo:   activityRepo,
		couponRepo:     couponRepo,
		groupService:   groupService,
		emailService:   emailService,
		settingService: settingService,
		userRepo:       userRepo,
		redisClient:    redisClient,
		stopCh:         make(chan struct{}),
	}
}

// Start begins the background check loop.
func (s *LotteryScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("[LotteryScheduler] Started")
}

// Stop gracefully stops the scheduler.
func (s *LotteryScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
	log.Printf("[LotteryScheduler] Stopped")
}

func (s *LotteryScheduler) run() {
	defer s.wg.Done()

	// Delay first check by 3 minutes after startup
	select {
	case <-time.After(3 * time.Minute):
	case <-s.stopCh:
		return
	}

	s.checkAll()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAll()
		}
	}
}

func (s *LotteryScheduler) checkAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s.checkStuckDrawing(ctx)
	s.checkAutoDraw(ctx)
	s.checkExpiredActivities(ctx)
	s.checkCouponExpiryReminder(ctx)
}

// checkStuckDrawing 恢复卡在 drawing 状态超过 10 分钟的活动
// 当 ExecuteDraw 中途失败时，活动可能永久卡在 drawing 状态
// 此方法将其重置为 active，允许下次调度重试开奖
func (s *LotteryScheduler) checkStuckDrawing(ctx context.Context) {
	activities, err := s.activityRepo.ListByStatus(ctx, LotteryStatusDrawing)
	if err != nil {
		log.Printf("[LotteryScheduler] Error listing drawing activities: %v", err)
		return
	}

	for _, activity := range activities {
		// 仅恢复卡住超过 10 分钟的活动（正常开奖不会超过这个时间）
		if time.Since(activity.UpdatedAt) < 10*time.Minute {
			continue
		}

		log.Printf("[LotteryScheduler] Recovering stuck activity %d (stuck in drawing since %s)",
			activity.ID, activity.UpdatedAt.Format(time.RFC3339))

		a := activity
		a.Status = LotteryStatusActive
		if err := s.activityRepo.Update(ctx, &a); err != nil {
			log.Printf("[LotteryScheduler] Failed to recover stuck activity %d: %v", activity.ID, err)
		}
	}
}

// checkAutoDraw 检查并自动开奖到达 draw_at 的活动
func (s *LotteryScheduler) checkAutoDraw(ctx context.Context) {
	// Redis dedup: 每小时最多执行一次
	redisKey := fmt.Sprintf("lottery_auto_draw_check:%s", time.Now().Format("2006-01-02-15"))
	isFirst, err := s.redisClient.SetNX(ctx, redisKey, "1", 2*time.Hour).Result()
	if err != nil {
		log.Printf("[LotteryScheduler] Redis SetNX error for auto draw: %v", err)
		return
	}
	if !isFirst {
		return
	}

	activities, err := s.activityRepo.ListActiveForDraw(ctx)
	if err != nil {
		log.Printf("[LotteryScheduler] Error listing activities for draw: %v", err)
		s.redisClient.Del(ctx, redisKey)
		return
	}

	for _, activity := range activities {
		log.Printf("[LotteryScheduler] Auto-drawing activity %d: %s", activity.ID, activity.Title)
		result, err := s.drawService.ExecuteDraw(ctx, activity.ID)
		if err != nil {
			log.Printf("[LotteryScheduler] Failed to draw activity %d: %v", activity.ID, err)
			continue
		}
		log.Printf("[LotteryScheduler] Activity %d drawn: %d participants, %d winners",
			activity.ID, result.TotalParticipants, result.WinnerCount)
	}
}

// checkExpiredActivities 检查已过期的活动，过期优惠券和禁用临时分组
func (s *LotteryScheduler) checkExpiredActivities(ctx context.Context) {
	activities, err := s.activityRepo.ListExpired(ctx)
	if err != nil {
		log.Printf("[LotteryScheduler] Error listing expired activities: %v", err)
		return
	}

	for _, activity := range activities {
		log.Printf("[LotteryScheduler] Processing expired activity %d: %s", activity.ID, activity.Title)

		// 过期所有未使用的优惠券
		if err := s.couponRepo.ExpireByActivity(ctx, activity.ID); err != nil {
			log.Printf("[LotteryScheduler] Failed to expire coupons for activity %d: %v", activity.ID, err)
		}

		// 禁用临时分组
		deleteTempGroup(ctx, s.groupService, activity.TempGroupID, activity.ID)

		// 更新活动状态为 expired（正常结束后过期，区别于人为 cancelled）
		a := activity
		a.Status = LotteryStatusExpired
		if err := s.activityRepo.Update(ctx, &a); err != nil {
			log.Printf("[LotteryScheduler] Failed to update expired activity %d: %v", activity.ID, err)
		}
	}

	// 若无其他活跃活动，关闭强制绑定邮箱开关
	if len(activities) > 0 {
		syncForceEmailBindOff(ctx, s.settingService, s.activityRepo, s.redisClient)
	}
}

// checkCouponExpiryReminder 检查即将过期的优惠券并发送提醒
func (s *LotteryScheduler) checkCouponExpiryReminder(ctx context.Context) {
	// Redis dedup
	redisKey := fmt.Sprintf("lottery_coupon_reminder:%s", time.Now().Format("2006-01-02-15"))
	isFirst, err := s.redisClient.SetNX(ctx, redisKey, "1", 2*time.Hour).Result()
	if err != nil {
		log.Printf("[LotteryScheduler] Redis SetNX error for coupon reminder: %v", err)
		return
	}
	if !isFirst {
		return
	}

	// 查找 24 小时内过期且未提醒的优惠券
	coupons, err := s.couponRepo.ListUnremindedExpiringSoon(ctx, 24)
	if err != nil {
		log.Printf("[LotteryScheduler] Error listing expiring coupons: %v", err)
		s.redisClient.Del(ctx, redisKey)
		return
	}

	if len(coupons) == 0 {
		return
	}

	siteName := s.settingService.GetSiteName(ctx)

	for _, coupon := range coupons {
		user, err := s.userRepo.GetByID(ctx, coupon.UserID)
		if err != nil {
			log.Printf("[LotteryScheduler] Failed to get user %d: %v", coupon.UserID, err)
			continue
		}
		if user.Email == "" {
			continue
		}

		subject := fmt.Sprintf("[%s] 您的抽奖优惠券即将过期", siteName)
		body := s.buildCouponReminderEmail(siteName, user, &coupon)

		if err := s.emailService.SendEmail(ctx, user.Email, subject, body); err != nil {
			log.Printf("[LotteryScheduler] Failed to send coupon reminder to %s: %v", user.Email, err)
			continue
		}

		if err := s.couponRepo.MarkReminded(ctx, coupon.ID); err != nil {
			log.Printf("[LotteryScheduler] Failed to mark coupon %d as reminded: %v", coupon.ID, err)
		}
	}

	log.Printf("[LotteryScheduler] Sent %d coupon expiry reminders", len(coupons))
}

func (s *LotteryScheduler) buildCouponReminderEmail(siteName string, user *User, coupon *LotteryCoupon) string {
	var couponDesc string
	if coupon.CouponType == LotteryCouponTypeWinnerDiscount {
		if coupon.DiscountPercent != nil {
			couponDesc = fmt.Sprintf("充值 %d 折券", *coupon.DiscountPercent)
		}
	} else {
		if coupon.ReductionAmount != nil {
			couponDesc = fmt.Sprintf("包月订阅 ¥%.0f 立减券", *coupon.ReductionAmount)
		}
	}

	expiresAt := coupon.ExpiresAt.Format("2006-01-02 15:04")

	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h2>%s - 优惠券即将过期提醒</h2>
<p>亲爱的 %s，</p>
<p>您的抽奖 <strong>%s</strong> 将于 <strong>%s</strong> 过期，请尽快使用！</p>
<p style="color:#e53e3e;font-weight:bold;">过期后优惠券将无法使用。</p>
<p style="color:#666;margin-top:16px;">此邮件由系统自动发送，请勿回复。</p>
</body></html>`, siteName, user.Username, couponDesc, expiresAt)
}
