package httpapi

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"context"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	authinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/auth"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	ratelimitinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/ratelimit"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
)

func newRateLimitTestRouter(t *testing.T) http.Handler {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "rl-test.sqlite3")
	db, err := sqliteinfra.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		t.Fatalf("sqlite migrate: %v", err)
	}

	cfg := config.Config{
		BackendCORSOrigins:    []string{"http://localhost:3000"},
		JWTSecretKey:          "test-secret-key",
		JWTAlgorithm:          "HS256",
		JWTExpiresMinutes:     60,
		JWTCookieName:         "kick_logs_session",
		JWTCookieSameSite:     "lax",
		RateLimitEnabled:      true,
		RateLimitStoreMaxKeys: 1000,
		RateLimitTrustProxy:   false,
	}

	adminRepo := sqliteinfra.NewAdminUserRepository(db)
	if _, err := sqliteinfra.SeedSuperAdmin(ctx, adminRepo, "admin@kicklogs.local", "admin123"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	tokenService := authinfra.NewJWTTokenService(cfg)
	authService := authusecase.NewService(adminRepo, authinfra.NewBcryptPasswordHasher(), tokenService)

	rl, err := ratelimitinfra.NewGCRA(cfg.RateLimitStoreMaxKeys)
	if err != nil {
		t.Fatalf("NewGCRA: %v", err)
	}

	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:       cfg,
		Auth:         authService,
		TokenService: tokenService,
		RateLimiter:  rl,
	})
}

func loginBody(email, password string) *bytes.Buffer {
	return bytes.NewBufferString(`{"email":"` + email + `","password":"` + password + `"}`)
}

func TestLoginRateLimit_IPBlocked(t *testing.T) {
	h := newRateLimitTestRouter(t)

	// Use a distinct email per request so the IP+email handler limit
	// (8/10min burst 3) never trips; only the middleware IP-only policy
	// (login-ip: 20/10min burst 5 -> effective capacity 6) gates here.
	// This makes the test actually exercise the IP policy, not the email one.
	login := func(email string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/auth/login", loginBody(email, "wrong"))
		r.Header.Set("Content-Type", "application/json")
		r.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	for i := 0; i < 6; i++ {
		email := "user" + strconv.Itoa(i) + "@example.com"
		if code := login(email).Code; code == http.StatusTooManyRequests {
			t.Fatalf("request %d should pass the IP limit (capacity 6), got 429", i)
		}
	}

	w := login("user6@example.com")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after IP limit exhausted, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing on 429")
	}
}

func TestLoginRateLimit_DifferentIPSameEmail(t *testing.T) {
	h := newRateLimitTestRouter(t)

	for i := 0; i < 9; i++ {
		r := httptest.NewRequest(http.MethodPost, "/auth/login", loginBody("victim@example.com", "wrong"))
		r.Header.Set("Content-Type", "application/json")
		r.RemoteAddr = "10.0.0.2:1111"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}

	r := httptest.NewRequest(http.MethodPost, "/auth/login", loginBody("victim@example.com", "wrong"))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "10.0.0.3:2222"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code == http.StatusTooManyRequests {
		t.Fatal("different IP should not be blocked even when same email was hammered from another IP")
	}
}

func TestHealthEndpoint_NeverLimited(t *testing.T) {
	h := newRateLimitTestRouter(t)

	for i := 0; i < 50; i++ {
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.RemoteAddr = "10.0.0.4:9999"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("health request %d should never be limited, got %d", i, w.Code)
		}
	}
}
