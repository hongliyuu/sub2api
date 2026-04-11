package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type dbPoolSettings struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func buildDBPoolSettings(cfg *config.Config) dbPoolSettings {
	return dbPoolSettings{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Database.ConnMaxLifetimeMinutes) * time.Minute,
		ConnMaxIdleTime: time.Duration(cfg.Database.ConnMaxIdleTimeMinutes) * time.Minute,
	}
}

func applyDBPoolSettings(db *sql.DB, cfg *config.Config) {
	if db == nil || cfg == nil {
		return
	}

	settings := buildDBPoolSettings(cfg)
	db.SetMaxOpenConns(settings.MaxOpenConns)
	db.SetMaxIdleConns(settings.MaxIdleConns)
	db.SetConnMaxLifetime(settings.ConnMaxLifetime)
	db.SetConnMaxIdleTime(settings.ConnMaxIdleTime)
	logDBPoolStartupWarnings(settings)
	logDBPoolRuntimeWarnings(db)
}

const (
	dbPoolStartMaxOpenWarningThreshold    = 500
	dbPoolStartIdleRatioWarningThreshold  = 0.9
	dbPoolRuntimeIdleRatioWarningThreshold = 0.85
)

func logDBPoolStartupWarnings(settings dbPoolSettings) {
	warnings := evaluateDBPoolConfigWarnings(settings)
	if len(warnings) == 0 {
		return
	}
	logger.WriteSinkEvent("warn", "repository.db_pool", "database pool configuration guard", map[string]any{
		"warnings":        warnings,
		"max_open_conns":  settings.MaxOpenConns,
		"max_idle_conns":  settings.MaxIdleConns,
		"conn_max_life":   settings.ConnMaxLifetime.String(),
		"conn_max_idle":   settings.ConnMaxIdleTime.String(),
	})
}

func logDBPoolRuntimeWarnings(db *sql.DB) {
	if db == nil {
		return
	}
	stats := db.Stats()
	warnings := evaluateDBPoolStatsWarnings(stats)
	if len(warnings) == 0 {
		return
	}
	logger.WriteSinkEvent("warn", "repository.db_pool", "database pool runtime guard", map[string]any{
		"warnings":          warnings,
		"open_connections":  stats.OpenConnections,
		"idle_connections":  stats.Idle,
		"max_open_conns":    stats.MaxOpenConnections,
		"in_use":            stats.InUse,
	})
}

func evaluateDBPoolConfigWarnings(settings dbPoolSettings) []string {
	var warnings []string
	if settings.MaxOpenConns > dbPoolStartMaxOpenWarningThreshold {
		warnings = append(warnings, fmt.Sprintf("max_open_conns (%d) exceeds %d safety threshold", settings.MaxOpenConns, dbPoolStartMaxOpenWarningThreshold))
	}
	if settings.MaxOpenConns > 0 && settings.MaxIdleConns > 0 {
		ratio := float64(settings.MaxIdleConns) / float64(settings.MaxOpenConns)
		if ratio >= dbPoolStartIdleRatioWarningThreshold {
			warnings = append(warnings, fmt.Sprintf("max_idle_conns (%d) is %.0f%% of max_open_conns", settings.MaxIdleConns, ratio*100))
		}
	}
	if settings.MaxIdleConns >= settings.MaxOpenConns && settings.MaxOpenConns > 0 {
		warnings = append(warnings, fmt.Sprintf("max_idle_conns (%d) >= max_open_conns (%d)", settings.MaxIdleConns, settings.MaxOpenConns))
	}
	return warnings
}

func evaluateDBPoolStatsWarnings(stats sql.DBStats) []string {
	var warnings []string
	if stats.MaxOpenConnections > 0 {
		ratio := float64(stats.Idle) / float64(stats.MaxOpenConnections)
		if ratio >= dbPoolRuntimeIdleRatioWarningThreshold {
			warnings = append(warnings, fmt.Sprintf("idle_connections (%.0f%%) close to max_open_conns", ratio*100))
		}
	}
	return warnings
}
