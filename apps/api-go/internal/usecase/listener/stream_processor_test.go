package listener

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

func TestStreamProcessorWritesClickHouseBatchesThenAcks(t *testing.T) {
	unit := newFakeListenerUnit()
	messages := []*fakeRawEventStreamMessage{
		newFakeRawEventStreamMessage(t, "stream-1", buildPayload("message-stream-1")),
		newFakeRawEventStreamMessage(t, "stream-2", buildPayload("message-stream-2")),
	}
	processor := newTestStreamProcessor(unit, &fakeRawEventStreamConsumer{messages: messages})

	result, err := processor.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}
	if result.Fetched != 2 || result.RawInserted != 2 || result.Processed != 2 || result.Acked != 2 || result.Nacked != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(unit.rawEvents.events) != 2 {
		t.Fatalf("raw events = %#v", unit.rawEvents.events)
	}
	if len(unit.messages.messages) != 2 {
		t.Fatalf("messages = %#v", unit.messages.messages)
	}
	if len(unit.rawEvents.attempts) != 2 || unit.rawEvents.attempts[0].Status != "processed" {
		t.Fatalf("attempts = %#v", unit.rawEvents.attempts)
	}
	for _, message := range messages {
		if !message.acked || message.nacked || message.termed {
			t.Fatalf("message ack state = %#v", message)
		}
	}
}

func TestStreamProcessorNacksTransientClickHouseFailure(t *testing.T) {
	unit := newFakeListenerUnit()
	unit.messages.failBatch = errors.New("clickhouse unavailable")
	messages := []*fakeRawEventStreamMessage{
		newFakeRawEventStreamMessage(t, "stream-fail", buildPayload("message-stream-fail")),
	}
	processor := newTestStreamProcessor(unit, &fakeRawEventStreamConsumer{messages: messages})

	result, err := processor.ProcessOnce(context.Background())
	if err == nil {
		t.Fatal("expected transient ClickHouse failure")
	}
	if result.Fetched != 1 || result.RawInserted != 1 || result.Nacked != 1 || result.Acked != 0 {
		t.Fatalf("result = %#v", result)
	}
	if !messages[0].nacked || messages[0].acked || messages[0].termed {
		t.Fatalf("message ack state = %#v", messages[0])
	}
	if len(unit.rawEvents.attempts) != 0 {
		t.Fatalf("attempts should wait for successful durable attempt write = %#v", unit.rawEvents.attempts)
	}
}

func TestStreamProcessorTermsTerminalInvalidPayload(t *testing.T) {
	unit := newFakeListenerUnit()
	payload := buildPayload("message-invalid-stream")
	delete(payload, "id")
	messages := []*fakeRawEventStreamMessage{
		newFakeRawEventStreamMessage(t, "stream-invalid", payload),
	}
	processor := newTestStreamProcessor(unit, &fakeRawEventStreamConsumer{messages: messages})

	result, err := processor.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}
	if result.Fetched != 1 || result.Ignored != 1 || result.Termed != 1 || result.Acked != 0 {
		t.Fatalf("result = %#v", result)
	}
	if !messages[0].termed || messages[0].acked || messages[0].nacked {
		t.Fatalf("message ack state = %#v", messages[0])
	}
	if len(unit.messages.messages) != 0 {
		t.Fatalf("messages = %#v", unit.messages.messages)
	}
	if len(unit.rawEvents.attempts) != 1 || unit.rawEvents.attempts[0].Status != "ignored" {
		t.Fatalf("attempts = %#v", unit.rawEvents.attempts)
	}
}

func newTestStreamProcessor(unit *fakeListenerUnit, stream *fakeRawEventStreamConsumer) *StreamProcessorService {
	return NewStreamProcessorService(StreamProcessorDependencies{
		Stream:     stream,
		RawEvents:  unit.rawEvents,
		Messages:   unit.messages,
		Channels:   unit.channels,
		Senders:    unit.senders,
		Heartbeats: unit.heartbeats,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: StreamProcessorConfig{
			BatchSize:                10,
			IdleDelay:                time.Millisecond,
			HeartbeatInterval:        time.Millisecond,
			HeartbeatServiceName:     "processor",
			NakDelay:                 time.Millisecond,
			SenderProfileCacheTTL:    time.Minute,
			ClickHouseBackoffInitial: time.Millisecond,
			ClickHouseBackoffMax:     time.Millisecond,
			ClickHouseBackoffFactor:  1,
			ClickHouseBreakerThresh:  1,
		},
	})
}

func newFakeRawEventStreamMessage(t *testing.T, id string, payload map[string]any) *fakeRawEventStreamMessage {
	t.Helper()
	rawEvent := ChatMessageEvent{
		EventName:     chatMessageEventName,
		PusherChannel: "chatrooms.123.v2",
		Payload:       payload,
		RawJSON:       buildPusherEvent(payload),
	}
	envelope := rawChatEventEnvelopeFromEvent(rawEvent, testChannel(), 123, time.Now().UTC())
	envelope.RawEventID = id
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return &fakeRawEventStreamMessage{
		id:               id,
		subject:          "kick.raw.chat",
		data:             data,
		streamSequence:   1,
		consumerSequence: 1,
		numDelivered:     1,
		timestamp:        time.Now().UTC(),
	}
}

type fakeRawEventStreamConsumer struct {
	messages []*fakeRawEventStreamMessage
	err      error
}

func (consumer *fakeRawEventStreamConsumer) Fetch(_ context.Context, limit int) ([]ports.RawEventStreamMessage, error) {
	if consumer.err != nil {
		return nil, consumer.err
	}
	if limit <= 0 || limit > len(consumer.messages) {
		limit = len(consumer.messages)
	}
	out := make([]ports.RawEventStreamMessage, 0, limit)
	for _, message := range consumer.messages[:limit] {
		out = append(out, message)
	}
	return out, nil
}

type fakeRawEventStreamMessage struct {
	id               string
	subject          string
	data             []byte
	streamSequence   uint64
	consumerSequence uint64
	numDelivered     uint64
	numPending       uint64
	timestamp        time.Time
	acked            bool
	nacked           bool
	termed           bool
}

func (message *fakeRawEventStreamMessage) ID() string {
	return message.id
}

func (message *fakeRawEventStreamMessage) Subject() string {
	return message.subject
}

func (message *fakeRawEventStreamMessage) Data() []byte {
	return message.data
}

func (message *fakeRawEventStreamMessage) StreamSequence() uint64 {
	return message.streamSequence
}

func (message *fakeRawEventStreamMessage) ConsumerSequence() uint64 {
	return message.consumerSequence
}

func (message *fakeRawEventStreamMessage) NumDelivered() uint64 {
	return message.numDelivered
}

func (message *fakeRawEventStreamMessage) NumPending() uint64 {
	return message.numPending
}

func (message *fakeRawEventStreamMessage) Timestamp() time.Time {
	return message.timestamp
}

func (message *fakeRawEventStreamMessage) Ack(_ context.Context) error {
	message.acked = true
	return nil
}

func (message *fakeRawEventStreamMessage) Nak(_ context.Context, _ time.Duration) error {
	message.nacked = true
	return nil
}

func (message *fakeRawEventStreamMessage) Term(_ context.Context, _ string) error {
	message.termed = true
	return nil
}
