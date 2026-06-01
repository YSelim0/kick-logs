package routes

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

const (
	headerMessageID  = "Kick-Event-Message-Id"
	headerTimestamp  = "Kick-Event-Message-Timestamp"
	headerEventType  = "Kick-Event-Type"
	headerVersion    = "Kick-Event-Version"
	headerSignature  = "Kick-Event-Signature"
	maxWebhookBodyBytes = 1 << 20 // 1 MiB
)

func RegisterWebhookRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("POST /webhooks/kick", func(w http.ResponseWriter, r *http.Request) {
		handleKickWebhook(w, r, deps)
	})
}

func handleKickWebhook(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	skipVerification := deps.Config.KickWebhookSkipVerification
	if deps.WebhookVerifier == nil && !skipVerification {
		writeError(w, http.StatusServiceUnavailable, "Webhook signature verification not configured.")
		return
	}
	// Log all Kick headers to determine signing mechanism.
	if skipVerification {
		for name, values := range r.Header {
			if strings.HasPrefix(strings.ToLower(name), "kick-") {
				logWebhookHeader(name, values)
			}
		}
	}

	messageID := strings.TrimSpace(r.Header.Get(headerMessageID))
	timestamp := strings.TrimSpace(r.Header.Get(headerTimestamp))
	eventType := strings.TrimSpace(r.Header.Get(headerEventType))
	eventVersion := strings.TrimSpace(r.Header.Get(headerVersion))
	signature := strings.TrimSpace(r.Header.Get(headerSignature))

	if messageID == "" || timestamp == "" || eventType == "" || eventVersion == "" {
		writeError(w, http.StatusBadRequest, "Missing required Kick webhook headers.")
		return
	}
	if !skipVerification && signature == "" {
		writeError(w, http.StatusBadRequest, "Missing required Kick webhook headers.")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read request body.")
		return
	}

	if !skipVerification {
		if err := deps.WebhookVerifier.Verify(messageID, timestamp, body, signature); err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid webhook signature.")
			return
		}
	}

	if deps.WebhookEvents == nil {
		writeError(w, http.StatusServiceUnavailable, "Webhook event storage not available.")
		return
	}

	event := domain.KickWebhookEvent{
		MessageID:      messageID,
		SubscriptionID: strings.TrimSpace(r.Header.Get("Kick-Event-Subscription-Id")),
		EventType:      eventType,
		EventVersion:   eventVersion,
		RawPayloadJSON: string(body),
		ReceivedAt:     time.Now().UTC(),
	}

	if err := deps.WebhookEvents.InsertIdempotent(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to store webhook event.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func logWebhookHeader(name string, values []string) {
	slog.Info("kick webhook header", "name", name, "value", strings.Join(values, ", "))
	fmt.Fprintf(os.Stderr, "[webhook-debug] header %s = %s\n", name, strings.Join(values, ", "))
}

