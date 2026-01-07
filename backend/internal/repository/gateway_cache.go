package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/redis/go-redis/v9"
)

const stickySessionPrefix = "sticky_session:"

type gatewayCache struct {
	rdb                *redis.Client
	userAgentKeyPrefix string
}

func NewGatewayCache(rdb *redis.Client, cfg *config.Config) *gatewayCache {
	return &gatewayCache{
		rdb:                rdb,
		userAgentKeyPrefix: buildUserAgentKeyPrefix(cfg),
	}
}

func buildUserAgentKeyPrefix(cfg *config.Config) string {
	if cfg == nil {
		return "sub2api"
	}
	if prefix := strings.TrimSpace(cfg.Gateway.UserAgentCacheKeyPrefix); prefix != "" {
		return prefix
	}
	runMode := strings.TrimSpace(cfg.RunMode)
	serverMode := strings.TrimSpace(cfg.Server.Mode)
	if runMode == "" && serverMode == "" {
		return "sub2api"
	}
	return fmt.Sprintf("sub2api:%s:%s", runMode, serverMode)
}

func (c *gatewayCache) GetSessionAccountID(ctx context.Context, sessionHash string) (int64, error) {
	key := stickySessionPrefix + sessionHash
	return c.rdb.Get(ctx, key).Int64()
}

func (c *gatewayCache) SetSessionAccountID(ctx context.Context, sessionHash string, accountID int64, ttl time.Duration) error {
	key := stickySessionPrefix + sessionHash
	return c.rdb.Set(ctx, key, accountID, ttl).Err()
}

func (c *gatewayCache) RefreshSessionTTL(ctx context.Context, sessionHash string, ttl time.Duration) error {
	key := stickySessionPrefix + sessionHash
	return c.rdb.Expire(ctx, key, ttl).Err()
}

// GetLatestUserAgent 获取缓存的最新 User-Agent
func (c *gatewayCache) GetLatestUserAgent(ctx context.Context) (string, error) {
	value, err := c.rdb.Get(ctx, c.latestUserAgentKey()).Result()
	if err == redis.Nil {
		return "", nil
	}
	return value, err
}

// SetLatestUserAgent 设置最新 User-Agent 到缓存
func (c *gatewayCache) SetLatestUserAgent(ctx context.Context, userAgent string, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.latestUserAgentKey(), userAgent, ttl).Err()
}

// GetLatestUserAgentForAccount 获取账号维度缓存的 User-Agent
func (c *gatewayCache) GetLatestUserAgentForAccount(ctx context.Context, accountID int64) (string, error) {
	if accountID <= 0 {
		return "", nil
	}
	value, err := c.rdb.Get(ctx, c.latestUserAgentKeyForAccount(accountID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return value, err
}

// SetLatestUserAgentForAccount 设置账号维度的 User-Agent 到缓存
func (c *gatewayCache) SetLatestUserAgentForAccount(ctx context.Context, accountID int64, userAgent string, ttl time.Duration) error {
	if accountID <= 0 {
		return nil
	}
	return c.rdb.Set(ctx, c.latestUserAgentKeyForAccount(accountID), userAgent, ttl).Err()
}

func (c *gatewayCache) latestUserAgentKey() string {
	return fmt.Sprintf("%s:latest_user_agent", c.userAgentKeyPrefix)
}

func (c *gatewayCache) latestUserAgentKeyForAccount(accountID int64) string {
	return fmt.Sprintf("%s:latest_user_agent:%d", c.userAgentKeyPrefix, accountID)
}
