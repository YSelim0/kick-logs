package routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/middleware"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
)

func RegisterAuthRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("POST /auth/login", func(response http.ResponseWriter, request *http.Request) {
		login(response, request, deps)
	})
	mux.HandleFunc("POST /auth/logout", func(response http.ResponseWriter, request *http.Request) {
		logout(response, deps.Config)
	})
	mux.HandleFunc("GET /auth/me", func(response http.ResponseWriter, request *http.Request) {
		me(response, request, deps)
	})
}

func login(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	var payload schemas.LoginRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}

	if deps.RateLimiter != nil && payload.Email != "" {
		clientIP := middleware.ClientIP(request, deps.Config.RateLimitTrustProxy, deps.Config.RateLimitClientIPHeader)
		key := "login:email:" + clientIP + ":" + strings.ToLower(strings.TrimSpace(payload.Email))
		result, _ := deps.RateLimiter.RateLimit(key, 8, 600, 3)
		if result.Limited {
			response.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
			writeError(response, http.StatusTooManyRequests, "Too many requests.")
			return
		}
	}

	user, token, err := deps.Auth.Login(request.Context(), payload.Email, payload.Password)
	if err != nil {
		if errors.Is(err, authusecase.ErrInvalidCredentials) {
			writeError(response, http.StatusUnauthorized, "Invalid credentials.")
			return
		}
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	http.SetCookie(response, sessionCookie(deps.Config, token))
	writeJSON(response, http.StatusOK, schemas.AuthResponse{User: adminUserResponse(user)})
}

func logout(response http.ResponseWriter, cfg config.Config) {
	cookie := sessionCookie(cfg, "")
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0).UTC()
	http.SetCookie(response, cookie)
	writeJSON(response, http.StatusOK, statusResponse{Status: "ok"})
}

func me(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	user, ok := requireAdmin(response, request, deps.Auth, deps.Config)
	if !ok {
		return
	}
	writeJSON(response, http.StatusOK, adminUserResponse(user))
}

func sessionCookie(cfg config.Config, token string) *http.Cookie {
	return &http.Cookie{
		Name:     cfg.JWTCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cfg.JWTExpiresMinutes * 60,
		HttpOnly: true,
		Secure:   cfg.JWTCookieSecure,
		SameSite: sameSiteMode(cfg.JWTCookieSameSite),
	}
}

func sameSiteMode(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
