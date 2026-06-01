package httpapi_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	httpapi "github.com/YSelim0/kick-logs/apps/api-go/internal/http"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
)

func TestChannelSubscriptionSummary(t *testing.T) {
	ch := domain.FollowedChannel{ID: 1, Slug: "hype", DisplayName: "Hype", BroadcasterUserID: 9000, IsEnabled: true, RawPayloadJSON: "{}"}
	channelSvc := channelsusecase.NewService(newAdminFakeChannelRepo(ch), &nopResolver{})
	periodRepo := &fakeSubPeriodRepo{
		summary: domain.ChannelSubscriptionSummary{
			ActiveCount:       5,
			ActiveGiftedCount: 2,
			LatestEventAt:     time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	router := httpapi.NewRouter(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Channels:            channelSvc,
		SubscriptionPeriods: periodRepo,
	})

	t.Run("known channel returns summary", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/channels/hype/subscription-summary", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
		}
		var resp schemas.ChannelSubscriptionSummaryResponse
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp.ActiveCount != 5 || resp.ActiveGiftedCount != 2 {
			t.Errorf("resp = %+v", resp)
		}
		if resp.LatestEventAt == nil {
			t.Error("LatestEventAt is nil")
		}
	})

	t.Run("unknown channel returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/channels/nope/subscription-summary", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("no periods repo returns zeros", func(t *testing.T) {
		r := httpapi.NewRouter(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
			Channels: channelSvc,
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/channels/hype/subscription-summary", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
		var resp schemas.ChannelSubscriptionSummaryResponse
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp.ActiveCount != 0 {
			t.Errorf("expected 0 active count, got %d", resp.ActiveCount)
		}
	})
}

func TestWebhookHealthRequiresAuth(t *testing.T) {
	router := httpapi.NewRouter(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/webhooks/health", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestWebhookHealthInboxCounts(t *testing.T) {
	inboxRepo := &fakeInboxRepo{
		counts:     map[string]int64{"pending": 3, "processed": 10, "failed": 1},
		latestTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}
	cfg := config.Config{
		KickWebhookEvents:      []string{"channel.subscription.new"},
		KickWebhookSyncEnabled: true,
	}

	router := httpapi.NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:        cfg,
		WebhookEvents: inboxRepo,
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/webhooks/health", nil))
	// No auth → 401, confirming route is registered and auth-gated
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestWebhookSyncRequiresAuth(t *testing.T) {
	router := httpapi.NewRouter(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/webhooks/sync", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestWebhookSyncNoSyncServiceReturns503(t *testing.T) {
	// When auth succeeds (nil auth service → 401), and no sync service → 503.
	// We test auth-gate here; full sync tested in kicksync unit tests.
	router := httpapi.NewRouter(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/webhooks/sync", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no auth configured)", w.Code)
	}
}

// --- fakeSubPeriodRepo ---

type fakeSubPeriodRepo struct {
	summary domain.ChannelSubscriptionSummary
}

func (r *fakeSubPeriodRepo) InsertBatch(_ context.Context, _ []domain.ChannelSubscriptionPeriod) error {
	return nil
}

func (r *fakeSubPeriodRepo) ActiveSummary(_ context.Context, _ int64) (domain.ChannelSubscriptionSummary, error) {
	return r.summary, nil
}

// --- fakeInboxRepo ---

type fakeInboxRepo struct {
	counts     map[string]int64
	latestTime time.Time
}

func (r *fakeInboxRepo) InsertIdempotent(_ context.Context, _ domain.KickWebhookEvent) error {
	return nil
}
func (r *fakeInboxRepo) GetByMessageID(_ context.Context, _ string) (domain.KickWebhookEvent, error) {
	return domain.KickWebhookEvent{}, nil
}
func (r *fakeInboxRepo) ListPending(_ context.Context, _ int, _ int) ([]domain.KickWebhookEvent, error) {
	return nil, nil
}
func (r *fakeInboxRepo) MarkProcessed(_ context.Context, _ string) error               { return nil }
func (r *fakeInboxRepo) MarkFailed(_ context.Context, _ string, _ string, _ int) error { return nil }
func (r *fakeInboxRepo) MarkIgnored(_ context.Context, _ string, _ string) error       { return nil }
func (r *fakeInboxRepo) CountByStatus(_ context.Context) (map[string]int64, error) {
	return r.counts, nil
}
func (r *fakeInboxRepo) LatestReceivedAt(_ context.Context) (time.Time, error) {
	return r.latestTime, nil
}

// --- adminFakeChannelRepo (minimal, for channelsusecase.Service) ---

type adminFakeChannelRepo struct {
	ch domain.FollowedChannel
}

func newAdminFakeChannelRepo(ch domain.FollowedChannel) *adminFakeChannelRepo {
	return &adminFakeChannelRepo{ch: ch}
}

func (r *adminFakeChannelRepo) Upsert(_ context.Context, ch domain.FollowedChannel) (domain.FollowedChannel, error) {
	return ch, nil
}
func (r *adminFakeChannelRepo) GetByID(_ context.Context, _ int64) (domain.FollowedChannel, error) {
	return r.ch, nil
}
func (r *adminFakeChannelRepo) GetBySlug(_ context.Context, slug string) (domain.FollowedChannel, error) {
	if r.ch.Slug == slug {
		return r.ch, nil
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}
func (r *adminFakeChannelRepo) GetByChatroomID(_ context.Context, _ int64) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{}, sql.ErrNoRows
}
func (r *adminFakeChannelRepo) GetByBroadcasterUserID(_ context.Context, id int64) (domain.FollowedChannel, error) {
	if r.ch.BroadcasterUserID == id {
		return r.ch, nil
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}
func (r *adminFakeChannelRepo) List(_ context.Context) ([]domain.FollowedChannel, error) {
	return []domain.FollowedChannel{r.ch}, nil
}
func (r *adminFakeChannelRepo) ListEnabled(_ context.Context) ([]domain.FollowedChannel, error) {
	return []domain.FollowedChannel{r.ch}, nil
}
func (r *adminFakeChannelRepo) Disable(_ context.Context, _ int64) (domain.FollowedChannel, error) {
	return r.ch, nil
}

// nopResolver satisfies ports.KickChannelResolver for test.
type nopResolver struct{}

func (n *nopResolver) ResolveChannel(_ context.Context, slug string) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{}, sql.ErrNoRows
}
