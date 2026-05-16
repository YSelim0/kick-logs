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
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/kick"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	operationsinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/operations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
	messagesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messages"
	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
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

	channelRepo := sqliteinfra.NewFollowedChannelRepository(sqliteDB)
	senderRepo := sqliteinfra.NewSenderProfileRepository(sqliteDB)
	authService := authusecase.NewService(
		adminRepo,
		authinfra.NewBcryptPasswordHasher(),
		authinfra.NewJWTTokenService(cfg),
	)
	channelService := channelsusecase.NewService(channelRepo, kick.NewWebChannelResolver())
	var messageService *messagesusecase.Service
	var analyticsService *analyticsusecase.Service
	var profileService *profilesusecase.Service
	if clickHouseConn != nil {
		messageRepository := clickhouseinfra.NewMessageRepository(clickHouseConn)
		analyticsRepository := clickhouseinfra.NewAnalyticsRepository(clickHouseConn)
		messageService = messagesusecase.NewService(messageRepository)
		analyticsService = analyticsusecase.NewService(analyticsRepository)
		profileService = profilesusecase.NewService(analyticsRepository, channelRepo, senderRepo)
	}
	operationsRepo := operationsinfra.NewRepository(
		sqliteDB,
		cfg.SQLitePath,
		clickHouseConn,
		cfg.ListenerStaleAfter,
	)
	server := app.NewAPIServer(cfg, logger, routes.Dependencies{
		Config:     cfg,
		Auth:       authService,
		Analytics:  analyticsService,
		Channels:   channelService,
		Messages:   messageService,
		Profiles:   profileService,
		Operations: operationsRepo,
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
