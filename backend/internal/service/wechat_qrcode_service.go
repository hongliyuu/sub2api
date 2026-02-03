package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 微信二维码服务相关错误
var (
	ErrWeChatAppIDNotConfigured     = infraerrors.BadRequest("WECHAT_APPID_NOT_CONFIGURED", "微信 AppID 未配置")
	ErrWeChatAppSecretNotConfigured = infraerrors.BadRequest("WECHAT_APPSECRET_NOT_CONFIGURED", "微信 AppSecret 未配置")
	ErrWeChatAPIFailed              = infraerrors.ServiceUnavailable("WECHAT_API_FAILED", "微信 API 调用失败")
	ErrWeChatInvalidCredentials     = infraerrors.BadRequest("WECHAT_INVALID_CREDENTIALS", "微信 AppID 或 AppSecret 无效")
	ErrWeChatQRCodeGenerateFailed   = infraerrors.ServiceUnavailable("WECHAT_QRCODE_GENERATE_FAILED", "生成微信二维码失败")
)

// WeChatAccessTokenResponse 微信 access_token 响应
type WeChatAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // 过期时间（秒）
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WeChatQRCodeResponse 微信二维码创建响应
type WeChatQRCodeResponse struct {
	Ticket        string `json:"ticket"`
	ExpireSeconds int    `json:"expire_seconds"` // 永久码为0
	URL           string `json:"url"`            // 二维码图片解析后的地址
	ErrCode       int    `json:"errcode"`
	ErrMsg        string `json:"errmsg"`
}

// WeChatQRCodeResult 生成二维码的结果
type WeChatQRCodeResult struct {
	Ticket   string `json:"ticket"`
	URL      string `json:"url"`       // 二维码图片解析后的地址
	ImageURL string `json:"image_url"` // 二维码图片直接访问的 URL
}

// WeChatAPIClient 微信 API 客户端接口
type WeChatAPIClient interface {
	// GetAccessToken 获取 access_token
	GetAccessToken(ctx context.Context, appID, appSecret string) (*WeChatAccessTokenResponse, error)
	// CreatePermanentQRCode 创建永久二维码
	CreatePermanentQRCode(ctx context.Context, accessToken, sceneStr string) (*WeChatQRCodeResponse, error)
	// CreateTemporaryQRCode 创建临时二维码
	// expireSeconds: 有效期（秒），最长不超过 2592000（30天）
	CreateTemporaryQRCode(ctx context.Context, accessToken, sceneStr string, expireSeconds int) (*WeChatQRCodeResponse, error)
}

// accessTokenCache access_token 缓存
type accessTokenCache struct {
	token     string
	expiresAt time.Time
}

// WeChatQRCodeService 微信二维码服务
type WeChatQRCodeService struct {
	settingService *SettingService
	apiClient      WeChatAPIClient

	// access_token 缓存
	tokenMu    sync.RWMutex
	tokenCache map[string]*accessTokenCache // key: appID
}

// NewWeChatQRCodeService 创建微信二维码服务
func NewWeChatQRCodeService(settingService *SettingService, apiClient WeChatAPIClient) *WeChatQRCodeService {
	return &WeChatQRCodeService{
		settingService: settingService,
		apiClient:      apiClient,
		tokenCache:     make(map[string]*accessTokenCache),
	}
}

// GeneratePermanentQRCode 生成永久二维码
// sceneStr: 场景值字符串（最长64字符），如 "login"
func (s *WeChatQRCodeService) GeneratePermanentQRCode(ctx context.Context, appID, appSecret, sceneStr string) (*WeChatQRCodeResult, error) {
	if appID == "" {
		return nil, ErrWeChatAppIDNotConfigured
	}
	if appSecret == "" {
		return nil, ErrWeChatAppSecretNotConfigured
	}

	// 获取 access_token（优先使用缓存）
	accessToken, err := s.getAccessToken(ctx, appID, appSecret)
	if err != nil {
		return nil, err
	}

	// 创建永久二维码
	qrResp, err := s.apiClient.CreatePermanentQRCode(ctx, accessToken, sceneStr)
	if err != nil {
		return nil, fmt.Errorf("创建二维码失败: %w", err)
	}

	if qrResp.ErrCode != 0 {
		// access_token 可能过期，清除缓存后重试一次
		if qrResp.ErrCode == 40001 || qrResp.ErrCode == 42001 {
			s.invalidateTokenCache(appID)
			accessToken, err = s.getAccessToken(ctx, appID, appSecret)
			if err != nil {
				return nil, err
			}
			qrResp, err = s.apiClient.CreatePermanentQRCode(ctx, accessToken, sceneStr)
			if err != nil {
				return nil, fmt.Errorf("创建二维码失败: %w", err)
			}
		}
		if qrResp.ErrCode != 0 {
			return nil, fmt.Errorf("%w: %s (errcode: %d)", ErrWeChatQRCodeGenerateFailed, translateWeChatError(qrResp.ErrCode, qrResp.ErrMsg), qrResp.ErrCode)
		}
	}

	return &WeChatQRCodeResult{
		Ticket:   qrResp.Ticket,
		URL:      qrResp.URL,
		ImageURL: fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s", qrResp.Ticket),
	}, nil
}

// GenerateTemporaryQRCode 生成临时二维码
// sceneStr: 场景值字符串（最长64字符）
// expireSeconds: 有效期（秒），最长不超过 2592000（30天）
func (s *WeChatQRCodeService) GenerateTemporaryQRCode(ctx context.Context, appID, appSecret, sceneStr string, expireSeconds int) (*WeChatQRCodeResult, error) {
	if appID == "" {
		return nil, ErrWeChatAppIDNotConfigured
	}
	if appSecret == "" {
		return nil, ErrWeChatAppSecretNotConfigured
	}

	// 获取 access_token（优先使用缓存）
	accessToken, err := s.getAccessToken(ctx, appID, appSecret)
	if err != nil {
		return nil, err
	}

	// 创建临时二维码
	qrResp, err := s.apiClient.CreateTemporaryQRCode(ctx, accessToken, sceneStr, expireSeconds)
	if err != nil {
		return nil, fmt.Errorf("创建临时二维码失败: %w", err)
	}

	if qrResp.ErrCode != 0 {
		// access_token 可能过期，清除缓存后重试一次
		if qrResp.ErrCode == 40001 || qrResp.ErrCode == 42001 {
			s.invalidateTokenCache(appID)
			accessToken, err = s.getAccessToken(ctx, appID, appSecret)
			if err != nil {
				return nil, err
			}
			qrResp, err = s.apiClient.CreateTemporaryQRCode(ctx, accessToken, sceneStr, expireSeconds)
			if err != nil {
				return nil, fmt.Errorf("创建临时二维码失败: %w", err)
			}
		}
		if qrResp.ErrCode != 0 {
			return nil, fmt.Errorf("%w: %s (errcode: %d)", ErrWeChatQRCodeGenerateFailed, translateWeChatError(qrResp.ErrCode, qrResp.ErrMsg), qrResp.ErrCode)
		}
	}

	return &WeChatQRCodeResult{
		Ticket:   qrResp.Ticket,
		URL:      qrResp.URL,
		ImageURL: fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s", qrResp.Ticket),
	}, nil
}

// getAccessToken 获取 access_token（带缓存）
func (s *WeChatQRCodeService) getAccessToken(ctx context.Context, appID, appSecret string) (string, error) {
	// 先检查缓存
	s.tokenMu.RLock()
	if cache, ok := s.tokenCache[appID]; ok {
		// 提前 5 分钟失效，避免边界情况
		if time.Now().Add(5 * time.Minute).Before(cache.expiresAt) {
			s.tokenMu.RUnlock()
			return cache.token, nil
		}
	}
	s.tokenMu.RUnlock()

	// 缓存未命中或已过期，重新获取
	resp, err := s.apiClient.GetAccessToken(ctx, appID, appSecret)
	if err != nil {
		return "", fmt.Errorf("获取 access_token 失败: %w", err)
	}

	if resp.ErrCode != 0 {
		return "", fmt.Errorf("%w: %s (errcode: %d)", ErrWeChatInvalidCredentials, translateWeChatError(resp.ErrCode, resp.ErrMsg), resp.ErrCode)
	}

	// 更新缓存
	s.tokenMu.Lock()
	s.tokenCache[appID] = &accessTokenCache{
		token:     resp.AccessToken,
		expiresAt: time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
	}
	s.tokenMu.Unlock()

	return resp.AccessToken, nil
}

// invalidateTokenCache 使缓存失效
func (s *WeChatQRCodeService) invalidateTokenCache(appID string) {
	s.tokenMu.Lock()
	delete(s.tokenCache, appID)
	s.tokenMu.Unlock()
}

// InvalidateAllTokenCache 清除所有 access_token 缓存
func (s *WeChatQRCodeService) InvalidateAllTokenCache() {
	s.tokenMu.Lock()
	s.tokenCache = make(map[string]*accessTokenCache)
	s.tokenMu.Unlock()
}

// translateWeChatError 翻译微信错误码为中文
func translateWeChatError(errCode int, errMsg string) string {
	switch errCode {
	case 40001:
		return "access_token 无效或已过期"
	case 40013:
		return "无效的 AppID"
	case 40125:
		return "无效的 AppSecret"
	case 41002:
		return "缺少 AppID 参数"
	case 41004:
		return "缺少 AppSecret 参数"
	case 42001:
		return "access_token 已过期"
	case 45009:
		return "接口调用频率超限"
	case 61023:
		return "无效的 scene_str（场景值）"
	case 61024:
		return "scene_str 长度超过限制（最长64字符）"
	case -1:
		return "系统繁忙，请稍后重试"
	default:
		if errMsg != "" {
			return errMsg
		}
		return fmt.Sprintf("未知错误 (errcode: %d)", errCode)
	}
}
