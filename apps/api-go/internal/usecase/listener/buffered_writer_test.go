package listener

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestBufferedWriterFlushesOnSize(t *testing.T) {
	rawRepo := &recordingRawEventRepository{}
	queueRepo := newRecordingQueueRepository()
	writer := newBufferedRawWriter(BufferedWriterConfig{
		BatchSize:         3,
		FlushInterval:     time.Hour,
		QueueSize:         10,
		MaxRetries:        2,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}, rawRepo, queueRepo, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go writer.Run(ctx)

	for i := 0; i < 3; i++ {
		writer.Submit(domain.RawKickEvent{ID: idAt(i), KickMessageID: idAt(i), ReceivedAt: time.Now().UTC()})
	}
	waitFor(t, func() bool {
		return writer.Stats().FlushCount >= 1
	})
	cancel()
	writer.Wait()

	stats := writer.Stats()
	if stats.FlushCount < 1 {
		t.Fatalf("flush count = %d", stats.FlushCount)
	}
	if stats.LastFlushSize != 3 {
		t.Fatalf("last flush size = %d", stats.LastFlushSize)
	}
	if rawRepo.totalEvents() != 3 {
		t.Fatalf("raw events stored = %d", rawRepo.totalEvents())
	}
	if queueRepo.totalEnqueued() != 3 {
		t.Fatalf("queue rows = %d", queueRepo.totalEnqueued())
	}
}

func TestBufferedWriterFlushesOnInterval(t *testing.T) {
	rawRepo := &recordingRawEventRepository{}
	queueRepo := newRecordingQueueRepository()
	writer := newBufferedRawWriter(BufferedWriterConfig{
		BatchSize:         100,
		FlushInterval:     20 * time.Millisecond,
		QueueSize:         10,
		MaxRetries:        2,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}, rawRepo, queueRepo, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go writer.Run(ctx)

	writer.Submit(domain.RawKickEvent{ID: "a", KickMessageID: "a", ReceivedAt: time.Now().UTC()})
	waitFor(t, func() bool {
		return writer.Stats().FlushCount >= 1
	})
	cancel()
	writer.Wait()

	if rawRepo.totalEvents() != 1 {
		t.Fatalf("raw events stored = %d", rawRepo.totalEvents())
	}
	if queueRepo.totalEnqueued() != 1 {
		t.Fatalf("queue rows = %d", queueRepo.totalEnqueued())
	}
}

func TestBufferedWriterDrainsOnShutdown(t *testing.T) {
	rawRepo := &recordingRawEventRepository{}
	queueRepo := newRecordingQueueRepository()
	writer := newBufferedRawWriter(BufferedWriterConfig{
		BatchSize:         100,
		FlushInterval:     time.Hour,
		QueueSize:         10,
		MaxRetries:        2,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}, rawRepo, queueRepo, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go writer.Run(ctx)
	for i := 0; i < 5; i++ {
		writer.Submit(domain.RawKickEvent{ID: idAt(i), KickMessageID: idAt(i), ReceivedAt: time.Now().UTC()})
	}
	cancel()
	writer.Wait()

	if rawRepo.totalEvents() != 5 {
		t.Fatalf("raw events stored on drain = %d", rawRepo.totalEvents())
	}
	if queueRepo.totalEnqueued() != 5 {
		t.Fatalf("queue rows on drain = %d", queueRepo.totalEnqueued())
	}
}

func TestBufferedWriterDropsWhenQueueFull(t *testing.T) {
	rawRepo := &blockingRawEventRepository{block: make(chan struct{})}
	queueRepo := newRecordingQueueRepository()
	writer := newBufferedRawWriter(BufferedWriterConfig{
		BatchSize:         100,
		FlushInterval:     time.Hour,
		QueueSize:         2,
		MaxRetries:        1,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}, rawRepo, queueRepo, testLogger())

	for i := 0; i < 6; i++ {
		writer.Submit(domain.RawKickEvent{ID: idAt(i), KickMessageID: idAt(i), ReceivedAt: time.Now().UTC()})
	}
	close(rawRepo.block)

	if writer.Stats().DropCount == 0 {
		t.Fatal("expected drops when queue overflowed")
	}
}

func TestBufferedWriterRetriesAndDropsAfterMaxRetries(t *testing.T) {
	rawRepo := &failingRawEventRepository{maxFailures: 5}
	queueRepo := newRecordingQueueRepository()
	writer := newBufferedRawWriter(BufferedWriterConfig{
		BatchSize:         1,
		FlushInterval:     time.Hour,
		QueueSize:         10,
		MaxRetries:        3,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}, rawRepo, queueRepo, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go writer.Run(ctx)
	writer.Submit(domain.RawKickEvent{ID: "a", KickMessageID: "a", ReceivedAt: time.Now().UTC()})
	waitFor(t, func() bool {
		return writer.Stats().ClickHouseFailures >= 3
	})
	cancel()
	writer.Wait()

	if writer.Stats().DropCount < 1 {
		t.Fatalf("expected drop after max retries, stats = %#v", writer.Stats())
	}
	if queueRepo.totalEnqueued() != 0 {
		t.Fatalf("queue should not receive events from a failed CH batch, got %d", queueRepo.totalEnqueued())
	}
}

func TestBufferedWriterRetriesSqliteEnqueueAfterClickHouseSuccess(t *testing.T) {
	rawRepo := &recordingRawEventRepository{}
	queueRepo := newRecordingQueueRepository()
	queueRepo.failuresBeforeSuccess.Store(2)
	writer := newBufferedRawWriter(BufferedWriterConfig{
		BatchSize:         1,
		FlushInterval:     time.Hour,
		QueueSize:         10,
		MaxRetries:        3,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}, rawRepo, queueRepo, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go writer.Run(ctx)
	writer.Submit(domain.RawKickEvent{ID: "a", KickMessageID: "a", ReceivedAt: time.Now().UTC()})
	waitFor(t, func() bool {
		return queueRepo.totalEnqueued() >= 1
	})
	cancel()
	writer.Wait()

	if writer.Stats().QueueEnqueueFails < 2 {
		t.Fatalf("expected SQLite enqueue retries, stats = %#v", writer.Stats())
	}
	if rawRepo.totalEvents() != 1 {
		t.Fatalf("raw events stored = %d", rawRepo.totalEvents())
	}
	if queueRepo.totalEnqueued() != 1 {
		t.Fatalf("queue rows = %d", queueRepo.totalEnqueued())
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func idAt(i int) string {
	return string(rune('a' + i))
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

type recordingRawEventRepository struct {
	mu     sync.Mutex
	events []domain.RawKickEvent
}

func (repo *recordingRawEventRepository) totalEvents() int {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return len(repo.events)
}

func (repo *recordingRawEventRepository) InsertEvent(ctx context.Context, event domain.RawKickEvent) error {
	return repo.InsertEventsBatch(ctx, []domain.RawKickEvent{event})
}

func (repo *recordingRawEventRepository) InsertEventsBatch(_ context.Context, events []domain.RawKickEvent) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.events = append(repo.events, events...)
	return nil
}

func (repo *recordingRawEventRepository) InsertAttempt(_ context.Context, _ domain.RawEventAttempt) error {
	return nil
}

func (repo *recordingRawEventRepository) InsertAttemptsBatch(_ context.Context, _ []domain.RawEventAttempt) error {
	return nil
}

func (repo *recordingRawEventRepository) ListUnprocessed(_ context.Context, _ uint64, _ uint16) ([]domain.RawKickEvent, error) {
	return nil, nil
}

func (repo *recordingRawEventRepository) CountUnprocessed(_ context.Context, _ uint16) (int64, error) {
	return 0, nil
}

func (repo *recordingRawEventRepository) AttemptCount(_ context.Context, _ string) (uint16, error) {
	return 0, nil
}

func (repo *recordingRawEventRepository) GetByID(_ context.Context, _ string) (domain.RawKickEvent, error) {
	return domain.RawKickEvent{}, nil
}

func (repo *recordingRawEventRepository) GetByIDs(_ context.Context, ids []string) (map[string]domain.RawKickEvent, error) {
	return map[string]domain.RawKickEvent{}, nil
}

type blockingRawEventRepository struct {
	recordingRawEventRepository
	block chan struct{}
}

func (repo *blockingRawEventRepository) InsertEventsBatch(ctx context.Context, events []domain.RawKickEvent) error {
	select {
	case <-repo.block:
	case <-ctx.Done():
		return ctx.Err()
	}
	return repo.recordingRawEventRepository.InsertEventsBatch(ctx, events)
}

type failingRawEventRepository struct {
	recordingRawEventRepository
	mu          sync.Mutex
	failures    int
	maxFailures int
}

func (repo *failingRawEventRepository) InsertEventsBatch(ctx context.Context, events []domain.RawKickEvent) error {
	repo.mu.Lock()
	repo.failures++
	if repo.failures <= repo.maxFailures {
		repo.mu.Unlock()
		return errors.New("clickhouse unavailable")
	}
	repo.mu.Unlock()
	return repo.recordingRawEventRepository.InsertEventsBatch(ctx, events)
}

type recordingQueueRepository struct {
	mu                    sync.Mutex
	items                 map[string]domain.RawEventQueueItem
	failuresBeforeSuccess atomic.Int32
}

func newRecordingQueueRepository() *recordingQueueRepository {
	return &recordingQueueRepository{items: make(map[string]domain.RawEventQueueItem)}
}

func (repo *recordingQueueRepository) totalEnqueued() int {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return len(repo.items)
}

func (repo *recordingQueueRepository) Enqueue(ctx context.Context, item domain.RawEventQueueItem) error {
	return repo.EnqueueBatch(ctx, []domain.RawEventQueueItem{item})
}

func (repo *recordingQueueRepository) EnqueueBatch(_ context.Context, items []domain.RawEventQueueItem) error {
	if repo.failuresBeforeSuccess.Load() > 0 {
		repo.failuresBeforeSuccess.Add(-1)
		return errors.New("sqlite unavailable")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, item := range items {
		if _, exists := repo.items[item.RawEventID]; exists {
			continue
		}
		repo.items[item.RawEventID] = item
	}
	return nil
}

func (repo *recordingQueueRepository) ListPending(_ context.Context, _ uint64, _ uint16) ([]domain.RawEventQueueItem, error) {
	return nil, nil
}

func (repo *recordingQueueRepository) Claim(_ context.Context, _ string, _ string) (bool, error) {
	return false, nil
}

func (repo *recordingQueueRepository) Release(_ context.Context, _ string, _ string) error {
	return nil
}

func (repo *recordingQueueRepository) MarkProcessed(_ context.Context, _ string) error {
	return nil
}

func (repo *recordingQueueRepository) MarkFailed(_ context.Context, _ string, _ string, _ uint16) error {
	return nil
}

func (repo *recordingQueueRepository) CountPending(_ context.Context, _ uint16) (int64, error) {
	return 0, nil
}

func (repo *recordingQueueRepository) OldestPendingAge(_ context.Context, _ uint16) (time.Duration, error) {
	return 0, nil
}

func (repo *recordingQueueRepository) RecoverStaleClaims(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

func (repo *recordingQueueRepository) GetByID(_ context.Context, _ string) (domain.RawEventQueueItem, error) {
	return domain.RawEventQueueItem{}, nil
}
