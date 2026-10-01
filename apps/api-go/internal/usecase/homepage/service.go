package homepage

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
)

const (
	RefreshInterval = 15 * time.Minute
	maxSnapshotAge  = 24 * time.Hour
	refreshTimeout  = 90 * time.Second
	snapshotVersion = 1
)

var ErrNotReady = errors.New("homepage analytics are not ready")

type Service struct {
	repository ports.AnalyticsRepository
	store      ports.HomepageSnapshotStore
	logger     *slog.Logger
	now        func() time.Time
	mu         sync.RWMutex
	refreshMu  sync.Mutex
	snapshot   domain.HomepageSnapshot
}

func NewService(repository ports.AnalyticsRepository, store ports.HomepageSnapshotStore, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repository: repository, store: store, logger: logger, now: time.Now}
}

// Get never performs I/O or waits for a refresh. The reporting interval belongs to
// the returned snapshot, including when it is stale across a UTC day boundary.
func (service *Service) Get() (domain.HomepageSnapshot, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if !validSnapshot(service.snapshot, service.now()) {
		return domain.HomepageSnapshot{}, ErrNotReady
	}
	snapshot := service.snapshot
	snapshot.Volume = append([]domain.MessageVolumePoint(nil), snapshot.Volume...)
	snapshot.TopChannels = append([]domain.TopChannelAnalytics(nil), snapshot.TopChannels...)
	snapshot.TopSenders = append([]domain.TopSenderAnalytics(nil), snapshot.TopSenders...)
	snapshot.TopEmotes = append([]domain.TopEmoteAnalytics(nil), snapshot.TopEmotes...)
	return snapshot, nil
}

func (service *Service) Restore() error {
	if service.store == nil {
		return ErrNotReady
	}
	snapshot, err := service.store.Load()
	if err != nil {
		return err
	}
	if !validSnapshot(snapshot, service.now()) {
		return ErrNotReady
	}
	service.mu.Lock()
	service.snapshot = snapshot
	service.mu.Unlock()
	return nil
}

// Run is started once by the API lifetime, not once per visitor. Queries execute
// sequentially; a failing database receives exponential retry delays up to 15m.
func (service *Service) Run(ctx context.Context) {
	if err := service.Restore(); err != nil {
		service.logger.Debug("homepage snapshot unavailable at startup", "error", err)
	}
	if snapshot, err := service.Get(); err == nil {
		if delay := RefreshInterval - service.now().Sub(snapshot.AsOf); delay > 0 && !wait(ctx, delay) {
			return
		}
	}
	backoff := time.Minute
	for ctx.Err() == nil {
		delay := RefreshInterval
		if err := service.Refresh(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			service.logger.Warn("homepage analytics refresh failed", "error", err, "retry_in", backoff)
			delay = backoff
			backoff = min(backoff*2, RefreshInterval)
		} else {
			backoff = time.Minute
		}
		if !wait(ctx, delay) {
			return
		}
	}
}

func (service *Service) Refresh(ctx context.Context) error {
	if service.repository == nil {
		return errors.New("homepage analytics database unavailable")
	}
	if !service.refreshMu.TryLock() {
		return nil
	}
	defer service.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	// Match the second precision used by the existing analytics query bindings.
	asOf := service.now().UTC().Truncate(time.Second)
	filter := domain.AnalyticsFilter{Start: startOfWindow(asOf), End: asOf}
	snapshot := domain.HomepageSnapshot{Version: snapshotVersion, AsOf: asOf, Start: filter.Start, End: filter.End}
	var err error
	if snapshot.Overview, err = service.repository.Overview(ctx, filter); err != nil {
		return err
	}
	if snapshot.Volume, err = service.repository.MessageVolume(ctx, filter, domain.AnalyticsBucketDay); err != nil {
		return err
	}
	snapshot.Volume = analyticsusecase.FillMessageVolumeRange(snapshot.Volume, filter, domain.AnalyticsBucketDay)
	if snapshot.TopChannels, err = service.repository.TopChannels(ctx, filter, 5); err != nil {
		return err
	}
	if snapshot.TopSenders, err = service.repository.TopSenders(ctx, filter, 5); err != nil {
		return err
	}
	if snapshot.TopEmotes, err = service.repository.TopEmotes(ctx, filter, 5); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	service.mu.Lock()
	service.snapshot = snapshot
	service.mu.Unlock()
	if service.store != nil {
		if err := service.store.Save(snapshot); err != nil {
			service.logger.Warn("homepage snapshot persistence failed; serving memory snapshot", "error", err)
		}
	}
	return nil
}

func startOfWindow(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -13)
}

func validSnapshot(snapshot domain.HomepageSnapshot, now time.Time) bool {
	age := now.Sub(snapshot.AsOf)
	if snapshot.Version != snapshotVersion || snapshot.AsOf.IsZero() || age < 0 || age > maxSnapshotAge ||
		!snapshot.Start.Equal(startOfWindow(snapshot.AsOf)) || !snapshot.End.Equal(snapshot.AsOf) || len(snapshot.Volume) != 14 {
		return false
	}
	for i, point := range snapshot.Volume {
		if point.MessageCount < 0 || !point.BucketStart.Equal(snapshot.Start.AddDate(0, 0, i)) {
			return false
		}
	}
	return true
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
