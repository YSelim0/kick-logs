package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/app"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	postgresinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/postgres"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/datamigration"
)

func main() {
	target := flag.String("target", "all", "migration target: all, sqlite, clickhouse, or data")
	sourcePostgresURL := flag.String("source-postgres-url", "", "PostgreSQL source URL for target=data")
	dryRun := flag.Bool("dry-run", false, "read and validate source data without writing migrated rows")
	execute := flag.Bool("execute", false, "execute PostgreSQL to SQLite/ClickHouse data migration")
	validationOnly := flag.Bool("validation-only", false, "validate destination data against PostgreSQL source without migrating")
	batchSize := flag.Int("batch-size", 500, "data migration batch size")
	sampleSize := flag.Int("sample-size", 5, "number of rows per table to sample during validation")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := app.NewLogger(cfg.LogLevel)
	ctx := context.Background()

	if *target == "all" || *target == "sqlite" {
		db, err := sqliteinfra.Open(ctx, cfg.SQLitePath)
		if err != nil {
			logger.Error("failed to open sqlite", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		if err := migrations.ApplySQLite(ctx, db); err != nil {
			logger.Error("failed to apply sqlite migrations", "error", err)
			os.Exit(1)
		}

		adminRepo := sqliteinfra.NewAdminUserRepository(db)
		if _, err := sqliteinfra.SeedSuperAdmin(ctx, adminRepo, cfg.DefaultAdminEmail, cfg.DefaultAdminPassword); err != nil {
			logger.Error("failed to seed default super admin", "error", err)
			os.Exit(1)
		}
		logger.Info("SQLite migrations applied", "path", cfg.SQLitePath)
	}

	if *target == "all" || *target == "clickhouse" {
		conn, err := clickhouseinfra.Open(ctx, cfg)
		if err != nil {
			logger.Error("failed to open clickhouse", "error", err)
			os.Exit(1)
		}

		if err := migrations.ApplyClickHouse(ctx, conn); err != nil {
			logger.Error("failed to apply clickhouse migrations", "error", err)
			os.Exit(1)
		}
		logger.Info("ClickHouse migrations applied", "addr", cfg.ClickHouseAddr, "database", cfg.ClickHouseDatabase)
	}

	if *target == "data" {
		if err := runDataMigration(
			ctx,
			cfg,
			logger,
			*sourcePostgresURL,
			datamigration.Options{
				DryRun:         *dryRun,
				Execute:        *execute,
				ValidationOnly: *validationOnly,
				BatchSize:      *batchSize,
				SampleSize:     *sampleSize,
			},
		); err != nil {
			logger.Error("data migration failed", "error", err)
			os.Exit(1)
		}
	}

	if *target != "all" && *target != "sqlite" && *target != "clickhouse" && *target != "data" {
		logger.Error("invalid migration target", "target", *target)
		os.Exit(1)
	}
}

func runDataMigration(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	sourcePostgresURL string,
	options datamigration.Options,
) error {
	if selectedModes(options) > 1 {
		return fmt.Errorf("choose only one of -dry-run, -execute, or -validation-only")
	}

	dsn := sourcePostgresURL
	if dsn == "" {
		dsn = cfg.PostgresSourceDSN
	}
	source, err := postgresinfra.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer source.Close()

	sqliteDB, err := sqliteinfra.Open(ctx, cfg.SQLitePath)
	if err != nil {
		return err
	}
	defer sqliteDB.Close()
	if err := migrations.ApplySQLite(ctx, sqliteDB); err != nil {
		return err
	}

	clickhouseConn, err := clickhouseinfra.Open(ctx, cfg)
	if err != nil {
		return err
	}
	if err := migrations.ApplyClickHouse(ctx, clickhouseConn); err != nil {
		return err
	}

	service := datamigration.NewService(datamigration.Dependencies{
		Source:  source,
		Control: sqliteinfra.NewDataMigrationRepository(sqliteDB),
		Data:    clickhouseinfra.NewDataMigrationRepository(clickhouseConn),
	})
	report, err := service.Run(ctx, options)
	if err != nil {
		return err
	}

	logger.Info(
		"data migration completed",
		"mode", report.Mode,
		"source_counts", report.SourceCounts,
		"destination_counts", report.DestinationCounts,
		"written_counts", report.WrittenCounts,
	)
	return nil
}

func selectedModes(options datamigration.Options) int {
	count := 0
	if options.DryRun {
		count++
	}
	if options.Execute {
		count++
	}
	if options.ValidationOnly {
		count++
	}
	return count
}
