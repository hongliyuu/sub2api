package service

import (
	"context"
	"crypto/rand"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrAffCodeNotFound = infraerrors.NotFound("AFF_CODE_NOT_FOUND", "referral code not found")
	ErrSelfReferral    = infraerrors.BadRequest("SELF_REFERRAL", "cannot use your own referral code")
)

// ReferralRepository 邀请返利数据访问接口（UserRepository 的子集）
type ReferralRepository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByAffCode(ctx context.Context, affCode string) (*User, error)
	SetAffCode(ctx context.Context, userID int64, affCode string) error
	SetInviterID(ctx context.Context, userID int64, inviterID int64) error
	IncrementAffStats(ctx context.Context, userID int64, bonusAmount float64) error
	UpdateBalance(ctx context.Context, id int64, amount float64) error
}

// ReferralService 邀请返利服务
type ReferralService struct {
	userRepo       ReferralRepository
	settingService *SettingService
}

// NewReferralService 创建邀请返利服务实例
func NewReferralService(userRepo ReferralRepository, settingService *SettingService) *ReferralService {
	return &ReferralService{
		userRepo:       userRepo,
		settingService: settingService,
	}
}

// GetAffCodeByUserID 获取用户的推荐码，若不存在则懒加载生成一个
func (s *ReferralService) GetAffCodeByUserID(ctx context.Context, userID int64) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if user.AffCode != "" {
		return user.AffCode, nil
	}

	// 懒加载生成唯一推荐码
	code, err := s.generateUniqueAffCode(ctx)
	if err != nil {
		return "", fmt.Errorf("generate aff code: %w", err)
	}

	if err := s.userRepo.SetAffCode(ctx, userID, code); err != nil {
		return "", fmt.Errorf("save aff code: %w", err)
	}

	return code, nil
}

// BindInviter 注册时记录邀请关系（绑定 inviter_id，不发余额奖励）
// affCode 是邀请人的推荐码，inviteeID 是新注册用户的ID
func (s *ReferralService) BindInviter(ctx context.Context, affCode string, inviteeID int64) error {
	if affCode == "" {
		return nil
	}

	inviter, err := s.userRepo.GetByAffCode(ctx, affCode)
	if err != nil {
		logger.LegacyPrintf("service.referral", "[Referral] Aff code not found: %s", affCode)
		return ErrAffCodeNotFound
	}

	if inviter.ID == inviteeID {
		return ErrSelfReferral
	}

	// 绑定邀请人ID
	if err := s.userRepo.SetInviterID(ctx, inviteeID, inviter.ID); err != nil {
		logger.LegacyPrintf("service.referral", "[Referral] Failed to bind inviter %d for invitee %d: %v", inviter.ID, inviteeID, err)
		return fmt.Errorf("bind inviter: %w", err)
	}

	// 累计邀请人的邀请次数（不加余额）
	if err := s.userRepo.IncrementAffStats(ctx, inviter.ID, 0); err != nil {
		logger.LegacyPrintf("service.referral", "[Referral] Failed to increment aff stats for inviter %d: %v", inviter.ID, err)
	}

	logger.LegacyPrintf("service.referral", "[Referral] Bound inviter=%d for invitee=%d", inviter.ID, inviteeID)
	return nil
}

// RewardRecharge 被邀请人充值成功后，按比例给邀请人返利
// inviteeID 是充值用户的ID，rechargeAmount 是实际到账余额（USD）
func (s *ReferralService) RewardRecharge(ctx context.Context, inviteeID int64, rechargeAmount float64) error {
	if rechargeAmount <= 0 || s.settingService == nil {
		return nil
	}
	if !s.settingService.IsReferralEnabled(ctx) {
		return nil
	}
	rate := s.settingService.GetReferralRebateRate(ctx)
	if rate <= 0 {
		return nil
	}

	invitee, err := s.userRepo.GetByID(ctx, inviteeID)
	if err != nil || invitee.InviterID == nil {
		return nil
	}

	rebate := rechargeAmount * rate
	if err := s.userRepo.IncrementAffStats(ctx, *invitee.InviterID, rebate); err != nil {
		logger.LegacyPrintf("service.referral", "[Referral] Failed to reward inviter %d rebate %.4f: %v", *invitee.InviterID, rebate, err)
		return fmt.Errorf("reward inviter rebate: %w", err)
	}

	logger.LegacyPrintf("service.referral", "[Referral] Rebate: invitee=%d recharged=%.4f rate=%.4f inviter=%d rebate=%.4f",
		inviteeID, rechargeAmount, rate, *invitee.InviterID, rebate)
	return nil
}

// generateUniqueAffCode 生成唯一的 6 位字母数字推荐码，最多重试 10 次
func (s *ReferralService) generateUniqueAffCode(ctx context.Context) (string, error) {
	const maxRetries = 10
	for i := 0; i < maxRetries; i++ {
		code := randomAlphaNum(6)
		_, err := s.userRepo.GetByAffCode(ctx, code)
		if err != nil {
			// 未找到表示该码可用
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique aff code after %d retries", maxRetries)
}

// randomAlphaNum 生成指定长度的随机字母数字字符串（大写字母 + 数字，去除易混淆字符 0/O/I/1）
func randomAlphaNum(n int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	rb := make([]byte, n*2)
	_, _ = rand.Read(rb)
	for i := 0; i < n; i++ {
		b[i] = charset[int(rb[i])%len(charset)]
	}
	return string(b)
}
