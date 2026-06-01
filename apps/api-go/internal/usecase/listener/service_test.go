package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestEventParserCoversChatRepliesEmotesAndMissingFields(t *testing.T) {
	parser := NewEventParser()
	event, ok := parser.Parse(buildPusherEvent(buildPayload("message-1")))
	if !ok {
		t.Fatal("expected chat event")
	}
	if event.EventName != chatMessageEventName || event.PusherChannel != "chatrooms.123.v2" {
		t.Fatalf("event = %#v", event)
	}
	if event.Payload["content"] != "hello [emote:37226:KEKW]" {
		t.Fatalf("payload = %#v", event.Payload)
	}

	message, err := normalizeMessagePayload(event.Payload, testChannel(), domain.SenderProfile{
		ID:              10,
		KickUserID:      456,
		Username:        "Yavuz",
		Slug:            "yavuz",
		ProfileImageURL: "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("normalizeMessagePayload() error = %v", err)
	}
	if message.MessageType != "reply" ||
		message.ReplyToSender != "Other" ||
		message.ReplyToContent != "previous" ||
		message.ThreadParentID != "thread-1" {
		t.Fatalf("reply message = %#v", message)
	}
	if len(message.Emotes) != 1 || message.Emotes[0].ImageURL != "https://files.kick.com/emotes/37226/fullsize" {
		t.Fatalf("emotes = %#v", message.Emotes)
	}

	if _, ok := parser.Parse("not-json"); ok {
		t.Fatal("malformed JSON parsed")
	}
	incomplete := map[string]any{"id": "message-1"}
	if _, ok := parser.Parse(buildPusherEvent(incomplete)); ok {
		t.Fatal("incomplete payload parsed")
	}
}

func TestListenerRunOnceStoresRawEventBeforeNormalization(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{events: []string{buildPusherEvent(buildPayload("message-1"))}})

	stored, err := service.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if stored != 1 {
		t.Fatalf("stored = %d", stored)
	}
	if len(unit.rawEvents.events) != 1 {
		t.Fatalf("raw events = %#v", unit.rawEvents.events)
	}
	if len(unit.messages.messages) != 0 {
		t.Fatalf("messages should not be normalized inline = %#v", unit.messages.messages)
	}
	if unit.rawEvents.events[0].KickMessageID != "message-1" ||
		unit.rawEvents.events[0].ChatroomID != 123 ||
		unit.rawEvents.events[0].ChannelID != 1 {
		t.Fatalf("stored raw event = %#v", unit.rawEvents.events[0])
	}
	enqueued, err := unit.queue.ListPending(context.Background(), 10, 5)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(enqueued) != 1 || enqueued[0].RawEventID != unit.rawEvents.events[0].ID {
		t.Fatalf("queue did not receive enqueue = %#v", enqueued)
	}
}

func TestRawEventProcessorNormalizesAndDeduplicatesMessages(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-duplicate")
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-1",
		EventName:     chatMessageEventName,
		KickMessageID: "message-duplicate",
		ChatroomID:    123,
		ChannelID:     1,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-2",
		EventName:     chatMessageEventName,
		KickMessageID: "message-duplicate",
		ChatroomID:    123,
		ChannelID:     1,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 2 || result.Processed != 2 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.messages.messages) != 1 {
		t.Fatalf("messages = %#v", unit.messages.messages)
	}
	message := unit.messages.messages[0]
	if message.KickMessageID != "message-duplicate" ||
		message.SenderBadgesJSON == "[]" ||
		len(message.Emotes) != 1 ||
		message.ReplyMetadataJSON == "{}" {
		t.Fatalf("message = %#v", message)
	}
	if len(unit.rawEvents.attempts) != 2 || unit.rawEvents.attempts[0].Status != "processed" {
		t.Fatalf("attempts = %#v", unit.rawEvents.attempts)
	}
	if len(unit.senders.values) != 1 {
		t.Fatalf("senders = %#v", unit.senders.values)
	}
}

func TestRawEventProcessorKeepsMessageWhenSenderCacheFails(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	unit.senders.failUpsert = errors.New("sqlite busy")

	payload := buildPayload("message-cache-failure")
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-cache-failure",
		EventName:     chatMessageEventName,
		KickMessageID: "message-cache-failure",
		ChatroomID:    123,
		ChannelID:     1,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 1 || result.Processed != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.messages.messages) != 1 {
		t.Fatalf("messages = %#v", unit.messages.messages)
	}
	message := unit.messages.messages[0]
	if message.KickMessageID != "message-cache-failure" || message.SenderKickID != 456 {
		t.Fatalf("message = %#v", message)
	}
	if message.SenderID != message.SenderKickID {
		t.Fatalf("sender id = %d, want payload kick id %d", message.SenderID, message.SenderKickID)
	}
	if len(unit.rawEvents.attempts) != 1 || unit.rawEvents.attempts[0].Status != "processed" {
		t.Fatalf("attempts = %#v", unit.rawEvents.attempts)
	}
	if len(unit.senders.values) != 0 {
		t.Fatalf("sender cache should not store failed upsert = %#v", unit.senders.values)
	}
}

func TestRawEventProcessorSkipsAlreadyClaimedRawEvents(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-claimed")
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-claimed",
		EventName:     chatMessageEventName,
		KickMessageID: "message-claimed",
		ChatroomID:    123,
		ChannelID:     1,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})
	if _, err := unit.queue.Claim(context.Background(), "raw-claimed", "other-worker"); err != nil {
		t.Fatalf("Claim() pre-claim error = %v", err)
	}

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 0 || result.Processed != 0 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.messages.messages) != 0 || len(unit.rawEvents.attempts) != 0 {
		t.Fatalf("messages=%#v attempts=%#v", unit.messages.messages, unit.rawEvents.attempts)
	}
}

func TestRawEventProcessorClaimsDuplicateRowsOnce(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-same-raw")
	rawEvent := domain.RawKickEvent{
		ID:            "raw-same",
		EventName:     chatMessageEventName,
		KickMessageID: "message-same-raw",
		ChatroomID:    123,
		ChannelID:     1,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	}
	enqueueRawEvent(t, unit, rawEvent)
	enqueueRawEvent(t, unit, rawEvent)

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 1 || result.Processed != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.messages.messages) != 1 || len(unit.rawEvents.attempts) != 1 {
		t.Fatalf("messages=%#v attempts=%#v", unit.messages.messages, unit.rawEvents.attempts)
	}
	item, err := unit.queue.GetByID(context.Background(), "raw-same")
	if err != nil {
		t.Fatalf("queue GetByID() error = %v", err)
	}
	if item.Status != domain.RawEventQueueStatusProcessed {
		t.Fatalf("queue status = %q, want processed", item.Status)
	}
}

func TestRawEventProcessorDoesNotRetryDuplicateRowsInSameBatch(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-same-failing-raw")
	payload["chatroom_id"] = 999
	rawEvent := domain.RawKickEvent{
		ID:            "raw-same-failing",
		EventName:     chatMessageEventName,
		KickMessageID: "message-same-failing-raw",
		ChatroomID:    999,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	}
	enqueueRawEvent(t, unit, rawEvent)
	enqueueRawEvent(t, unit, rawEvent)

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 1 || result.Processed != 0 || result.Failed != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.rawEvents.attempts) != 1 || unit.rawEvents.attempts[0].Status != "failed" {
		t.Fatalf("attempts = %#v", unit.rawEvents.attempts)
	}
}

func TestRawEventProcessorWritesOneBatchPerTick(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	for i := 0; i < 5; i++ {
		messageID := "message-batch-" + idAt(i)
		payload := buildPayload(messageID)
		enqueueRawEvent(t, unit, domain.RawKickEvent{
			ID:            "raw-batch-" + idAt(i),
			EventName:     chatMessageEventName,
			KickMessageID: messageID,
			ChatroomID:    123,
			ChannelID:     1,
			PayloadJSON:   rawPayloadJSON(payload),
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		})
	}

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 5 || result.Processed != 5 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.messages.batchCallSizes) != 1 || unit.messages.batchCallSizes[0] != 5 {
		t.Fatalf("message batch calls = %#v", unit.messages.batchCallSizes)
	}
	if len(unit.rawEvents.attemptBatchSizes) != 1 || unit.rawEvents.attemptBatchSizes[0] != 5 {
		t.Fatalf("attempt batch calls = %#v", unit.rawEvents.attemptBatchSizes)
	}
	if unit.messages.insertCallCount != 0 {
		t.Fatalf("single-row Insert called %d times in batch path", unit.messages.insertCallCount)
	}
}

func TestRawEventProcessorMixesProcessedAndFailedInOneTick(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})

	goodPayload := buildPayload("message-good")
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-good",
		EventName:     chatMessageEventName,
		KickMessageID: "message-good",
		ChatroomID:    123,
		ChannelID:     1,
		PayloadJSON:   rawPayloadJSON(goodPayload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})

	badPayload := buildPayload("message-bad")
	badPayload["chatroom_id"] = 999
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-bad",
		EventName:     chatMessageEventName,
		KickMessageID: "message-bad",
		ChatroomID:    999,
		PayloadJSON:   rawPayloadJSON(badPayload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Claimed != 2 || result.Processed != 1 || result.Failed != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.messages.batchCallSizes) != 1 || unit.messages.batchCallSizes[0] != 1 {
		t.Fatalf("message batch calls = %#v", unit.messages.batchCallSizes)
	}
	if len(unit.rawEvents.attemptBatchSizes) != 1 || unit.rawEvents.attemptBatchSizes[0] != 2 {
		t.Fatalf("attempt batch calls = %#v", unit.rawEvents.attemptBatchSizes)
	}
	good, err := unit.queue.GetByID(context.Background(), "raw-good")
	if err != nil {
		t.Fatalf("GetByID good error = %v", err)
	}
	if good.Status != domain.RawEventQueueStatusProcessed {
		t.Fatalf("good status = %q", good.Status)
	}
	bad, err := unit.queue.GetByID(context.Background(), "raw-bad")
	if err != nil {
		t.Fatalf("GetByID bad error = %v", err)
	}
	if bad.Status != domain.RawEventQueueStatusPending || bad.Attempts != 1 {
		t.Fatalf("bad item = %#v", bad)
	}
}

func TestRawEventProcessorReleasesClaimsWhenBatchInsertFails(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	unit.messages.failBatch = errors.New("clickhouse unavailable")

	for i := 0; i < 3; i++ {
		messageID := "message-fail-" + idAt(i)
		payload := buildPayload(messageID)
		enqueueRawEvent(t, unit, domain.RawKickEvent{
			ID:            "raw-fail-batch-" + idAt(i),
			EventName:     chatMessageEventName,
			KickMessageID: messageID,
			ChatroomID:    123,
			ChannelID:     1,
			PayloadJSON:   rawPayloadJSON(payload),
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		})
	}

	_, err := service.ProcessRawEventsOnce(context.Background())
	if err == nil {
		t.Fatal("expected batch insert failure to surface as error")
	}
	for i := 0; i < 3; i++ {
		item, err := unit.queue.GetByID(context.Background(), "raw-fail-batch-"+idAt(i))
		if err != nil {
			t.Fatalf("GetByID error = %v", err)
		}
		if item.Status != domain.RawEventQueueStatusPending {
			t.Fatalf("status = %q for id %d, want pending after batch failure", item.Status, i)
		}
	}
}

func TestRawEventProcessorMarksFailuresAndHeartbeat(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-failure")
	payload["chatroom_id"] = 999
	enqueueRawEvent(t, unit, domain.RawKickEvent{
		ID:            "raw-fail",
		EventName:     chatMessageEventName,
		KickMessageID: "message-failure",
		ChatroomID:    999,
		PayloadJSON:   rawPayloadJSON(payload),
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	})

	result, err := service.ProcessRawEventsOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessRawEventsOnce() error = %v", err)
	}
	if result.Failed != 1 || len(unit.rawEvents.attempts) != 1 || unit.rawEvents.attempts[0].Status != "failed" {
		t.Fatalf("failed result=%#v attempts=%#v", result, unit.rawEvents.attempts)
	}
	if !strings.Contains(unit.rawEvents.attempts[0].ErrorMessage, "not followed") {
		t.Fatalf("error = %q", unit.rawEvents.attempts[0].ErrorMessage)
	}

	if err := service.RecordHeartbeat(context.Background()); err != nil {
		t.Fatalf("RecordHeartbeat() error = %v", err)
	}
	if unit.heartbeats.last.ServiceName != "listener" || unit.heartbeats.last.MetadataJSON == "" {
		t.Fatalf("heartbeat = %#v", unit.heartbeats.last)
	}
}

func TestListenerRunOnceReturnsAfterResyncContext(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, blockingPusherClient{})
	service.config.ChannelResyncInterval = time.Millisecond

	stored, err := service.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if stored != 0 {
		t.Fatalf("stored = %d", stored)
	}
}

func TestListenerRunOnceReportsNoEnabledChannels(t *testing.T) {
	unit := newFakeListenerUnit()
	unit.channels.channels = nil
	service := newTestService(unit, fakePusherClient{})

	stored, err := service.RunOnce(context.Background())
	if !errors.Is(err, errNoEnabledChannels) {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if stored != 0 {
		t.Fatalf("stored = %d", stored)
	}
}

func enqueueRawEvent(t *testing.T, unit *fakeListenerUnit, event domain.RawKickEvent) {
	t.Helper()
	if err := unit.rawEvents.InsertEvent(context.Background(), event); err != nil {
		t.Fatalf("InsertEvent() error = %v", err)
	}
	if err := unit.queue.Enqueue(context.Background(), domain.RawEventQueueItem{
		RawEventID:    event.ID,
		ChannelID:     event.ChannelID,
		ChatroomID:    event.ChatroomID,
		ChannelSlug:   event.ChannelSlug,
		KickMessageID: event.KickMessageID,
		EnqueuedAt:    event.ReceivedAt,
	}); err != nil {
		t.Fatalf("queue.Enqueue() error = %v", err)
	}
}

func buildPayload(messageID string) map[string]any {
	return map[string]any{
		"id":               messageID,
		"chatroom_id":      float64(123),
		"content":          "hello [emote:37226:KEKW]",
		"type":             "reply",
		"created_at":       "2026-05-10T01:02:03Z",
		"thread_parent_id": "thread-1",
		"sender": map[string]any{
			"id":       float64(456),
			"username": "Yavuz_User",
			"slug":     "yavuz_user",
			"identity": map[string]any{
				"color":  "#fff600",
				"badges": []any{map[string]any{"type": "moderator"}},
			},
		},
		"metadata": map[string]any{
			"message_ref":      "ref-1",
			"original_sender":  map[string]any{"username": "Other"},
			"original_message": map[string]any{"content": "previous"},
		},
	}
}

func buildPusherEvent(payload map[string]any) string {
	payloadJSON, _ := json.Marshal(payload)
	eventJSON, _ := json.Marshal(map[string]any{
		"event":   chatMessageEventName,
		"channel": "chatrooms.123.v2",
		"data":    string(payloadJSON),
	})
	return string(eventJSON)
}

func testChannel() domain.FollowedChannel {
	return domain.FollowedChannel{
		ID:              1,
		KickChannelID:   100,
		KickChatroomID:  123,
		Slug:            "hype",
		DisplayName:     "Hype",
		ProfileImageURL: "https://example.com/channel.png",
		BannerImageURL:  "https://example.com/banner.png",
		IsEnabled:       true,
		RawPayloadJSON:  "{}",
		LastResolvedAt:  time.Now().UTC(),
		LastMessageAt:   time.Now().UTC(),
	}
}

func newTestService(unit *fakeListenerUnit, pusher fakePusher) *Service {
	service := NewService(Dependencies{
		Channels:        unit.channels,
		RawEvents:       unit.rawEvents,
		Queue:           unit.queue,
		Messages:        unit.messages,
		Senders:         unit.senders,
		Heartbeats:      unit.heartbeats,
		ChannelResolver: fakeChannelResolver{},
		SenderResolver:  fakeSenderResolver{},
		Pusher:          pusher,
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: ServiceConfig{
			WorkerCount:               0,
			RawEventBatchSize:         10,
			RawEventMaxAttempts:       2,
			RawEventProcessingTimeout: time.Minute,
			ChannelResyncInterval:     time.Second,
			RawEventWorkerIdleDelay:   time.Millisecond,
			HeartbeatInterval:         time.Millisecond,
			ReconnectInitialDelay:     time.Millisecond,
			ReconnectMaxDelay:         time.Millisecond,
			ReconnectMultiplier:       1,
			HeartbeatServiceName:      "listener",
		},
	})
	service.writer = nil
	return service
}

type fakePusher interface {
	Listen(context.Context, []domain.ListenerChannel, func(string) error) error
}

type fakePusherClient struct {
	events []string
}

func (client fakePusherClient) Listen(ctx context.Context, _ []domain.ListenerChannel, handle func(string) error) error {
	for _, event := range client.events {
		if err := handle(event); err != nil {
			return err
		}
	}
	return ctx.Err()
}

type blockingPusherClient struct{}

func (blockingPusherClient) Listen(ctx context.Context, _ []domain.ListenerChannel, _ func(string) error) error {
	<-ctx.Done()
	return ctx.Err()
}

type fakeListenerUnit struct {
	channels   *fakeChannelRepository
	rawEvents  *fakeRawEventRepository
	queue      *fakeRawEventQueueRepository
	messages   *fakeMessageRepository
	senders    *fakeSenderRepository
	heartbeats *fakeHeartbeatRepository
}

func newFakeListenerUnit() *fakeListenerUnit {
	return &fakeListenerUnit{
		channels:   &fakeChannelRepository{channels: []domain.FollowedChannel{testChannel()}},
		rawEvents:  &fakeRawEventRepository{},
		queue:      newFakeRawEventQueueRepository(),
		messages:   &fakeMessageRepository{},
		senders:    &fakeSenderRepository{},
		heartbeats: &fakeHeartbeatRepository{},
	}
}

type fakeChannelRepository struct {
	channels []domain.FollowedChannel
}

func (repo *fakeChannelRepository) Upsert(_ context.Context, channel domain.FollowedChannel) (domain.FollowedChannel, error) {
	if channel.ID == 0 {
		channel.ID = int64(len(repo.channels) + 1)
	}
	repo.channels = append(repo.channels, channel)
	return channel, nil
}

func (repo *fakeChannelRepository) GetByID(_ context.Context, id int64) (domain.FollowedChannel, error) {
	for _, channel := range repo.channels {
		if channel.ID == id {
			return channel, nil
		}
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeChannelRepository) GetBySlug(_ context.Context, slug string) (domain.FollowedChannel, error) {
	for _, channel := range repo.channels {
		if channel.Slug == slug {
			return channel, nil
		}
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeChannelRepository) GetByChatroomID(_ context.Context, kickChatroomID int64) (domain.FollowedChannel, error) {
	for _, channel := range repo.channels {
		if channel.KickChatroomID == kickChatroomID {
			return channel, nil
		}
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeChannelRepository) GetByBroadcasterUserID(_ context.Context, broadcasterUserID int64) (domain.FollowedChannel, error) {
	for _, channel := range repo.channels {
		if channel.BroadcasterUserID == broadcasterUserID {
			return channel, nil
		}
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeChannelRepository) List(_ context.Context) ([]domain.FollowedChannel, error) {
	return repo.channels, nil
}

func (repo *fakeChannelRepository) ListEnabled(_ context.Context) ([]domain.FollowedChannel, error) {
	enabled := make([]domain.FollowedChannel, 0, len(repo.channels))
	for _, channel := range repo.channels {
		if channel.IsEnabled {
			enabled = append(enabled, channel)
		}
	}
	return enabled, nil
}

func (repo *fakeChannelRepository) Disable(_ context.Context, id int64) (domain.FollowedChannel, error) {
	for index, channel := range repo.channels {
		if channel.ID == id {
			repo.channels[index].IsEnabled = false
			return repo.channels[index], nil
		}
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

type fakeRawEventRepository struct {
	events            []domain.RawKickEvent
	attempts          []domain.RawEventAttempt
	attemptBatchSizes []int
}

func (repo *fakeRawEventRepository) InsertEvent(_ context.Context, event domain.RawKickEvent) error {
	repo.events = append(repo.events, event)
	return nil
}

func (repo *fakeRawEventRepository) GetByID(_ context.Context, rawEventID string) (domain.RawKickEvent, error) {
	for _, event := range repo.events {
		if event.ID == rawEventID {
			event.Attempts = repo.attemptsFor(event.ID)
			return event, nil
		}
	}
	return domain.RawKickEvent{}, sql.ErrNoRows
}

func (repo *fakeRawEventRepository) InsertEventsBatch(_ context.Context, events []domain.RawKickEvent) error {
	repo.events = append(repo.events, events...)
	return nil
}

func (repo *fakeRawEventRepository) InsertAttempt(_ context.Context, attempt domain.RawEventAttempt) error {
	repo.attempts = append(repo.attempts, attempt)
	return nil
}

func (repo *fakeRawEventRepository) InsertAttemptsBatch(_ context.Context, attempts []domain.RawEventAttempt) error {
	repo.attemptBatchSizes = append(repo.attemptBatchSizes, len(attempts))
	repo.attempts = append(repo.attempts, attempts...)
	return nil
}

func (repo *fakeRawEventRepository) ListUnprocessed(_ context.Context, limit uint64, maxAttempts uint16) ([]domain.RawKickEvent, error) {
	processed := make(map[string]bool)
	for _, attempt := range repo.attempts {
		if attempt.Status == "processed" {
			processed[attempt.RawEventID] = true
		}
	}
	events := make([]domain.RawKickEvent, 0)
	for _, event := range repo.events {
		if processed[event.ID] {
			continue
		}
		event.Attempts = repo.attemptsFor(event.ID)
		if event.Attempts >= maxAttempts {
			continue
		}
		events = append(events, event)
		if uint64(len(events)) >= limit {
			break
		}
	}
	return events, nil
}

func (repo *fakeRawEventRepository) CountUnprocessed(_ context.Context, maxAttempts uint16) (int64, error) {
	events, _ := repo.ListUnprocessed(context.Background(), 100, maxAttempts)
	var count int64
	for _, event := range events {
		if event.Attempts < maxAttempts {
			count++
		}
	}
	return count, nil
}

func (repo *fakeRawEventRepository) AttemptCount(_ context.Context, rawEventID string) (uint16, error) {
	return repo.attemptsFor(rawEventID), nil
}

func (repo *fakeRawEventRepository) GetByIDs(_ context.Context, rawEventIDs []string) (map[string]domain.RawKickEvent, error) {
	result := make(map[string]domain.RawKickEvent, len(rawEventIDs))
	for _, id := range rawEventIDs {
		for _, event := range repo.events {
			if event.ID == id {
				event.Attempts = repo.attemptsFor(event.ID)
				result[id] = event
				break
			}
		}
	}
	return result, nil
}

func (repo *fakeRawEventRepository) attemptsFor(rawEventID string) uint16 {
	var count uint16
	for _, attempt := range repo.attempts {
		if attempt.RawEventID == rawEventID {
			count++
		}
	}
	return count
}

type fakeRawEventQueueRepository struct {
	mu    sync.Mutex
	items map[string]*domain.RawEventQueueItem
	order []string
}

func newFakeRawEventQueueRepository() *fakeRawEventQueueRepository {
	return &fakeRawEventQueueRepository{items: make(map[string]*domain.RawEventQueueItem)}
}

func (repo *fakeRawEventQueueRepository) Enqueue(_ context.Context, item domain.RawEventQueueItem) error {
	return repo.EnqueueBatch(context.Background(), []domain.RawEventQueueItem{item})
}

func (repo *fakeRawEventQueueRepository) EnqueueBatch(_ context.Context, items []domain.RawEventQueueItem) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, item := range items {
		if _, exists := repo.items[item.RawEventID]; exists {
			continue
		}
		if item.EnqueuedAt.IsZero() {
			item.EnqueuedAt = time.Now().UTC()
		}
		item.Status = domain.RawEventQueueStatusPending
		copy := item
		repo.items[item.RawEventID] = &copy
		repo.order = append(repo.order, item.RawEventID)
	}
	return nil
}

func (repo *fakeRawEventQueueRepository) ListPending(_ context.Context, limit uint64, maxAttempts uint16) ([]domain.RawEventQueueItem, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	out := make([]domain.RawEventQueueItem, 0)
	for _, id := range repo.order {
		item := repo.items[id]
		if item == nil {
			continue
		}
		if item.Status != domain.RawEventQueueStatusPending {
			continue
		}
		if item.Attempts >= maxAttempts {
			continue
		}
		out = append(out, *item)
		if uint64(len(out)) >= limit {
			break
		}
	}
	return out, nil
}

func (repo *fakeRawEventQueueRepository) Claim(_ context.Context, rawEventID string, workerID string) (bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	item := repo.items[rawEventID]
	if item == nil || item.Status != domain.RawEventQueueStatusPending {
		return false, nil
	}
	item.Status = domain.RawEventQueueStatusClaimed
	item.ClaimedBy = workerID
	item.ClaimedAt = time.Now().UTC()
	return true, nil
}

func (repo *fakeRawEventQueueRepository) Release(_ context.Context, rawEventID string, workerID string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	item := repo.items[rawEventID]
	if item == nil || item.Status != domain.RawEventQueueStatusClaimed {
		return nil
	}
	if workerID != "" && item.ClaimedBy != workerID {
		return nil
	}
	item.Status = domain.RawEventQueueStatusPending
	item.ClaimedBy = ""
	item.ClaimedAt = time.Time{}
	return nil
}

func (repo *fakeRawEventQueueRepository) MarkProcessed(_ context.Context, rawEventID string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	item := repo.items[rawEventID]
	if item == nil {
		return nil
	}
	item.Status = domain.RawEventQueueStatusProcessed
	item.Attempts++
	item.ClaimedBy = ""
	item.ClaimedAt = time.Time{}
	item.LastError = ""
	return nil
}

func (repo *fakeRawEventQueueRepository) MarkFailed(_ context.Context, rawEventID string, errMessage string, maxAttempts uint16) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	item := repo.items[rawEventID]
	if item == nil {
		return nil
	}
	item.Attempts++
	item.LastError = errMessage
	item.ClaimedBy = ""
	item.ClaimedAt = time.Time{}
	if item.Attempts >= maxAttempts {
		item.Status = domain.RawEventQueueStatusFailed
	} else {
		item.Status = domain.RawEventQueueStatusPending
	}
	return nil
}

func (repo *fakeRawEventQueueRepository) CountPending(_ context.Context, maxAttempts uint16) (int64, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var count int64
	for _, item := range repo.items {
		if (item.Status == domain.RawEventQueueStatusPending || item.Status == domain.RawEventQueueStatusClaimed) && item.Attempts < maxAttempts {
			count++
		}
	}
	return count, nil
}

func (repo *fakeRawEventQueueRepository) OldestPendingAge(_ context.Context, maxAttempts uint16) (time.Duration, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var oldest time.Time
	for _, item := range repo.items {
		if (item.Status == domain.RawEventQueueStatusPending || item.Status == domain.RawEventQueueStatusClaimed) && item.Attempts < maxAttempts {
			if oldest.IsZero() || item.EnqueuedAt.Before(oldest) {
				oldest = item.EnqueuedAt
			}
		}
	}
	if oldest.IsZero() {
		return 0, nil
	}
	return time.Since(oldest), nil
}

func (repo *fakeRawEventQueueRepository) RecoverStaleClaims(_ context.Context, olderThan time.Duration) (int64, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	cutoff := time.Now().UTC().Add(-olderThan)
	var recovered int64
	for _, item := range repo.items {
		if item.Status == domain.RawEventQueueStatusClaimed && !item.ClaimedAt.IsZero() && !item.ClaimedAt.After(cutoff) {
			item.Status = domain.RawEventQueueStatusPending
			item.ClaimedBy = ""
			item.ClaimedAt = time.Time{}
			recovered++
		}
	}
	return recovered, nil
}

func (repo *fakeRawEventQueueRepository) GetByID(_ context.Context, rawEventID string) (domain.RawEventQueueItem, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	item := repo.items[rawEventID]
	if item == nil {
		return domain.RawEventQueueItem{}, sql.ErrNoRows
	}
	return *item, nil
}

type fakeMessageRepository struct {
	messages        []domain.ChatMessage
	batchCallSizes  []int
	insertCallCount int
	failBatch       error
}

func (repo *fakeMessageRepository) Insert(_ context.Context, message domain.ChatMessage) error {
	repo.insertCallCount++
	repo.messages = append(repo.messages, message)
	return nil
}

func (repo *fakeMessageRepository) InsertMessagesBatch(_ context.Context, messages []domain.ChatMessage) error {
	if repo.failBatch != nil {
		return repo.failBatch
	}
	repo.batchCallSizes = append(repo.batchCallSizes, len(messages))
	repo.messages = append(repo.messages, messages...)
	return nil
}

func (repo *fakeMessageRepository) ExistsByKickMessageID(_ context.Context, kickMessageID string) (bool, error) {
	for _, message := range repo.messages {
		if message.KickMessageID == kickMessageID {
			return true, nil
		}
	}
	return false, nil
}

func (repo *fakeMessageRepository) ExistingKickMessageIDs(_ context.Context, kickMessageIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(kickMessageIDs))
	for _, id := range kickMessageIDs {
		for _, message := range repo.messages {
			if message.KickMessageID == id {
				result[id] = true
				break
			}
		}
	}
	return result, nil
}

func (repo *fakeMessageRepository) Search(_ context.Context, _ domain.MessageSearchFilter) ([]domain.ChatMessage, error) {
	return repo.messages, nil
}

type fakeSenderRepository struct {
	values     []domain.SenderProfile
	failUpsert error
}

func (repo *fakeSenderRepository) Upsert(_ context.Context, sender domain.SenderProfile) (domain.SenderProfile, error) {
	if repo.failUpsert != nil {
		return domain.SenderProfile{}, repo.failUpsert
	}
	for index, existing := range repo.values {
		if existing.KickUserID == sender.KickUserID {
			sender.ID = existing.ID
			repo.values[index] = sender
			return sender, nil
		}
	}
	sender.ID = int64(len(repo.values) + 1)
	repo.values = append(repo.values, sender)
	return sender, nil
}

func (repo *fakeSenderRepository) GetByKickUserID(_ context.Context, kickUserID int64) (domain.SenderProfile, error) {
	for _, sender := range repo.values {
		if sender.KickUserID == kickUserID {
			return sender, nil
		}
	}
	return domain.SenderProfile{}, sql.ErrNoRows
}

func (repo *fakeSenderRepository) GetBySlug(_ context.Context, slug string) (domain.SenderProfile, error) {
	for _, sender := range repo.values {
		if sender.Slug == slug {
			return sender, nil
		}
	}
	return domain.SenderProfile{}, sql.ErrNoRows
}

type fakeHeartbeatRepository struct {
	last domain.ListenerHeartbeat
}

func (repo *fakeHeartbeatRepository) Upsert(_ context.Context, heartbeat domain.ListenerHeartbeat) error {
	repo.last = heartbeat
	return nil
}

type fakeChannelResolver struct{}

func (fakeChannelResolver) ResolveChannel(_ context.Context, slug string) (domain.FollowedChannel, error) {
	channel := testChannel()
	channel.Slug = slug
	return channel, nil
}

type fakeSenderResolver struct{}

func (fakeSenderResolver) ResolveSender(_ context.Context, slug string) (domain.SenderProfile, error) {
	if slug == "" {
		return domain.SenderProfile{}, errors.New("missing slug")
	}
	return domain.SenderProfile{
		Username:              "Yavuz_User",
		Slug:                  slug,
		ProfileImageURL:       "https://example.com/avatar.png",
		RawProfilePayloadJSON: `{"user":{"profile_pic":"https://example.com/avatar.png"}}`,
	}, nil
}
