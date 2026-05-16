package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
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
}

func TestRawEventProcessorNormalizesAndDeduplicatesMessages(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-duplicate")
	unit.rawEvents.events = append(unit.rawEvents.events,
		domain.RawKickEvent{
			ID:            "raw-1",
			EventName:     chatMessageEventName,
			KickMessageID: "message-duplicate",
			ChatroomID:    123,
			ChannelID:     1,
			PayloadJSON:   rawPayloadJSON(payload),
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		},
		domain.RawKickEvent{
			ID:            "raw-2",
			EventName:     chatMessageEventName,
			KickMessageID: "message-duplicate",
			ChatroomID:    123,
			ChannelID:     1,
			PayloadJSON:   rawPayloadJSON(payload),
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		},
	)

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
	if len(unit.senders.values) != 1 || unit.senders.values[0].ProfileImageURL == "" {
		t.Fatalf("senders = %#v", unit.senders.values)
	}
}

func TestRawEventProcessorMarksFailuresAndHeartbeat(t *testing.T) {
	unit := newFakeListenerUnit()
	service := newTestService(unit, fakePusherClient{})
	payload := buildPayload("message-failure")
	payload["chatroom_id"] = 999
	unit.rawEvents.events = append(unit.rawEvents.events, domain.RawKickEvent{
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
	return NewService(Dependencies{
		Channels:        unit.channels,
		RawEvents:       unit.rawEvents,
		Messages:        unit.messages,
		Senders:         unit.senders,
		Heartbeats:      unit.heartbeats,
		ChannelResolver: fakeChannelResolver{},
		SenderResolver:  fakeSenderResolver{},
		Pusher:          pusher,
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: ServiceConfig{
			WorkerCount:             0,
			RawEventBatchSize:       10,
			RawEventMaxAttempts:     2,
			ChannelResyncInterval:   time.Second,
			RawEventWorkerIdleDelay: time.Millisecond,
			HeartbeatInterval:       time.Millisecond,
			ReconnectInitialDelay:   time.Millisecond,
			ReconnectMaxDelay:       time.Millisecond,
			ReconnectMultiplier:     1,
			HeartbeatServiceName:    "listener",
		},
	})
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
	messages   *fakeMessageRepository
	senders    *fakeSenderRepository
	heartbeats *fakeHeartbeatRepository
}

func newFakeListenerUnit() *fakeListenerUnit {
	return &fakeListenerUnit{
		channels:   &fakeChannelRepository{channels: []domain.FollowedChannel{testChannel()}},
		rawEvents:  &fakeRawEventRepository{},
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
	events   []domain.RawKickEvent
	attempts []domain.RawEventAttempt
}

func (repo *fakeRawEventRepository) InsertEvent(_ context.Context, event domain.RawKickEvent) error {
	repo.events = append(repo.events, event)
	return nil
}

func (repo *fakeRawEventRepository) InsertAttempt(_ context.Context, attempt domain.RawEventAttempt) error {
	repo.attempts = append(repo.attempts, attempt)
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

func (repo *fakeRawEventRepository) attemptsFor(rawEventID string) uint16 {
	var count uint16
	for _, attempt := range repo.attempts {
		if attempt.RawEventID == rawEventID {
			count++
		}
	}
	return count
}

type fakeMessageRepository struct {
	messages []domain.ChatMessage
}

func (repo *fakeMessageRepository) Insert(_ context.Context, message domain.ChatMessage) error {
	repo.messages = append(repo.messages, message)
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

func (repo *fakeMessageRepository) Search(_ context.Context, _ domain.MessageSearchFilter) ([]domain.ChatMessage, error) {
	return repo.messages, nil
}

type fakeSenderRepository struct {
	values []domain.SenderProfile
}

func (repo *fakeSenderRepository) Upsert(_ context.Context, sender domain.SenderProfile) (domain.SenderProfile, error) {
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
