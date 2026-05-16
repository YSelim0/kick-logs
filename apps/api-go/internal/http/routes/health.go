package routes

import (
	"encoding/json"
	"net/http"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
)

func RegisterHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", health)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(schemas.HealthResponse{Status: "ok"})
}
