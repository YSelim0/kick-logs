package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/app"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
)

func main() {
	target := flag.String("target", "all", "migration target: all, sqlite, or clickhouse")
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

	if *target != "all" && *target != "sqlite" && *target != "clickhouse" {
		logger.Error("invalid migration target", "target", *target)
		os.Exit(1)
	}
}
