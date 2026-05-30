package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	ratelimitinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/ratelimit"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type fakeTokenService struct {
	userID int64
	ok     bool
}

func (f fakeTokenService) CreateAccessToken(_ int64) (string, error) { return "", nil }
func (f fakeTokenService) GetUserID(_ string) (int64, bool)          { return f.userID, f.ok }

type fakeRateLimiter struct {
	result ports.RateLimitResult
	err    error
}

func (f *fakeRateLimiter) RateLimit(_ string, _, _, _ int) (ports.RateLimitResult, error) {
	return f.result, f.err
}

func newTestLimiter(t *testing.T) ports.RateLimiter {
	t.Helper()
	l, err := ratelimitinfra.NewGCRA(1000)
	if err != nil {
		t.Fatalf("NewGCRA: %v", err)
	}
	return l
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func cfg() config.Config {
	return config.Config{
		JWTCookieName:       "kick_logs_session",
		RateLimitTrustProxy: true,
	}
}

func TestClientIP_CFConnectingIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	r.RemoteAddr = "127.0.0.1:12345"

	if got := ClientIP(r, true); got != "1.2.3.4" {
		t.Fatalf("got %q want 1.2.3.4", got)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	r.RemoteAddr = "5.6.7.8:9999"

	if got := ClientIP(r, false); got != "5.6.7.8" {
		t.Fatalf("got %q want 5.6.7.8", got)
	}
}

func TestClientIP_FallbackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "9.10.11.12:8080"

	if got := ClientIP(r, true); got != "9.10.11.12" {
		t.Fatalf("got %q want 9.10.11.12", got)
	}
}

func TestClientIP_RemoteAddrPortStrip(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"

	if got := ClientIP(r, false); got != "192.168.1.1" {
		t.Fatalf("got %q want 192.168.1.1", got)
	}
}

func TestRateLimit_Returns429WithRetryAfter(t *testing.T) {
	limiter := &fakeRateLimiter{result: ports.RateLimitResult{Limited: true, RetryAfter: 42}}
	policies := []RateLimitPolicy{
		{
			Name:          "test",
			Match:         func(m, p string) bool { return true },
			Key:           func(r *http.Request, _ ports.TokenService, _ config.Config) string { return "key" },
			PerPeriod:     1,
			PeriodSeconds: 60,
			MaxBurst:      0,
		},
	}

	h := RateLimit(limiter, policies, nil, cfg(), discardLogger())(okHandler())
	r := httptest.NewRequest(http.MethodGet, "/messages", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "42" {
		t.Fatalf("Retry-After = %q", got)
	}
	var body struct {
		Detail string `json:"detail"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Detail != "Too many requests." {
		t.Fatalf("detail = %q", body.Detail)
	}
}

func TestRateLimit_BurstTolerance(t *testing.T) {
	limiter := newTestLimiter(t)
	policies := DefaultPolicies(false)

	h := RateLimit(limiter, policies, nil, cfg(), discardLogger())(okHandler())

	for i := 0; i <= 2; i++ {
		r := httptest.NewRequest(http.MethodGet, "/messages/export", nil)
		r.RemoteAddr = "1.1.1.1:9999"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d should pass (burst=2, first 3 allowed), got %d", i, w.Code)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/messages/export", nil)
	r.RemoteAddr = "1.1.1.1:9999"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request should be limited, got %d", w.Code)
	}
}

func TestRateLimit_HealthUnlimited(t *testing.T) {
	limiter := newTestLimiter(t)
	policies := DefaultPolicies(false)
	h := RateLimit(limiter, policies, nil, cfg(), discardLogger())(okHandler())

	for i := 0; i < 50; i++ {
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.RemoteAddr = "2.2.2.2:1111"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("health request %d should never be limited", i)
		}
	}
}

func TestRateLimit_AdminUserIDKeying(t *testing.T) {
	limiter := newTestLimiter(t)

	user1TS := fakeTokenService{userID: 1, ok: true}
	user2TS := fakeTokenService{userID: 2, ok: true}

	policiesUser1 := DefaultPolicies(false)
	policiesUser2 := DefaultPolicies(false)

	cfg1 := cfg()
	h1 := RateLimit(limiter, policiesUser1, user1TS, cfg1, discardLogger())(okHandler())
	h2 := RateLimit(limiter, policiesUser2, user2TS, cfg1, discardLogger())(okHandler())

	reqWithCookie := func(handler http.Handler) int {
		r := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		r.RemoteAddr = "3.3.3.3:1111"
		r.AddCookie(&http.Cookie{Name: "kick_logs_session", Value: "token"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	for i := 0; i <= 30; i++ {
		reqWithCookie(h1)
	}

	if code := reqWithCookie(h2); code != http.StatusOK {
		t.Fatalf("user2 should be independent from user1, got %d", code)
	}
}

func TestRateLimit_AdminIPFallback(t *testing.T) {
	var capturedKey string
	limiter := &fakeRateLimiter{}
	policies := []RateLimitPolicy{
		{
			Name:  "admin-read",
			Match: func(m, p string) bool { return m == "GET" && p == "/admin/users" },
			Key: func(r *http.Request, ts ports.TokenService, c config.Config) string {
				if ts != nil {
					if cookie, err := r.Cookie(c.JWTCookieName); err == nil {
						if uid, ok := ts.GetUserID(cookie.Value); ok {
							return "admin:uid:" + strconv.FormatInt(uid, 10)
						}
					}
				}
				return "admin:ip:" + ClientIP(r, false)
			},
			PerPeriod:     120,
			PeriodSeconds: 60,
			MaxBurst:      30,
		},
	}

	ts := fakeTokenService{ok: false}
	h := RateLimit(limiter, policies, ts, cfg(), discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := range policies {
			if policies[i].Match(r.Method, r.URL.Path) {
				capturedKey = policies[i].Key(r, ts, cfg())
				break
			}
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.RemoteAddr = "4.4.4.4:9999"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if !startsWithPrefix(capturedKey, "admin:ip:") {
		t.Fatalf("key should use IP fallback, got %q", capturedKey)
	}
}

func TestRateLimit_CleanupConfirmTighterThanAdminWrite(t *testing.T) {
	limiter := newTestLimiter(t)
	policies := DefaultPolicies(false)

	ts := fakeTokenService{userID: 10, ok: true}
	h := RateLimit(limiter, policies, ts, cfg(), discardLogger())(okHandler())

	cleanupReq := func() int {
		r := httptest.NewRequest(http.MethodPost, "/admin/data-management/cleanup/confirm", nil)
		r.RemoteAddr = "5.5.5.5:1111"
		r.AddCookie(&http.Cookie{Name: "kick_logs_session", Value: "tok"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}

	adminWriteReq := func() int {
		r := httptest.NewRequest(http.MethodPost, "/admin/channels", nil)
		r.RemoteAddr = "5.5.5.5:1111"
		r.AddCookie(&http.Cookie{Name: "kick_logs_session", Value: "tok"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}

	cleanupReq()
	cleanupReq()
	if code := cleanupReq(); code != http.StatusTooManyRequests {
		t.Fatalf("cleanup/confirm should be limited after burst=1, got %d", code)
	}

	if code := adminWriteReq(); code != http.StatusOK {
		t.Fatalf("admin-write should still be allowed (independent policy), got %d", code)
	}
}

func TestRateLimit_FailOpen(t *testing.T) {
	limiter := &fakeRateLimiter{err: errTest("store unavailable")}
	policies := DefaultPolicies(false)

	h := RateLimit(limiter, policies, nil, cfg(), discardLogger())(okHandler())
	r := httptest.NewRequest(http.MethodGet, "/messages", nil)
	r.RemoteAddr = "6.6.6.6:1111"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("limiter error should fail open, got %d", w.Code)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func startsWithPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
