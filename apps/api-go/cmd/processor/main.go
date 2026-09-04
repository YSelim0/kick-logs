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
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/natsstream"
	notifyinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/notify"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	listenerusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/listener"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/watchlist"
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

	rawEventStream, err := natsstream.Open(ctx, cfg)
	if err != nil {
		logger.Error("failed to open NATS JetStream", "error", err)
		os.Exit(1)
	}
	defer rawEventStream.Close()

	// Watched-sender email notification is opt-in: it activates only when a
	// watchlist, an SMTP host, and a recipient are all configured. Any one
	// missing leaves the feature fully disabled (nil watchlist), matching the
	// Kick webhook client's "missing credentials -> feature off" pattern.
	var senderWatchlist *watchlist.WatchlistService
	if len(cfg.WatchedSenderUsernames) > 0 && cfg.SMTPHost != "" && cfg.NotifyEmailTo != "" {
		smtpClient := notifyinfra.NewSMTPClient(notifyinfra.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			To:       cfg.NotifyEmailTo,
		})
		senderWatchlist = watchlist.NewWatchlistService(
			cfg.WatchedSenderUsernames,
			durationFromSeconds(float64(cfg.NotifyEmailCooldownSeconds)),
			smtpClient,
			logger,
		)
		logger.Info("watched-sender email notification enabled", "watched_senders", len(cfg.WatchedSenderUsernames))
	}

	service := listenerusecase.NewStreamProcessorService(listenerusecase.StreamProcessorDependencies{
		Stream:     rawEventStream,
		RawEvents:  clickhouseinfra.NewRawEventRepository(clickHouseConn),
		Messages:   clickhouseinfra.NewMessageRepository(clickHouseConn),
		Channels:   sqliteinfra.NewFollowedChannelRepository(sqliteDB),
		Senders:    sqliteinfra.NewSenderProfileRepository(sqliteDB),
		Heartbeats: sqliteinfra.NewWorkerHeartbeatRepository(sqliteDB),
		Logger:     logger,
		Watchlist:  senderWatchlist,
		Config: listenerusecase.StreamProcessorConfig{
			BatchSize:                cfg.NATSRawEventFetchBatchSize,
			IdleDelay:                durationFromSeconds(cfg.ListenerRawEventWorkerIdleDelay),
			HeartbeatInterval:        durationFromSeconds(cfg.ListenerHeartbeatInterval),
			HeartbeatServiceName:     "processor",
			NakDelay:                 durationFromSeconds(cfg.ListenerReconnectInitialDelaySeconds),
			SenderProfileCacheTTL:    10 * time.Minute,
			ClickHouseBackoffInitial: time.Duration(cfg.ListenerClickHouseBackoffInitialMS) * time.Millisecond,
			ClickHouseBackoffMax:     time.Duration(cfg.ListenerClickHouseBackoffMaxMS) * time.Millisecond,
			ClickHouseBackoffFactor:  cfg.ListenerClickHouseBackoffMultiplier,
			ClickHouseBreakerThresh:  cfg.ListenerClickHouseBreakerThreshold,
		},
	})

	logger.Info("starting Go raw event processor", "env", cfg.AppEnv)
	if err := service.RunForever(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Go raw event processor failed", "error", err)
		os.Exit(1)
	}
	logger.Info("Go raw event processor stopped")
}

func durationFromSeconds(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
