// Command loadgen drives the Kick listener with a synthetic chat-event stream
// to exercise the buffered writer, SQLite work queue, batched worker output,
// and ClickHouse circuit breaker introduced in GitHub issue #9.
//
// Usage example:
//
//	go run ./cmd/loadgen -events-per-second=2000 -duration=60s -channels=5
//
// Loadgen wires the same listener.Service used by cmd/listener but replaces the
// real Pusher client with a deterministic event emitter. ClickHouse and SQLite
// migrations are applied at startup; followed channels listed under -channels
// are upserted as enabled channels with synthetic Kick ids.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/app"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	listenerusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/listener"
)

func main() {
	eventsPerSecond := flag.Int("events-per-second", 500, "synthetic events per second across all channels")
	duration := flag.Duration("duration", 30*time.Second, "load test duration")
	channelCount := flag.Int("channels", 3, "number of synthetic followed channels")
	burstFactor := flag.Float64("burst-factor", 1.0, "multiplier applied to the configured rate during the second half of the run")
	reportEvery := flag.Duration("report-every", 5*time.Second, "interval between metrics snapshots")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := app.NewLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sqliteDB, err := sqliteinfra.Open(ctx, cfg.SQLitePath)
	if err != nil {
		logger.Error("open sqlite", "error", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	if err := migrations.ApplySQLite(ctx, sqliteDB); err != nil {
		logger.Error("apply sqlite migrations", "error", err)
		os.Exit(1)
	}

	clickHouseConn, err := clickhouseinfra.Open(ctx, cfg)
	if err != nil {
		logger.Error("open clickhouse", "error", err)
		os.Exit(1)
	}
	if err := migrations.ApplyClickHouse(ctx, clickHouseConn); err != nil {
		logger.Error("apply clickhouse migrations", "error", err)
		os.Exit(1)
	}

	channels := make([]domain.FollowedChannel, 0, *channelCount)
	channelRepo := sqliteinfra.NewFollowedChannelRepository(sqliteDB)
	for i := 0; i < *channelCount; i++ {
		slug := fmt.Sprintf("loadgen-%d", i+1)
		channel := domain.FollowedChannel{
			KickChannelID:  1_000_000 + int64(i),
			KickChatroomID: 2_000_000 + int64(i),
			Slug:           slug,
			DisplayName:    "Loadgen " + slug,
			IsEnabled:      true,
			RawPayloadJSON: "{}",
			LastResolvedAt: time.Now().UTC(),
		}
		stored, err := channelRepo.Upsert(ctx, channel)
		if err != nil {
			logger.Error("upsert loadgen channel", "slug", slug, "error", err)
			os.Exit(1)
		}
		channels = append(channels, stored)
	}

	emitter := newSyntheticPusher(*eventsPerSecond, *duration, *burstFactor, channels, logger)

	service := listenerusecase.NewService(listenerusecase.Dependencies{
		Channels:        channelRepo,
		RawEvents:       clickhouseinfra.NewRawEventRepository(clickHouseConn),
		Queue:           sqliteinfra.NewRawEventQueueRepository(sqliteDB),
		Messages:        clickhouseinfra.NewMessageRepository(clickHouseConn),
		Senders:         sqliteinfra.NewSenderProfileRepository(sqliteDB),
		Heartbeats:      sqliteinfra.NewWorkerHeartbeatRepository(sqliteDB),
		ChannelResolver: passthroughChannelResolver{},
		SenderResolver:  noopSenderResolver{},
		Pusher:          emitter,
		Logger:          logger,
		Config: listenerusecase.ServiceConfig{
			WorkerCount:               cfg.ListenerWorkerCount,
			RawEventBatchSize:         cfg.ListenerRawEventBatchSize,
			RawEventProcessingTimeout: time.Duration(cfg.ListenerRawEventProcessingTimeout) * time.Second,
			RawEventMaxAttempts:       uint16(cfg.ListenerRawEventMaxAttempts),
			RawEventWorkerIdleDelay:   durationFromSeconds(cfg.ListenerRawEventWorkerIdleDelay),
			ChannelResyncInterval:     *duration + 5*time.Second,
			HeartbeatInterval:         durationFromSeconds(cfg.ListenerHeartbeatInterval),
			ReconnectInitialDelay:     durationFromSeconds(cfg.ListenerReconnectInitialDelaySeconds),
			ReconnectMaxDelay:         durationFromSeconds(cfg.ListenerReconnectMaxDelaySeconds),
			ReconnectMultiplier:       cfg.ListenerReconnectMultiplier,
			HeartbeatServiceName:      "loadgen",
			WriteBatchSize:            cfg.ListenerRawEventWriteBatchSize,
			WriteFlushInterval:        time.Duration(cfg.ListenerRawEventWriteFlushIntervalMS) * time.Millisecond,
			WriteQueueSize:            cfg.ListenerRawEventWriteQueueSize,
			WriteMaxRetries:           cfg.ListenerRawEventWriteMaxRetries,
			ClickHouseBackoffInitial:  time.Duration(cfg.ListenerClickHouseBackoffInitialMS) * time.Millisecond,
			ClickHouseBackoffMax:      time.Duration(cfg.ListenerClickHouseBackoffMaxMS) * time.Millisecond,
			ClickHouseBackoffFactor:   cfg.ListenerClickHouseBackoffMultiplier,
			ClickHouseBreakerThresh:   cfg.ListenerClickHouseBreakerThreshold,
		},
	})

	logger.Info(
		"loadgen starting",
		"events_per_second", *eventsPerSecond,
		"duration", duration.String(),
		"channels", *channelCount,
		"burst_factor", *burstFactor,
	)

	runCtx, cancelRun := context.WithTimeout(ctx, *duration+10*time.Second)
	defer cancelRun()

	go reportStats(runCtx, service, emitter, *reportEvery, logger)

	if err := service.RunForever(runCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		logger.Error("loadgen run failed", "error", err)
		os.Exit(1)
	}

	final := service.WriterStats()
	logger.Info(
		"loadgen finished",
		"emitted", emitter.Emitted(),
		"writer_queue_depth", final.QueueDepth,
		"writer_high_water_mark", final.QueueHighWaterMark,
		"writer_drops", final.DropCount,
		"writer_flushes", final.FlushCount,
		"clickhouse_failures", final.ClickHouseFailures,
		"sqlite_enqueue_failures", final.QueueEnqueueFails,
	)
}

func reportStats(
	ctx context.Context,
	service *listenerusecase.Service,
	emitter *syntheticPusher,
	interval time.Duration,
	logger *slog.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := service.WriterStats()
			logger.Info(
				"loadgen snapshot",
				"emitted", emitter.Emitted(),
				"writer_queue_depth", stats.QueueDepth,
				"writer_drops", stats.DropCount,
				"writer_flushes", stats.FlushCount,
				"last_flush_size", stats.LastFlushSize,
				"last_flush_ms", stats.LastFlushNanos/int64(time.Millisecond),
				"clickhouse_failures", stats.ClickHouseFailures,
			)
		}
	}
}

func durationFromSeconds(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

type syntheticPusher struct {
	eventsPerSecond int
	duration        time.Duration
	burstFactor     float64
	channels        []domain.FollowedChannel
	logger          *slog.Logger
	emitted         atomic.Int64
}

func newSyntheticPusher(eventsPerSecond int, duration time.Duration, burstFactor float64, channels []domain.FollowedChannel, logger *slog.Logger) *syntheticPusher {
	if eventsPerSecond < 1 {
		eventsPerSecond = 1
	}
	if burstFactor < 1 {
		burstFactor = 1
	}
	return &syntheticPusher{
		eventsPerSecond: eventsPerSecond,
		duration:        duration,
		burstFactor:     burstFactor,
		channels:        channels,
		logger:          logger,
	}
}

func (emitter *syntheticPusher) Emitted() int64 {
	return emitter.emitted.Load()
}

func (emitter *syntheticPusher) Listen(ctx context.Context, _ []domain.ListenerChannel, handle func(string) error) error {
	start := time.Now()
	deadline := start.Add(emitter.duration)
	half := start.Add(emitter.duration / 2)

	tick := time.Second / time.Duration(emitter.eventsPerSecond)
	if tick <= 0 {
		tick = time.Microsecond
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if now.After(deadline) {
				return nil
			}
			eventCount := 1
			if emitter.burstFactor > 1 && now.After(half) {
				eventCount = int(emitter.burstFactor)
			}
			for i := 0; i < eventCount; i++ {
				channel := emitter.channels[(emitter.emitted.Load())%int64(len(emitter.channels))]
				event := emitter.buildPayload(channel)
				if err := handle(event); err != nil {
					return err
				}
				emitter.emitted.Add(1)
			}
		}
	}
}

func (emitter *syntheticPusher) buildPayload(channel domain.FollowedChannel) string {
	id := emitter.emitted.Load() + 1
	idStr := strconv.FormatInt(id, 10)
	payload := map[string]any{
		"id":          "loadgen-" + idStr,
		"chatroom_id": channel.KickChatroomID,
		"content":     "loadgen synthetic message " + idStr,
		"type":        "message",
		"created_at":  time.Now().UTC().Format(time.RFC3339Nano),
		"sender": map[string]any{
			"id":       1_000_000 + (id % 1000),
			"username": "loadgen_user_" + idStr,
			"slug":     "loadgen-user-" + idStr,
			"identity": map[string]any{
				"color":  "#FFF600",
				"badges": []any{},
			},
		},
		"metadata": map[string]any{},
	}
	data, _ := json.Marshal(payload)
	envelope := map[string]any{
		"event":   "App\\Events\\ChatMessageEvent",
		"channel": fmt.Sprintf("chatrooms.%d.v2", channel.KickChatroomID),
		"data":    string(data),
	}
	encoded, _ := json.Marshal(envelope)
	return string(encoded)
}

type passthroughChannelResolver struct{}

func (passthroughChannelResolver) ResolveChannel(_ context.Context, slug string) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{
		Slug:        slug,
		DisplayName: strings.ToUpper(slug),
	}, nil
}

type noopSenderResolver struct{}

func (noopSenderResolver) ResolveSender(_ context.Context, _ string) (domain.SenderProfile, error) {
	return domain.SenderProfile{}, errors.New("noop")
}
