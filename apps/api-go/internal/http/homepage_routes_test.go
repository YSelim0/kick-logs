package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	ratelimitinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/ratelimit"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/homepage"
)

func TestHomepageRegisteredAndRateLimitedWithoutDatabaseReads(t *testing.T) {
	limiter, err := ratelimitinfra.NewGCRA(1000)
	if err != nil {
		t.Fatal(err)
	}
	router := NewRouter(config.Config{RateLimitEnabled: true}, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Homepage: homepage.NewService(nil, nil, nil), RateLimiter: limiter,
	})
	for i := 0; i < 17; i++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/analytics/homepage", nil))
		want := http.StatusAccepted
		if i == 16 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want || response.Header().Get("Retry-After") == "" {
			t.Fatalf("request %d: status %d, want %d; %s", i, response.Code, want, response.Body.String())
		}
	}
}
