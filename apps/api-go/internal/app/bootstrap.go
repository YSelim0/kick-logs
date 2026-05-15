package app

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	httpapi "github.com/YSelim0/kick-logs/apps/api-go/internal/http"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
)

func NewLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "WARN", "WARNING":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
}

func NewAPIServer(cfg config.Config, logger *slog.Logger, dependencySets ...routes.Dependencies) *http.Server {
	return &http.Server{
		Addr:              cfg.APIAddress(),
		Handler:           httpapi.NewRouter(cfg, logger, dependencySets...),
		ReadHeaderTimeout: 5 * time.Second,
	}
}
