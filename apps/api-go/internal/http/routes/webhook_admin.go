package routes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
)

func RegisterWebhookAdminRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/webhooks/health", func(w http.ResponseWriter, r *http.Request) {
		webhookHealth(w, r, deps)
	})
	mux.HandleFunc("POST /admin/webhooks/sync", func(w http.ResponseWriter, r *http.Request) {
		webhookSync(w, r, deps)
	})
	mux.HandleFunc("GET /channels/{slug}/subscription-summary", func(w http.ResponseWriter, r *http.Request) {
		channelSubscriptionSummary(w, r, deps)
	})
}

func channelSubscriptionSummary(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusNotFound, "Channel not found.")
		return
	}

	if deps.Channels == nil {
		writeJSON(w, http.StatusOK, schemas.ChannelSubscriptionSummaryResponse{ChannelSlug: slug})
		return
	}

	ch, err := deps.Channels.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, channelsusecase.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "Channel not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	resp := schemas.ChannelSubscriptionSummaryResponse{
		ChannelSlug: ch.Slug,
	}

	if deps.SubscriptionPeriods != nil {
		summary, err := deps.SubscriptionPeriods.ActiveSummary(r.Context(), ch.ID)
		if err == nil {
			resp.ActiveCount = summary.ActiveCount
			resp.ActiveGiftedCount = summary.ActiveGiftedCount
			if !summary.LatestEventAt.IsZero() {
				s := summary.LatestEventAt.UTC().Format(time.RFC3339)
				resp.LatestEventAt = &s
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func webhookHealth(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(w, r, deps.Auth, deps.Config); !ok {
		return
	}

	resp := schemas.WebhookHealthResponse{
		ConfiguredEventTypes:     deps.Config.KickWebhookEvents,
		MissingClientCredentials: deps.Config.KickClientID == "" || deps.Config.KickClientSecret == "",
		MissingWebhookPublicKey:  deps.WebhookVerifier == nil && !deps.Config.KickWebhookSkipVerification,
		SyncEnabled:              deps.Config.KickWebhookSyncEnabled,
		InboxCounts:              map[string]int64{},
		Channels:                 []schemas.ChannelSyncStatus{},
	}

	if deps.WebhookEvents != nil {
		if counts, err := deps.WebhookEvents.CountByStatus(r.Context()); err == nil {
			resp.InboxCounts = counts
		}
		if latest, err := deps.WebhookEvents.LatestReceivedAt(r.Context()); err == nil && !latest.IsZero() {
			s := latest.UTC().Format(time.RFC3339)
			resp.LatestWebhookReceivedAt = &s
		}
	}

	if deps.WebhookEventSubs != nil && deps.Channels != nil {
		resp.Channels = buildChannelSyncStatuses(r.Context(), deps)
	}

	writeJSON(w, http.StatusOK, resp)
}

func buildChannelSyncStatuses(ctx context.Context, deps Dependencies) []schemas.ChannelSyncStatus {
	channels, err := deps.Channels.List(ctx)
	if err != nil {
		return nil
	}

	allSubs, err := deps.WebhookEventSubs.List(ctx)
	if err != nil {
		allSubs = nil
	}

	subsByChannel := make(map[int64][]domain.KickEventSubscription)
	for _, sub := range allSubs {
		subsByChannel[sub.FollowedChannelID] = append(subsByChannel[sub.FollowedChannelID], sub)
	}

	statuses := make([]schemas.ChannelSyncStatus, 0, len(channels))
	for _, ch := range channels {
		cs := schemas.ChannelSyncStatus{
			FollowedChannelID: ch.ID,
			Slug:              ch.Slug,
			BroadcasterUserID: ch.BroadcasterUserID,
			Subscriptions:     []schemas.EventSubStatus{},
		}
		for _, sub := range subsByChannel[ch.ID] {
			es := schemas.EventSubStatus{
				EventType:          sub.EventType,
				KickSubscriptionID: sub.KickSubscriptionID,
				Status:             sub.Status,
				LatestSyncError:    sub.LatestSyncError,
			}
			if !sub.SyncedAt.IsZero() {
				s := sub.SyncedAt.UTC().Format(time.RFC3339)
				es.SyncedAt = &s
			}
			cs.Subscriptions = append(cs.Subscriptions, es)
		}
		statuses = append(statuses, cs)
	}
	return statuses
}

func webhookSync(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(w, r, deps.Auth, deps.Config); !ok {
		return
	}

	if deps.KickSync == nil {
		writeError(w, http.StatusServiceUnavailable, "Kick subscription sync not configured.")
		return
	}

	deps.KickSync.SyncAll(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "sync triggered"})
}
