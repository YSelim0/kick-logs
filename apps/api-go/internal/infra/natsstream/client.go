package natsstream

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type Config struct {
	URL                 string
	StreamName          string
	Subject             string
	ConsumerName        string
	AckWait             time.Duration
	FetchBatchSize      int
	FetchTimeout        time.Duration
	DuplicateWindow     time.Duration
	ConnectionName      string
	ConnectionTimeout   time.Duration
	MaxAckPending       int
	MaxRequestBatchSize int
}

type Client struct {
	conn *nats.Conn
	js   jetstream.JetStream
	cfg  Config
}

func ConfigFromAppConfig(cfg config.Config) Config {
	return NormalizeConfig(Config{
		URL:                 cfg.NATSURL,
		StreamName:          cfg.NATSRawEventStream,
		Subject:             cfg.NATSRawEventSubject,
		ConsumerName:        cfg.NATSRawEventConsumer,
		AckWait:             time.Duration(cfg.NATSRawEventAckWaitSeconds) * time.Second,
		FetchBatchSize:      cfg.NATSRawEventFetchBatchSize,
		FetchTimeout:        time.Duration(cfg.NATSRawEventFetchTimeoutSeconds) * time.Second,
		DuplicateWindow:     10 * time.Minute,
		ConnectionName:      "kick-logs",
		ConnectionTimeout:   10 * time.Second,
		MaxAckPending:       cfg.NATSRawEventFetchBatchSize * 4,
		MaxRequestBatchSize: cfg.NATSRawEventFetchBatchSize,
	})
}

func NormalizeConfig(cfg Config) Config {
	if cfg.URL == "" {
		cfg.URL = "nats://127.0.0.1:4222"
	}
	if cfg.StreamName == "" {
		cfg.StreamName = "KICK_RAW_EVENTS"
	}
	if cfg.Subject == "" {
		cfg.Subject = "kick.raw.chat"
	}
	if cfg.ConsumerName == "" {
		cfg.ConsumerName = "kick-raw-event-processor"
	}
	if cfg.AckWait <= 0 {
		cfg.AckWait = 60 * time.Second
	}
	if cfg.FetchBatchSize <= 0 {
		cfg.FetchBatchSize = 500
	}
	if cfg.FetchTimeout <= 0 {
		cfg.FetchTimeout = 2 * time.Second
	}
	if cfg.DuplicateWindow <= 0 {
		cfg.DuplicateWindow = 10 * time.Minute
	}
	if cfg.ConnectionName == "" {
		cfg.ConnectionName = "kick-logs"
	}
	if cfg.ConnectionTimeout <= 0 {
		cfg.ConnectionTimeout = 10 * time.Second
	}
	if cfg.MaxAckPending <= 0 {
		cfg.MaxAckPending = cfg.FetchBatchSize * 4
	}
	if cfg.MaxRequestBatchSize <= 0 {
		cfg.MaxRequestBatchSize = cfg.FetchBatchSize
	}
	return cfg
}

func Open(ctx context.Context, appCfg config.Config) (*Client, error) {
	cfg := ConfigFromAppConfig(appCfg)
	conn, err := nats.Connect(
		cfg.URL,
		nats.Name(cfg.ConnectionName),
		nats.Timeout(cfg.ConnectionTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create jetstream client: %w", err)
	}

	client := &Client{conn: conn, js: js, cfg: cfg}
	if err := client.Ensure(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	return client, nil
}

func (client *Client) Close() {
	if client != nil && client.conn != nil {
		client.conn.Close()
	}
}

func (client *Client) Ensure(ctx context.Context) error {
	stream, err := client.js.CreateOrUpdateStream(ctx, client.streamConfig())
	if err != nil {
		return fmt.Errorf("ensure raw event stream: %w", err)
	}
	if _, err := stream.CreateOrUpdateConsumer(ctx, client.consumerConfig()); err != nil {
		return fmt.Errorf("ensure raw event consumer: %w", err)
	}
	return nil
}

func (client *Client) Publish(ctx context.Context, event domain.RawStreamEvent) (domain.RawStreamPublishAck, error) {
	subject := event.Subject
	if subject == "" {
		subject = client.cfg.Subject
	}
	if len(event.Payload) == 0 {
		return domain.RawStreamPublishAck{}, fmt.Errorf("raw stream event payload is required")
	}

	opts := []jetstream.PublishOpt{jetstream.WithExpectStream(client.cfg.StreamName)}
	if event.ID != "" {
		opts = append(opts, jetstream.WithMsgID(event.ID))
	}

	msg := nats.NewMsg(subject)
	msg.Data = event.Payload
	for key, value := range event.Headers {
		msg.Header.Set(key, value)
	}

	ack, err := client.js.PublishMsg(ctx, msg, opts...)
	if err != nil {
		return domain.RawStreamPublishAck{}, fmt.Errorf("publish raw event to jetstream: %w", err)
	}
	return domain.RawStreamPublishAck{
		Stream:    ack.Stream,
		Sequence:  ack.Sequence,
		Duplicate: ack.Duplicate,
	}, nil
}

func (client *Client) Fetch(ctx context.Context, limit int) ([]ports.RawEventStreamMessage, error) {
	if limit <= 0 {
		limit = client.cfg.FetchBatchSize
	}

	stream, err := client.js.Stream(ctx, client.cfg.StreamName)
	if err != nil {
		return nil, fmt.Errorf("get raw event stream: %w", err)
	}
	consumer, err := stream.Consumer(ctx, client.cfg.ConsumerName)
	if err != nil {
		return nil, fmt.Errorf("get raw event consumer: %w", err)
	}

	batch, err := consumer.Fetch(
		limit,
		jetstream.FetchContext(ctx),
		jetstream.FetchMaxWait(client.cfg.FetchTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("fetch raw events from jetstream: %w", err)
	}

	messages := make([]ports.RawEventStreamMessage, 0, limit)
	for msg := range batch.Messages() {
		messages = append(messages, streamMessage{msg: msg})
	}
	if err := batch.Error(); err != nil {
		return nil, fmt.Errorf("iterate fetched raw events: %w", err)
	}
	return messages, nil
}

func (client *Client) Stats(ctx context.Context) (domain.RawStreamStats, error) {
	stream, err := client.js.Stream(ctx, client.cfg.StreamName)
	if err != nil {
		return domain.RawStreamStats{}, fmt.Errorf("get raw event stream stats: %w", err)
	}
	streamInfo, err := stream.Info(ctx)
	if err != nil {
		return domain.RawStreamStats{}, fmt.Errorf("read raw event stream stats: %w", err)
	}
	consumer, err := stream.Consumer(ctx, client.cfg.ConsumerName)
	if err != nil {
		return domain.RawStreamStats{}, fmt.Errorf("get raw event consumer stats: %w", err)
	}
	consumerInfo, err := consumer.Info(ctx)
	if err != nil {
		return domain.RawStreamStats{}, fmt.Errorf("read raw event consumer stats: %w", err)
	}

	now := time.Now().UTC()
	stats := domain.RawStreamStats{
		StreamName:               streamInfo.Config.Name,
		ConsumerName:             consumerInfo.Name,
		Messages:                 int64(streamInfo.State.Msgs),
		Bytes:                    int64(streamInfo.State.Bytes),
		ConsumerPending:          int64(consumerInfo.NumPending),
		ConsumerAckPending:       int64(consumerInfo.NumAckPending),
		ConsumerRedelivered:      int64(consumerInfo.NumRedelivered),
		LatestConsumerUpdateTime: consumerInfo.TimeStamp.UTC(),
	}
	if !streamInfo.State.FirstTime.IsZero() {
		stats.OldestPendingAgeSeconds = int64(now.Sub(streamInfo.State.FirstTime.UTC()).Seconds())
	}
	if !streamInfo.State.LastTime.IsZero() {
		stats.LatestMessageAgeSeconds = int64(now.Sub(streamInfo.State.LastTime.UTC()).Seconds())
	}
	return stats, nil
}

func (client *Client) streamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:       client.cfg.StreamName,
		Subjects:   []string{client.cfg.Subject},
		Retention:  jetstream.WorkQueuePolicy,
		Discard:    jetstream.DiscardNew,
		Storage:    jetstream.FileStorage,
		Replicas:   1,
		Duplicates: client.cfg.DuplicateWindow,
	}
}

func (client *Client) consumerConfig() jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Durable:           client.cfg.ConsumerName,
		Name:              client.cfg.ConsumerName,
		DeliverPolicy:     jetstream.DeliverAllPolicy,
		AckPolicy:         jetstream.AckExplicitPolicy,
		AckWait:           client.cfg.AckWait,
		MaxDeliver:        -1,
		FilterSubject:     client.cfg.Subject,
		ReplayPolicy:      jetstream.ReplayInstantPolicy,
		MaxAckPending:     client.cfg.MaxAckPending,
		MaxRequestBatch:   client.cfg.MaxRequestBatchSize,
		MaxRequestExpires: client.cfg.FetchTimeout,
	}
}

type streamMessage struct {
	msg jetstream.Msg
}

func (message streamMessage) ID() string {
	return message.msg.Headers().Get(jetstream.MsgIDHeader)
}

func (message streamMessage) Subject() string {
	return message.msg.Subject()
}

func (message streamMessage) Data() []byte {
	return message.msg.Data()
}

func (message streamMessage) StreamSequence() uint64 {
	metadata, err := message.msg.Metadata()
	if err != nil {
		return 0
	}
	return metadata.Sequence.Stream
}

func (message streamMessage) ConsumerSequence() uint64 {
	metadata, err := message.msg.Metadata()
	if err != nil {
		return 0
	}
	return metadata.Sequence.Consumer
}

func (message streamMessage) NumDelivered() uint64 {
	metadata, err := message.msg.Metadata()
	if err != nil {
		return 0
	}
	return metadata.NumDelivered
}

func (message streamMessage) NumPending() uint64 {
	metadata, err := message.msg.Metadata()
	if err != nil {
		return 0
	}
	return metadata.NumPending
}

func (message streamMessage) Timestamp() time.Time {
	metadata, err := message.msg.Metadata()
	if err != nil {
		return time.Time{}
	}
	return metadata.Timestamp.UTC()
}

func (message streamMessage) Ack(ctx context.Context) error {
	return message.msg.DoubleAck(ctx)
}

func (message streamMessage) Nak(_ context.Context, delay time.Duration) error {
	if delay > 0 {
		return message.msg.NakWithDelay(delay)
	}
	return message.msg.Nak()
}

func (message streamMessage) Term(_ context.Context, reason string) error {
	if reason != "" {
		return message.msg.TermWithReason(reason)
	}
	return message.msg.Term()
}
