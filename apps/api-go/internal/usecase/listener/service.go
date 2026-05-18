package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type Service struct {
	channels        ports.FollowedChannelRepository
	rawEvents       ports.RawEventRepository
	rawEventClaims  ports.RawEventClaimRepository
	messages        ports.MessageRepository
	senders         ports.SenderProfileRepository
	heartbeats      ports.WorkerHeartbeatRepository
	channelResolver ports.KickChannelResolver
	senderResolver  ports.KickSenderProfileResolver
	pusher          ports.PusherClient
	parser          EventParser
	logger          *slog.Logger
	config          ServiceConfig
}

type ServiceConfig struct {
	WorkerCount               int
	RawEventBatchSize         int
	RawEventProcessingTimeout time.Duration
	RawEventMaxAttempts       uint16
	RawEventWorkerIdleDelay   time.Duration
	ChannelResyncInterval     time.Duration
	HeartbeatInterval         time.Duration
	ReconnectInitialDelay     time.Duration
	ReconnectMaxDelay         time.Duration
	ReconnectMultiplier       float64
	HeartbeatServiceName      string
}

type Dependencies struct {
	Channels        ports.FollowedChannelRepository
	RawEvents       ports.RawEventRepository
	RawEventClaims  ports.RawEventClaimRepository
	Messages        ports.MessageRepository
	Senders         ports.SenderProfileRepository
	Heartbeats      ports.WorkerHeartbeatRepository
	ChannelResolver ports.KickChannelResolver
	SenderResolver  ports.KickSenderProfileResolver
	Pusher          ports.PusherClient
	Logger          *slog.Logger
	Config          ServiceConfig
}

type RawEventProcessingResult struct {
	Claimed      int
	Processed    int
	Failed       int
	PendingCount int64
}

var errNoEnabledChannels = errors.New("no enabled Kick channels are ready for listener subscription")

func NewService(deps Dependencies) *Service {
	cfg := deps.Config.withDefaults()
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		channels:        deps.Channels,
		rawEvents:       deps.RawEvents,
		rawEventClaims:  deps.RawEventClaims,
		messages:        deps.Messages,
		senders:         deps.Senders,
		heartbeats:      deps.Heartbeats,
		channelResolver: deps.ChannelResolver,
		senderResolver:  deps.SenderResolver,
		pusher:          deps.Pusher,
		parser:          NewEventParser(),
		logger:          logger,
		config:          cfg,
	}
}

func (service *Service) RunForever(ctx context.Context) error {
	for workerID := 1; workerID <= service.config.WorkerCount; workerID++ {
		go service.processRawEventsForever(ctx, workerID)
	}
	go service.recordHeartbeatForever(ctx)

	attempt := 1
	for ctx.Err() == nil {
		stored, err := service.RunOnce(ctx)
		delay := service.reconnectDelay(attempt)
		if errors.Is(err, errNoEnabledChannels) {
			service.logger.Info(errNoEnabledChannels.Error())
			attempt = 1
			delay = service.config.ChannelResyncInterval
		} else if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			service.logger.Error("Kick listener failed", "error", err)
			attempt++
		} else {
			service.logger.Info("Kick listener stream ended", "stored_raw_events", stored)
			attempt = 1
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return ctx.Err()
}

func (service *Service) RunOnce(ctx context.Context) (int, error) {
	channels, err := service.loadEnabledChannels(ctx)
	if err != nil {
		return 0, err
	}
	if len(channels) == 0 {
		return 0, errNoEnabledChannels
	}
	if service.pusher == nil {
		return 0, fmt.Errorf("pusher client is not configured")
	}

	listenerChannels := make([]domain.ListenerChannel, 0, len(channels))
	channelsByChatroomID := make(map[int64]domain.FollowedChannel, len(channels))
	for _, channel := range channels {
		listenerChannels = append(listenerChannels, domain.ListenerChannel{
			ID:             channel.ID,
			KickChannelID:  channel.KickChannelID,
			KickChatroomID: channel.KickChatroomID,
			Slug:           channel.Slug,
			DisplayName:    channel.DisplayName,
		})
		channelsByChatroomID[channel.KickChatroomID] = channel
	}

	resyncCtx, cancel := context.WithTimeout(ctx, service.config.ChannelResyncInterval)
	defer cancel()

	storedCount := 0
	err = service.pusher.Listen(resyncCtx, listenerChannels, func(raw string) error {
		event, ok := service.parser.Parse(raw)
		if !ok {
			return nil
		}
		chatroomID := asInt64(event.Payload["chatroom_id"])
		channel := channelsByChatroomID[chatroomID]
		if channel.ID == 0 {
			var err error
			channel, err = service.channels.GetByChatroomID(ctx, chatroomID)
			if err != nil {
				return fmt.Errorf("resolve raw event channel: %w", err)
			}
		}
		rawEvent := domain.RawKickEvent{
			ID:            uuid.NewString(),
			ChannelSlug:   channel.Slug,
			EventType:     "pusher",
			EventName:     event.EventName,
			KickMessageID: cleanText(event.Payload["id"]),
			ChatroomID:    chatroomID,
			ChannelID:     channel.ID,
			PayloadJSON:   rawPayloadJSON(event.Payload),
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		}
		if err := service.rawEvents.InsertEvent(ctx, rawEvent); err != nil {
			return err
		}
		storedCount++
		return nil
	})
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return storedCount, nil
	}
	return storedCount, err
}

func (service *Service) ProcessRawEventsOnce(ctx context.Context) (RawEventProcessingResult, error) {
	return service.processRawEventsOnce(ctx, 0)
}

func (service *Service) processRawEventsOnce(ctx context.Context, workerID int) (RawEventProcessingResult, error) {
	if service.rawEventClaims == nil {
		return RawEventProcessingResult{}, errors.New("raw event claim repository is not configured")
	}
	events, err := service.rawEvents.ListUnprocessed(
		ctx,
		uint64(service.config.RawEventBatchSize),
		service.config.RawEventMaxAttempts,
	)
	if err != nil {
		return RawEventProcessingResult{}, err
	}

	result := RawEventProcessingResult{}
	seenRawEventIDs := make(map[string]bool, len(events))
	for _, rawEvent := range events {
		if seenRawEventIDs[rawEvent.ID] {
			continue
		}
		seenRawEventIDs[rawEvent.ID] = true
		if rawEvent.Attempts >= service.config.RawEventMaxAttempts {
			continue
		}
		claimWorkerID := service.rawEventClaimWorkerID(workerID)
		claimed, err := service.rawEventClaims.TryClaim(
			ctx,
			rawEvent.ID,
			claimWorkerID,
			service.config.RawEventProcessingTimeout,
		)
		if err != nil {
			return RawEventProcessingResult{}, err
		}
		if !claimed {
			continue
		}
		result.Claimed++
		if err := service.processRawEvent(ctx, rawEvent); err != nil {
			result.Failed++
			if releaseErr := service.rawEventClaims.Release(ctx, rawEvent.ID, claimWorkerID); releaseErr != nil {
				service.logger.Error("failed to release raw Kick event claim", "raw_event_id", rawEvent.ID, "error", releaseErr)
			}
			service.logger.Error("Raw Kick event processing failed", "raw_event_id", rawEvent.ID, "error", err)
			continue
		}
		if err := service.rawEventClaims.MarkCompleted(ctx, rawEvent.ID, claimWorkerID); err != nil {
			return RawEventProcessingResult{}, err
		}
		result.Processed++
	}

	pendingCount, err := service.rawEvents.CountUnprocessed(ctx, service.config.RawEventMaxAttempts)
	if err != nil {
		return RawEventProcessingResult{}, err
	}
	result.PendingCount = pendingCount
	return result, nil
}

func (service *Service) rawEventClaimWorkerID(workerID int) string {
	if workerID < 1 {
		return service.config.HeartbeatServiceName + "-manual"
	}
	return fmt.Sprintf("%s-%d", service.config.HeartbeatServiceName, workerID)
}

func (service *Service) RecordHeartbeat(ctx context.Context) error {
	metadata := map[string]any{
		"raw_event_worker_count":              service.config.WorkerCount,
		"raw_event_batch_size":                service.config.RawEventBatchSize,
		"channel_resync_interval_seconds":     service.config.ChannelResyncInterval.Seconds(),
		"raw_event_worker_idle_delay_seconds": service.config.RawEventWorkerIdleDelay.Seconds(),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return service.heartbeats.Upsert(ctx, domain.ListenerHeartbeat{
		ServiceName:  service.config.HeartbeatServiceName,
		LastSeenAt:   time.Now().UTC(),
		MetadataJSON: string(encoded),
	})
}

func (service *Service) loadEnabledChannels(ctx context.Context) ([]domain.FollowedChannel, error) {
	channels, err := service.channels.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	ready := make([]domain.FollowedChannel, 0, len(channels))
	for _, channel := range channels {
		if channel.KickChannelID == 0 || channel.KickChatroomID == 0 {
			if service.channelResolver == nil {
				continue
			}
			resolved, err := service.channelResolver.ResolveChannel(ctx, channel.Slug)
			if err != nil {
				service.logger.Warn("failed to resolve followed channel", "slug", channel.Slug, "error", err)
				continue
			}
			resolved.ID = channel.ID
			resolved.IsEnabled = true
			resolved, err = service.channels.Upsert(ctx, resolved)
			if err != nil {
				return nil, err
			}
			channel = resolved
		}
		if channel.ID == 0 || channel.KickChannelID == 0 || channel.KickChatroomID == 0 {
			continue
		}
		ready = append(ready, channel)
	}
	return ready, nil
}

func (service *Service) processRawEvent(ctx context.Context, rawEvent domain.RawKickEvent) error {
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawEvent.PayloadJSON), &payload); err != nil {
		return service.markRawEventFailed(ctx, rawEvent, fmt.Errorf("decode raw payload: %w", err))
	}

	kickMessageID := cleanText(payload["id"])
	if kickMessageID == "" {
		return service.markRawEventFailed(ctx, rawEvent, fmt.Errorf("raw event payload missing message id"))
	}
	exists, err := service.messages.ExistsByKickMessageID(ctx, kickMessageID)
	if err != nil {
		return service.markRawEventFailed(ctx, rawEvent, err)
	}
	if exists {
		return service.markRawEventProcessed(ctx, rawEvent)
	}

	chatroomID := rawEvent.ChatroomID
	if chatroomID == 0 {
		chatroomID = asInt64(payload["chatroom_id"])
	}
	channel, err := service.channels.GetByChatroomID(ctx, chatroomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = fmt.Errorf("message channel is not followed")
		}
		return service.markRawEventFailed(ctx, rawEvent, err)
	}

	sender, err := senderProfileFromPayload(payload)
	if err != nil {
		return service.markRawEventFailed(ctx, rawEvent, err)
	}
	if service.senderResolver != nil && sender.Slug != "" {
		if resolved, err := service.senderResolver.ResolveSender(ctx, sender.Slug); err == nil {
			sender.Slug = resolved.Slug
			if resolved.Username != "" {
				sender.Username = resolved.Username
			}
			if resolved.ProfileImageURL != "" {
				sender.ProfileImageURL = resolved.ProfileImageURL
			}
			if resolved.RawProfilePayloadJSON != "" {
				sender.RawProfilePayloadJSON = resolved.RawProfilePayloadJSON
			}
		}
	}
	sender, err = service.senders.Upsert(ctx, sender)
	if err != nil {
		return service.markRawEventFailed(ctx, rawEvent, err)
	}

	message, err := normalizeMessagePayload(payload, channel, sender)
	if err != nil {
		return service.markRawEventFailed(ctx, rawEvent, err)
	}
	if err := service.messages.Insert(ctx, message); err != nil {
		return service.markRawEventFailed(ctx, rawEvent, err)
	}
	return service.markRawEventProcessed(ctx, rawEvent)
}

func (service *Service) markRawEventProcessed(ctx context.Context, rawEvent domain.RawKickEvent) error {
	attempt := rawEvent.Attempts + 1
	return service.rawEvents.InsertAttempt(ctx, domain.RawEventAttempt{
		RawEventID: rawEvent.ID,
		Attempt:    attempt,
		Status:     "processed",
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
	})
}

func (service *Service) markRawEventFailed(ctx context.Context, rawEvent domain.RawKickEvent, cause error) error {
	attempt := rawEvent.Attempts + 1
	status := "failed"
	if err := service.rawEvents.InsertAttempt(ctx, domain.RawEventAttempt{
		RawEventID:   rawEvent.ID,
		Attempt:      attempt,
		Status:       status,
		ErrorMessage: fmt.Sprintf("%T: %v", cause, cause),
		StartedAt:    time.Now().UTC(),
		FinishedAt:   time.Now().UTC(),
	}); err != nil {
		return err
	}
	return cause
}

func (service *Service) processRawEventsForever(ctx context.Context, workerID int) {
	for ctx.Err() == nil {
		result, err := service.processRawEventsOnce(ctx, workerID)
		if err != nil {
			service.logger.Error("raw Kick event worker failed", "worker_id", workerID, "error", err)
		} else if result.Claimed > 0 {
			service.logger.Info(
				"raw Kick event worker processed batch",
				"worker_id", workerID,
				"claimed", result.Claimed,
				"processed", result.Processed,
				"failed", result.Failed,
				"pending", result.PendingCount,
			)
		}
		timer := time.NewTimer(service.config.RawEventWorkerIdleDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (service *Service) recordHeartbeatForever(ctx context.Context) {
	for ctx.Err() == nil {
		if err := service.RecordHeartbeat(ctx); err != nil {
			service.logger.Error("failed to record listener heartbeat", "error", err)
		}
		timer := time.NewTimer(service.config.HeartbeatInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (service *Service) reconnectDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := service.config.ReconnectInitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * service.config.ReconnectMultiplier)
		if delay > service.config.ReconnectMaxDelay {
			return service.config.ReconnectMaxDelay
		}
	}
	return delay
}

func (cfg ServiceConfig) withDefaults() ServiceConfig {
	if cfg.WorkerCount < 0 {
		cfg.WorkerCount = 0
	}
	if cfg.RawEventBatchSize < 1 {
		cfg.RawEventBatchSize = 100
	}
	if cfg.RawEventProcessingTimeout <= 0 {
		cfg.RawEventProcessingTimeout = 300 * time.Second
	}
	if cfg.RawEventMaxAttempts == 0 {
		cfg.RawEventMaxAttempts = 5
	}
	if cfg.RawEventWorkerIdleDelay <= 0 {
		cfg.RawEventWorkerIdleDelay = 250 * time.Millisecond
	}
	if cfg.ChannelResyncInterval <= 0 {
		cfg.ChannelResyncInterval = time.Minute
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 15 * time.Second
	}
	if cfg.ReconnectInitialDelay <= 0 {
		cfg.ReconnectInitialDelay = time.Second
	}
	if cfg.ReconnectMaxDelay <= 0 {
		cfg.ReconnectMaxDelay = 30 * time.Second
	}
	if cfg.ReconnectMultiplier < 1 {
		cfg.ReconnectMultiplier = 2
	}
	if cfg.HeartbeatServiceName == "" {
		cfg.HeartbeatServiceName = "listener"
	}
	return cfg
}
