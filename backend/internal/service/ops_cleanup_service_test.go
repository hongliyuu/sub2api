package service

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpsCleanupRunCleanupOnceSnapshotsMaxRowsWithoutRetention(t *testing.T) {
	resetCleanupStatsForTest()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(st.n_live_tup::bigint, c.reltuples::bigint, 0\\)").
		WithArgs("ops_error_logs").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1200)))
	mock.ExpectQuery("SELECT COALESCE\\(st.n_live_tup::bigint, c.reltuples::bigint, 0\\)").
		WithArgs("ops_system_logs").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(3400)))

	svc := &OpsCleanupService{
		db: db,
		cfg: &config.Config{
			Ops: config.OpsConfig{
				Cleanup: config.OpsCleanupConfig{
					Enabled:               true,
					ErrorLogRetentionDays: 0,
					SystemLogMaxRows:      3000,
					ErrorLogMaxRows:       1000,
					MaxRowsEnabled:        true,
					MaxRowsDryRun:         true,
				},
			},
		},
	}

	_, err = svc.runCleanupOnce(context.Background())
	require.NoError(t, err)

	stats := SnapshotCleanupStats()
	require.Equal(t, int64(3400), stats.SystemLogRows)
	require.Equal(t, int64(1200), stats.ErrorLogRows)
	require.Equal(t, int64(3000), stats.SystemLogLimit)
	require.Equal(t, int64(1000), stats.ErrorLogLimit)
	require.True(t, stats.MaxRowsEnabled)
	require.True(t, stats.MaxRowsDryRun)
	require.True(t, stats.MaxRowsHit)

	require.NoError(t, mock.ExpectationsWereMet())
}
