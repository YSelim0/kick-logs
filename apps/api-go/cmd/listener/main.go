package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/app"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/kick"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	listenerusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/listener"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := app.NewLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sqliteDB, err := sqliteinfra.Open(ctx, cfg.SQLitePath)
	if err != nil {
		logger.Error("failed to open sqlite", "error", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	if err := migrations.ApplySQLite(ctx, sqliteDB); err != nil {
		logger.Error("failed to apply sqlite migrations", "error", err)
		os.Exit(1)
	}

	clickHouseConn, err := clickhouseinfra.Open(ctx, cfg)
	if err != nil {
		logger.Error("failed to open clickhouse", "error", err)
		os.Exit(1)
	}
	if err := migrations.ApplyClickHouse(ctx, clickHouseConn); err != nil {
		logger.Error("failed to apply clickhouse migrations", "error", err)
		os.Exit(1)
	}

	service := listenerusecase.NewService(listenerusecase.Dependencies{
		Channels:        sqliteinfra.NewFollowedChannelRepository(sqliteDB),
		RawEvents:       clickhouseinfra.NewRawEventRepository(clickHouseConn),
		RawEventClaims:  sqliteinfra.NewRawEventClaimRepository(sqliteDB),
		Messages:        clickhouseinfra.NewMessageRepository(clickHouseConn),
		Senders:         sqliteinfra.NewSenderProfileRepository(sqliteDB),
		Heartbeats:      sqliteinfra.NewWorkerHeartbeatRepository(sqliteDB),
		ChannelResolver: kick.NewWebChannelResolver(),
		SenderResolver:  kick.NewWebSenderProfileResolver(),
		Pusher:          kick.NewPusherClient(cfg.KickPusherURL),
		Logger:          logger,
		Config: listenerusecase.ServiceConfig{
			WorkerCount:               cfg.ListenerWorkerCount,
			RawEventBatchSize:         cfg.ListenerRawEventBatchSize,
			RawEventProcessingTimeout: time.Duration(cfg.ListenerRawEventProcessingTimeout) * time.Second,
			RawEventMaxAttempts:       uint16(cfg.ListenerRawEventMaxAttempts),
			RawEventWorkerIdleDelay:   durationFromSeconds(cfg.ListenerRawEventWorkerIdleDelay),
			ChannelResyncInterval:     durationFromSeconds(cfg.ListenerChannelResyncInterval),
			HeartbeatInterval:         durationFromSeconds(cfg.ListenerHeartbeatInterval),
			ReconnectInitialDelay:     durationFromSeconds(cfg.ListenerReconnectInitialDelaySeconds),
			ReconnectMaxDelay:         durationFromSeconds(cfg.ListenerReconnectMaxDelaySeconds),
			ReconnectMultiplier:       cfg.ListenerReconnectMultiplier,
			HeartbeatServiceName:      "listener",
		},
	})

	logger.Info("starting Go listener", "env", cfg.AppEnv)
	if err := service.RunForever(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Go listener failed", "error", err)
		os.Exit(1)
	}
	logger.Info("Go listener stopped")
}

func durationFromSeconds(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
