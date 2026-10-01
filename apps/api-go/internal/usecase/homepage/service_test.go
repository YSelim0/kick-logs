package homepage

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

func TestSnapshotUsesOneFourteenDayWindowAndFillsEmptyDays(t *testing.T) {
	repo := &testAnalytics{}
	store := &testStore{}
	service := newTestService(repo, store)
	if _, err := service.Get(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("uninitialized Get = %v", err)
	}
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.Get()
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	if snapshot.Start != start || snapshot.End != service.now() || snapshot.Overview.TotalMessages != 42 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if len(snapshot.Volume) != 14 || snapshot.Volume[0].MessageCount != 0 || snapshot.Volume[13].MessageCount != 42 {
		t.Fatalf("unexpected volume: %+v", snapshot.Volume)
	}
	if len(repo.filters) != 5 {
		t.Fatalf("query count = %d", len(repo.filters))
	}
	for _, filter := range repo.filters {
		if filter.Start != start || filter.End != snapshot.End || filter.Sender != "" || filter.Channel != "" {
			t.Fatalf("inconsistent scope: %+v", filter)
		}
	}
	for range 100 {
		if _, err := service.Get(); err != nil {
			t.Fatal(err)
		}
	}
	if len(repo.filters) != 5 {
		t.Fatal("visitors triggered database reads")
	}
	if store.saved.Overview.TotalMessages != 42 {
		t.Fatal("snapshot not persisted")
	}
}

func TestRefreshFailureKeepsLastGoodSnapshot(t *testing.T) {
	repo := &testAnalytics{}
	service := newTestService(repo, &testStore{})
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	repo.err = errors.New("database unavailable")
	if err := service.Refresh(context.Background()); err == nil {
		t.Fatal("expected refresh failure")
	}
	snapshot, err := service.Get()
	if err != nil || snapshot.Overview.TotalMessages != 42 {
		t.Fatalf("lost last good snapshot: %v", err)
	}
}

func TestGetDoesNotWaitForBackgroundRefresh(t *testing.T) {
	repo := &testAnalytics{}
	service := newTestService(repo, &testStore{})
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	repo.started, repo.release = make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- service.Refresh(context.Background()) }()
	<-repo.started
	if _, err := service.Get(); err != nil {
		t.Fatal(err)
	}
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	close(repo.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(repo.filters) != 10 {
		t.Fatalf("overlapping refreshes: %d reads", len(repo.filters))
	}
}

func TestRestoreSurvivesRestartAndRejectsExpiredOrFutureSnapshot(t *testing.T) {
	store := &testStore{}
	service := newTestService(&testAnalytics{}, store)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	restarted := newTestService(&testAnalytics{err: errors.New("offline")}, store)
	if err := restarted.Restore(); err != nil {
		t.Fatal(err)
	}
	if snapshot, err := restarted.Get(); err != nil || snapshot.Overview.TotalMessages != 42 {
		t.Fatal("restore failed", err)
	}
	for _, delta := range []time.Duration{25 * time.Hour, -time.Hour} {
		now := service.now().Add(delta)
		restarted.now = func() time.Time { return now }
		if _, err := restarted.Get(); !errors.Is(err, ErrNotReady) {
			t.Fatalf("invalid snapshot age: %v", err)
		}
	}
	store.saved.Version++
	if err := restarted.Restore(); err == nil {
		t.Fatal("accepted incompatible cache version")
	}
}

func TestPersistFailureDoesNotHideFreshSnapshot(t *testing.T) {
	service := newTestService(&testAnalytics{}, &testStore{saveErr: errors.New("read-only filesystem")})
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(); err != nil {
		t.Fatal(err)
	}
}

func TestLateRefreshFailureDoesNotPublishMixedSnapshot(t *testing.T) {
	repo := &testAnalytics{}
	service := newTestService(repo, &testStore{})
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	original, _ := service.Get()
	service.now = func() time.Time { return original.AsOf.Add(time.Hour) }
	repo.emoteErr = errors.New("last panel failed")
	if err := service.Refresh(context.Background()); err == nil {
		t.Fatal("expected last query failure")
	}
	actual, err := service.Get()
	if err != nil || !actual.AsOf.Equal(original.AsOf) {
		t.Fatalf("published partial snapshot: %+v, %v", actual, err)
	}
}

func TestRunRestoresFreshSnapshotWithoutQueryingDatabase(t *testing.T) {
	store := &testStore{}
	seed := newTestService(&testAnalytics{}, store)
	if err := seed.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	restarted := newTestService(nil, store)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { restarted.Run(ctx); close(done) }()
	deadline := time.After(time.Second)
	for {
		if _, err := restarted.Get(); err == nil {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("snapshot not restored")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fresh snapshot wait ignored cancellation")
	}
}

func TestMissingDatabaseStillServesRestoredSnapshot(t *testing.T) {
	store := &testStore{}
	seed := newTestService(&testAnalytics{}, store)
	if err := seed.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := newTestService(nil, store)
	if err := service.Restore(); err != nil {
		t.Fatal(err)
	}
	if err := service.Refresh(context.Background()); err == nil {
		t.Fatal("missing database must report an error")
	}
	if _, err := service.Get(); err != nil {
		t.Fatal("missing database discarded restored snapshot", err)
	}
}

func TestRunCancelsBlockedRefresh(t *testing.T) {
	repo := &testAnalytics{started: make(chan struct{}), release: make(chan struct{})}
	service := newTestService(repo, &testStore{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { service.Run(ctx); close(done) }()
	<-repo.started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh ignored shutdown")
	}
}

func newTestService(repo ports.AnalyticsRepository, store ports.HomepageSnapshotStore) *Service {
	service := NewService(repo, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.now = func() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC) }
	return service
}

type testStore struct {
	saved   domain.HomepageSnapshot
	saveErr error
}

func (store *testStore) Load() (domain.HomepageSnapshot, error) { return store.saved, nil }
func (store *testStore) Save(snapshot domain.HomepageSnapshot) error {
	store.saved = snapshot
	return store.saveErr
}

type testAnalytics struct {
	ports.AnalyticsRepository
	mu               sync.Mutex
	filters          []domain.AnalyticsFilter
	err              error
	emoteErr         error
	started, release chan struct{}
}

func (repo *testAnalytics) record(filter domain.AnalyticsFilter) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.filters = append(repo.filters, filter)
}
func (repo *testAnalytics) Overview(ctx context.Context, filter domain.AnalyticsFilter) (domain.AnalyticsOverview, error) {
	repo.record(filter)
	if repo.started != nil {
		close(repo.started)
		select {
		case <-repo.release:
		case <-ctx.Done():
			return domain.AnalyticsOverview{}, ctx.Err()
		}
	}
	return domain.AnalyticsOverview{TotalMessages: 42}, repo.err
}
func (repo *testAnalytics) MessageVolume(_ context.Context, filter domain.AnalyticsFilter, _ domain.AnalyticsBucket) ([]domain.MessageVolumePoint, error) {
	repo.record(filter)
	return []domain.MessageVolumePoint{{BucketStart: filter.End, MessageCount: 42}}, nil
}
func (repo *testAnalytics) TopSenders(_ context.Context, filter domain.AnalyticsFilter, limit uint64) ([]domain.TopSenderAnalytics, error) {
	repo.record(filter)
	return []domain.TopSenderAnalytics{{Username: "tester", MessageCount: int64(limit)}}, nil
}
func (repo *testAnalytics) TopChannels(_ context.Context, filter domain.AnalyticsFilter, _ uint64) ([]domain.TopChannelAnalytics, error) {
	repo.record(filter)
	return nil, nil
}
func (repo *testAnalytics) TopEmotes(_ context.Context, filter domain.AnalyticsFilter, _ uint64) ([]domain.TopEmoteAnalytics, error) {
	repo.record(filter)
	return nil, repo.emoteErr
}
