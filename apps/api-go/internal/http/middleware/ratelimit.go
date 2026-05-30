package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type RateLimitPolicy struct {
	Name          string
	Match         func(method, path string) bool
	Key           func(r *http.Request, ts ports.TokenService, cfg config.Config) string
	PerPeriod     int
	PeriodSeconds int
	MaxBurst      int
}

func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func DefaultPolicies(trustProxy bool) []RateLimitPolicy {
	ip := func(r *http.Request, _ ports.TokenService, _ config.Config) string {
		return "ip:" + ClientIP(r, trustProxy)
	}

	adminKey := func(r *http.Request, ts ports.TokenService, cfg config.Config) string {
		if ts != nil {
			if cookie, err := r.Cookie(cfg.JWTCookieName); err == nil {
				if uid, ok := ts.GetUserID(cookie.Value); ok {
					return "admin:uid:" + strconv.FormatInt(uid, 10)
				}
			}
		}
		return "admin:ip:" + ClientIP(r, trustProxy)
	}

	exactMatch := func(method, path string) func(string, string) bool {
		return func(m, p string) bool { return m == method && p == path }
	}

	prefixMatch := func(method, prefix string) func(string, string) bool {
		return func(m, p string) bool { return m == method && strings.HasPrefix(p, prefix) }
	}

	return []RateLimitPolicy{
		{
			Name:          "cleanup-confirm",
			Match:         exactMatch("POST", "/admin/data-management/cleanup/confirm"),
			Key:           adminKey,
			PerPeriod:     3,
			PeriodSeconds: 60,
			MaxBurst:      1,
		},
		{
			Name:          "login-ip",
			Match:         exactMatch("POST", "/auth/login"),
			Key:           ip,
			PerPeriod:     20,
			PeriodSeconds: 600,
			MaxBurst:      5,
		},
		{
			Name:          "messages-export",
			Match:         exactMatch("GET", "/messages/export"),
			Key:           ip,
			PerPeriod:     3,
			PeriodSeconds: 60,
			MaxBurst:      2,
		},
		{
			Name:          "messages",
			Match:         exactMatch("GET", "/messages"),
			Key:           ip,
			PerPeriod:     20,
			PeriodSeconds: 60,
			MaxBurst:      10,
		},
		{
			Name:          "analytics",
			Match:         prefixMatch("GET", "/analytics/"),
			Key:           ip,
			PerPeriod:     60,
			PeriodSeconds: 60,
			MaxBurst:      15,
		},
		{
			Name: "profile-analytics",
			Match: func(method, path string) bool {
				if method != "GET" {
					return false
				}
				return strings.HasSuffix(path, "/analytics") &&
					(strings.HasPrefix(path, "/users/") || strings.HasPrefix(path, "/channels/"))
			},
			Key:           ip,
			PerPeriod:     60,
			PeriodSeconds: 60,
			MaxBurst:      15,
		},
		{
			Name: "admin-write",
			Match: func(method, path string) bool {
				return (method == "POST" || method == "PUT" || method == "DELETE") &&
					strings.HasPrefix(path, "/admin/")
			},
			Key:           adminKey,
			PerPeriod:     30,
			PeriodSeconds: 60,
			MaxBurst:      10,
		},
		{
			Name:          "admin-read",
			Match:         prefixMatch("GET", "/admin/"),
			Key:           adminKey,
			PerPeriod:     120,
			PeriodSeconds: 60,
			MaxBurst:      30,
		},
	}
}

func RateLimit(limiter ports.RateLimiter, policies []RateLimitPolicy, ts ports.TokenService, cfg config.Config, logger *slog.Logger) func(http.Handler) http.Handler {
	type errBody struct {
		Detail string `json:"detail"`
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i := range policies {
				p := &policies[i]
				if !p.Match(r.Method, r.URL.Path) {
					continue
				}

				key := p.Key(r, ts, cfg)
				result, err := limiter.RateLimit(key, p.PerPeriod, p.PeriodSeconds, p.MaxBurst)
				if err != nil {
					logger.Warn("rate limit check failed", "policy", p.Name, "error", err)
					break
				}

				if result.Limited {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
					w.WriteHeader(http.StatusTooManyRequests)
					_ = json.NewEncoder(w).Encode(errBody{Detail: "Too many requests."})
					return
				}

				break
			}

			next.ServeHTTP(w, r)
		})
	}
}
