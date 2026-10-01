package routes

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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
	mux.HandleFunc("GET /channels/{slug}/subscribers/export", func(w http.ResponseWriter, r *http.Request) {
		exportChannelSubscribers(w, r, deps)
	})
	mux.HandleFunc("GET /channels/{slug}/subscribers", func(w http.ResponseWriter, r *http.Request) {
		channelSubscribers(w, r, deps)
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

func channelSubscribers(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	ch, ok := resolvePublicChannel(w, r, deps)
	if !ok {
		return
	}

	giftOnly, err := optionalBool(r.URL.Query().Get("gift_only"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
		return
	}
	limit, offset, err := parseSubscriberPagination(r)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
		return
	}

	if deps.SubscriptionPeriods == nil {
		writeJSON(w, http.StatusOK, schemas.ChannelSubscribersResponse{
			ChannelSlug: ch.Slug,
			GiftOnly:    giftOnly,
			Limit:       int64(limit),
			Offset:      int64(offset),
			Items:       []schemas.ChannelSubscriberResponse{},
		})
		return
	}

	page, err := deps.SubscriptionPeriods.ListActiveSubscribers(r.Context(), domain.ChannelSubscriberFilter{
		FollowedChannelID: ch.ID,
		GiftOnly:          giftOnly,
		Limit:             uint64(limit),
		Offset:            uint64(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	writeJSON(w, http.StatusOK, channelSubscribersResponse(ch.Slug, giftOnly, page))
}

func exportChannelSubscribers(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	ch, ok := resolvePublicChannel(w, r, deps)
	if !ok {
		return
	}

	giftOnly, err := optionalBool(r.URL.Query().Get("gift_only"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "txt"
	}
	if format != "json" && format != "csv" && format != "txt" {
		writeError(w, http.StatusUnprocessableEntity, "Invalid export format.")
		return
	}

	items := []domain.ChannelSubscriber{}
	if deps.SubscriptionPeriods != nil {
		exported, err := deps.SubscriptionPeriods.ExportActiveSubscribers(r.Context(), ch.ID, giftOnly)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Internal server error.")
			return
		}
		items = exported
	}

	generatedAt := time.Now().UTC()
	filename := subscriberExportFilename(ch.Slug, format)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(channelSubscribersCSV(ch.Slug, items)))
	case "json":
		payload := schemas.ChannelSubscribersExportResponse{
			ChannelSlug: ch.Slug,
			GiftOnly:    giftOnly,
			GeneratedAt: generatedAt.Format(time.RFC3339),
			Count:       int64(len(items)),
			Items:       channelSubscriberResponses(items),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(channelSubscribersTXT(ch.Slug, giftOnly, generatedAt, items)))
	}
}

func resolvePublicChannel(w http.ResponseWriter, r *http.Request, deps Dependencies) (domain.FollowedChannel, bool) {
	slug := r.PathValue("slug")
	if slug == "" || deps.Channels == nil {
		writeError(w, http.StatusNotFound, "Channel not found.")
		return domain.FollowedChannel{}, false
	}

	ch, err := deps.Channels.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, channelsusecase.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "Channel not found.")
			return domain.FollowedChannel{}, false
		}
		writeError(w, http.StatusInternalServerError, "Internal server error.")
		return domain.FollowedChannel{}, false
	}
	return ch, true
}

func parseSubscriberPagination(r *http.Request) (int, int, error) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, errors.New("invalid limit")
		}
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return 0, 0, errors.New("invalid offset")
		}
		offset = parsed
	}
	return limit, offset, nil
}

func channelSubscribersResponse(
	channelSlug string,
	giftOnly bool,
	page domain.ChannelSubscriberPage,
) schemas.ChannelSubscribersResponse {
	return schemas.ChannelSubscribersResponse{
		ChannelSlug: channelSlug,
		GiftOnly:    giftOnly,
		Count:       page.Count,
		Limit:       int64(page.Limit),
		Offset:      int64(page.Offset),
		Items:       channelSubscriberResponses(page.Items),
	}
}

func channelSubscriberResponses(items []domain.ChannelSubscriber) []schemas.ChannelSubscriberResponse {
	resp := make([]schemas.ChannelSubscriberResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, channelSubscriberResponse(item))
	}
	return resp
}

func channelSubscriberResponse(item domain.ChannelSubscriber) schemas.ChannelSubscriberResponse {
	return schemas.ChannelSubscriberResponse{
		SubscriberKickUserID:  item.SubscriberKickUserID,
		Username:              item.Username,
		Slug:                  item.Slug,
		ProfileImageURL:       item.ProfileImageURL,
		IsGift:                item.IsGift,
		GifterKickUserID:      nullableInt64(item.GifterKickUserID),
		GifterUsername:        nullableString(item.GifterUsername),
		GifterSlug:            nullableString(item.GifterSlug),
		GifterProfileImageURL: nullableString(item.GifterProfileImageURL),
		StartedAt:             item.StartedAt.UTC().Format(time.RFC3339),
		ExpiresAt:             item.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

func channelSubscribersCSV(channelSlug string, items []domain.ChannelSubscriber) string {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write([]string{
		"channel_slug",
		"subscriber_kick_user_id",
		"username",
		"slug",
		"profile_image_url",
		"is_gift",
		"gifter_kick_user_id",
		"gifter_username",
		"gifter_slug",
		"started_at",
		"expires_at",
	})
	for _, item := range items {
		gifterID := ""
		if item.GifterKickUserID > 0 {
			gifterID = strconv.FormatInt(item.GifterKickUserID, 10)
		}
		_ = writer.Write([]string{
			safeCSVValue(channelSlug),
			strconv.FormatInt(item.SubscriberKickUserID, 10),
			safeCSVValue(item.Username),
			safeCSVValue(item.Slug),
			safeCSVValue(item.ProfileImageURL),
			strconv.FormatBool(item.IsGift),
			gifterID,
			safeCSVValue(item.GifterUsername),
			safeCSVValue(item.GifterSlug),
			item.StartedAt.UTC().Format(time.RFC3339),
			item.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	writer.Flush()
	return builder.String()
}

func channelSubscribersTXT(
	channelSlug string,
	giftOnly bool,
	generatedAt time.Time,
	items []domain.ChannelSubscriber,
) string {
	var builder strings.Builder
	builder.WriteString("Kick Logs Aktif Abone Listesi\n")
	builder.WriteString("Kanal: #")
	builder.WriteString(channelSlug)
	builder.WriteString("\n")
	builder.WriteString("Filtre: ")
	if giftOnly {
		builder.WriteString("Hediye aboneler")
	} else {
		builder.WriteString("Tüm aktif aboneler")
	}
	builder.WriteString("\n")
	builder.WriteString("Olusturulma: ")
	builder.WriteString(generatedAt.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	builder.WriteString("Toplam: ")
	builder.WriteString(strconv.Itoa(len(items)))
	builder.WriteString("\n\n")

	if len(items) == 0 {
		builder.WriteString("Bu kanal için henüz aktif abonelik kaydı yok.\n")
		return builder.String()
	}

	for i, item := range items {
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString(". ")
		builder.WriteString(item.Username)
		builder.WriteString(" (ID: ")
		builder.WriteString(strconv.FormatInt(item.SubscriberKickUserID, 10))
		builder.WriteString(")")
		if item.Slug != "" {
			builder.WriteString(" - kick.com/")
			builder.WriteString(item.Slug)
		}
		builder.WriteString("\n")
		if item.IsGift && item.GifterUsername != "" {
			builder.WriteString("   Hediye eden: ")
			builder.WriteString(item.GifterUsername)
			builder.WriteString("\n")
		}
		builder.WriteString("   Baslangic: ")
		builder.WriteString(item.StartedAt.UTC().Format(time.RFC3339))
		builder.WriteString("\n")
		builder.WriteString("   Bitis: ")
		builder.WriteString(item.ExpiresAt.UTC().Format(time.RFC3339))
		builder.WriteString("\n\n")
	}
	return builder.String()
}

func subscriberExportFilename(channelSlug string, format string) string {
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, channelSlug)
	if cleaned == "" {
		cleaned = "channel"
	}
	return "kick-logs-" + cleaned + "-active-subscribers." + format
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
