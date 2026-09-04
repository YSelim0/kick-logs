package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	authinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/auth"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messageimport"
)

const importExportPayload = `{
  "items": [
    {
      "kick_message_id": "import-new",
      "chatroom_id": 7552533,
      "content": "yeni mesaj",
      "message_type": "message",
      "message_created_at": "2026-08-31T04:38:07Z",
      "ingested_at": "2026-08-31T04:38:10Z",
      "sender": {"id": 74431315, "kick_user_id": 74431315, "username": "prenses_elif", "slug": "prenses-elif"},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi"}
    },
    {
      "kick_message_id": "import-existing",
      "content": "zaten var",
      "message_type": "message",
      "message_created_at": "2026-08-31T04:39:07Z",
      "sender": {"id": 1, "kick_user_id": 1, "username": "someone", "slug": "someone"},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi"}
    }
  ],
  "count": 2,
  "max_rows": 1000,
  "truncated": false
}`

func TestMessageImportPreviewAndConfirm(t *testing.T) {
	handler, repository := newMessageImportTestRouter(t)
	cookie := loginAsAdmin(t, handler)

	previewResponse := httptest.NewRecorder()
	handler.ServeHTTP(previewResponse, importRequest(t, "/admin/data-management/import/preview", cookie, "", ""))
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body = %s", previewResponse.Code, previewResponse.Body.String())
	}

	var preview struct {
		RecordsRead      int    `json:"records_read"`
		ToInsert         int    `json:"to_insert"`
		AlreadyExists    int    `json:"already_exists"`
		Invalid          int    `json:"invalid"`
		CanExecute       bool   `json:"can_execute"`
		ConfirmationText string `json:"confirmation_text"`
	}
	if err := json.Unmarshal(previewResponse.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if preview.RecordsRead != 2 || preview.ToInsert != 1 || preview.AlreadyExists != 1 || preview.Invalid != 0 {
		t.Fatalf("unexpected preview counts: %+v", preview)
	}
	if !preview.CanExecute || preview.ConfirmationText != messageimport.ConfirmationText {
		t.Fatalf("unexpected preview gate: %+v", preview)
	}
	if len(repository.messages) != 1 {
		t.Fatalf("preview must not write; store has %d messages", len(repository.messages))
	}

	confirmResponse := httptest.NewRecorder()
	handler.ServeHTTP(confirmResponse, importRequest(
		t,
		"/admin/data-management/import/confirm",
		cookie,
		messageimport.ConfirmationText,
		"",
	))
	if confirmResponse.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, body = %s", confirmResponse.Code, confirmResponse.Body.String())
	}

	var result struct {
		Written       int `json:"written"`
		AlreadyExists int `json:"already_exists"`
	}
	if err := json.Unmarshal(confirmResponse.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Written != 1 || result.AlreadyExists != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(repository.messages) != 2 {
		t.Fatalf("store should hold 2 messages, has %d", len(repository.messages))
	}
	if repository.messages[1].KickMessageID != "import-new" {
		t.Fatalf("unexpected inserted message: %+v", repository.messages[1])
	}
}

func TestMessageImportConfirmRejectsWrongConfirmationText(t *testing.T) {
	handler, repository := newMessageImportTestRouter(t)
	cookie := loginAsAdmin(t, handler)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, importRequest(
		t,
		"/admin/data-management/import/confirm",
		cookie,
		"nope",
		"",
	))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(repository.messages) != 1 {
		t.Fatalf("nothing should be written, store has %d messages", len(repository.messages))
	}
}

func TestMessageImportRequiresAuthentication(t *testing.T) {
	handler, _ := newMessageImportTestRouter(t)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, importRequest(t, "/admin/data-management/import/preview", nil, "", ""))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestMessageImportPreviewRespectsLimit(t *testing.T) {
	handler, _ := newMessageImportTestRouter(t)
	cookie := loginAsAdmin(t, handler)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, importRequest(t, "/admin/data-management/import/preview", cookie, "", "1"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var preview struct {
		RecordsRead int `json:"records_read"`
		TotalInFile int `json:"total_in_file"`
		ToInsert    int `json:"to_insert"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if preview.RecordsRead != 1 || preview.TotalInFile != 2 || preview.ToInsert != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func TestMessageImportRejectsMissingFile(t *testing.T) {
	handler, _ := newMessageImportTestRouter(t)
	cookie := loginAsAdmin(t, handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("confirmation_text", messageimport.ConfirmationText); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/admin/data-management/import/preview", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(cookie)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func importRequest(t *testing.T, path string, cookie *http.Cookie, confirmationText string, limit string) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "export.json")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte(importExportPayload)); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if confirmationText != "" {
		if err := writer.WriteField("confirmation_text", confirmationText); err != nil {
			t.Fatalf("write confirmation: %v", err)
		}
	}
	if limit != "" {
		if err := writer.WriteField("limit", limit); err != nil {
			t.Fatalf("write limit: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if cookie != nil {
		request.AddCookie(cookie)
	}
	return request
}

func newMessageImportTestRouter(t *testing.T) (http.Handler, *fakeMessageRepository) {
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
		BackendCORSOrigins:          []string{"http://localhost:3000"},
		JWTSecretKey:                "test-secret-key",
		JWTAlgorithm:                "HS256",
		JWTExpiresMinutes:           60,
		JWTCookieName:               "kick_logs_session",
		JWTCookieSameSite:           "lax",
		MessageImportMaxRows:        1000,
		MessageImportMaxUploadBytes: 1024 * 1024,
	}

	adminRepo := sqliteinfra.NewAdminUserRepository(db)
	if _, err := sqliteinfra.SeedSuperAdmin(ctx, adminRepo, "admin@kicklogs.local", "admin123"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	tokenService := authinfra.NewJWTTokenService(cfg)
	authService := authusecase.NewService(adminRepo, authinfra.NewBcryptPasswordHasher(), tokenService)

	repository := &fakeMessageRepository{messages: []domain.ChatMessage{{KickMessageID: "import-existing"}}}
	handler := NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:        cfg,
		Auth:          authService,
		MessageImport: messageimport.NewService(repository, cfg.MessageImportMaxRows),
		TokenService:  tokenService,
	})
	return handler, repository
}
