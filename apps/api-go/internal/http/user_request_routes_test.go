package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	authinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/auth"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
	requestsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/requests"
)

func TestUserRequestRoutesPublicSubmitAndAdminWorkflow(t *testing.T) {
	handler, repo := newUserRequestTestRouter(t)

	submitResponse := httptest.NewRecorder()
	submitRequest := jsonRequest(t, http.MethodPost, "/requests", `{
		"type":"channel_request",
		"title":"Kanal eklensin",
		"message":"Bu kanali uygulamaya ekleyebilir miyiz?",
		"channel_slug":"https://kick.com/NuriBen",
		"contact":"mod@example.com"
	}`)
	submitRequest.RemoteAddr = "10.20.30.40:1234"
	submitRequest.Header.Set("User-Agent", "route-test-agent")
	handler.ServeHTTP(submitResponse, submitRequest)
	if submitResponse.Code != http.StatusCreated {
		t.Fatalf("submit status = %d body = %s", submitResponse.Code, submitResponse.Body.String())
	}

	var createPayload struct {
		RequestID string `json:"request_id"`
	}
	decodeResponse(t, submitResponse, &createPayload)
	if createPayload.RequestID == "" {
		t.Fatal("request_id is empty")
	}
	stored := repo.requests[createPayload.RequestID]
	if stored.ChannelSlug != "nuriben" {
		t.Fatalf("stored channel slug = %q", stored.ChannelSlug)
	}
	if stored.IPHash == "" || stored.IPHash == "10.20.30.40" {
		t.Fatalf("IPHash was not hashed = %q", stored.IPHash)
	}

	adminCookie := loginAsAdmin(t, handler)

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/admin/requests?type=channel_request&archived=false", nil)
	listRequest.AddCookie(adminCookie)
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listResponse.Code, listResponse.Body.String())
	}
	if !strings.Contains(listResponse.Body.String(), `"current_status":"new"`) ||
		!strings.Contains(listResponse.Body.String(), `"channel_slug":"nuriben"`) {
		t.Fatalf("list body = %s", listResponse.Body.String())
	}

	statusResponse := httptest.NewRecorder()
	statusRequest := jsonRequest(
		t,
		http.MethodPost,
		"/admin/requests/"+createPayload.RequestID+"/status",
		`{"status":"reviewing"}`,
	)
	statusRequest.AddCookie(adminCookie)
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("status update = %d body = %s", statusResponse.Code, statusResponse.Body.String())
	}
	if !strings.Contains(statusResponse.Body.String(), `"current_status":"reviewing"`) {
		t.Fatalf("status body = %s", statusResponse.Body.String())
	}

	noteResponse := httptest.NewRecorder()
	noteRequest := jsonRequest(t, http.MethodPost, "/admin/requests/"+createPayload.RequestID+"/notes", `{"note":"Kontrol edilecek."}`)
	noteRequest.AddCookie(adminCookie)
	handler.ServeHTTP(noteResponse, noteRequest)
	if noteResponse.Code != http.StatusOK {
		t.Fatalf("note status = %d body = %s", noteResponse.Code, noteResponse.Body.String())
	}
	if !strings.Contains(noteResponse.Body.String(), "Kontrol edilecek.") {
		t.Fatalf("note body = %s", noteResponse.Body.String())
	}

	archiveResponse := httptest.NewRecorder()
	archiveRequest := httptest.NewRequest(http.MethodPost, "/admin/requests/"+createPayload.RequestID+"/archive", nil)
	archiveRequest.AddCookie(adminCookie)
	handler.ServeHTTP(archiveResponse, archiveRequest)
	if archiveResponse.Code != http.StatusOK {
		t.Fatalf("archive status = %d body = %s", archiveResponse.Code, archiveResponse.Body.String())
	}
	if !strings.Contains(archiveResponse.Body.String(), `"is_archived":true`) {
		t.Fatalf("archive body = %s", archiveResponse.Body.String())
	}
}

func TestUserRequestRoutesRejectHoneypotAndRequireAdmin(t *testing.T) {
	handler, _ := newUserRequestTestRouter(t)

	honeypotResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		honeypotResponse,
		jsonRequest(t, http.MethodPost, "/requests", `{"type":"feedback","title":"Feedback","message":"Guzel uygulama.","website":"bot"}`),
	)
	if honeypotResponse.Code != http.StatusBadRequest {
		t.Fatalf("honeypot status = %d body = %s", honeypotResponse.Code, honeypotResponse.Body.String())
	}

	adminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminResponse, httptest.NewRequest(http.MethodGet, "/admin/requests", nil))
	if adminResponse.Code != http.StatusUnauthorized {
		t.Fatalf("admin status = %d body = %s", adminResponse.Code, adminResponse.Body.String())
	}
}

func newUserRequestTestRouter(t *testing.T) (http.Handler, *routeMemoryRequestRepo) {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "kick-logs-requests.sqlite3")
	db, err := sqliteinfra.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		t.Fatalf("sqlite migrate: %v", err)
	}

	cfg := config.Config{
		BackendCORSOrigins:            []string{"http://localhost:3000"},
		JWTSecretKey:                  "test-secret-key",
		JWTAlgorithm:                  "HS256",
		JWTExpiresMinutes:             60,
		JWTCookieName:                 "kick_logs_session",
		JWTCookieSameSite:             "lax",
		RateLimitTrustProxy:           false,
		RateLimitClientIPHeader:       "CF-Connecting-IP",
		RateLimitStoreMaxKeys:         1000,
		MessageExportMaxRows:          1000,
		ListenerStaleAfter:            45,
		ClickHouseDatabase:            "kick_logs",
		DefaultAdminEmail:             "admin@kicklogs.local",
		DefaultAdminPassword:          "admin123",
		SeedSuperAdmin:                true,
		KickWebhookProcessBatchSize:   50,
		KickWebhookProcessMaxAttempts: 5,
	}

	adminRepo := sqliteinfra.NewAdminUserRepository(db)
	if _, err := sqliteinfra.SeedSuperAdmin(ctx, adminRepo, "admin@kicklogs.local", "admin123"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	tokenService := authinfra.NewJWTTokenService(cfg)
	authService := authusecase.NewService(adminRepo, authinfra.NewBcryptPasswordHasher(), tokenService)
	requestRepo := newRouteMemoryRequestRepo()

	handler := NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:       cfg,
		Auth:         authService,
		Requests:     requestsusecase.NewService(requestRepo),
		TokenService: tokenService,
	})
	return handler, requestRepo
}

type routeMemoryRequestRepo struct {
	requests map[string]domain.UserRequest
	events   map[string][]domain.UserRequestEvent
}

func newRouteMemoryRequestRepo() *routeMemoryRequestRepo {
	return &routeMemoryRequestRepo{
		requests: map[string]domain.UserRequest{},
		events:   map[string][]domain.UserRequestEvent{},
	}
}

func (repo *routeMemoryRequestRepo) Create(_ context.Context, request domain.UserRequest) error {
	repo.requests[request.ID] = request
	return nil
}

func (repo *routeMemoryRequestRepo) List(
	_ context.Context,
	filter domain.UserRequestListFilter,
) ([]domain.UserRequestState, error) {
	states := make([]domain.UserRequestState, 0, len(repo.requests))
	for requestID := range repo.requests {
		state := repo.state(requestID)
		if filter.Type != "" && state.Request.Type != filter.Type {
			continue
		}
		if filter.Status != "" && state.CurrentStatus != filter.Status {
			continue
		}
		if filter.Archived != nil && state.IsArchived != *filter.Archived {
			continue
		}
		states = append(states, state)
	}
	return states, nil
}

func (repo *routeMemoryRequestRepo) Get(_ context.Context, requestID string) (domain.UserRequestState, error) {
	if _, ok := repo.requests[requestID]; !ok {
		return domain.UserRequestState{}, sql.ErrNoRows
	}
	return repo.state(requestID), nil
}

func (repo *routeMemoryRequestRepo) ListEvents(_ context.Context, requestID string) ([]domain.UserRequestEvent, error) {
	return append([]domain.UserRequestEvent(nil), repo.events[requestID]...), nil
}

func (repo *routeMemoryRequestRepo) AppendEvent(_ context.Context, event domain.UserRequestEvent) error {
	if _, ok := repo.requests[event.RequestID]; !ok {
		return sql.ErrNoRows
	}
	repo.events[event.RequestID] = append(repo.events[event.RequestID], event)
	return nil
}

func (repo *routeMemoryRequestRepo) state(requestID string) domain.UserRequestState {
	request := repo.requests[requestID]
	state := domain.UserRequestState{
		Request:       request,
		CurrentStatus: domain.UserRequestStatusNew,
		LatestEventAt: request.CreatedAt,
	}
	for _, event := range repo.events[requestID] {
		if event.EventType == domain.UserRequestEventStatusChanged && event.Status != "" {
			state.CurrentStatus = event.Status
		}
		if event.EventType == domain.UserRequestEventArchived {
			state.IsArchived = true
		}
		if event.CreatedAt.After(state.LatestEventAt) {
			state.LatestEventAt = event.CreatedAt
		}
	}
	if state.LatestEventAt.IsZero() {
		state.LatestEventAt = time.Now().UTC()
	}
	return state
}

func decodeJSONBody(t *testing.T, body string, target any) {
	t.Helper()
	if err := json.NewDecoder(strings.NewReader(body)).Decode(target); err != nil {
		t.Fatalf("decode json body: %v body = %s", err, body)
	}
}
