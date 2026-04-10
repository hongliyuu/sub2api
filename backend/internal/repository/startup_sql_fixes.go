package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/Wei-Shaw/sub2api/migrations"
)

type startupSQLFix struct {
	filename string
	needsRun func(context.Context, *sql.DB) (bool, error)
}

var startupSQLFixes = []startupSQLFix{
	{
		filename: "083_disable_error_passthrough_body.sql",
		needsRun: needsDisableErrorPassthroughBodyFix,
	},
	{
		filename: "084_migrate_openai_passthrough_key.sql",
		needsRun: needsMigrateOpenAIPassthroughKeyFix,
	},
}

func applyEmbeddedStartupSQLFixes(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("apply embedded startup sql fixes: nil sql db")
	}

	for _, fix := range startupSQLFixes {
		need, err := fix.needsRun(ctx, db)
		if err != nil {
			return fmt.Errorf("check startup sql fix %s: %w", fix.filename, err)
		}
		if !need {
			continue
		}

		sqlText, err := readEmbeddedMigrationSQL(fix.filename)
		if err != nil {
			return fmt.Errorf("load startup sql fix %s: %w", fix.filename, err)
		}
		if _, err := db.ExecContext(ctx, sqlText); err != nil {
			return fmt.Errorf("apply startup sql fix %s: %w", fix.filename, err)
		}
		log.Printf("[DBInit] Applied startup SQL fix: %s", fix.filename)
	}

	return nil
}

func readEmbeddedMigrationSQL(filename string) (string, error) {
	contentBytes, err := fs.ReadFile(migrations.FS, filename)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(contentBytes))
	if content == "" {
		return "", fmt.Errorf("empty sql content")
	}
	return content, nil
}

func needsDisableErrorPassthroughBodyFix(ctx context.Context, db *sql.DB) (bool, error) {
	hasTable, err := tableExists(ctx, db, "error_passthrough_rules")
	if err != nil {
		return false, fmt.Errorf("check error_passthrough_rules table: %w", err)
	}
	if !hasTable {
		return false, nil
	}

	hasColumn, err := columnExists(ctx, db, "error_passthrough_rules", "passthrough_body")
	if err != nil {
		return false, fmt.Errorf("check passthrough_body column: %w", err)
	}
	if !hasColumn {
		return false, nil
	}

	var columnDefault sql.NullString
	if err := db.QueryRowContext(ctx, `
		SELECT column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
	`, "error_passthrough_rules", "passthrough_body").Scan(&columnDefault); err != nil {
		return false, fmt.Errorf("query passthrough_body default: %w", err)
	}

	defaultExpr := strings.ToLower(strings.TrimSpace(columnDefault.String))
	defaultIsFalse := strings.Contains(defaultExpr, "false")

	var hasEnabledRows bool
	if err := db.QueryRowContext(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM error_passthrough_rules WHERE passthrough_body = true)",
	).Scan(&hasEnabledRows); err != nil {
		return false, fmt.Errorf("query passthrough_body data: %w", err)
	}

	return !defaultIsFalse || hasEnabledRows, nil
}

func needsMigrateOpenAIPassthroughKeyFix(ctx context.Context, db *sql.DB) (bool, error) {
	hasTable, err := tableExists(ctx, db, "accounts")
	if err != nil {
		return false, fmt.Errorf("check accounts table: %w", err)
	}
	if !hasTable {
		return false, nil
	}

	hasExtraColumn, err := columnExists(ctx, db, "accounts", "extra")
	if err != nil {
		return false, fmt.Errorf("check accounts.extra column: %w", err)
	}
	if !hasExtraColumn {
		return false, nil
	}

	var hasLegacyKeys bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM accounts
			WHERE platform = 'openai'
			  AND extra IS NOT NULL
			  AND (extra ? 'openai_passthrough' OR extra ? 'openai_oauth_passthrough')
		)
	`).Scan(&hasLegacyKeys); err != nil {
		return false, fmt.Errorf("query openai passthrough legacy keys: %w", err)
	}
	return hasLegacyKeys, nil
}

func columnExists(ctx context.Context, db *sql.DB, tableName, columnName string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}
