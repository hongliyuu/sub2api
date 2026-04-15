package routes

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	iputil "github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const healthProbeTimeout = 500 * time.Millisecond

type CommonRouteOptions struct {
	DBProbe           func(context.Context) error
	RedisProbe        func(context.Context) error
	DBSnapshot        func() repository.DBPoolSnapshot
	RedisSnapshot     func() repository.RedisPoolSnapshot
	OpsErrorLog       func() OpsErrorLogSnapshot
	SchedulerSnapshot func() service.SchedulerOutboxRuntimeMetrics
	Now               func() time.Time
	PublicHealth      bool
	PublicMetrics     bool
	AllowCIDRs        []string
	ClientIP          func(*gin.Context) string
}

type OpsErrorLogSnapshot struct {
	QueueLength           int64
	QueueCapacity         int
	DroppedTotal          int64
	EnqueuedTotal         int64
	ProcessedTotal        int64
	PersistedSuccessTotal int64
	PersistedFailureTotal int64
}

// RegisterCommonRoutes 注册通用路由（健康检查、状态等）
func RegisterCommonRoutes(r *gin.Engine, provided ...CommonRouteOptions) {
	opts := resolveCommonRouteOptions(provided...)
	healthGuard := observabilityGuard(opts.PublicHealth, opts.AllowCIDRs, opts.ClientIP)
	metricsGuard := observabilityGuard(opts.PublicMetrics, opts.AllowCIDRs, opts.ClientIP)

	readinessHandler := func(c *gin.Context) {
		payload, ready := buildReadinessPayload(c.Request.Context(), opts)
		statusCode := http.StatusOK
		if !ready {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, payload)
	}
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, buildHealthPayload(c.Request.Context(), opts))
	}

	// 健康检查
	r.GET("/health", healthGuard(healthHandler))
	r.GET("/readyz", readinessHandler)
	r.GET("/api/health", healthGuard(healthHandler))
	r.GET("/api/readyz", readinessHandler)

	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"live":       true,
			"checked_at": opts.Now().UTC().Format(time.RFC3339),
		})
	})

	// Runtime metrics in Prometheus exposition format.
	r.GET("/metrics", metricsGuard(metricsHandler()))

	// Claude Code 遥测日志（忽略，直接返回200）
	r.POST("/api/event_logging/batch", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Setup status endpoint (always returns needs_setup: false in normal mode)
	// This is used by the frontend to detect when the service has restarted after setup
	r.GET("/setup/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"needs_setup": false,
				"step":        "completed",
			},
		})
	})
}

func resolveCommonRouteOptions(provided ...CommonRouteOptions) CommonRouteOptions {
	var opts CommonRouteOptions
	if len(provided) > 0 {
		opts = provided[0]
	}
	if opts.DBProbe == nil {
		opts.DBProbe = repository.ProbeDefaultDB
	}
	if opts.RedisProbe == nil {
		opts.RedisProbe = repository.ProbeDefaultRedis
	}
	if opts.DBSnapshot == nil {
		opts.DBSnapshot = repository.SnapshotDefaultDBPoolStats
	}
	if opts.RedisSnapshot == nil {
		opts.RedisSnapshot = repository.SnapshotDefaultRedisPoolStats
	}
	if opts.OpsErrorLog == nil {
		opts.OpsErrorLog = func() OpsErrorLogSnapshot {
			return OpsErrorLogSnapshot{
				QueueLength:           handler.OpsErrorLogQueueLength(),
				QueueCapacity:         handler.OpsErrorLogQueueCapacity(),
				DroppedTotal:          handler.OpsErrorLogDroppedTotal(),
				EnqueuedTotal:         handler.OpsErrorLogEnqueuedTotal(),
				ProcessedTotal:        handler.OpsErrorLogProcessedTotal(),
				PersistedSuccessTotal: handler.OpsErrorLogPersistedSuccessTotal(),
				PersistedFailureTotal: handler.OpsErrorLogPersistedFailureTotal(),
			}
		}
	}
	if opts.SchedulerSnapshot == nil {
		opts.SchedulerSnapshot = service.SnapshotSchedulerOutboxRuntimeMetrics
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.ClientIP == nil {
		opts.ClientIP = iputil.GetTrustedClientIP
	}
	return opts
}

func observabilityGuard(public bool, allowCIDRs []string, clientIP func(*gin.Context) string) func(gin.HandlerFunc) gin.HandlerFunc {
	compiledAllow := iputil.CompileIPRules(allowCIDRs)
	return func(next gin.HandlerFunc) gin.HandlerFunc {
		if public {
			return next
		}
		return func(c *gin.Context) {
			if compiledAllow == nil || compiledAllow.PatternCount == 0 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "forbidden",
					"message": "observability endpoint is not exposed",
				})
				return
			}
			requestIP := ""
			if clientIP != nil {
				requestIP = strings.TrimSpace(clientIP(c))
			}
			allowed, _ := iputil.CheckIPRestrictionWithCompiledRules(requestIP, compiledAllow, nil)
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "forbidden",
					"message": "observability endpoint is not exposed",
				})
				return
			}
			next(c)
		}
	}
}

func buildReadinessPayload(baseCtx context.Context, opts CommonRouteOptions) (gin.H, bool) {
	payload := buildHealthPayload(baseCtx, opts)
	ready, _ := payload["ready"].(bool)
	checkedAt, _ := payload["checked_at"].(string)
	status := "ok"
	if !ready {
		status = "not_ready"
	}
	return gin.H{
		"status":     status,
		"ready":      ready,
		"checked_at": checkedAt,
	}, ready
}

func buildHealthPayload(baseCtx context.Context, opts CommonRouteOptions) gin.H {
	dbSnapshot := opts.DBSnapshot()
	redisSnapshot := opts.RedisSnapshot()
	opsErrorLogSnapshot := opts.OpsErrorLog()
	schedulerSnapshot := opts.SchedulerSnapshot()
	dbErr := probeHealthComponent(baseCtx, opts.DBProbe)
	redisErr := probeHealthComponent(baseCtx, opts.RedisProbe)

	dbReady, dbStatus, dbNote := summarizeDBHealth(dbSnapshot, dbErr)
	redisReady, redisStatus, redisNote := summarizeRedisHealth(redisSnapshot, redisErr)
	opsErrorLogStatus, opsErrorLogNote, opsErrorLogUsagePercent := summarizeOpsErrorLogHealth(opsErrorLogSnapshot)
	schedulerStatus, schedulerNote := summarizeSchedulerHealth(schedulerSnapshot)

	ready := dbReady && redisReady
	status := "ok"
	if !ready || schedulerStatus != "ok" || opsErrorLogStatus != "ok" {
		status = "degraded"
	}

	return gin.H{
		"status":           status,
		"ready":            ready,
		"checked_at":       opts.Now().UTC().Format(time.RFC3339),
		"probe_timeout_ms": healthProbeTimeout.Milliseconds(),
		"checks": gin.H{
			"db": gin.H{
				"status":               dbStatus,
				"ready":                dbReady,
				"note":                 dbNote,
				"open_connections":     dbSnapshot.OpenConnections,
				"in_use":               dbSnapshot.InUse,
				"idle":                 dbSnapshot.Idle,
				"max_open_connections": dbSnapshot.MaxOpenConnections,
				"wait_count":           dbSnapshot.WaitCount,
				"usage_percent":        dbSnapshot.UsagePercent,
				"in_use_percent":       dbSnapshot.InUsePercent,
			},
			"redis": gin.H{
				"status":           redisStatus,
				"ready":            redisReady,
				"note":             redisNote,
				"total_conns":      redisSnapshot.TotalConns,
				"idle_conns":       redisSnapshot.IdleConns,
				"timeouts":         redisSnapshot.Timeouts,
				"stalls":           redisSnapshot.Stalls,
				"usage_percent":    redisSnapshot.UsagePercent,
				"hit_rate_percent": redisSnapshot.HitRatePercent,
			},
			"ops_error_logger": gin.H{
				"status":                  opsErrorLogStatus,
				"note":                    opsErrorLogNote,
				"queue_length":            opsErrorLogSnapshot.QueueLength,
				"queue_capacity":          opsErrorLogSnapshot.QueueCapacity,
				"queue_usage_percent":     opsErrorLogUsagePercent,
				"enqueued_total":          opsErrorLogSnapshot.EnqueuedTotal,
				"processed_total":         opsErrorLogSnapshot.ProcessedTotal,
				"dropped_total":           opsErrorLogSnapshot.DroppedTotal,
				"persisted_success_total": opsErrorLogSnapshot.PersistedSuccessTotal,
				"persisted_failure_total": opsErrorLogSnapshot.PersistedFailureTotal,
			},
			"scheduler_outbox": gin.H{
				"status":                     schedulerStatus,
				"note":                       schedulerNote,
				"lag_seconds":                schedulerSnapshot.LagSeconds,
				"backlog_rows":               schedulerSnapshot.BacklogRows,
				"watermark_drift":            schedulerSnapshot.WatermarkDrift,
				"lag_failure_streak":         schedulerSnapshot.LagFailureStreak,
				"checkpoint_fallback_streak": schedulerSnapshot.CheckpointFallbackStreak,
				"checkpoint_write_failures":  schedulerSnapshot.CheckpointWriteFailureTotal,
			},
		},
	}
}

func probeHealthComponent(baseCtx context.Context, probe func(context.Context) error) error {
	if probe == nil {
		return errors.New("probe not configured")
	}
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, healthProbeTimeout)
	defer cancel()
	return probe(ctx)
}

func summarizeDBHealth(snapshot repository.DBPoolSnapshot, err error) (bool, string, string) {
	if err != nil {
		return false, "critical", compactHealthError(err)
	}
	if snapshot.MaxOpenConnections > 0 {
		if snapshot.InUsePercent != nil && *snapshot.InUsePercent >= 95 {
			return true, "warning", "db pool nearly saturated"
		}
		if snapshot.UsagePercent != nil && *snapshot.UsagePercent >= 95 {
			return true, "warning", "db open connections near max"
		}
	}
	return true, "ok", "db probe healthy"
}

func summarizeRedisHealth(snapshot repository.RedisPoolSnapshot, err error) (bool, string, string) {
	if err != nil {
		return false, "critical", compactHealthError(err)
	}
	if snapshot.ConnectionPressurePercent != nil && *snapshot.ConnectionPressurePercent >= 95 {
		return true, "warning", "redis pool nearly saturated"
	}
	if snapshot.UsagePercent != nil && *snapshot.UsagePercent >= 95 {
		return true, "warning", "redis connections near pool limit"
	}
	return true, "ok", "redis probe healthy"
}

func summarizeOpsErrorLogHealth(snapshot OpsErrorLogSnapshot) (string, string, *float64) {
	var usagePercent *float64
	if snapshot.QueueCapacity > 0 {
		usage := float64(snapshot.QueueLength) / float64(snapshot.QueueCapacity) * 100
		usagePercent = &usage
		if usage >= 95 {
			return "critical", "ops error log queue nearly full", usagePercent
		}
		if usage >= 80 {
			return "warning", "ops error log queue under pressure", usagePercent
		}
	}

	notes := make([]string, 0, 2)
	if snapshot.DroppedTotal > 0 {
		notes = append(notes, "historical_dropped_total="+formatHealthInt(snapshot.DroppedTotal))
	}
	if snapshot.PersistedFailureTotal > 0 {
		notes = append(notes, "historical_persist_fail_total="+formatHealthInt(snapshot.PersistedFailureTotal))
	}
	if len(notes) > 0 {
		return "ok", strings.Join(notes, ", "), usagePercent
	}
	return "ok", "ops error logger healthy", usagePercent
}

func summarizeSchedulerHealth(snapshot service.SchedulerOutboxRuntimeMetrics) (string, string) {
	switch {
	case snapshot.LagSeconds >= 300 || snapshot.BacklogRows >= 1000 || snapshot.CheckpointFallbackStreak >= 3:
		return "critical", healthSchedulerNote(snapshot)
	case snapshot.LagSeconds > 0 || snapshot.BacklogRows > 0 || snapshot.WatermarkDrift != 0 || snapshot.LagFailureStreak > 0 || snapshot.CheckpointFallbackStreak > 0:
		return "warning", healthSchedulerNote(snapshot)
	default:
		return "ok", "scheduler outbox healthy"
	}
}

func healthSchedulerNote(snapshot service.SchedulerOutboxRuntimeMetrics) string {
	parts := make([]string, 0, 4)
	if snapshot.LagSeconds > 0 {
		parts = append(parts, "lag="+formatHealthInt(snapshot.LagSeconds)+"s")
	}
	if snapshot.BacklogRows > 0 {
		parts = append(parts, "backlog="+formatHealthInt(snapshot.BacklogRows))
	}
	if snapshot.WatermarkDrift != 0 {
		parts = append(parts, "drift="+formatHealthInt(snapshot.WatermarkDrift))
	}
	if snapshot.CheckpointFallbackStreak > 0 {
		parts = append(parts, "checkpoint_fallback_streak="+formatHealthInt(snapshot.CheckpointFallbackStreak))
	}
	if len(parts) == 0 {
		return "scheduler outbox requires attention"
	}
	return strings.Join(parts, ", ")
}

func compactHealthError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "dependency probe failed"
	}
	return msg
}

func formatHealthInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
