package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryGetAllGroupUsageSummary_UsesAggregatesAndTail(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	watermark := time.Date(2026, 4, 12, 10, 30, 0, 0, time.UTC)

	mock.ExpectQuery("SELECT last_aggregated_at FROM usage_dashboard_aggregation_watermark").
		WillReturnRows(sqlmock.NewRows([]string{"last_aggregated_at"}).AddRow(watermark))

	rows := sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
		AddRow(int64(1), 12.5, 1.5).
		AddRow(int64(2), 0.0, 0.0)
	mock.ExpectQuery("usage_group_daily_costs").
		WithArgs(watermark, todayStart).
		WillReturnRows(rows)

	got, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, int64(1), got[0].GroupID)
	require.Equal(t, 12.5, got[0].TotalCost)
	require.Equal(t, 1.5, got[0].TodayCost)
	require.Equal(t, int64(2), got[1].GroupID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummary_FallsBackToRawWhenAggregateTableUnavailable(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	watermark := time.Date(2026, 4, 12, 10, 30, 0, 0, time.UTC)

	mock.ExpectQuery("SELECT last_aggregated_at FROM usage_dashboard_aggregation_watermark").
		WillReturnRows(sqlmock.NewRows([]string{"last_aggregated_at"}).AddRow(watermark))
	mock.ExpectQuery("usage_group_daily_costs").
		WithArgs(watermark, todayStart).
		WillReturnError(&pq.Error{Code: "42P01"})

	rows := sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
		AddRow(int64(1), 8.5, 2.0)
	mock.ExpectQuery("WITH usage AS").
		WithArgs(todayStart).
		WillReturnRows(rows)

	got, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(1), got[0].GroupID)
	require.Equal(t, 8.5, got[0].TotalCost)
	require.Equal(t, 2.0, got[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}
