package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

func TestBuildDBPoolSettings(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MaxOpenConns:           50,
			MaxIdleConns:           10,
			ConnMaxLifetimeMinutes: 30,
			ConnMaxIdleTimeMinutes: 5,
		},
	}

	settings := buildDBPoolSettings(cfg)
	require.Equal(t, 50, settings.MaxOpenConns)
	require.Equal(t, 10, settings.MaxIdleConns)
	require.Equal(t, 30*time.Minute, settings.ConnMaxLifetime)
	require.Equal(t, 5*time.Minute, settings.ConnMaxIdleTime)
}

func TestApplyDBPoolSettings(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MaxOpenConns:           40,
			MaxIdleConns:           8,
			ConnMaxLifetimeMinutes: 15,
			ConnMaxIdleTimeMinutes: 3,
		},
	}

	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	applyDBPoolSettings(db, cfg)
	stats := db.Stats()
	require.Equal(t, 40, stats.MaxOpenConnections)
}

func TestEvaluateDBPoolConfigWarnings(t *testing.T) {
	warnings := evaluateDBPoolConfigWarnings(dbPoolSettings{
		MaxOpenConns: 600,
		MaxIdleConns: 550,
	})
	require.NotEmpty(t, warnings)
	contains := false
	for _, warn := range warnings {
		if strings.Contains(warn, "exceeds") {
			contains = true
		}
	}
	require.True(t, contains)
}

func TestEvaluateDBPoolStatsWarnings(t *testing.T) {
	warnings := evaluateDBPoolStatsWarnings(sql.DBStats{
		MaxOpenConnections: 100,
		Idle:               90,
	})
	require.NotEmpty(t, warnings)
}
