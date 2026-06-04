package httpapi_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	httpapi "github.com/YSelim0/kick-logs/apps/api-go/internal/http"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/kick"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

func TestKickWebhookReceiver(t *testing.T) {
	publicKeyPEM, privateKey, err := newWebhookTestKey()
	if err != nil {
		t.Fatalf("newWebhookTestKey: %v", err)
	}
	verifier, err := kick.NewWebhookVerifier(publicKeyPEM)
	if err != nil {
		t.Fatalf("NewWebhookVerifier: %v", err)
	}

	store := &inMemoryWebhookStore{}

	router := newWebhookRouter(verifier, store)

	body := []byte(`{"event":"test"}`)
	messageID := "msg-test-001"
	timestamp := time.Now().UTC().Format(time.RFC3339)

	msg := buildMsg(messageID, timestamp, body)
	sig := signWebhookMessage(t, privateKey, msg)

	t.Run("valid event stored", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest(body, messageID, timestamp, "channel.subscription.new", "v1", sig)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
		}
		if len(store.events) != 1 {
			t.Fatalf("events stored = %d", len(store.events))
		}
		if store.events[0].MessageID != messageID {
			t.Fatalf("stored message_id = %q", store.events[0].MessageID)
		}
	})

	t.Run("duplicate message id is 204", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest(body, messageID, timestamp, "channel.subscription.new", "v1", sig)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("duplicate status = %d", w.Code)
		}
		if len(store.events) != 1 {
			t.Fatalf("events stored after duplicate = %d (want 1)", len(store.events))
		}
	})

	t.Run("invalid signature returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest(body, messageID, timestamp, "channel.subscription.new", "v1", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("0", 128))))
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("missing message-id header returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest(body, "", timestamp, "channel.subscription.new", "v1", sig)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("missing timestamp header returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest(body, messageID, "", "channel.subscription.new", "v1", sig)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("missing event-type header returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest(body, messageID, timestamp, "", "v1", sig)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("tampered body returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := webhookRequest([]byte(`{"event":"tampered"}`), messageID, timestamp, "channel.subscription.new", "v1", sig)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("no verifier configured returns 503", func(t *testing.T) {
		r := newWebhookRouter(nil, store)
		w := httptest.NewRecorder()
		req := webhookRequest(body, messageID, timestamp, "channel.subscription.new", "v1", sig)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestWebhookVerifierSignatureFormats(t *testing.T) {
	publicKeyPEM, privateKey, err := newWebhookTestKey()
	if err != nil {
		t.Fatalf("newWebhookTestKey: %v", err)
	}
	body := []byte(`{"test":1}`)
	messageID := "msg-fmt-test"
	timestamp := "2026-06-01T12:00:00Z"
	msg := buildMsg(messageID, timestamp, body)
	rawSig := signWebhookRaw(t, privateKey, msg)

	tests := []struct {
		name      string
		keyStr    string
		sigStr    string
		expectErr bool
	}{
		{
			name:   "pem key + base64 sig",
			keyStr: publicKeyPEM,
			sigStr: base64.StdEncoding.EncodeToString(rawSig),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := kick.NewWebhookVerifier(tc.keyStr)
			if err != nil {
				if !tc.expectErr {
					t.Fatalf("NewWebhookVerifier: %v", err)
				}
				return
			}
			err = v.Verify(messageID, timestamp, body, tc.sigStr)
			if tc.expectErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Verify: %v", err)
			}
		})
	}
}

func buildMsg(messageID, timestamp string, body []byte) []byte {
	msg := make([]byte, 0, len(messageID)+len(timestamp)+len(body)+2)
	msg = append(msg, []byte(messageID)...)
	msg = append(msg, '.')
	msg = append(msg, []byte(timestamp)...)
	msg = append(msg, '.')
	msg = append(msg, body...)
	return msg
}

func newWebhookTestKey() (string, *rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return string(pemBytes), privateKey, nil
}

func signWebhookMessage(t *testing.T, privateKey *rsa.PrivateKey, msg []byte) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(signWebhookRaw(t, privateKey, msg))
}

func signWebhookRaw(t *testing.T, privateKey *rsa.PrivateKey, msg []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(msg)
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign webhook message: %v", err)
	}
	return sig
}

func webhookRequest(body []byte, messageID, timestamp, eventType, version, sig string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kick", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if messageID != "" {
		req.Header.Set("Kick-Event-Message-Id", messageID)
	}
	if timestamp != "" {
		req.Header.Set("Kick-Event-Message-Timestamp", timestamp)
	}
	if eventType != "" {
		req.Header.Set("Kick-Event-Type", eventType)
	}
	if version != "" {
		req.Header.Set("Kick-Event-Version", version)
	}
	if sig != "" {
		req.Header.Set("Kick-Event-Signature", sig)
	}
	return req
}

func newWebhookRouter(verifier ports.KickWebhookVerifier, store ports.KickWebhookEventRepository) http.Handler {
	cfg := config.Config{}
	return httpapi.NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:          cfg,
		WebhookVerifier: verifier,
		WebhookEvents:   store,
	})
}

// inMemoryWebhookStore implements ports.KickWebhookEventRepository for tests.
type inMemoryWebhookStore struct {
	events []domain.KickWebhookEvent
}

func (s *inMemoryWebhookStore) InsertIdempotent(_ context.Context, event domain.KickWebhookEvent) error {
	for _, e := range s.events {
		if e.MessageID == event.MessageID {
			return nil
		}
	}
	s.events = append(s.events, event)
	return nil
}

func (s *inMemoryWebhookStore) GetByMessageID(_ context.Context, messageID string) (domain.KickWebhookEvent, error) {
	for _, e := range s.events {
		if e.MessageID == messageID {
			return e, nil
		}
	}
	return domain.KickWebhookEvent{}, fmt.Errorf("not found")
}

func (s *inMemoryWebhookStore) ListPending(_ context.Context, _ int, _ int) ([]domain.KickWebhookEvent, error) {
	return nil, nil
}

func (s *inMemoryWebhookStore) MarkProcessed(_ context.Context, _ string) error { return nil }
func (s *inMemoryWebhookStore) MarkFailed(_ context.Context, _ string, _ string, _ int) error {
	return nil
}
func (s *inMemoryWebhookStore) MarkIgnored(_ context.Context, _ string, _ string) error { return nil }
func (s *inMemoryWebhookStore) PruneTerminalBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (s *inMemoryWebhookStore) CountByStatus(_ context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *inMemoryWebhookStore) LatestReceivedAt(_ context.Context) (time.Time, error) {
	return time.Time{}, nil
}
