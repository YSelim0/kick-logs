package listener

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type BufferedWriterConfig struct {
	BatchSize         int
	FlushInterval     time.Duration
	QueueSize         int
	MaxRetries        int
	RetryInitialDelay time.Duration
	RetryMaxDelay     time.Duration
}

type BufferedWriterStats struct {
	QueueDepth         int64
	QueueHighWaterMark int64
	DropCount          int64
	FlushCount         int64
	LastFlushSize      int64
	LastFlushNanos     int64
	ClickHouseFailures int64
	QueueEnqueueFails  int64
}

type bufferedRawWriter struct {
	cfg       BufferedWriterConfig
	rawEvents ports.RawEventRepository
	queue     ports.RawEventQueueRepository
	logger    *slog.Logger
	breaker   *CircuitBreaker

	events chan domain.RawKickEvent
	done   chan struct{}
	stats  atomicStats
}

type atomicStats struct {
	depth         atomic.Int64
	highWaterMark atomic.Int64
	dropCount     atomic.Int64
	flushCount    atomic.Int64
	lastFlushSize atomic.Int64
	lastFlushNs   atomic.Int64
	chFailures    atomic.Int64
	queueFailures atomic.Int64
}

func newBufferedRawWriter(
	cfg BufferedWriterConfig,
	rawEvents ports.RawEventRepository,
	queue ports.RawEventQueueRepository,
	logger *slog.Logger,
) *bufferedRawWriter {
	cfg = cfg.withDefaults()
	if logger == nil {
		logger = slog.Default()
	}
	return &bufferedRawWriter{
		cfg:       cfg,
		rawEvents: rawEvents,
		queue:     queue,
		logger:    logger,
		events:    make(chan domain.RawKickEvent, cfg.QueueSize),
		done:      make(chan struct{}),
	}
}

func (cfg BufferedWriterConfig) withDefaults() BufferedWriterConfig {
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 500
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 500 * time.Millisecond
	}
	if cfg.QueueSize < 1 {
		cfg.QueueSize = 50000
	}
	if cfg.MaxRetries < 1 {
		cfg.MaxRetries = 10
	}
	if cfg.RetryInitialDelay <= 0 {
		cfg.RetryInitialDelay = 100 * time.Millisecond
	}
	if cfg.RetryMaxDelay <= 0 {
		cfg.RetryMaxDelay = 5 * time.Second
	}
	return cfg
}

func (writer *bufferedRawWriter) Submit(event domain.RawKickEvent) {
	select {
	case writer.events <- event:
		depth := writer.stats.depth.Add(1)
		for {
			high := writer.stats.highWaterMark.Load()
			if depth <= high {
				break
			}
			if writer.stats.highWaterMark.CompareAndSwap(high, depth) {
				break
			}
		}
	default:
		select {
		case dropped := <-writer.events:
			writer.stats.depth.Add(-1)
			writer.stats.dropCount.Add(1)
			writer.logger.Warn(
				"buffered raw writer queue full, dropping oldest event",
				"dropped_raw_event_id", dropped.ID,
				"drop_count", writer.stats.dropCount.Load(),
			)
		default:
		}
		select {
		case writer.events <- event:
			writer.stats.depth.Add(1)
		default:
			writer.stats.dropCount.Add(1)
			writer.logger.Warn(
				"buffered raw writer queue still full, dropping incoming event",
				"raw_event_id", event.ID,
				"drop_count", writer.stats.dropCount.Load(),
			)
		}
	}
}

func (writer *bufferedRawWriter) Run(ctx context.Context) {
	defer close(writer.done)

	timer := time.NewTimer(writer.cfg.FlushInterval)
	defer timer.Stop()

	buffer := make([]domain.RawKickEvent, 0, writer.cfg.BatchSize)
	flush := func(reason string) {
		if len(buffer) == 0 {
			return
		}
		writer.flush(ctx, buffer, reason)
		buffer = buffer[:0]
		resetTimer(timer, writer.cfg.FlushInterval)
	}

	for {
		select {
		case <-ctx.Done():
			writer.drain(buffer)
			return
		case event := <-writer.events:
			writer.stats.depth.Add(-1)
			buffer = append(buffer, event)
			if len(buffer) >= writer.cfg.BatchSize {
				flush("size")
			}
		case <-timer.C:
			flush("interval")
			resetTimer(timer, writer.cfg.FlushInterval)
		}
	}
}

func (writer *bufferedRawWriter) drain(remaining []domain.RawKickEvent) {
	drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	buffer := make([]domain.RawKickEvent, 0, writer.cfg.BatchSize)
	buffer = append(buffer, remaining...)
	for {
		select {
		case event := <-writer.events:
			writer.stats.depth.Add(-1)
			buffer = append(buffer, event)
			if len(buffer) >= writer.cfg.BatchSize {
				writer.flush(drainCtx, buffer, "shutdown")
				buffer = buffer[:0]
			}
		default:
			if len(buffer) > 0 {
				writer.flush(drainCtx, buffer, "shutdown")
			}
			return
		}
	}
}

func (writer *bufferedRawWriter) flush(ctx context.Context, batch []domain.RawKickEvent, reason string) {
	start := time.Now()
	delay := writer.cfg.RetryInitialDelay
	var chErr error
	for attempt := 1; attempt <= writer.cfg.MaxRetries; attempt++ {
		if writer.breaker != nil {
			if err := writer.breaker.Wait(ctx); err != nil {
				return
			}
		}
		chErr = writer.rawEvents.InsertEventsBatch(ctx, batch)
		if chErr == nil {
			if writer.breaker != nil {
				writer.breaker.RecordSuccess()
			}
			break
		}
		if writer.breaker != nil {
			writer.breaker.RecordFailure()
		}
		writer.stats.chFailures.Add(1)
		writer.logger.Error(
			"ClickHouse raw event batch insert failed",
			"reason", reason,
			"batch_size", len(batch),
			"attempt", attempt,
			"error", chErr,
		)
		if errors.Is(chErr, context.Canceled) || errors.Is(chErr, context.DeadlineExceeded) {
			return
		}
		if attempt >= writer.cfg.MaxRetries {
			break
		}
		if err := sleepCtx(ctx, delay); err != nil {
			return
		}
		delay *= 2
		if delay > writer.cfg.RetryMaxDelay {
			delay = writer.cfg.RetryMaxDelay
		}
	}
	if chErr != nil {
		writer.stats.dropCount.Add(int64(len(batch)))
		writer.logger.Error(
			"buffered raw writer dropping batch after max retries",
			"reason", reason,
			"batch_size", len(batch),
			"drop_count", writer.stats.dropCount.Load(),
		)
		return
	}

	items := make([]domain.RawEventQueueItem, 0, len(batch))
	for _, event := range batch {
		items = append(items, domain.RawEventQueueItem{
			RawEventID:    event.ID,
			ChannelID:     event.ChannelID,
			ChatroomID:    event.ChatroomID,
			ChannelSlug:   event.ChannelSlug,
			KickMessageID: event.KickMessageID,
			EnqueuedAt:    event.ReceivedAt,
		})
	}
	delay = writer.cfg.RetryInitialDelay
	for attempt := 1; ; attempt++ {
		err := writer.queue.EnqueueBatch(ctx, items)
		if err == nil {
			break
		}
		writer.stats.queueFailures.Add(1)
		writer.logger.Error(
			"SQLite raw event queue enqueue failed after CH archive",
			"reason", reason,
			"batch_size", len(batch),
			"attempt", attempt,
			"error", err,
		)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		if err := sleepCtx(ctx, delay); err != nil {
			return
		}
		delay *= 2
		if delay > writer.cfg.RetryMaxDelay {
			delay = writer.cfg.RetryMaxDelay
		}
	}

	writer.stats.flushCount.Add(1)
	writer.stats.lastFlushSize.Store(int64(len(batch)))
	writer.stats.lastFlushNs.Store(time.Since(start).Nanoseconds())
}

func (writer *bufferedRawWriter) Wait() {
	<-writer.done
}

func (writer *bufferedRawWriter) Stats() BufferedWriterStats {
	return BufferedWriterStats{
		QueueDepth:         writer.stats.depth.Load(),
		QueueHighWaterMark: writer.stats.highWaterMark.Load(),
		DropCount:          writer.stats.dropCount.Load(),
		FlushCount:         writer.stats.flushCount.Load(),
		LastFlushSize:      writer.stats.lastFlushSize.Load(),
		LastFlushNanos:     writer.stats.lastFlushNs.Load(),
		ClickHouseFailures: writer.stats.chFailures.Load(),
		QueueEnqueueFails:  writer.stats.queueFailures.Load(),
	}
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
