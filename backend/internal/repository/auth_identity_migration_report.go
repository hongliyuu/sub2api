package repository

import (
	"context"
	"time"
)

type AuthIdentityMigrationReport struct {
	ID                              int64
	EmailIdentityBackfilledCount    int64
	EmailIdentityMismatchCount      int64
	WeChatUnionIDMigratedCount      int64
	WeChatLegacyRepairableCount     int64
	WeChatManualRepairRequiredCount int64
	CreatedAt                       time.Time
}

func (r *userRepository) LoadLatestAuthIdentityMigrationReport(ctx context.Context) (*AuthIdentityMigrationReport, error) {
	if r == nil || r.sql == nil {
		return nil, nil
	}

	report := &AuthIdentityMigrationReport{}
	if err := scanSingleRow(ctx, r.sql, `
SELECT
	id,
	email_identity_backfilled_count,
	email_identity_mismatch_count,
	wechat_unionid_migrated_count,
	wechat_legacy_repairable_count,
	wechat_manual_repair_required_count,
	created_at
FROM auth_identity_migration_reports
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		nil,
		&report.ID,
		&report.EmailIdentityBackfilledCount,
		&report.EmailIdentityMismatchCount,
		&report.WeChatUnionIDMigratedCount,
		&report.WeChatLegacyRepairableCount,
		&report.WeChatManualRepairRequiredCount,
		&report.CreatedAt,
	); err != nil {
		return nil, err
	}

	return report, nil
}
