package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type Service struct {
	channels        ports.FollowedChannelRepository
	rawEvents       ports.RawEventRepository
	queue           ports.RawEventQueueRepository
	streamPublisher ports.RawEventStreamPublisher
	messages        ports.MessageRepository
	senders         ports.SenderProfileRepository
	heartbeats      ports.WorkerHeartbeatRepository
	channelResolver ports.KickChannelResolver
	senderResolver  ports.KickSenderProfileResolver
	pusher          ports.PusherClient
	parser          EventParser
	logger          *slog.Logger
	config          ServiceConfig
	writer          *bufferedRawWriter
	breaker         *CircuitBreaker
	senderCacheGate *senderProfileWriteGate
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
	WriteBatchSize            int
	WriteFlushInterval        time.Duration
	WriteQueueSize            int
	WriteMaxRetries           int
	BootstrapRawQueueOnStart  bool
	ClickHouseBackoffInitial  time.Duration
	ClickHouseBackoffMax      time.Duration
	ClickHouseBackoffFactor   float64
	ClickHouseBreakerThresh   int
	SenderProfileCacheTTL     time.Duration
}

type Dependencies struct {
	Channels        ports.FollowedChannelRepository
	RawEvents       ports.RawEventRepository
	Queue           ports.RawEventQueueRepository
	StreamPublisher ports.RawEventStreamPublisher
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
	Ignored      int
	Failed       int
	PendingCount int64
}

var errNoEnabledChannels = errors.New("no enabled Kick channels are ready for listener subscription")
var errChannelSetChanged = errors.New("enabled Kick channel set changed")

func NewService(deps Dependencies) *Service {
	cfg := deps.Config.withDefaults()
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	breaker := NewCircuitBreaker(
		"clickhouse",
		cfg.ClickHouseBreakerThresh,
		NewBackoff(cfg.ClickHouseBackoffInitial, cfg.ClickHouseBackoffMax, cfg.ClickHouseBackoffFactor),
		logger,
	)
	service := &Service{
		channels:        deps.Channels,
		rawEvents:       deps.RawEvents,
		queue:           deps.Queue,
		streamPublisher: deps.StreamPublisher,
		messages:        deps.Messages,
		senders:         deps.Senders,
		heartbeats:      deps.Heartbeats,
		channelResolver: deps.ChannelResolver,
		senderResolver:  deps.SenderResolver,
		pusher:          deps.Pusher,
		parser:          NewEventParser(),
		logger:          logger,
		config:          cfg,
		breaker:         breaker,
		senderCacheGate: newSenderProfileWriteGate(cfg.SenderProfileCacheTTL),
	}
	if deps.StreamPublisher == nil && deps.RawEvents != nil && deps.Queue != nil {
		service.writer = newBufferedRawWriter(BufferedWriterConfig{
			BatchSize:     cfg.WriteBatchSize,
			FlushInterval: cfg.WriteFlushInterval,
			QueueSize:     cfg.WriteQueueSize,
			MaxRetries:    cfg.WriteMaxRetries,
		}, deps.RawEvents, deps.Queue, logger)
		service.writer.breaker = breaker
	}
	return service
}

func (service *Service) RunForever(ctx context.Context) error {
	if service.writer != nil {
		go service.writer.Run(ctx)
	}
	if service.usesLegacyRawEventQueue() {
		if service.config.BootstrapRawQueueOnStart {
			if err := service.bootstrapQueue(ctx); err != nil && !errors.Is(err, context.Canceled) {
				service.logger.Error("raw event queue bootstrap failed", "error", err)
			}
		} else {
			service.logger.Info("raw event queue bootstrap skipped")
		}
		if recovered, err := service.queue.RecoverStaleClaims(ctx, service.config.RawEventProcessingTimeout); err != nil {
			service.logger.Error("raw event stale claim recovery failed", "error", err)
		} else if recovered > 0 {
			service.logger.Info("recovered stale raw event claims at startup", "count", recovered)
		}

		for workerID := 1; workerID <= service.config.WorkerCount; workerID++ {
			go service.processRawEventsForever(ctx, workerID)
		}
		go service.recoverStaleClaimsForever(ctx)
	} else if service.streamPublisher != nil {
		service.logger.Info("legacy raw event queue disabled; listener publishes to JetStream")
	}
	go service.recordHeartbeatForever(ctx)

	attempt := 1
	for ctx.Err() == nil {
		stored, err := service.RunOnce(ctx)
		delay := service.reconnectDelay(attempt)
		if errors.Is(err, errChannelSetChanged) {
			service.logger.Info("Kick listener channel set changed; reconnecting stream", "stored_raw_events", stored)
			attempt = 1
			delay = 0
		} else if errors.Is(err, errNoEnabledChannels) {
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

	listenerChannels, channelsByChatroomID := listenerChannelsFromFollowed(channels)

	listenCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	channelSetChanged := service.watchChannelSetChanges(
		listenCtx,
		cancel,
		listenerChannelSignature(listenerChannels),
	)

	storedCount := 0
	err = service.pusher.Listen(listenCtx, listenerChannels, func(raw string) error {
		event, ok := service.parser.Parse(raw)
		if !ok {
			return nil
		}
		chatroomID := asInt64(event.Payload["chatroom_id"])
		if chatroomID == 0 {
			chatroomID = chatroomIDFromPusherChannel(event.PusherChannel)
		}
		channel := channelsByChatroomID[chatroomID]
		if channel.ID == 0 && chatroomID > 0 {
			var err error
			channel, err = service.channels.GetByChatroomID(ctx, chatroomID)
			if err != nil {
				service.logger.Warn(
					"failed to resolve raw event channel; capturing payload without channel metadata",
					"chatroom_id", chatroomID,
					"error", err,
				)
			}
		}
		receivedAt := time.Now().UTC()
		envelope := rawChatEventEnvelopeFromEvent(event, channel, chatroomID, receivedAt)
		rawEvent := rawKickEventFromEnvelope(envelope)
		if service.streamPublisher != nil {
			streamEvent, err := rawStreamEventFromEnvelope(envelope)
			if err != nil {
				return err
			}
			if _, err := service.streamPublisher.Publish(ctx, streamEvent); err != nil {
				return fmt.Errorf("publish raw event stream: %w", err)
			}
		} else if service.writer != nil {
			service.writer.Submit(rawEvent)
		} else {
			if service.rawEvents == nil {
				return fmt.Errorf("raw event repository is not configured")
			}
			if err := service.rawEvents.InsertEvent(ctx, rawEvent); err != nil {
				return err
			}
			if service.queue != nil {
				if err := service.queue.Enqueue(ctx, domain.RawEventQueueItem{
					RawEventID:    rawEvent.ID,
					ChannelID:     rawEvent.ChannelID,
					ChatroomID:    rawEvent.ChatroomID,
					ChannelSlug:   rawEvent.ChannelSlug,
					KickMessageID: rawEvent.KickMessageID,
					EnqueuedAt:    rawEvent.ReceivedAt,
				}); err != nil {
					return fmt.Errorf("enqueue raw event: %w", err)
				}
			}
		}
		storedCount++
		return nil
	})
	if errors.Is(err, context.Canceled) && ctx.Err() == nil {
		select {
		case <-channelSetChanged:
			return storedCount, errChannelSetChanged
		default:
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return storedCount, nil
	}
	return storedCount, err
}

func (service *Service) usesLegacyRawEventQueue() bool {
	return service.streamPublisher == nil &&
		service.queue != nil &&
		service.rawEvents != nil &&
		service.messages != nil
}

func listenerChannelsFromFollowed(
	channels []domain.FollowedChannel,
) ([]domain.ListenerChannel, map[int64]domain.FollowedChannel) {
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
	return listenerChannels, channelsByChatroomID
}

func rawEventIDFromChatEvent(event ChatMessageEvent, receivedAt time.Time) string {
	kickMessageID := cleanText(event.Payload["id"])
	if kickMessageID != "" {
		return "kick:" + kickMessageID
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(event.EventName))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(event.PusherChannel))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(event.RawJSON))
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(receivedAt.UTC().Format(time.RFC3339Nano)))
	return fmt.Sprintf("raw:%x", hash.Sum64())
}

func rawChatEventEnvelopeFromEvent(
	event ChatMessageEvent,
	channel domain.FollowedChannel,
	chatroomID int64,
	receivedAt time.Time,
) domain.RawChatEventEnvelope {
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	return domain.RawChatEventEnvelope{
		RawEventID:        rawEventIDFromChatEvent(event, receivedAt),
		KickMessageID:     cleanText(event.Payload["id"]),
		EventName:         event.EventName,
		PusherChannel:     event.PusherChannel,
		FollowedChannelID: channel.ID,
		ChannelSlug:       channel.Slug,
		KickChannelID:     channel.KickChannelID,
		KickChatroomID:    chatroomID,
		ReceivedAt:        receivedAt.UTC(),
		PayloadJSON:       rawPayloadJSON(event.Payload),
		RawPusherJSON:     event.RawJSON,
	}
}

func rawKickEventFromEnvelope(envelope domain.RawChatEventEnvelope) domain.RawKickEvent {
	return domain.RawKickEvent{
		ID:            envelope.RawEventID,
		ChannelSlug:   envelope.ChannelSlug,
		EventType:     "pusher",
		EventName:     envelope.EventName,
		KickMessageID: envelope.KickMessageID,
		ChatroomID:    envelope.KickChatroomID,
		ChannelID:     envelope.FollowedChannelID,
		PayloadJSON:   envelope.PayloadJSON,
		Status:        "pending",
		ReceivedAt:    envelope.ReceivedAt.UTC(),
	}
}

func rawStreamEventFromEnvelope(envelope domain.RawChatEventEnvelope) (domain.RawStreamEvent, error) {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return domain.RawStreamEvent{}, fmt.Errorf("encode raw stream event envelope: %w", err)
	}
	headers := map[string]string{
		"event_name": envelope.EventName,
	}
	if envelope.KickMessageID != "" {
		headers["kick_message_id"] = envelope.KickMessageID
	}
	if envelope.ChannelSlug != "" {
		headers["channel_slug"] = envelope.ChannelSlug
	}
	return domain.RawStreamEvent{
		ID:      envelope.RawEventID,
		Payload: payload,
		Headers: headers,
	}, nil
}

func listenerChannelSignature(channels []domain.ListenerChannel) string {
	parts := make([]string, 0, len(channels))
	for _, channel := range channels {
		parts = append(parts, fmt.Sprintf(
			"%d:%d:%d:%s",
			channel.ID,
			channel.KickChannelID,
			channel.KickChatroomID,
			channel.Slug,
		))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func (service *Service) watchChannelSetChanges(
	ctx context.Context,
	cancel context.CancelFunc,
	initialSignature string,
) <-chan struct{} {
	changed := make(chan struct{})
	go func() {
		timer := time.NewTicker(service.config.ChannelResyncInterval)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}

			channels, err := service.loadEnabledChannels(ctx)
			if err != nil {
				service.logger.Warn("failed to check Kick listener channel set", "error", err)
				continue
			}
			nextChannels, _ := listenerChannelsFromFollowed(channels)
			if listenerChannelSignature(nextChannels) == initialSignature {
				continue
			}

			close(changed)
			cancel()
			return
		}
	}()
	return changed
}

func (service *Service) ProcessRawEventsOnce(ctx context.Context) (RawEventProcessingResult, error) {
	return service.processRawEventsOnce(ctx, 0)
}

type rawEventOutcome struct {
	item       domain.RawEventQueueItem
	processed  bool
	terminal   bool
	message    domain.ChatMessage
	hasMessage bool
	failure    error
}

func (service *Service) processRawEventsOnce(ctx context.Context, workerID int) (RawEventProcessingResult, error) {
	if service.queue == nil {
		return RawEventProcessingResult{}, errors.New("raw event queue repository is not configured")
	}
	items, err := service.queue.ListPending(
		ctx,
		uint64(service.config.RawEventBatchSize),
		service.config.RawEventMaxAttempts,
	)
	if err != nil {
		return RawEventProcessingResult{}, err
	}

	result := RawEventProcessingResult{}
	claimWorkerID := service.rawEventClaimWorkerID(workerID)

	claimed := make([]domain.RawEventQueueItem, 0, len(items))
	for _, item := range items {
		ok, err := service.queue.Claim(ctx, item.RawEventID, claimWorkerID)
		if err != nil {
			service.releaseClaims(ctx, claimed, claimWorkerID)
			return RawEventProcessingResult{}, err
		}
		if !ok {
			continue
		}
		claimed = append(claimed, item)
	}
	result.Claimed = len(claimed)

	rawEventIDs := make([]string, 0, len(claimed))
	for _, item := range claimed {
		rawEventIDs = append(rawEventIDs, item.RawEventID)
	}
	rawEventsMap, err := service.rawEvents.GetByIDs(ctx, rawEventIDs)
	if err != nil {
		service.releaseClaims(ctx, claimed, claimWorkerID)
		return RawEventProcessingResult{}, fmt.Errorf("batch load raw events: %w", err)
	}

	kickMessageIDsToCheck := make([]string, 0, len(claimed))
	for _, item := range claimed {
		if re, ok := rawEventsMap[item.RawEventID]; ok && re.KickMessageID != "" {
			kickMessageIDsToCheck = append(kickMessageIDsToCheck, re.KickMessageID)
		}
	}
	existingIDs, err := service.messages.ExistingKickMessageIDs(ctx, kickMessageIDsToCheck)
	if err != nil {
		service.releaseClaims(ctx, claimed, claimWorkerID)
		return RawEventProcessingResult{}, fmt.Errorf("batch check existing messages: %w", err)
	}

	outcomes := make([]rawEventOutcome, 0, len(claimed))
	messages := make([]domain.ChatMessage, 0, len(claimed))
	attempts := make([]domain.RawEventAttempt, 0, len(claimed))
	seenKickMessageIDs := make(map[string]bool, len(claimed))
	for _, item := range claimed {
		outcome := rawEventOutcome{item: item}
		rawEvent, ok := rawEventsMap[item.RawEventID]
		if !ok {
			outcome.failure = fmt.Errorf("load raw event: raw event not found in batch")
			outcomes = append(outcomes, outcome)
			attempts = append(attempts, buildAttempt(item, "failed", outcome.failure))
			continue
		}
		message, alreadyExists, normErr := service.prepareMessage(ctx, rawEvent, existingIDs)
		if normErr != nil {
			outcome.failure = normErr
			outcome.terminal = isTerminalRawEventError(normErr)
			outcomes = append(outcomes, outcome)
			status := "failed"
			if outcome.terminal {
				status = "ignored"
			}
			attempts = append(attempts, buildAttempt(item, status, normErr))
			continue
		}
		outcome.processed = true
		if !alreadyExists && !seenKickMessageIDs[message.KickMessageID] {
			seenKickMessageIDs[message.KickMessageID] = true
			outcome.message = message
			outcome.hasMessage = true
			messages = append(messages, message)
		}
		outcomes = append(outcomes, outcome)
		attempts = append(attempts, buildAttempt(item, "processed", nil))
	}

	if len(messages) > 0 {
		if err := service.messages.InsertMessagesBatch(ctx, messages); err != nil {
			service.releaseClaims(ctx, claimed, claimWorkerID)
			return RawEventProcessingResult{}, fmt.Errorf("insert chat messages batch: %w", err)
		}
	}
	if len(attempts) > 0 {
		if err := service.rawEvents.InsertAttemptsBatch(ctx, attempts); err != nil {
			service.logger.Error("raw event attempts batch insert failed", "batch_size", len(attempts), "error", err)
			service.releaseClaims(ctx, claimed, claimWorkerID)
			return RawEventProcessingResult{}, fmt.Errorf("insert raw event attempts batch: %w", err)
		}
	}

	for _, outcome := range outcomes {
		if outcome.processed || outcome.terminal {
			if err := service.queue.MarkProcessed(ctx, outcome.item.RawEventID); err != nil {
				service.logger.Error("failed to remove completed queue item", "raw_event_id", outcome.item.RawEventID, "error", err)
				continue
			}
			if outcome.terminal {
				result.Ignored++
				service.logger.Warn("raw Kick event ignored", "raw_event_id", outcome.item.RawEventID, "error", outcome.failure)
			} else {
				result.Processed++
			}
		} else {
			message := ""
			if outcome.failure != nil {
				message = outcome.failure.Error()
			}
			if err := service.queue.MarkFailed(ctx, outcome.item.RawEventID, message, service.config.RawEventMaxAttempts); err != nil {
				service.logger.Error("failed to mark queue item failed", "raw_event_id", outcome.item.RawEventID, "error", err)
				continue
			}
			result.Failed++
			service.logger.Error("raw Kick event processing failed", "raw_event_id", outcome.item.RawEventID, "error", outcome.failure)
		}
	}

	pendingCount, err := service.queue.CountPending(ctx, service.config.RawEventMaxAttempts)
	if err != nil {
		return RawEventProcessingResult{}, err
	}
	result.PendingCount = pendingCount
	return result, nil
}

func (service *Service) releaseClaims(ctx context.Context, items []domain.RawEventQueueItem, workerID string) {
	for _, item := range items {
		if err := service.queue.Release(ctx, item.RawEventID, workerID); err != nil {
			service.logger.Error("failed to release queue claim", "raw_event_id", item.RawEventID, "error", err)
		}
	}
}

func buildAttempt(item domain.RawEventQueueItem, status string, cause error) domain.RawEventAttempt {
	attempt := domain.RawEventAttempt{
		RawEventID: item.RawEventID,
		Attempt:    item.Attempts + 1,
		Status:     status,
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
	}
	if cause != nil {
		attempt.ErrorMessage = fmt.Sprintf("%T: %v", cause, cause)
	}
	return attempt
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
	if service.writer != nil {
		stats := service.writer.Stats()
		metadata["write_queue_depth"] = stats.QueueDepth
		metadata["write_queue_high_water_mark"] = stats.QueueHighWaterMark
		metadata["write_drop_count"] = stats.DropCount
		metadata["write_flush_count"] = stats.FlushCount
		metadata["last_flush_size"] = stats.LastFlushSize
		metadata["last_flush_millis"] = stats.LastFlushNanos / int64(time.Millisecond)
		metadata["clickhouse_insert_failures"] = stats.ClickHouseFailures
		metadata["queue_enqueue_failures"] = stats.QueueEnqueueFails
	}
	if service.breaker != nil {
		state := service.breaker.State()
		metadata["breaker_state"] = state.State
		metadata["breaker_current_delay_ms"] = state.CurrentDelay / time.Millisecond
		metadata["breaker_failures"] = state.Failures
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

type terminalRawEventError struct {
	err error
}

func (err *terminalRawEventError) Error() string {
	if err == nil || err.err == nil {
		return "terminal raw event error"
	}
	return err.err.Error()
}

func (err *terminalRawEventError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.err
}

func terminalRawEvent(err error) error {
	return &terminalRawEventError{err: err}
}

func isTerminalRawEventError(err error) bool {
	var terminal *terminalRawEventError
	return errors.As(err, &terminal)
}

func (service *Service) prepareMessage(ctx context.Context, rawEvent domain.RawKickEvent, existingIDs map[string]bool) (domain.ChatMessage, bool, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawEvent.PayloadJSON), &payload); err != nil {
		return domain.ChatMessage{}, false, terminalRawEvent(fmt.Errorf("decode raw payload: %w", err))
	}

	kickMessageID := cleanText(payload["id"])
	if kickMessageID == "" {
		return domain.ChatMessage{}, false, terminalRawEvent(errors.New("raw event payload missing message id"))
	}
	if existingIDs[kickMessageID] {
		return domain.ChatMessage{}, true, nil
	}

	chatroomID := rawEvent.ChatroomID
	if chatroomID == 0 {
		chatroomID = asInt64(payload["chatroom_id"])
	}
	channel, err := service.channels.GetByChatroomID(ctx, chatroomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ChatMessage{}, false, terminalRawEvent(errors.New("message channel is not followed"))
		}
		return domain.ChatMessage{}, false, err
	}

	sender, err := senderProfileFromPayload(payload)
	if err != nil {
		return domain.ChatMessage{}, false, terminalRawEvent(err)
	}
	if service.senders != nil && service.senderCacheGate.ShouldWrite(sender.KickUserID) {
		upserted, err := service.senders.Upsert(ctx, sender)
		if err != nil {
			service.logger.Warn(
				"sender profile cache upsert failed; continuing with payload snapshot",
				"raw_event_id", rawEvent.ID,
				"sender_kick_user_id", sender.KickUserID,
				"error", err,
			)
		} else {
			sender = upserted
		}
	}

	message, err := normalizeMessagePayload(payload, channel, sender)
	if err != nil {
		return domain.ChatMessage{}, false, terminalRawEvent(err)
	}
	return message, false, nil
}

func (service *Service) processRawEventsForever(ctx context.Context, workerID int) {
	for ctx.Err() == nil {
		if service.breaker != nil {
			if err := service.breaker.Wait(ctx); err != nil {
				return
			}
		}
		result, err := service.processRawEventsOnce(ctx, workerID)
		if err != nil {
			service.logger.Error("raw Kick event worker failed", "worker_id", workerID, "error", err)
			if service.breaker != nil {
				service.breaker.RecordFailure()
			}
		} else {
			if service.breaker != nil {
				service.breaker.RecordSuccess()
			}
			if result.Claimed > 0 {
				service.logger.Info(
					"raw Kick event worker processed batch",
					"worker_id", workerID,
					"claimed", result.Claimed,
					"processed", result.Processed,
					"ignored", result.Ignored,
					"failed", result.Failed,
					"pending", result.PendingCount,
				)
			}
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

func (service *Service) bootstrapQueue(ctx context.Context) error {
	if service.queue == nil {
		return nil
	}
	const pageSize = uint64(1000)
	maxAttempts := service.config.RawEventMaxAttempts
	for {
		events, err := service.rawEvents.ListUnprocessed(ctx, pageSize, maxAttempts)
		if err != nil {
			return fmt.Errorf("bootstrap list unprocessed: %w", err)
		}
		if len(events) == 0 {
			return nil
		}
		items := make([]domain.RawEventQueueItem, 0, len(events))
		for _, event := range events {
			items = append(items, domain.RawEventQueueItem{
				RawEventID:    event.ID,
				ChannelID:     event.ChannelID,
				ChatroomID:    event.ChatroomID,
				ChannelSlug:   event.ChannelSlug,
				KickMessageID: event.KickMessageID,
				EnqueuedAt:    event.ReceivedAt,
			})
		}
		if err := service.queue.EnqueueBatch(ctx, items); err != nil {
			return fmt.Errorf("bootstrap enqueue batch: %w", err)
		}
		service.logger.Info("bootstrapped raw event queue page", "count", len(events))
		if uint64(len(events)) < pageSize {
			return nil
		}
	}
}

func (service *Service) recoverStaleClaimsForever(ctx context.Context) {
	if service.queue == nil {
		return
	}
	interval := service.config.RawEventProcessingTimeout / 2
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}
	for ctx.Err() == nil {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if recovered, err := service.queue.RecoverStaleClaims(ctx, service.config.RawEventProcessingTimeout); err != nil {
			service.logger.Error("raw event stale claim recovery failed", "error", err)
		} else if recovered > 0 {
			service.logger.Info("recovered stale raw event claims", "count", recovered)
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

// WriterStats returns the buffered writer counters or zero when no writer is wired.
func (service *Service) WriterStats() BufferedWriterStats {
	if service.writer == nil {
		return BufferedWriterStats{}
	}
	return service.writer.Stats()
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
	if cfg.ClickHouseBackoffInitial <= 0 {
		cfg.ClickHouseBackoffInitial = time.Second
	}
	if cfg.ClickHouseBackoffMax <= 0 {
		cfg.ClickHouseBackoffMax = 30 * time.Second
	}
	if cfg.ClickHouseBackoffFactor < 1 {
		cfg.ClickHouseBackoffFactor = 2
	}
	if cfg.ClickHouseBreakerThresh < 1 {
		cfg.ClickHouseBreakerThresh = 5
	}
	if cfg.SenderProfileCacheTTL <= 0 {
		cfg.SenderProfileCacheTTL = 10 * time.Minute
	}
	return cfg
}
