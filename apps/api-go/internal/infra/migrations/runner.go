package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type SQLiteMigration struct {
	Version    int
	Name       string
	Statements []string
}

type ClickHouseMigration struct {
	Version    int
	Name       string
	Statements []string
}

func ApplySQLite(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, sqliteMigrationTableSQL); err != nil {
		return fmt.Errorf("create sqlite migration table: %w", err)
	}

	for _, migration := range SQLiteMigrations() {
		if err := applySQLiteMigration(ctx, db, migration); err != nil {
			return err
		}
	}
	return nil
}

func ApplyClickHouse(ctx context.Context, conn driver.Conn) error {
	if err := conn.Exec(ctx, clickHouseMigrationTableSQL); err != nil {
		return fmt.Errorf("create clickhouse migration table: %w", err)
	}

	for _, migration := range ClickHouseMigrations() {
		if err := applyClickHouseMigration(ctx, conn, migration); err != nil {
			return err
		}
	}
	return nil
}

func applySQLiteMigration(ctx context.Context, db *sql.DB, migration SQLiteMigration) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migration.Version).Scan(&count); err != nil {
		return fmt.Errorf("check sqlite migration %d: %w", migration.Version, err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite migration %d: %w", migration.Version, err)
	}
	defer rollbackQuietly(tx)

	for _, statement := range migration.Statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply sqlite migration %d %s: %w", migration.Version, migration.Name, err)
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
		migration.Version,
		migration.Name,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("record sqlite migration %d: %w", migration.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite migration %d: %w", migration.Version, err)
	}
	return nil
}

func applyClickHouseMigration(ctx context.Context, conn driver.Conn, migration ClickHouseMigration) error {
	var count uint64
	if err := conn.QueryRow(ctx, "SELECT count() FROM schema_migrations WHERE version = ?", uint32(migration.Version)).Scan(&count); err != nil {
		return fmt.Errorf("check clickhouse migration %d: %w", migration.Version, err)
	}
	if count > 0 {
		return nil
	}

	for _, statement := range migration.Statements {
		if err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply clickhouse migration %d %s: %w", migration.Version, migration.Name, err)
		}
	}

	if err := conn.Exec(
		ctx,
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
		uint32(migration.Version),
		migration.Name,
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("record clickhouse migration %d: %w", migration.Version, err)
	}
	return nil
}

func rollbackQuietly(tx *sql.Tx) {
	_ = tx.Rollback()
}
