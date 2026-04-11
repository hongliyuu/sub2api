package repository

import (
	"crypto/tls"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
)

var defaultRedisClient atomic.Pointer[redis.Client]

// InitRedis 初始化 Redis 客户端
//
// 性能优化说明：
// 原实现使用 go-redis 默认配置，未设置连接池和超时参数：
// 1. 默认连接池大小可能不足以支撑高并发
// 2. 无超时控制可能导致慢操作阻塞
//
// 新实现支持可配置的连接池和超时参数：
// 1. PoolSize: 控制最大并发连接数（默认 128）
// 2. MinIdleConns: 保持最小空闲连接，减少冷启动延迟（默认 10）
// 3. DialTimeout/ReadTimeout/WriteTimeout: 精确控制各阶段超时
func InitRedis(cfg *config.Config) *redis.Client {
	client := redis.NewClient(buildRedisOptions(cfg))
	defaultRedisClient.Store(client)
	return client
}

const (
	defaultRedisPoolSize        = 128
	defaultRedisPoolTimeout     = 30 * time.Second
	defaultRedisConnMaxIdleTime = 5 * time.Minute
)

// buildRedisOptions 构建 Redis 连接选项
// 从配置文件读取连接池和超时参数，支持生产环境调优
func buildRedisOptions(cfg *config.Config) *redis.Options {
	if cfg == nil {
		cfg = &config.Config{}
	}
	poolSize := cfg.Redis.PoolSize
	if poolSize <= 0 {
		poolSize = defaultRedisPoolSize
	}
	minIdle := cfg.Redis.MinIdleConns
	if minIdle < 0 {
		minIdle = 0
	}
	if minIdle > poolSize {
		minIdle = poolSize
	}

	opts := &redis.Options{
		Addr:            cfg.Redis.Address(),
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.DB,
		DialTimeout:     time.Duration(cfg.Redis.DialTimeoutSeconds) * time.Second,  // 建连超时
		ReadTimeout:     time.Duration(cfg.Redis.ReadTimeoutSeconds) * time.Second,  // 读取超时
		WriteTimeout:    time.Duration(cfg.Redis.WriteTimeoutSeconds) * time.Second, // 写入超时
		PoolSize:        poolSize,
		MinIdleConns:    minIdle,
		PoolTimeout:     defaultRedisPoolTimeout,
		ConnMaxIdleTime: defaultRedisConnMaxIdleTime,
	}

	if cfg.Redis.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Redis.Host,
		}
	}
	warnings := evaluateRedisPoolWarnings(opts.PoolSize, opts.MinIdleConns)
	logRedisPoolWarnings(warnings, opts.PoolSize, opts.MinIdleConns)

	return opts
}

const (
	redisPoolSizeWarningThreshold      = 512
	redisMinIdleRatioWarningThreshold  = 0.8
	redisMinIdleHardLimitWarningThresh = 1.0
)

func evaluateRedisPoolWarnings(poolSize, minIdle int) []string {
	var warnings []string
	if poolSize > redisPoolSizeWarningThreshold {
		warnings = append(warnings, fmt.Sprintf("pool_size (%d) exceeds %d threshold", poolSize, redisPoolSizeWarningThreshold))
	}
	if poolSize > 0 {
		ratio := float64(minIdle) / float64(poolSize)
		if ratio >= redisMinIdleRatioWarningThreshold {
			warnings = append(warnings, fmt.Sprintf("min_idle_conns (%d) is %.0f%% of pool_size", minIdle, ratio*100))
		}
		if ratio >= redisMinIdleHardLimitWarningThresh {
			warnings = append(warnings, "min_idle_conns equals or exceeds pool_size")
		}
	}
	return warnings
}

func logRedisPoolWarnings(warnings []string, poolSize, minIdle int) {
	if len(warnings) == 0 {
		return
	}
	logger.WriteSinkEvent("warn", "repository.redis", "redis pool configuration guard", map[string]any{
		"warnings": warnings,
		"configured": map[string]any{
			"pool_size": poolSize,
			"min_idle":  minIdle,
		},
	})
}

// RedisPoolSnapshot captures runtime pool metrics that are valuable to ops.
type RedisPoolSnapshot struct {
	Hits       int64 `json:"hits"`
	Misses     int64 `json:"misses"`
	Stalls     int64 `json:"stalls"`
	Timeouts   int64 `json:"timeouts"`
	TotalConns int64 `json:"total_conns"`
	IdleConns  int64 `json:"idle_conns"`
}

// SnapshotRedisPoolStats returns pool metrics from the given client.
func SnapshotRedisPoolStats(client *redis.Client) RedisPoolSnapshot {
	if client == nil {
		return RedisPoolSnapshot{}
	}
	return RedisPoolSnapshotFromStats(client.PoolStats())
}

// SnapshotDefaultRedisPoolStats returns pool stats for the default initialized client.
func SnapshotDefaultRedisPoolStats() RedisPoolSnapshot {
	return SnapshotRedisPoolStats(defaultRedisClient.Load())
}

// RedisPoolSnapshotFromStats converts go-redis stats into our snapshot structure.
func RedisPoolSnapshotFromStats(stats *redis.PoolStats) RedisPoolSnapshot {
	if stats == nil {
		return RedisPoolSnapshot{}
	}
	return RedisPoolSnapshot{
		Hits:   int64(stats.Hits),
		Misses: int64(stats.Misses),
		// go-redis v9 exposes wait count instead of a direct "stalls" field.
		// We keep the outward-facing name stable for existing ops payloads.
		Stalls:     int64(stats.WaitCount),
		Timeouts:   int64(stats.Timeouts),
		TotalConns: int64(stats.TotalConns),
		IdleConns:  int64(stats.IdleConns),
	}
}
