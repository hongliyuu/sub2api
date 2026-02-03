package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	// wechatScanKeyPrefix 微信扫码登录缓存 key 前缀
	wechatScanKeyPrefix = "wechat:scan:"
	// wechatScanDefaultTTL 扫码结果默认有效期 (5 分钟)
	wechatScanDefaultTTL = 5 * time.Minute
)

// wechatScanCache 微信扫码登录 Redis 缓存实现
type wechatScanCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewWeChatScanCache 创建微信扫码缓存实例
func NewWeChatScanCache(rdb *redis.Client) service.WeChatScanCache {
	return &wechatScanCache{
		rdb: rdb,
		ttl: wechatScanDefaultTTL,
	}
}

// SetScanResult 保存扫码结果 (openID)
// sceneID: 临时二维码的场景值 (UUID)
// openID: 扫码用户的 OpenID
func (c *wechatScanCache) SetScanResult(ctx context.Context, sceneID, openID string) error {
	key := wechatScanKeyPrefix + sceneID
	return c.rdb.Set(ctx, key, openID, c.ttl).Err()
}

// GetScanResult 获取扫码结果
// 返回 openID，如果不存在或已过期返回空字符串
func (c *wechatScanCache) GetScanResult(ctx context.Context, sceneID string) (string, error) {
	key := wechatScanKeyPrefix + sceneID
	result, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get scan result: %w", err)
	}
	return result, nil
}

// DeleteScanResult 删除扫码结果 (登录成功后调用)
func (c *wechatScanCache) DeleteScanResult(ctx context.Context, sceneID string) error {
	key := wechatScanKeyPrefix + sceneID
	return c.rdb.Del(ctx, key).Err()
}
