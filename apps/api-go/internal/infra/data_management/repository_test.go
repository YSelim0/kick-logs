package datamanagement

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	_ "modernc.org/sqlite"
)

func TestSummaryDoesNotWriteRetentionSettingsWhenRowExists(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/kick-logs.sqlite3"

	db := openTestSQLite(t, path)
	defer db.Close()
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	repo := NewRepository(db, path, nil)
	if _, err := repo.UpdateRetentionSettings(ctx, domain.RetentionSettings{}); err != nil {
		t.Fatalf("seed retention settings: %v", err)
	}

	lockDB := openTestSQLite(t, path)
	defer lockDB.Close()
	tx, err := lockDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin lock tx: %v", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO worker_heartbeats (
			service_name, last_seen_at, metadata_json, created_at, updated_at
		) VALUES ('lock-holder', ?, '{}', ?, ?)`,
		now,
		now,
		now,
	); err != nil {
		t.Fatalf("hold sqlite write lock: %v", err)
	}

	if _, err := repo.Summary(ctx); err != nil {
		t.Fatalf("summary under concurrent sqlite writer: %v", err)
	}
}

func openTestSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(
		"PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 50;",
	); err != nil {
		_ = db.Close()
		t.Fatalf("configure sqlite: %v", err)
	}
	return db
}
