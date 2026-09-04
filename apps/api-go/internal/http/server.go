package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/middleware"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
)

func NewRouter(cfg config.Config, logger *slog.Logger, dependencySets ...routes.Dependencies) http.Handler {
	mux := http.NewServeMux()
	routes.RegisterHealthRoutes(mux)

	var deps routes.Dependencies
	if len(dependencySets) > 0 {
		deps = dependencySets[0]
		routes.RegisterWebhookRoutes(mux, deps)
		routes.RegisterWebhookAdminRoutes(mux, deps)
		routes.RegisterAuthRoutes(mux, deps)
		routes.RegisterMessageRoutes(mux, deps)
		routes.RegisterAnalyticsRoutes(mux, deps)
		routes.RegisterProfileRoutes(mux, deps)
		routes.RegisterAdminUserRoutes(mux, deps)
		routes.RegisterAdminChannelRoutes(mux, deps)
		routes.RegisterAdminWatchedSenderRoutes(mux, deps)
		routes.RegisterAdminOperationRoutes(mux, deps)
		routes.RegisterAdminDataManagementRoutes(mux, deps)
	}

	var handler http.Handler = mux
	if cfg.RateLimitEnabled && deps.RateLimiter != nil {
		handler = middleware.RateLimit(
			deps.RateLimiter,
			middleware.DefaultPolicies(cfg.RateLimitTrustProxy, cfg.RateLimitClientIPHeader),
			deps.TokenService,
			cfg,
			logger,
		)(handler)
	}
	handler = middleware.Recover(logger)(handler)
	handler = middleware.RequestLogger(logger)(handler)
	handler = middleware.CORS(cfg.BackendCORSOrigins)(handler)
	return handler
}
