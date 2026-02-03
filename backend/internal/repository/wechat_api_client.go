package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	wechatAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"
	wechatQRCodeURL      = "https://api.weixin.qq.com/cgi-bin/qrcode/create"
)

// wechatAPIClient 微信 API 客户端实现
type wechatAPIClient struct {
	httpClient *http.Client
}

// NewWeChatAPIClient 创建微信 API 客户端
func NewWeChatAPIClient() service.WeChatAPIClient {
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:            30 * time.Second,
		ValidateResolvedIP: false, // 微信 API 是外部服务，不需要校验
	})
	if err != nil {
		sharedClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &wechatAPIClient{
		httpClient: sharedClient,
	}
}

// GetAccessToken 获取微信 access_token
func (c *wechatAPIClient) GetAccessToken(ctx context.Context, appID, appSecret string) (*service.WeChatAccessTokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "client_credential")
	params.Set("appid", appID)
	params.Set("secret", appSecret)

	reqURL := fmt.Sprintf("%s?%s", wechatAccessTokenURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result service.WeChatAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// CreatePermanentQRCode 创建永久二维码
func (c *wechatAPIClient) CreatePermanentQRCode(ctx context.Context, accessToken, sceneStr string) (*service.WeChatQRCodeResponse, error) {
	reqURL := fmt.Sprintf("%s?access_token=%s", wechatQRCodeURL, url.QueryEscape(accessToken))

	// 构建请求体
	reqBody := map[string]interface{}{
		"action_name": "QR_LIMIT_STR_SCENE",
		"action_info": map[string]interface{}{
			"scene": map[string]string{
				"scene_str": sceneStr,
			},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result service.WeChatQRCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// CreateTemporaryQRCode 创建临时二维码
// expireSeconds: 有效期（秒），最长不超过 2592000（30天）
func (c *wechatAPIClient) CreateTemporaryQRCode(ctx context.Context, accessToken, sceneStr string, expireSeconds int) (*service.WeChatQRCodeResponse, error) {
	reqURL := fmt.Sprintf("%s?access_token=%s", wechatQRCodeURL, url.QueryEscape(accessToken))

	// 构建请求体
	reqBody := map[string]interface{}{
		"expire_seconds": expireSeconds,
		"action_name":    "QR_STR_SCENE",
		"action_info": map[string]interface{}{
			"scene": map[string]string{
				"scene_str": sceneStr,
			},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result service.WeChatQRCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
