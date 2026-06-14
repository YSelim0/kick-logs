package analytics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestServiceCachesGlobalOverview(t *testing.T) {
	repo := &countingAnalyticsRepository{
		overview: domain.AnalyticsOverview{TotalMessages: 42},
	}
	service := NewService(repo)

	for range 2 {
		overview, err := service.Overview(context.Background(), domain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("Overview() error = %v", err)
		}
		if overview.TotalMessages != 42 {
			t.Fatalf("overview = %#v", overview)
		}
	}

	if repo.overviewCalls != 1 {
		t.Fatalf("overview calls = %d, want 1", repo.overviewCalls)
	}
}

func TestServiceDoesNotCacheScopedTopSenderSearch(t *testing.T) {
	repo := &countingAnalyticsRepository{
		topSenders: []domain.TopSenderAnalytics{{Username: "alice", Slug: "alice"}},
	}
	service := NewService(repo)
	filter := domain.AnalyticsFilter{Query: "alice"}

	for range 2 {
		senders, err := service.TopSenders(context.Background(), filter, 20)
		if err != nil {
			t.Fatalf("TopSenders() error = %v", err)
		}
		if len(senders) != 1 {
			t.Fatalf("senders = %#v", senders)
		}
	}

	if repo.topSendersCalls != 2 {
		t.Fatalf("top sender calls = %d, want 2", repo.topSendersCalls)
	}
}

func TestServiceCachesRepeatedGlobalVolumeRange(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 2, 8, 0, 0, 0, time.UTC)
	repo := &countingAnalyticsRepository{
		volume: []domain.MessageVolumePoint{{BucketStart: start, MessageCount: 12}},
	}
	service := NewService(repo)

	for range 2 {
		points, err := service.MessageVolume(
			context.Background(),
			domain.AnalyticsFilter{Start: start, End: end},
			domain.AnalyticsBucketDay,
		)
		if err != nil {
			t.Fatalf("MessageVolume() error = %v", err)
		}
		if len(points) != 2 {
			t.Fatalf("points = %#v", points)
		}
		if points[0].BucketStart != start || points[0].MessageCount != 12 {
			t.Fatalf("first point = %#v", points[0])
		}
		if points[1].BucketStart != start.AddDate(0, 0, 1) || points[1].MessageCount != 0 {
			t.Fatalf("second point = %#v", points[1])
		}
	}

	if repo.volumeCalls != 1 {
		t.Fatalf("volume calls = %d, want 1", repo.volumeCalls)
	}
}

func TestServiceFillsMissingHourlyVolumeBuckets(t *testing.T) {
	start := time.Date(2026, 6, 1, 10, 15, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 12, 45, 0, 0, time.UTC)
	repo := &countingAnalyticsRepository{
		volume: []domain.MessageVolumePoint{{
			BucketStart:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			MessageCount: 7,
		}},
	}
	service := NewService(repo)

	points, err := service.MessageVolume(
		context.Background(),
		domain.AnalyticsFilter{Start: start, End: end, Channel: "hype"},
		domain.AnalyticsBucketHour,
	)
	if err != nil {
		t.Fatalf("MessageVolume() error = %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("points = %#v", points)
	}
	expected := []int64{0, 0, 7}
	for index, wantCount := range expected {
		if points[index].MessageCount != wantCount {
			t.Fatalf("points[%d] = %#v, want count %d", index, points[index], wantCount)
		}
	}
}

func TestServiceReturnsStaleGlobalAnalyticsOnRefreshFailure(t *testing.T) {
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	repo := &countingAnalyticsRepository{
		overview: domain.AnalyticsOverview{TotalMessages: 42},
	}
	service := NewService(repo)
	service.cache.now = func() time.Time { return now }

	if _, err := service.Overview(context.Background(), domain.AnalyticsFilter{}); err != nil {
		t.Fatalf("Overview() warm error = %v", err)
	}

	now = now.Add(2 * time.Hour)
	repo.overviewErr = errors.New("clickhouse unavailable")
	overview, err := service.Overview(context.Background(), domain.AnalyticsFilter{})
	if err != nil {
		t.Fatalf("Overview() stale error = %v", err)
	}
	if overview.TotalMessages != 42 {
		t.Fatalf("overview = %#v", overview)
	}
	if repo.overviewCalls != 2 {
		t.Fatalf("overview calls = %d, want 2", repo.overviewCalls)
	}
}

func TestServiceCoalescesConcurrentGlobalOverviewRequests(t *testing.T) {
	repo := &countingAnalyticsRepository{
		overview:        domain.AnalyticsOverview{TotalMessages: 42},
		overviewStarted: make(chan struct{}),
		overviewRelease: make(chan struct{}),
	}
	service := NewService(repo)
	ctx := context.Background()

	firstDone := make(chan error, 1)
	go func() {
		_, err := service.Overview(ctx, domain.AnalyticsFilter{})
		firstDone <- err
	}()

	select {
	case <-repo.overviewStarted:
	case <-time.After(time.Second):
		t.Fatal("first overview request did not start")
	}

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Overview(ctx, domain.AnalyticsFilter{})
			errs <- err
		}()
	}

	close(repo.overviewRelease)
	if err := <-firstDone; err != nil {
		t.Fatalf("first Overview() error = %v", err)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("coalesced Overview() error = %v", err)
		}
	}
	if repo.overviewCalls != 1 {
		t.Fatalf("overview calls = %d, want 1", repo.overviewCalls)
	}
}

type countingAnalyticsRepository struct {
	mu sync.Mutex

	overview        domain.AnalyticsOverview
	overviewErr     error
	overviewCalls   int
	overviewStarted chan struct{}
	overviewRelease chan struct{}
	startedClosed   bool

	volume      []domain.MessageVolumePoint
	volumeCalls int

	topSenders      []domain.TopSenderAnalytics
	topSendersCalls int
}

func (repo *countingAnalyticsRepository) Overview(
	ctx context.Context,
	filter domain.AnalyticsFilter,
) (domain.AnalyticsOverview, error) {
	repo.mu.Lock()
	repo.overviewCalls++
	if repo.overviewStarted != nil && !repo.startedClosed {
		close(repo.overviewStarted)
		repo.startedClosed = true
	}
	release := repo.overviewRelease
	overview := repo.overview
	err := repo.overviewErr
	repo.mu.Unlock()

	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return domain.AnalyticsOverview{}, ctx.Err()
		}
	}
	return overview, err
}

func (repo *countingAnalyticsRepository) MessageVolume(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	bucket domain.AnalyticsBucket,
) ([]domain.MessageVolumePoint, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.volumeCalls++
	return repo.volume, nil
}

func (repo *countingAnalyticsRepository) TopSenders(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopSenderAnalytics, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.topSendersCalls++
	return repo.topSenders, nil
}

func (repo *countingAnalyticsRepository) TopChannels(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopChannelAnalytics, error) {
	return nil, nil
}

func (repo *countingAnalyticsRepository) TopEmotes(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopEmoteAnalytics, error) {
	return nil, nil
}

func (repo *countingAnalyticsRepository) LatestMessages(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.ChatMessage, error) {
	return nil, nil
}
