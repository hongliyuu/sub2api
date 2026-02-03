package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 微信扫码登录相关错误
var (
	ErrWeChatScanNotFound    = infraerrors.NotFound("WECHAT_SCAN_NOT_FOUND", "扫码记录不存在或已过期")
	ErrWeChatScanPending     = infraerrors.NotFound("WECHAT_SCAN_PENDING", "等待用户扫码")
	ErrWeChatScanInitFailed  = infraerrors.ServiceUnavailable("WECHAT_SCAN_INIT_FAILED", "初始化扫码登录失败")
	ErrWeChatAccountTypeMismatch = infraerrors.BadRequest("WECHAT_ACCOUNT_TYPE_MISMATCH", "当前账号类型不支持扫码自动登录")
)

// WeChatScanCache 微信扫码登录缓存接口
type WeChatScanCache interface {
	// SetScanResult 保存扫码结果
	SetScanResult(ctx context.Context, sceneID, openID string) error
	// GetScanResult 获取扫码结果
	GetScanResult(ctx context.Context, sceneID string) (string, error)
	// DeleteScanResult 删除扫码结果
	DeleteScanResult(ctx context.Context, sceneID string) error
}

// WeChatScanSession 扫码登录会话信息
type WeChatScanSession struct {
	SceneID       string `json:"scene_id"`        // 场景值 (UUID)
	QRCodeURL     string `json:"qr_code_url"`     // 二维码图片 URL
	ExpireSeconds int    `json:"expire_seconds"`  // 过期时间（秒）
}

// WeChatScanResult 扫码结果
type WeChatScanResult struct {
	Status string `json:"status"` // "waiting", "confirmed"
	OpenID string `json:"-"`      // 仅内部使用，不返回给前端
}

// WeChatScanLoginService 微信扫码登录服务
type WeChatScanLoginService struct {
	qrcodeService  *WeChatQRCodeService
	scanCache      WeChatScanCache
	settingService *SettingService
}

// NewWeChatScanLoginService 创建微信扫码登录服务
func NewWeChatScanLoginService(
	qrcodeService *WeChatQRCodeService,
	scanCache WeChatScanCache,
	settingService *SettingService,
) *WeChatScanLoginService {
	return &WeChatScanLoginService{
		qrcodeService:  qrcodeService,
		scanCache:      scanCache,
		settingService: settingService,
	}
}

// InitScanSession 初始化扫码登录会话
// 生成临时二维码，返回场景值和二维码 URL
func (s *WeChatScanLoginService) InitScanSession(ctx context.Context) (*WeChatScanSession, error) {
	// 获取微信配置
	appID, appSecret, err := s.settingService.GetWeChatAppCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("get wechat credentials: %w", err)
	}

	if appID == "" || appSecret == "" {
		return nil, ErrWeChatAppIDNotConfigured
	}

	// 生成唯一场景值 (UUID)
	sceneID := uuid.New().String()

	// 创建临时二维码 (有效期 300 秒)
	expireSeconds := 300
	result, err := s.qrcodeService.GenerateTemporaryQRCode(ctx, appID, appSecret, sceneID, expireSeconds)
	if err != nil {
		return nil, fmt.Errorf("generate temporary qrcode: %w", err)
	}

	return &WeChatScanSession{
		SceneID:       sceneID,
		QRCodeURL:     result.ImageURL,
		ExpireSeconds: expireSeconds,
	}, nil
}

// HandleScanEvent 处理扫码事件
// 当微信推送 SCAN 或 subscribe 事件时调用
func (s *WeChatScanLoginService) HandleScanEvent(ctx context.Context, sceneStr, openID string) error {
	if sceneStr == "" || openID == "" {
		return fmt.Errorf("invalid scan event: sceneStr=%q, openID=%q", sceneStr, openID)
	}

	// 将扫码结果写入 Redis
	if err := s.scanCache.SetScanResult(ctx, sceneStr, openID); err != nil {
		return fmt.Errorf("set scan result: %w", err)
	}

	return nil
}

// CheckScanResult 检查扫码结果
// 前端轮询调用，返回状态和 openID（如果已扫码）
func (s *WeChatScanLoginService) CheckScanResult(ctx context.Context, sceneID string) (*WeChatScanResult, error) {
	if sceneID == "" {
		return nil, ErrWeChatScanNotFound
	}

	openID, err := s.scanCache.GetScanResult(ctx, sceneID)
	if err != nil {
		return nil, fmt.Errorf("get scan result: %w", err)
	}

	if openID == "" {
		return &WeChatScanResult{
			Status: "waiting",
		}, nil
	}

	return &WeChatScanResult{
		Status: "confirmed",
		OpenID: openID,
	}, nil
}

// ConsumeScanResult 消费扫码结果（登录成功后删除）
func (s *WeChatScanLoginService) ConsumeScanResult(ctx context.Context, sceneID string) error {
	return s.scanCache.DeleteScanResult(ctx, sceneID)
}

// IsSubscriptionAccountType 检查当前配置是否为订阅号类型
func (s *WeChatScanLoginService) IsSubscriptionAccountType(ctx context.Context) (bool, error) {
	settings, err := s.settingService.GetPublicSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings.WeChatAccountType == WeChatAccountTypeSubscription, nil
}
