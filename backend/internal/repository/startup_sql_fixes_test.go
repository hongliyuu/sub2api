package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestApplyEmbeddedStartupSQLFixes_AppliesWhenNeeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("error_passthrough_rules").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("error_passthrough_rules", "passthrough_body").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT column_default\\s+FROM information_schema.columns").
		WithArgs("error_passthrough_rules", "passthrough_body").
		WillReturnRows(sqlmock.NewRows([]string{"column_default"}).AddRow("true"))
	mock.ExpectQuery("SELECT EXISTS \\(SELECT 1 FROM error_passthrough_rules WHERE passthrough_body = true\\)").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec("ALTER TABLE error_passthrough_rules").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("accounts").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("accounts", "extra").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS \\(\\s*SELECT 1\\s*FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE accounts").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = applyEmbeddedStartupSQLFixes(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyEmbeddedStartupSQLFixes_SkipsWhenAlreadySatisfied(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("error_passthrough_rules").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("error_passthrough_rules", "passthrough_body").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT column_default\\s+FROM information_schema.columns").
		WithArgs("error_passthrough_rules", "passthrough_body").
		WillReturnRows(sqlmock.NewRows([]string{"column_default"}).AddRow("'false'::boolean"))
	mock.ExpectQuery("SELECT EXISTS \\(SELECT 1 FROM error_passthrough_rules WHERE passthrough_body = true\\)").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("accounts").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("accounts", "extra").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS \\(\\s*SELECT 1\\s*FROM accounts").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err = applyEmbeddedStartupSQLFixes(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
