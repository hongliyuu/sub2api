package service

import (
	"context"
	"fmt"
	"html"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/redis/go-redis/v9"
)

// BalanceLotService 余额批次服务
type BalanceLotService struct {
	lotRepo        BalanceLotRepository
	userRepo       UserRepository
	balanceLogRepo BalanceLogRepository
	billingCache   *BillingCacheService
	settingService *SettingService
	entClient      *dbent.Client
	emailService   *EmailService
	redisClient    *redis.Client
}

// NewBalanceLotService 创建余额批次服务
func NewBalanceLotService(
	lotRepo BalanceLotRepository,
	userRepo UserRepository,
	balanceLogRepo BalanceLogRepository,
	billingCache *BillingCacheService,
	settingService *SettingService,
	entClient *dbent.Client,
	emailService *EmailService,
	redisClient *redis.Client,
) *BalanceLotService {
	return &BalanceLotService{
		lotRepo:        lotRepo,
		userRepo:       userRepo,
		balanceLogRepo: balanceLogRepo,
		billingCache:   billingCache,
		settingService: settingService,
		entClient:      entClient,
		emailService:   emailService,
		redisClient:    redisClient,
	}
}

// GetExpiryDays 获取配置的过期天数
func (s *BalanceLotService) GetExpiryDays(ctx context.Context) int {
	value, err := s.settingService.GetSettingValue(ctx, SettingKeyBalanceLotExpiryDays)
	if err != nil || value == "" {
		return 30 // 默认30天
	}
	days, err := strconv.Atoi(value)
	if err != nil || days <= 0 {
		return 30
	}
	return days
}

// CreateLot 创建余额批次
// 注意：user.balance 的增加由调用方负责（保持现有 UpdateBalance 逻辑不变）
func (s *BalanceLotService) CreateLot(ctx context.Context, userID int64, amount float64, sourceType string, sourceRef *string, description string) error {
	if amount <= 0 {
		return nil
	}

	days := s.GetExpiryDays(ctx)
	expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	lot := &BalanceLot{
		UserID:          userID,
		SourceType:      sourceType,
		SourceRef:       sourceRef,
		OriginalAmount:  amount,
		RemainingAmount: amount,
		Status:          BalanceLotStatusActive,
		ExpiresAt:       expiresAt,
		Description:     description,
	}

	if err := s.lotRepo.Create(ctx, lot); err != nil {
		return fmt.Errorf("create balance lot: %w", err)
	}

	return nil
}

// DeductFromLots 从用户余额批次中扣减（FEFO：先过期先扣）
// 在事务中完成：锁行 → 按过期时间顺序扣减 → 更新 user.balance
func (s *BalanceLotService) DeductFromLots(ctx context.Context, userID int64, amount float64) error {
	if amount <= 0 {
		return nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)

	// 获取活跃批次并加行锁
	lots, err := s.lotRepo.ListActiveByUserForUpdate(txCtx, userID)
	if err != nil {
		return fmt.Errorf("list active lots: %w", err)
	}

	remaining := amount
	for _, lot := range lots {
		if remaining <= 0 {
			break
		}

		deduct := math.Min(lot.RemainingAmount, remaining)
		lot.RemainingAmount -= deduct
		remaining -= deduct

		status := BalanceLotStatusActive
		if lot.RemainingAmount <= 0 {
			status = BalanceLotStatusDepleted
			lot.RemainingAmount = 0
		}

		if err := s.lotRepo.UpdateRemainingAndStatus(txCtx, lot.ID, lot.RemainingAmount, status); err != nil {
			return fmt.Errorf("update lot %d: %w", lot.ID, err)
		}
	}

	// 扣减 user.balance（与批次扣减在同一事务中）
	if err := s.userRepo.DeductBalance(txCtx, userID, amount); err != nil {
		return fmt.Errorf("deduct user balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// DeductFromLotsOnly 仅从余额批次中扣减，不修改 user.balance
// 用于管理员操作等已单独处理了 user.balance 的场景
func (s *BalanceLotService) DeductFromLotsOnly(ctx context.Context, userID int64, amount float64) error {
	if amount <= 0 {
		return nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)

	// 获取活跃批次并加行锁
	lots, err := s.lotRepo.ListActiveByUserForUpdate(txCtx, userID)
	if err != nil {
		return fmt.Errorf("list active lots: %w", err)
	}

	remaining := amount
	for _, lot := range lots {
		if remaining <= 0 {
			break
		}

		deduct := math.Min(lot.RemainingAmount, remaining)
		lot.RemainingAmount -= deduct
		remaining -= deduct

		status := BalanceLotStatusActive
		if lot.RemainingAmount <= 0 {
			status = BalanceLotStatusDepleted
			lot.RemainingAmount = 0
		}

		if err := s.lotRepo.UpdateRemainingAndStatus(txCtx, lot.ID, lot.RemainingAmount, status); err != nil {
			return fmt.Errorf("update lot %d: %w", lot.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ExpireLots 处理过期批次，由 Scheduler 调用
// 批量查询过期批次 → 按用户分组 → 事务中更新状态并重算余额 → 记录日志 → 失效缓存
func (s *BalanceLotService) ExpireLots(ctx context.Context) (int, error) {
	totalProcessed := 0
	failedLotIDs := make(map[int64]bool)

	for {
		// 每次处理一批
		lots, err := s.lotRepo.BatchExpire(ctx, 100)
		if err != nil {
			return totalProcessed, fmt.Errorf("batch expire: %w", err)
		}

		if len(lots) == 0 {
			break
		}

		// 过滤掉之前处理失败的 lot，避免无限循环
		var newLots []*BalanceLot
		for _, lot := range lots {
			if !failedLotIDs[lot.ID] {
				newLots = append(newLots, lot)
			}
		}
		if len(newLots) == 0 {
			break // 所有返回的 lot 都已尝试过且失败，跳出循环
		}

		// 按 user_id 分组
		userLots := make(map[int64][]*BalanceLot)
		for _, lot := range newLots {
			userLots[lot.UserID] = append(userLots[lot.UserID], lot)
		}

		// 逐用户处理
		for userID, uLots := range userLots {
			if err := s.expireUserLots(ctx, userID, uLots); err != nil {
				log.Printf("[BalanceLotService] Failed to expire lots for user %d: %v", userID, err)
				// 记录失败的 lot ID，防止下次迭代重复处理
				for _, lot := range uLots {
					failedLotIDs[lot.ID] = true
				}
				continue
			}
			totalProcessed += len(uLots)
		}
	}

	return totalProcessed, nil
}

// expireUserLots 在事务中处理单个用户的过期批次
func (s *BalanceLotService) expireUserLots(ctx context.Context, userID int64, lots []*BalanceLot) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	now := time.Now()

	var totalExpired float64
	for _, lot := range lots {
		totalExpired += lot.RemainingAmount
		if err := s.lotRepo.UpdateRemainingAndStatus(txCtx, lot.ID, 0, BalanceLotStatusExpired); err != nil {
			return fmt.Errorf("expire lot %d: %w", lot.ID, err)
		}
	}

	if totalExpired <= 0 {
		return tx.Commit()
	}

	// 重新计算用户余额（基于活跃批次总和）
	newBalance, err := s.lotRepo.SumActiveRemaining(txCtx, userID)
	if err != nil {
		return fmt.Errorf("sum active remaining: %w", err)
	}

	// 获取当前余额（用于日志）
	user, err := s.userRepo.GetByID(txCtx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// 设置余额
	if err := s.userRepo.SetBalance(txCtx, userID, newBalance); err != nil {
		return fmt.Errorf("set balance: %w", err)
	}

	// 创建余额变动日志
	balanceLog := &BalanceLog{
		UserID:        userID,
		ChangeType:    BalanceChangeTypeExpire,
		Amount:        -totalExpired,
		BalanceBefore: user.Balance,
		BalanceAfter:  newBalance,
		Description:   fmt.Sprintf("余额批次过期，过期金额: %.2f，涉及 %d 个批次", totalExpired, len(lots)),
		OperatorID:    0,
		OperatorType:  OperatorTypeSystem,
	}
	_ = s.balanceLogRepo.Create(txCtx, balanceLog)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// 失效 Redis 缓存（事务提交后）
	_ = s.billingCache.InvalidateUserBalance(ctx, userID)

	// 设置 expired_at 时间戳（非关键操作，失败不影响主流程）
	s.setExpiredAt(ctx, lots, now)

	log.Printf("[BalanceLotService] Expired %.2f balance for user %d (%d lots), new balance: %.2f",
		totalExpired, userID, len(lots), newBalance)

	// 异步发送过期通知邮件（非关键操作，失败不影响主流程）
	// 深拷贝 lots 数据，避免 goroutine 引用事务作用域内的指针
	lotsCopy := make([]*BalanceLot, len(lots))
	for i, lot := range lots {
		copied := *lot
		lotsCopy[i] = &copied
	}
	go s.sendExpiryNotification(userID, user.Email, totalExpired, newBalance, lotsCopy)

	return nil
}

// setExpiredAt 设置已过期批次的 expired_at 字段
func (s *BalanceLotService) setExpiredAt(ctx context.Context, lots []*BalanceLot, now time.Time) {
	for _, lot := range lots {
		_, err := s.entClient.BalanceLot.UpdateOneID(lot.ID).
			SetExpiredAt(now).
			Save(ctx)
		if err != nil {
			log.Printf("[BalanceLotService] Failed to set expired_at for lot %d: %v", lot.ID, err)
		}
	}
}

// RecalculateBalance 重新计算用户余额（基于活跃批次总和）
func (s *BalanceLotService) RecalculateBalance(ctx context.Context, userID int64) (float64, error) {
	return s.lotRepo.SumActiveRemaining(ctx, userID)
}

// GetUserLots 获取用户余额批次列表（分页），status 为空时查询所有状态
func (s *BalanceLotService) GetUserLots(ctx context.Context, userID int64, status string, page, pageSize int) ([]*BalanceLot, int64, error) {
	return s.lotRepo.ListByUser(ctx, userID, status, page, pageSize)
}

// GetUserSummary 获取用户余额概览
func (s *BalanceLotService) GetUserSummary(ctx context.Context, userID int64) (*BalanceLotSummary, error) {
	totalBalance, err := s.lotRepo.SumActiveRemaining(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("sum active remaining: %w", err)
	}

	summary := &BalanceLotSummary{
		TotalBalance: totalBalance,
	}

	// 查询当前用户即将过期的批次
	advanceDays := 3
	value, verr := s.settingService.GetSettingValue(ctx, SettingKeyBalanceExpiryReminderAdvanceDays)
	if verr == nil && value != "" {
		if d, err := strconv.Atoi(value); err == nil && d > 0 {
			advanceDays = d
		}
	}

	expiring, err := s.lotRepo.ListExpiringSoonByUser(ctx, userID, advanceDays)
	if err != nil {
		return nil, fmt.Errorf("list expiring soon: %w", err)
	}

	var expiringAmount float64
	var earliestExpiry time.Time
	for _, lot := range expiring {
		expiringAmount += lot.RemainingAmount
		if earliestExpiry.IsZero() || lot.ExpiresAt.Before(earliestExpiry) {
			earliestExpiry = lot.ExpiresAt
		}
	}

	if expiringAmount > 0 {
		summary.ExpiringSoon = &ExpiringSoonDetail{
			Amount:         expiringAmount,
			EarliestExpiry: earliestExpiry,
		}
	}

	return summary, nil
}

// CountByUser 统计用户的批次数量
func (s *BalanceLotService) CountByUser(ctx context.Context, userID int64) (int64, error) {
	return s.lotRepo.CountByUser(ctx, userID)
}

// sendExpiryNotification 发送余额过期通知邮件
func (s *BalanceLotService) sendExpiryNotification(userID int64, email string, expiredAmount, newBalance float64, lots []*BalanceLot) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 检查是否启用过期提醒
	enabled, err := s.settingService.GetSettingValue(ctx, SettingKeyBalanceExpiryReminderEnabled)
	if err != nil || enabled != "true" {
		return
	}

	// 跳过无邮箱用户
	if email == "" {
		return
	}

	// Redis 去重：每天每用户只发一次过期通知
	dedupeKey := fmt.Sprintf("balance_expiry_notification:%d:%s", userID, time.Now().Format("2006-01-02"))
	ok, err := s.redisClient.SetNX(ctx, dedupeKey, "1", 25*time.Hour).Result()
	if err != nil || !ok {
		return
	}

	// 获取站点名称
	siteName, _ := s.settingService.GetSettingValue(ctx, SettingKeySiteName)
	if siteName == "" {
		siteName = "Sub2API"
	}

	subject := fmt.Sprintf("[%s] 余额过期通知", siteName)
	body := s.buildExpiryNotificationEmail(siteName, email, expiredAmount, newBalance, lots)

	if err := s.emailService.SendEmail(ctx, email, subject, body); err != nil {
		log.Printf("[BalanceLotService] Failed to send expiry notification to user %d (%s): %v", userID, email, err)
		// 发送失败时删除去重键，允许下次重试
		s.redisClient.Del(ctx, dedupeKey)
		return
	}

	log.Printf("[BalanceLotService] Sent expiry notification to user %d (%s), expired: %.2f", userID, email, expiredAmount)
}

// buildExpiryNotificationEmail 构建余额已过期通知邮件 HTML
func (s *BalanceLotService) buildExpiryNotificationEmail(siteName, email string, expiredAmount, newBalance float64, lots []*BalanceLot) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head><body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">`)

	// 标题
	b.WriteString(fmt.Sprintf(`<h2 style="color: #c62828;">%s - 余额过期通知</h2>`, html.EscapeString(siteName)))

	// 概要
	b.WriteString(fmt.Sprintf(`<p>您好，%s：</p>`, html.EscapeString(email)))
	b.WriteString(fmt.Sprintf(`<p>您有 <strong style="color: #c62828;">%.2f</strong> 元余额已过期并被清零，当前剩余余额为 <strong>%.2f</strong> 元。</p>`,
		expiredAmount, newBalance))

	// 批次明细表格
	b.WriteString(`<table style="width: 100%; border-collapse: collapse; margin: 16px 0;">`)
	b.WriteString(`<tr style="background: #f5f5f5;">`)
	b.WriteString(`<th style="padding: 8px; border: 1px solid #ddd; text-align: left;">来源</th>`)
	b.WriteString(`<th style="padding: 8px; border: 1px solid #ddd; text-align: right;">过期金额</th>`)
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
		</tr>`, html.EscapeString(sourceName), lot.RemainingAmount, lot.ExpiresAt.Format("2006-01-02 15:04")))
	}

	b.WriteString(`</table>`)

	// 提示
	b.WriteString(`<p style="color: #666;">如需继续使用服务，请及时充值。</p>`)
	b.WriteString(fmt.Sprintf(`<p style="color: #999; font-size: 12px;">此邮件由 %s 系统自动发送，请勿回复。</p>`, html.EscapeString(siteName)))

	b.WriteString(`</body></html>`)

	return b.String()
}
