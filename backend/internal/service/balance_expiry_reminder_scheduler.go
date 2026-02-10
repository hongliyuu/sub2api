package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// BalanceExpiryReminderScheduler 余额过期提醒定时任务
type BalanceExpiryReminderScheduler struct {
	lotRepo        BalanceLotRepository
	userRepo       UserRepository
	emailService   *EmailService
	settingService *SettingService
	redisClient    *redis.Client

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewBalanceExpiryReminderScheduler 创建余额过期提醒调度器
func NewBalanceExpiryReminderScheduler(
	lotRepo BalanceLotRepository,
	userRepo UserRepository,
	emailService *EmailService,
	settingService *SettingService,
	redisClient *redis.Client,
) *BalanceExpiryReminderScheduler {
	return &BalanceExpiryReminderScheduler{
		lotRepo:        lotRepo,
		userRepo:       userRepo,
		emailService:   emailService,
		settingService: settingService,
		redisClient:    redisClient,
		stopCh:         make(chan struct{}),
	}
}

// Start 启动调度器
func (s *BalanceExpiryReminderScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("[BalanceExpiryReminder] Started")
}

// Stop 停止调度器
func (s *BalanceExpiryReminderScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
	log.Printf("[BalanceExpiryReminder] Stopped")
}

func (s *BalanceExpiryReminderScheduler) run() {
	defer s.wg.Done()

	// 延迟 5 分钟首次执行
	select {
	case <-time.After(5 * time.Minute):
	case <-s.stopCh:
		return
	}

	s.checkAndNotify()

	// 每小时执行一次
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndNotify()
		}
	}
}

func (s *BalanceExpiryReminderScheduler) checkAndNotify() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 检查是否启用过期提醒
	enabled, err := s.settingService.GetSettingValue(ctx, SettingKeyBalanceExpiryReminderEnabled)
	if err != nil || enabled != "true" {
		return
	}

	// 获取提前提醒天数
	advanceDays := 3
	if raw, err := s.settingService.GetSettingValue(ctx, SettingKeyBalanceExpiryReminderAdvanceDays); err == nil && raw != "" {
		if d, err := strconv.Atoi(raw); err == nil && d > 0 {
			advanceDays = d
		}
	}

	// 查询即将过期的批次
	lots, err := s.lotRepo.ListExpiringSoon(ctx, advanceDays)
	if err != nil {
		log.Printf("[BalanceExpiryReminder] Failed to list expiring lots: %v", err)
		return
	}

	if len(lots) == 0 {
		return
	}

	// 按 user_id 聚合
	type userExpiryInfo struct {
		totalAmount    float64
		earliestExpiry time.Time
		lots           []*BalanceLot
	}
	userInfoMap := make(map[int64]*userExpiryInfo)

	for _, lot := range lots {
		info, ok := userInfoMap[lot.UserID]
		if !ok {
			info = &userExpiryInfo{earliestExpiry: lot.ExpiresAt}
			userInfoMap[lot.UserID] = info
		}
		info.totalAmount += lot.RemainingAmount
		info.lots = append(info.lots, lot)
		if lot.ExpiresAt.Before(info.earliestExpiry) {
			info.earliestExpiry = lot.ExpiresAt
		}
	}

	// 获取站点名称
	siteName, _ := s.settingService.GetSettingValue(ctx, SettingKeySiteName)
	if siteName == "" {
		siteName = "Sub2API"
	}

	notifiedCount := 0
	for userID, info := range userInfoMap {
		// Redis 去重：每天每用户只发一次
		dedupeKey := fmt.Sprintf("balance_expiry_reminder:%d:%s", userID, time.Now().Format("2006-01-02"))
		ok, err := s.redisClient.SetNX(ctx, dedupeKey, "1", 25*time.Hour).Result()
		if err != nil || !ok {
			continue
		}

		// 获取用户邮箱
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user.Email == "" {
			continue
		}

		// 构建并发送邮件
		body := s.buildEmailBody(siteName, user.Email, info.totalAmount, info.earliestExpiry, info.lots)
		subject := fmt.Sprintf("[%s] 余额即将过期提醒", siteName)

		if err := s.emailService.SendEmail(ctx, user.Email, subject, body); err != nil {
			log.Printf("[BalanceExpiryReminder] Failed to send email to user %d (%s): %v", userID, user.Email, err)
			// 发送失败时删除去重键，允许下次重试
			s.redisClient.Del(ctx, dedupeKey)
			continue
		}

		notifiedCount++
	}

	if notifiedCount > 0 {
		log.Printf("[BalanceExpiryReminder] Sent %d expiry reminder emails", notifiedCount)
	}
}

// buildEmailBody 构建过期提醒邮件 HTML 内容
func (s *BalanceExpiryReminderScheduler) buildEmailBody(siteName, email string, totalAmount float64, earliestExpiry time.Time, lots []*BalanceLot) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head><body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">`)

	// 标题
	b.WriteString(fmt.Sprintf(`<h2 style="color: #e65100;">%s - 余额即将过期提醒</h2>`, siteName))

	// 概要
	b.WriteString(fmt.Sprintf(`<p>您好，%s：</p>`, email))
	b.WriteString(fmt.Sprintf(`<p>您有 <strong style="color: #e65100;">%.2f</strong> 元余额即将过期，最早过期时间为 <strong>%s</strong>。</p>`,
		totalAmount, earliestExpiry.Format("2006-01-02 15:04")))

	// 批次明细表格
	b.WriteString(`<table style="width: 100%; border-collapse: collapse; margin: 16px 0;">`)
	b.WriteString(`<tr style="background: #f5f5f5;">`)
	b.WriteString(`<th style="padding: 8px; border: 1px solid #ddd; text-align: left;">来源</th>`)
	b.WriteString(`<th style="padding: 8px; border: 1px solid #ddd; text-align: right;">剩余金额</th>`)
	b.WriteString(`<th style="padding: 8px; border: 1px solid #ddd; text-align: center;">过期时间</th>`)
	b.WriteString(`</tr>`)

	sourceTypeNames := map[string]string{
		BalanceLotSourceRecharge:  "充值",
		BalanceLotSourceRedeem:    "兑换码",
		BalanceLotSourcePromo:     "优惠码",
		BalanceLotSourceAdjust:    "管理员调整",
		BalanceLotSourceMigration: "历史迁移",
	}

	for _, lot := range lots {
		sourceName := sourceTypeNames[lot.SourceType]
		if sourceName == "" {
			sourceName = lot.SourceType
		}
		b.WriteString(fmt.Sprintf(`<tr>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			<td style="padding: 8px; border: 1px solid #ddd; text-align: right;">%.2f</td>
			<td style="padding: 8px; border: 1px solid #ddd; text-align: center;">%s</td>
		</tr>`, sourceName, lot.RemainingAmount, lot.ExpiresAt.Format("2006-01-02 15:04")))
	}

	b.WriteString(`</table>`)

	// 提示
	b.WriteString(`<p style="color: #666;">过期后未消费的余额将被自动清零，请及时使用。</p>`)
	b.WriteString(fmt.Sprintf(`<p style="color: #999; font-size: 12px;">此邮件由 %s 系统自动发送，请勿回复。</p>`, siteName))

	b.WriteString(`</body></html>`)

	return b.String()
}
