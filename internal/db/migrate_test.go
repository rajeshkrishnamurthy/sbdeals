package db

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestMigratorUpAppliesPendingMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"migrations/0001_create_suppliers.sql": {
			Data: []byte("CREATE TABLE suppliers (id BIGSERIAL PRIMARY KEY);"),
		},
	}

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`)).
		WithArgs("0001_create_suppliers.sql").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE suppliers (id BIGSERIAL PRIMARY KEY);")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migrations(version) VALUES($1)`)).
		WithArgs("0001_create_suppliers.sql").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	migrator := NewMigrator(migrations, "migrations")
	if err := migrator.Up(context.Background(), db); err != nil {
		t.Fatalf("up returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMigratorUpSkipsAlreadyAppliedMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"migrations/0001_create_suppliers.sql": {
			Data: []byte("CREATE TABLE suppliers (id BIGSERIAL PRIMARY KEY);"),
		},
	}

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`)).
		WithArgs("0001_create_suppliers.sql").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	migrator := NewMigrator(migrations, "migrations")
	if err := migrator.Up(context.Background(), db); err != nil {
		t.Fatalf("up returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestApplyMigrationRollsBackOnMigrationExecFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"migrations/0001.sql": {Data: []byte("CREATE TABLE broken;")},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE broken;")).
		WillReturnError(assertionErr("exec failed"))
	mock.ExpectRollback()

	err = applyMigration(context.Background(), db, migrations, "migrations/0001.sql", "0001.sql")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "execute migration 0001.sql") {
		t.Fatalf("expected execute migration context, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestApplyMigrationRollsBackOnVersionRecordFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"migrations/0001.sql": {Data: []byte("CREATE TABLE ok (id BIGSERIAL PRIMARY KEY);")},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE ok (id BIGSERIAL PRIMARY KEY);")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migrations(version) VALUES($1)`)).
		WithArgs("0001.sql").
		WillReturnError(assertionErr("insert failed"))
	mock.ExpectRollback()

	err = applyMigration(context.Background(), db, migrations, "migrations/0001.sql", "0001.sql")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "record migration 0001.sql") {
		t.Fatalf("expected record migration context, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestApplyMigrationReturnsCommitFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"migrations/0001.sql": {Data: []byte("CREATE TABLE ok (id BIGSERIAL PRIMARY KEY);")},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE ok (id BIGSERIAL PRIMARY KEY);")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migrations(version) VALUES($1)`)).
		WithArgs("0001.sql").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(assertionErr("commit failed"))

	err = applyMigration(context.Background(), db, migrations, "migrations/0001.sql", "0001.sql")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "commit migration 0001.sql") {
		t.Fatalf("expected commit migration context, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func assertionErr(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
