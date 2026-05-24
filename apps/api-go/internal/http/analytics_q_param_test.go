package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
)

type qFilterRecorder struct {
	now        time.Time
	lastFilter domain.AnalyticsFilter
}

func newQFilterRecorder() *qFilterRecorder {
	return &qFilterRecorder{now: time.Date(2040, 5, 1, 10, 0, 0, 0, time.UTC)}
}

func (r *qFilterRecorder) Overview(_ context.Context, f domain.AnalyticsFilter) (domain.AnalyticsOverview, error) {
	r.lastFilter = f
	return domain.AnalyticsOverview{}, nil
}

func (r *qFilterRecorder) MessageVolume(
	_ context.Context, f domain.AnalyticsFilter, _ domain.AnalyticsBucket,
) ([]domain.MessageVolumePoint, error) {
	r.lastFilter = f
	return []domain.MessageVolumePoint{}, nil
}

func (r *qFilterRecorder) TopSenders(
	_ context.Context, f domain.AnalyticsFilter, _ uint64,
) ([]domain.TopSenderAnalytics, error) {
	r.lastFilter = f
	return []domain.TopSenderAnalytics{}, nil
}

func (r *qFilterRecorder) TopChannels(
	_ context.Context, f domain.AnalyticsFilter, _ uint64,
) ([]domain.TopChannelAnalytics, error) {
	r.lastFilter = f
	return []domain.TopChannelAnalytics{}, nil
}

func (r *qFilterRecorder) TopEmotes(
	_ context.Context, f domain.AnalyticsFilter, _ uint64,
) ([]domain.TopEmoteAnalytics, error) {
	r.lastFilter = f
	return []domain.TopEmoteAnalytics{}, nil
}

func (r *qFilterRecorder) LatestMessages(
	_ context.Context, f domain.AnalyticsFilter, _ uint64,
) ([]domain.ChatMessage, error) {
	r.lastFilter = f
	return []domain.ChatMessage{}, nil
}

func newQTestRouter(repo *qFilterRecorder) http.Handler {
	cfg := config.Config{BackendCORSOrigins: []string{"http://localhost:3000"}}
	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:    cfg,
		Analytics: analyticsusecase.NewService(repo),
		Profiles:  profilesusecase.NewService(repo, newFakeProfileChannelRepository(), newFakeProfileSenderRepository()),
	})
}

func TestTopSendersQParamPassedToRepository(t *testing.T) {
	repo := newQFilterRecorder()
	handler := newQTestRouter(repo)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/analytics/top-senders?q=alice", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if repo.lastFilter.Query != "alice" {
		t.Fatalf("Query = %q, want %q", repo.lastFilter.Query, "alice")
	}
}

func TestTopChannelsQParamPassedToRepository(t *testing.T) {
	repo := newQFilterRecorder()
	handler := newQTestRouter(repo)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/analytics/top-channels?q=hype", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if repo.lastFilter.Query != "hype" {
		t.Fatalf("Query = %q, want %q", repo.lastFilter.Query, "hype")
	}
}

func TestTopSendersWithoutQParamHasEmptyQuery(t *testing.T) {
	repo := newQFilterRecorder()
	handler := newQTestRouter(repo)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/analytics/top-senders", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if repo.lastFilter.Query != "" {
		t.Fatalf("Query = %q, want empty", repo.lastFilter.Query)
	}
}

func TestTopChannelsWithoutQParamHasEmptyQuery(t *testing.T) {
	repo := newQFilterRecorder()
	handler := newQTestRouter(repo)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/analytics/top-channels", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if repo.lastFilter.Query != "" {
		t.Fatalf("Query = %q, want empty", repo.lastFilter.Query)
	}
}
