package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	authinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/auth"
	datamanagementinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/data_management"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	operationsinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/operations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
	datamanagementusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/data_management"
)

func TestAuthAndAdminRoutes(t *testing.T) {
	handler := newAdminTestRouter(t)

	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		loginResponse,
		jsonRequest(t, http.MethodPost, "/auth/login", `{"email":"admin@kicklogs.local","password":"admin123"}`),
	)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", loginResponse.Code, loginResponse.Body.String())
	}
	if !strings.Contains(loginResponse.Body.String(), `"role":"super_admin"`) {
		t.Fatalf("login body = %s", loginResponse.Body.String())
	}
	sessionCookie := findCookie(t, loginResponse.Result().Cookies(), "kick_logs_session")
	if !sessionCookie.HttpOnly || sessionCookie.Path != "/" || sessionCookie.MaxAge != 60*60 {
		t.Fatalf("session cookie = %#v", sessionCookie)
	}

	meResponse := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("me status = %d body = %s", meResponse.Code, meResponse.Body.String())
	}

	createResponse := httptest.NewRecorder()
	createRequest := jsonRequest(t, http.MethodPost, "/admin/users", `{"email":"mod@kicklogs.local","password":"strongpass123"}`)
	createRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create user status = %d body = %s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	listRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list users status = %d body = %s", listResponse.Code, listResponse.Body.String())
	}
	if !strings.Contains(listResponse.Body.String(), "mod@kicklogs.local") {
		t.Fatalf("list users body = %s", listResponse.Body.String())
	}

	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if logoutResponse.Code != http.StatusOK {
		t.Fatalf("logout status = %d", logoutResponse.Code)
	}
	logoutCookie := findCookie(t, logoutResponse.Result().Cookies(), "kick_logs_session")
	if logoutCookie.MaxAge != -1 {
		t.Fatalf("logout cookie = %#v", logoutCookie)
	}
}

func TestAuthRoutesRejectInvalidCredentials(t *testing.T) {
	handler := newAdminTestRouter(t)

	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		jsonRequest(t, http.MethodPost, "/auth/login", `{"email":"admin@kicklogs.local","password":"wrong"}`),
	)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Invalid credentials.") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestAdminRoutesRejectMissingSession(t *testing.T) {
	handler := newAdminTestRouter(t)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Authentication required.") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestAdminUserCreationRequiresSuperAdmin(t *testing.T) {
	handler := newAdminTestRouter(t)
	superAdminCookie := loginAsAdmin(t, handler)

	createNormalResponse := httptest.NewRecorder()
	createNormalRequest := jsonRequest(t, http.MethodPost, "/admin/users", `{"email":"normal@kicklogs.local","password":"strongpass123"}`)
	createNormalRequest.AddCookie(superAdminCookie)
	handler.ServeHTTP(createNormalResponse, createNormalRequest)
	if createNormalResponse.Code != http.StatusCreated {
		t.Fatalf("create normal admin status = %d body = %s", createNormalResponse.Code, createNormalResponse.Body.String())
	}

	loginNormalResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		loginNormalResponse,
		jsonRequest(t, http.MethodPost, "/auth/login", `{"email":"normal@kicklogs.local","password":"strongpass123"}`),
	)
	normalCookie := findCookie(t, loginNormalResponse.Result().Cookies(), "kick_logs_session")

	blockedResponse := httptest.NewRecorder()
	blockedRequest := jsonRequest(t, http.MethodPost, "/admin/users", `{"email":"blocked@kicklogs.local","password":"strongpass123"}`)
	blockedRequest.AddCookie(normalCookie)
	handler.ServeHTTP(blockedResponse, blockedRequest)
	if blockedResponse.Code != http.StatusForbidden {
		t.Fatalf("blocked create status = %d body = %s", blockedResponse.Code, blockedResponse.Body.String())
	}
	if !strings.Contains(blockedResponse.Body.String(), "Super admin role required.") {
		t.Fatalf("blocked body = %s", blockedResponse.Body.String())
	}
}

func TestChannelRoutesResolveAndDisable(t *testing.T) {
	handler := newAdminTestRouter(t)
	sessionCookie := loginAsAdmin(t, handler)

	addResponse := httptest.NewRecorder()
	addRequest := jsonRequest(t, http.MethodPost, "/admin/channels", `{"slug":"Hype"}`)
	addRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(addResponse, addRequest)
	if addResponse.Code != http.StatusCreated {
		t.Fatalf("add channel status = %d body = %s", addResponse.Code, addResponse.Body.String())
	}

	var channel struct {
		ID             int64  `json:"id"`
		Slug           string `json:"slug"`
		KickChatroomID int64  `json:"kick_chatroom_id"`
		IsEnabled      bool   `json:"is_enabled"`
	}
	if err := json.NewDecoder(addResponse.Body).Decode(&channel); err != nil {
		t.Fatalf("decode channel: %v", err)
	}
	if channel.Slug != "hype" || channel.KickChatroomID != 2001 || !channel.IsEnabled {
		t.Fatalf("channel = %#v", channel)
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/admin/channels/1", nil)
	deleteRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete channel status = %d body = %s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if !strings.Contains(deleteResponse.Body.String(), `"is_enabled":false`) {
		t.Fatalf("delete channel body = %s", deleteResponse.Body.String())
	}
}

func TestOperationsSummaryRoute(t *testing.T) {
	handler := newAdminTestRouter(t)
	sessionCookie := loginAsAdmin(t, handler)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/operations/summary", nil)
	request.AddCookie(sessionCookie)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"raw_event_status_counts":{}`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestDataManagementRoutes(t *testing.T) {
	handler := newAdminTestRouter(t)
	sessionCookie := loginAsAdmin(t, handler)

	summaryResponse := httptest.NewRecorder()
	summaryRequest := httptest.NewRequest(http.MethodGet, "/admin/data-management/summary", nil)
	summaryRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(summaryResponse, summaryRequest)
	if summaryResponse.Code != http.StatusOK {
		t.Fatalf("summary status = %d body = %s", summaryResponse.Code, summaryResponse.Body.String())
	}
	if !strings.Contains(summaryResponse.Body.String(), `"retention_settings"`) {
		t.Fatalf("summary body = %s", summaryResponse.Body.String())
	}

	updateResponse := httptest.NewRecorder()
	updateRequest := jsonRequest(t, http.MethodPut, "/admin/data-management/retention-settings", `{"message_retention_days":30,"raw_event_retention_days":90}`)
	updateRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update status = %d body = %s", updateResponse.Code, updateResponse.Body.String())
	}
	if !strings.Contains(updateResponse.Body.String(), `"message_retention_days":30`) {
		t.Fatalf("update body = %s", updateResponse.Body.String())
	}

	previewResponse := httptest.NewRecorder()
	previewRequest := jsonRequest(t, http.MethodPost, "/admin/data-management/cleanup/preview", `{"target":"old_messages"}`)
	previewRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview status = %d body = %s", previewResponse.Code, previewResponse.Body.String())
	}
	if !strings.Contains(previewResponse.Body.String(), `"confirmation_text":"DELETE OLD MESSAGES"`) {
		t.Fatalf("preview body = %s", previewResponse.Body.String())
	}

	confirmResponse := httptest.NewRecorder()
	confirmRequest := jsonRequest(t, http.MethodPost, "/admin/data-management/cleanup/confirm", `{"target":"old_messages","confirmation_text":"DELETE OLD MESSAGES"}`)
	confirmRequest.AddCookie(sessionCookie)
	handler.ServeHTTP(confirmResponse, confirmRequest)
	if confirmResponse.Code != http.StatusOK {
		t.Fatalf("confirm status = %d body = %s", confirmResponse.Code, confirmResponse.Body.String())
	}
	if !strings.Contains(confirmResponse.Body.String(), `"deleted":{"messages":0,"raw_events":0,"total":0}`) {
		t.Fatalf("confirm body = %s", confirmResponse.Body.String())
	}
}

func newAdminTestRouter(t *testing.T) http.Handler {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "kick-logs.sqlite3")
	db, err := sqliteinfra.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		t.Fatalf("sqlite migrate: %v", err)
	}

	cfg := config.Config{
		BackendCORSOrigins: []string{"http://localhost:3000"},
		JWTSecretKey:       "test-secret-key",
		JWTAlgorithm:       "HS256",
		JWTExpiresMinutes:  60,
		JWTCookieName:      "kick_logs_session",
		JWTCookieSameSite:  "lax",
		ListenerStaleAfter: 45,
	}

	adminRepo := sqliteinfra.NewAdminUserRepository(db)
	if _, err := sqliteinfra.SeedSuperAdmin(ctx, adminRepo, "admin@kicklogs.local", "admin123"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	channelRepo := sqliteinfra.NewFollowedChannelRepository(db)
	authService := authusecase.NewService(
		adminRepo,
		authinfra.NewBcryptPasswordHasher(),
		authinfra.NewJWTTokenService(cfg),
	)
	channelService := channelsusecase.NewService(channelRepo, fakeChannelResolver{})
	operationsRepo := operationsinfra.NewRepository(db, dbPath, nil, cfg.ListenerStaleAfter)
	dataManagementService := datamanagementusecase.NewService(datamanagementinfra.NewRepository(db, dbPath, nil))

	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:     cfg,
		Auth:       authService,
		Channels:   channelService,
		Data:       dataManagementService,
		Operations: operationsRepo,
	})
}

func loginAsAdmin(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()

	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		jsonRequest(t, http.MethodPost, "/auth/login", `{"email":"admin@kicklogs.local","password":"admin123"}`),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", response.Code, response.Body.String())
	}
	return findCookie(t, response.Result().Cookies(), "kick_logs_session")
}

func jsonRequest(t *testing.T, method string, path string, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found in %#v", name, cookies)
	return nil
}

type fakeChannelResolver struct{}

func (fakeChannelResolver) ResolveChannel(_ context.Context, slug string) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{
		KickChannelID:   1001,
		KickChatroomID:  2001,
		Slug:            strings.ToLower(slug),
		DisplayName:     "Hype",
		ProfileImageURL: "https://files.kick.com/images/channel/hype-profile.png",
		BannerImageURL:  "https://files.kick.com/images/channel/hype-banner.png",
		RawPayloadJSON:  `{"slug":"hype"}`,
	}, nil
}
