package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/app"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	authinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/auth"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	datamanagementinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/data_management"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/kick"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/natsstream"
	operationsinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/operations"
	ratelimitinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/ratelimit"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
	datamanagementusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/data_management"
	kicksyncusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/kicksync"
	messagesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messages"
	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
	webhookprocessorusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/webhookprocessor"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := app.NewLogger(cfg.LogLevel)

	sqliteDB, err := sqliteinfra.Open(context.Background(), cfg.SQLitePath)
	if err != nil {
		logger.Error("failed to open sqlite", "error", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	if err := migrations.ApplySQLite(context.Background(), sqliteDB); err != nil {
		logger.Error("failed to apply sqlite migrations", "error", err)
		os.Exit(1)
	}

	adminRepo := sqliteinfra.NewAdminUserRepository(sqliteDB)
	if cfg.SeedSuperAdmin {
		if _, err := sqliteinfra.SeedSuperAdmin(
			context.Background(),
			adminRepo,
			cfg.DefaultAdminEmail,
			cfg.DefaultAdminPassword,
		); err != nil {
			logger.Error("failed to seed default super admin", "error", err)
			os.Exit(1)
		}
	}

	var clickHouseConn driver.Conn
	clickHouseConn, err = clickhouseinfra.Open(context.Background(), cfg)
	if err != nil {
		logger.Warn("ClickHouse unavailable; operations summary will use SQLite-only data", "error", err)
	} else if err := migrations.ApplyClickHouse(context.Background(), clickHouseConn); err != nil {
		logger.Warn("ClickHouse migrations failed; operations summary will use SQLite-only data", "error", err)
		clickHouseConn = nil
	}
	var rawStreamStats ports.RawEventStreamStatsRepository
	rawEventStream, err := natsstream.Open(context.Background(), cfg)
	if err != nil {
		logger.Warn("NATS JetStream unavailable; operations summary will omit stream backlog data", "error", err)
	} else {
		defer rawEventStream.Close()
		rawStreamStats = rawEventStream
	}

	channelRepo := sqliteinfra.NewFollowedChannelRepository(sqliteDB)
	senderRepo := sqliteinfra.NewSenderProfileRepository(sqliteDB)
	tokenService := authinfra.NewJWTTokenService(cfg)
	authService := authusecase.NewService(
		adminRepo,
		authinfra.NewBcryptPasswordHasher(),
		tokenService,
	)

	var rateLimiter ports.RateLimiter
	if cfg.RateLimitEnabled {
		rl, err := ratelimitinfra.NewGCRA(cfg.RateLimitStoreMaxKeys)
		if err != nil {
			logger.Error("failed to create rate limiter", "error", err)
			os.Exit(1)
		}
		rateLimiter = rl
		logger.Info("rate limiter enabled", "max_keys", cfg.RateLimitStoreMaxKeys, "trust_proxy", cfg.RateLimitTrustProxy)
	}
	channelService := channelsusecase.NewService(channelRepo, kick.NewWebChannelResolver())

	webhookEventRepo := sqliteinfra.NewKickWebhookEventRepository(sqliteDB)
	eventSubRepo := sqliteinfra.NewKickEventSubscriptionRepository(sqliteDB)

	var kickSyncService *kicksyncusecase.Service
	if cfg.KickClientID != "" && cfg.KickClientSecret != "" {
		kickAPIClient := kick.NewEventSubscriptionClient(cfg.KickAPIBaseURL, cfg.KickOAuthTokenURL, cfg.KickClientID, cfg.KickClientSecret)
		kickSyncService = kicksyncusecase.NewService(logger, channelRepo, eventSubRepo, kickAPIClient, cfg.KickWebhookEvents)
		if cfg.KickWebhookSyncEnabled {
			go func() {
				kickSyncService.SyncAll(context.Background())
			}()
		}
	} else {
		logger.Warn("Kick client credentials not configured; webhook subscription sync is disabled")
	}
	var webhookVerifier ports.KickWebhookVerifier
	resolvedPublicKey := cfg.KickWebhookPublicKey
	if resolvedPublicKey == "" && kickSyncService != nil {
		// Auto-fetch the webhook public key from the Kick API when credentials are available.
		kickAPIClient := kick.NewEventSubscriptionClient(cfg.KickAPIBaseURL, cfg.KickOAuthTokenURL, cfg.KickClientID, cfg.KickClientSecret)
		if fetched, err := kickAPIClient.FetchWebhookPublicKey(context.Background()); err != nil {
			logger.Warn("could not auto-fetch KICK_WEBHOOK_PUBLIC_KEY; POST /webhooks/kick will reject all requests", "error", err)
		} else {
			resolvedPublicKey = fetched
			logger.Info("fetched Kick webhook public key from API")
		}
	}
	if resolvedPublicKey != "" {
		v, err := kick.NewWebhookVerifier(resolvedPublicKey)
		if err != nil {
			logger.Warn("KICK_WEBHOOK_PUBLIC_KEY is invalid; POST /webhooks/kick will reject all requests", "error", err)
		} else {
			webhookVerifier = v
		}
	} else if cfg.KickWebhookSkipVerification {
		logger.Warn("KICK_WEBHOOK_SKIP_VERIFICATION=true; webhook signature verification bypassed (test mode only)")
	} else {
		logger.Warn("KICK_WEBHOOK_PUBLIC_KEY not configured; POST /webhooks/kick will reject all requests")
	}
	var messageService *messagesusecase.Service
	var analyticsService *analyticsusecase.Service
	var profileService *profilesusecase.Service
	var subPeriodRepoForAPI ports.SubscriptionPeriodRepository
	if clickHouseConn != nil {
		messageRepository := clickhouseinfra.NewMessageRepository(clickHouseConn)
		analyticsRepository := clickhouseinfra.NewAnalyticsRepository(clickHouseConn)
		subPeriodRepo := clickhouseinfra.NewSubscriptionPeriodRepository(clickHouseConn)
		subPeriodRepoForAPI = subPeriodRepo
		messageService = messagesusecase.NewService(messageRepository)
		analyticsService = analyticsusecase.NewService(analyticsRepository)
		profileService = profilesusecase.NewService(analyticsRepository, channelRepo, senderRepo)

		processorSvc := webhookprocessorusecase.NewService(
			logger,
			webhookEventRepo,
			channelRepo,
			subPeriodRepo,
			cfg.KickWebhookProcessBatchSize,
			cfg.KickWebhookProcessMaxAttempts,
		)
		processorSvc.Start(context.Background())
	}
	operationsRepo := operationsinfra.NewRepository(
		sqliteDB,
		cfg.SQLitePath,
		clickHouseConn,
		cfg.ListenerStaleAfter,
		rawStreamStats,
	)
	dataManagementService := datamanagementusecase.NewService(
		datamanagementinfra.NewRepository(sqliteDB, cfg.SQLitePath, clickHouseConn),
	)
	server := app.NewAPIServer(cfg, logger, routes.Dependencies{
		Config:              cfg,
		Auth:                authService,
		Analytics:           analyticsService,
		Channels:            channelService,
		Messages:            messageService,
		Profiles:            profileService,
		Data:                dataManagementService,
		KickSync:            kickSyncService,
		WebhookEvents:       webhookEventRepo,
		WebhookVerifier:     webhookVerifier,
		WebhookEventSubs:    eventSubRepo,
		SubscriptionPeriods: subPeriodRepoForAPI,
		Operations:          operationsRepo,
		RateLimiter:         rateLimiter,
		TokenService:        tokenService,
	})

	errs := make(chan error, 1)
	go func() {
		logger.Info("starting Go API", "address", server.Addr, "env", cfg.AppEnv)
		errs <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shut down API server", "error", err)
			os.Exit(1)
		}
		logger.Info("Go API stopped")
	case err := <-errs:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Go API failed", "error", err)
			os.Exit(1)
		}
	}
}
