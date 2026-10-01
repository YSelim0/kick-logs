package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
)

func TestProfileCombinedQueriesPreserveResponseAndCache(t *testing.T) {
	for _, path := range []string{"/users/profile-user/analytics", "/channels/hype/analytics"} {
		t.Run(path, func(t *testing.T) {
			want := profileQueryResponse(t, newAnalyticsProfileTestRouter(), path)
			repo := &combinedProfileRepository{fakeAnalyticsRepository: newFakeAnalyticsRepository()}
			handler := combinedProfileRouter(repo)
			for range 2 {
				got := profileQueryResponse(t, handler, path)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("profile response changed: got=%#v want=%#v", got, want)
				}
			}
			if repo.combinedCalls != 1 || repo.legacyCalls != 0 {
				t.Fatalf("combined calls=%d legacy calls=%d; want 1 and 0", repo.combinedCalls, repo.legacyCalls)
			}
			if !repo.filter.Start.IsZero() || !repo.filter.End.IsZero() {
				t.Fatalf("profile summary must remain all-time: %+v", repo.filter)
			}
			if repo.limit != 5 || (repo.filter.Channel != "hype" && repo.filter.Sender != "profile_user") {
				t.Fatalf("unexpected summary filter=%+v limit=%d", repo.filter, repo.limit)
			}
		})
	}
}

func TestProfileCombinedQueryFailurePreservesLegacyResults(t *testing.T) {
	for _, path := range []string{"/users/profile-user/analytics", "/channels/hype/analytics"} {
		t.Run(path, func(t *testing.T) {
			want := profileQueryResponse(t, newAnalyticsProfileTestRouter(), path)
			repo := &combinedProfileRepository{
				fakeAnalyticsRepository: newFakeAnalyticsRepository(),
				combinedErr:             errors.New("combined query unavailable"),
			}
			got := profileQueryResponse(t, combinedProfileRouter(repo), path)
			if !reflect.DeepEqual(got, want) || repo.combinedCalls != 1 || repo.legacyCalls != 2 {
				t.Fatalf("fallback changed result: combined=%d legacy=%d got=%#v", repo.combinedCalls, repo.legacyCalls, got)
			}
		})
	}
}

func combinedProfileRouter(repo *combinedProfileRepository) http.Handler {
	cfg := config.Config{}
	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config: cfg,
		Profiles: profilesusecase.NewService(
			repo, newFakeProfileChannelRepository(), newFakeProfileSenderRepository(),
		),
	})
}

func profileQueryResponse(t *testing.T, handler http.Handler, path string) any {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

type combinedProfileRepository struct {
	*fakeAnalyticsRepository
	combinedCalls int
	legacyCalls   int
	combinedErr   error
	filter        domain.AnalyticsFilter
	limit         uint64
}

func (repo *combinedProfileRepository) Overview(ctx context.Context, filter domain.AnalyticsFilter) (domain.AnalyticsOverview, error) {
	repo.legacyCalls++
	return repo.fakeAnalyticsRepository.Overview(ctx, filter)
}

func (repo *combinedProfileRepository) TopSenders(ctx context.Context, filter domain.AnalyticsFilter, limit uint64) ([]domain.TopSenderAnalytics, error) {
	repo.legacyCalls++
	return repo.fakeAnalyticsRepository.TopSenders(ctx, filter, limit)
}

func (repo *combinedProfileRepository) TopChannels(ctx context.Context, filter domain.AnalyticsFilter, limit uint64) ([]domain.TopChannelAnalytics, error) {
	repo.legacyCalls++
	return repo.fakeAnalyticsRepository.TopChannels(ctx, filter, limit)
}

func (repo *combinedProfileRepository) OverviewAndTopSenders(ctx context.Context, filter domain.AnalyticsFilter, limit uint64) (domain.AnalyticsOverview, []domain.TopSenderAnalytics, error) {
	repo.combinedCalls++
	repo.filter, repo.limit = filter, limit
	if repo.combinedErr != nil {
		return domain.AnalyticsOverview{}, nil, repo.combinedErr
	}
	overview, _ := repo.fakeAnalyticsRepository.Overview(ctx, filter)
	senders, _ := repo.fakeAnalyticsRepository.TopSenders(ctx, filter, limit)
	return overview, senders, nil
}

func (repo *combinedProfileRepository) OverviewAndTopChannels(ctx context.Context, filter domain.AnalyticsFilter, limit uint64) (domain.AnalyticsOverview, []domain.TopChannelAnalytics, error) {
	repo.combinedCalls++
	repo.filter, repo.limit = filter, limit
	if repo.combinedErr != nil {
		return domain.AnalyticsOverview{}, nil, repo.combinedErr
	}
	overview, _ := repo.fakeAnalyticsRepository.Overview(ctx, filter)
	channels, _ := repo.fakeAnalyticsRepository.TopChannels(ctx, filter, limit)
	return overview, channels, nil
}
