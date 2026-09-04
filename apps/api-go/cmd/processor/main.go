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

	// Watched-sender email notification is opt-in: it activates only when an
	// SMTP host and a recipient are both configured, matching the Kick
	// webhook client's "missing credentials -> feature off" pattern. The
	// watched-username list itself is admin-managed (SQLite, edited from the
	// admin panel) and refreshed periodically below, so it can go from empty
	// to populated at runtime with no processor restart.
	var senderWatchlist *watchlist.WatchlistService
	if cfg.SMTPHost != "" && cfg.NotifyEmailTo != "" {
		smtpClient := notifyinfra.NewSMTPClient(notifyinfra.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			To:       cfg.NotifyEmailTo,
		})
		senderWatchlist = watchlist.NewWatchlistService(
			durationFromSeconds(float64(cfg.NotifyEmailCooldownSeconds)),
			smtpClient,
			logger,
		)
		watchedSenders := sqliteinfra.NewWatchedSenderRepository(sqliteDB)
		notificationSettings := sqliteinfra.NewNotificationSettingsRepository(sqliteDB, cfg.NotifyEmailCooldownSeconds)
		refreshInterval := durationFromSeconds(float64(cfg.WatchlistRefreshIntervalSeconds))
		go refreshWatchlistForever(ctx, senderWatchlist, watchedSenders, refreshInterval, logger)
		go refreshCooldownForever(ctx, senderWatchlist, notificationSettings, refreshInterval, logger)
		logger.Info("watched-sender email notification enabled")
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

// refreshWatchlistForever polls the admin-managed watched-senders table and
// pushes the current username list into the in-memory watchlist so
// additions/removals made from the admin panel take effect without a
// processor restart. A read failure is logged and retried on the next tick;
// it never clears the in-memory list, so a transient SQLite hiccup cannot
// silently disable notifications for the current watchlist.
func refreshWatchlistForever(ctx context.Context, watchlistService *watchlist.WatchlistService, repo *sqliteinfra.WatchedSenderRepository, interval time.Duration, logger *slog.Logger) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	for {
		usernames, err := repo.ListUsernames(ctx)
		if err != nil {
			logger.Error("failed to refresh watched-sender list", "error", err)
		} else {
			watchlistService.SetUsernames(usernames)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// refreshCooldownForever polls the admin-managed notification cooldown
// setting and pushes it into the in-memory watchlist so a cooldown change
// made from the admin panel takes effect without a processor restart. A
// read failure is logged and retried on the next tick; it never clears the
// in-memory cooldown, matching refreshWatchlistForever's failure handling.
func refreshCooldownForever(ctx context.Context, watchlistService *watchlist.WatchlistService, repo *sqliteinfra.NotificationSettingsRepository, interval time.Duration, logger *slog.Logger) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	for {
		settings, err := repo.GetNotificationSettings(ctx)
		if err != nil {
			logger.Error("failed to refresh notification cooldown", "error", err)
		} else {
			watchlistService.SetCooldown(durationFromSeconds(float64(settings.CooldownSeconds)))
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
