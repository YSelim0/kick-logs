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
	if len(dependencySets) > 0 {
		deps := dependencySets[0]
		routes.RegisterAuthRoutes(mux, deps)
		routes.RegisterMessageRoutes(mux, deps)
		routes.RegisterAdminUserRoutes(mux, deps)
		routes.RegisterAdminChannelRoutes(mux, deps)
		routes.RegisterAdminOperationRoutes(mux, deps)
	}

	var handler http.Handler = mux
	handler = middleware.Recover(logger)(handler)
	handler = middleware.RequestLogger(logger)(handler)
	handler = middleware.CORS(cfg.BackendCORSOrigins)(handler)
	return handler
}
