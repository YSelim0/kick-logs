package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type StreamProcessorService struct {
	stream     ports.RawEventStreamConsumer
	rawEvents  ports.RawEventRepository
	messages   ports.MessageRepository
	heartbeats ports.WorkerHeartbeatRepository
	logger     *slog.Logger
	config     StreamProcessorConfig
	core       *Service
	breaker    *CircuitBreaker
}

type StreamProcessorConfig struct {
	BatchSize                int
	IdleDelay                time.Duration
	HeartbeatInterval        time.Duration
	HeartbeatServiceName     string
	NakDelay                 time.Duration
	SenderProfileCacheTTL    time.Duration
	ClickHouseBackoffInitial time.Duration
	ClickHouseBackoffMax     time.Duration
	ClickHouseBackoffFactor  float64
	ClickHouseBreakerThresh  int
}

type StreamProcessorDependencies struct {
	Stream     ports.RawEventStreamConsumer
	RawEvents  ports.RawEventRepository
	Messages   ports.MessageRepository
	Channels   ports.FollowedChannelRepository
	Senders    ports.SenderProfileRepository
	Heartbeats ports.WorkerHeartbeatRepository
	Logger     *slog.Logger
	Config     StreamProcessorConfig
}

type StreamProcessingResult struct {
	Fetched     int
	RawInserted int
	Processed   int
	Ignored     int
	Failed      int
	Acked       int
	Nacked      int
	Termed      int
	Pending     int64
}

type streamRawEventOutcome struct {
	message     ports.RawEventStreamMessage
	rawEvent    domain.RawKickEvent
	processed   bool
	terminal    bool
	failure     error
	hasMessage  bool
	chatMessage domain.ChatMessage
}

func NewStreamProcessorService(deps StreamProcessorDependencies) *StreamProcessorService {
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
	core := NewService(Dependencies{
		Channels:   deps.Channels,
		RawEvents:  deps.RawEvents,
		Messages:   deps.Messages,
		Senders:    deps.Senders,
		Heartbeats: deps.Heartbeats,
		Logger:     logger,
		Config: ServiceConfig{
			HeartbeatServiceName:  cfg.HeartbeatServiceName,
			SenderProfileCacheTTL: cfg.SenderProfileCacheTTL,
		},
	})
	core.breaker = breaker
	return &StreamProcessorService{
		stream:     deps.Stream,
		rawEvents:  deps.RawEvents,
		messages:   deps.Messages,
		heartbeats: deps.Heartbeats,
		logger:     logger,
		config:     cfg,
		core:       core,
		breaker:    breaker,
	}
}

func (service *StreamProcessorService) RunForever(ctx context.Context) error {
	go service.recordHeartbeatForever(ctx)
	for ctx.Err() == nil {
		if service.breaker != nil {
			if err := service.breaker.Wait(ctx); err != nil {
				return err
			}
		}
		result, err := service.ProcessOnce(ctx)
		if err != nil {
			service.logger.Error("raw Kick stream processor failed", "error", err)
			if service.breaker != nil {
				service.breaker.RecordFailure()
			}
		} else {
			if service.breaker != nil {
				service.breaker.RecordSuccess()
			}
			if result.Fetched > 0 {
				service.logger.Info(
					"raw Kick stream processor handled batch",
					"fetched", result.Fetched,
					"processed", result.Processed,
					"ignored", result.Ignored,
					"failed", result.Failed,
					"acked", result.Acked,
					"nacked", result.Nacked,
					"termed", result.Termed,
					"pending", result.Pending,
				)
			}
		}
		timer := time.NewTimer(service.config.IdleDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return ctx.Err()
}

func (service *StreamProcessorService) ProcessOnce(ctx context.Context) (StreamProcessingResult, error) {
	if service.stream == nil {
		return StreamProcessingResult{}, errors.New("raw event stream consumer is not configured")
	}
	if service.rawEvents == nil {
		return StreamProcessingResult{}, errors.New("raw event repository is not configured")
	}
	if service.messages == nil {
		return StreamProcessingResult{}, errors.New("message repository is not configured")
	}

	streamMessages, err := service.stream.Fetch(ctx, service.config.BatchSize)
	if err != nil {
		return StreamProcessingResult{}, err
	}
	result := StreamProcessingResult{Fetched: len(streamMessages)}
	if len(streamMessages) == 0 {
		return result, nil
	}

	rawEvents := make([]domain.RawKickEvent, 0, len(streamMessages))
	outcomes := make([]streamRawEventOutcome, 0, len(streamMessages))
	for _, message := range streamMessages {
		rawEvent, failure := rawKickEventFromStreamMessage(message)
		outcome := streamRawEventOutcome{
			message:  message,
			rawEvent: rawEvent,
		}
		if failure != nil {
			outcome.failure = terminalRawEvent(failure)
			outcome.terminal = true
		}
		outcomes = append(outcomes, outcome)
		rawEvents = append(rawEvents, rawEvent)
	}
	if err := service.rawEvents.InsertEventsBatch(ctx, rawEvents); err != nil {
		nacked := service.nakAll(ctx, streamMessages)
		result.Nacked += nacked
		return result, fmt.Errorf("insert raw stream events batch: %w", err)
	}
	result.RawInserted = len(rawEvents)

	kickMessageIDsToCheck := make([]string, 0, len(rawEvents))
	for _, rawEvent := range rawEvents {
		if rawEvent.KickMessageID != "" {
			kickMessageIDsToCheck = append(kickMessageIDsToCheck, rawEvent.KickMessageID)
		}
	}
	existingIDs, err := service.messages.ExistingKickMessageIDs(ctx, kickMessageIDsToCheck)
	if err != nil {
		nacked := service.nakAll(ctx, streamMessages)
		result.Nacked += nacked
		return result, fmt.Errorf("batch check existing messages: %w", err)
	}

	chatMessages := make([]domain.ChatMessage, 0, len(outcomes))
	attempts := make([]domain.RawEventAttempt, 0, len(outcomes))
	seenKickMessageIDs := make(map[string]bool, len(outcomes))
	for index := range outcomes {
		outcome := &outcomes[index]
		if outcome.failure == nil {
			message, alreadyExists, normErr := service.core.prepareMessage(ctx, outcome.rawEvent, existingIDs)
			if normErr != nil {
				outcome.failure = normErr
				outcome.terminal = isTerminalRawEventError(normErr)
			} else {
				outcome.processed = true
				if !alreadyExists && !seenKickMessageIDs[message.KickMessageID] {
					seenKickMessageIDs[message.KickMessageID] = true
					outcome.hasMessage = true
					outcome.chatMessage = message
					chatMessages = append(chatMessages, message)
				}
			}
		}
		status := "processed"
		if outcome.failure != nil {
			status = "failed"
			if outcome.terminal {
				status = "ignored"
			}
		}
		attempts = append(attempts, buildStreamAttempt(outcome.rawEvent.ID, outcome.message.NumDelivered(), status, outcome.failure))
	}

	if len(chatMessages) > 0 {
		if err := service.messages.InsertMessagesBatch(ctx, chatMessages); err != nil {
			nacked := service.nakAll(ctx, streamMessages)
			result.Nacked += nacked
			return result, fmt.Errorf("insert chat messages batch: %w", err)
		}
	}
	if len(attempts) > 0 {
		if err := service.rawEvents.InsertAttemptsBatch(ctx, attempts); err != nil {
			nacked := service.nakAll(ctx, streamMessages)
			result.Nacked += nacked
			return result, fmt.Errorf("insert raw stream event attempts batch: %w", err)
		}
	}

	for _, outcome := range outcomes {
		if outcome.processed {
			if err := outcome.message.Ack(ctx); err != nil {
				return result, fmt.Errorf("ack processed raw stream event %s: %w", outcome.rawEvent.ID, err)
			}
			result.Processed++
			result.Acked++
			continue
		}
		if outcome.terminal {
			if err := outcome.message.Term(ctx, errorText(outcome.failure)); err != nil {
				return result, fmt.Errorf("term ignored raw stream event %s: %w", outcome.rawEvent.ID, err)
			}
			result.Ignored++
			result.Termed++
			service.logger.Warn("raw Kick stream event ignored", "raw_event_id", outcome.rawEvent.ID, "error", outcome.failure)
			continue
		}
		if err := outcome.message.Nak(ctx, service.config.NakDelay); err != nil {
			return result, fmt.Errorf("nak failed raw stream event %s: %w", outcome.rawEvent.ID, err)
		}
		result.Failed++
		result.Nacked++
		service.logger.Error("raw Kick stream event processing failed", "raw_event_id", outcome.rawEvent.ID, "error", outcome.failure)
	}
	if len(streamMessages) > 0 {
		result.Pending = int64(streamMessages[len(streamMessages)-1].NumPending())
	}
	return result, nil
}

func rawKickEventFromStreamMessage(message ports.RawEventStreamMessage) (domain.RawKickEvent, error) {
	var envelope domain.RawChatEventEnvelope
	if err := json.Unmarshal(message.Data(), &envelope); err != nil {
		return fallbackRawKickEventFromStreamMessage(message), fmt.Errorf("decode raw stream envelope: %w", err)
	}
	rawEvent := rawKickEventFromEnvelope(envelope)
	if rawEvent.ID == "" {
		rawEvent.ID = fallbackRawEventIDFromStreamMessage(message)
	}
	if rawEvent.ReceivedAt.IsZero() {
		rawEvent.ReceivedAt = streamMessageTime(message)
	}
	rawEvent.MetadataJSON = rawEnvelopeMetadataJSON(envelope)
	return rawEvent, nil
}

func fallbackRawKickEventFromStreamMessage(message ports.RawEventStreamMessage) domain.RawKickEvent {
	return domain.RawKickEvent{
		ID:           fallbackRawEventIDFromStreamMessage(message),
		EventType:    "jetstream",
		EventName:    "invalid_raw_stream_envelope",
		PayloadJSON:  string(message.Data()),
		MetadataJSON: rawStreamMessageMetadataJSON(message),
		Status:       "pending",
		ReceivedAt:   streamMessageTime(message),
	}
}

func fallbackRawEventIDFromStreamMessage(message ports.RawEventStreamMessage) string {
	if message.ID() != "" {
		return message.ID()
	}
	if message.StreamSequence() > 0 {
		return fmt.Sprintf("stream:%d", message.StreamSequence())
	}
	return fmt.Sprintf("stream:%d:%d", time.Now().UTC().UnixNano(), len(message.Data()))
}

func rawEnvelopeMetadataJSON(envelope domain.RawChatEventEnvelope) string {
	metadata := map[string]any{
		"pusher_channel":  envelope.PusherChannel,
		"kick_channel_id": envelope.KickChannelID,
	}
	if envelope.RawPusherJSON != "" {
		metadata["raw_pusher_json"] = envelope.RawPusherJSON
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func rawStreamMessageMetadataJSON(message ports.RawEventStreamMessage) string {
	metadata := map[string]any{
		"subject":           message.Subject(),
		"stream_sequence":   message.StreamSequence(),
		"consumer_sequence": message.ConsumerSequence(),
		"num_delivered":     message.NumDelivered(),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func streamMessageTime(message ports.RawEventStreamMessage) time.Time {
	timestamp := message.Timestamp()
	if timestamp.IsZero() {
		return time.Now().UTC()
	}
	return timestamp.UTC()
}

func buildStreamAttempt(rawEventID string, delivered uint64, status string, cause error) domain.RawEventAttempt {
	attemptNumber := delivered
	if attemptNumber == 0 {
		attemptNumber = 1
	}
	if attemptNumber > uint64(^uint16(0)) {
		attemptNumber = uint64(^uint16(0))
	}
	attempt := domain.RawEventAttempt{
		RawEventID: rawEventID,
		Attempt:    uint16(attemptNumber),
		Status:     status,
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
	}
	if cause != nil {
		attempt.ErrorMessage = fmt.Sprintf("%T: %v", cause, cause)
	}
	return attempt
}

func (service *StreamProcessorService) nakAll(ctx context.Context, messages []ports.RawEventStreamMessage) int {
	nacked := 0
	for _, message := range messages {
		if err := message.Nak(ctx, service.config.NakDelay); err != nil {
			service.logger.Error("failed to nak raw stream event", "stream_sequence", message.StreamSequence(), "error", err)
			continue
		}
		nacked++
	}
	return nacked
}

func (service *StreamProcessorService) RecordHeartbeat(ctx context.Context) error {
	metadata := map[string]any{
		"stream_batch_size":         service.config.BatchSize,
		"stream_idle_delay_seconds": service.config.IdleDelay.Seconds(),
		"stream_nak_delay_seconds":  service.config.NakDelay.Seconds(),
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

func (service *StreamProcessorService) recordHeartbeatForever(ctx context.Context) {
	if service.heartbeats == nil {
		return
	}
	for ctx.Err() == nil {
		if err := service.RecordHeartbeat(ctx); err != nil {
			service.logger.Error("failed to record processor heartbeat", "error", err)
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

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (cfg StreamProcessorConfig) withDefaults() StreamProcessorConfig {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.IdleDelay <= 0 {
		cfg.IdleDelay = 250 * time.Millisecond
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 15 * time.Second
	}
	if cfg.HeartbeatServiceName == "" {
		cfg.HeartbeatServiceName = "processor"
	}
	if cfg.NakDelay <= 0 {
		cfg.NakDelay = time.Second
	}
	if cfg.SenderProfileCacheTTL <= 0 {
		cfg.SenderProfileCacheTTL = 10 * time.Minute
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
	return cfg
}
