package ports

import (
	"context"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type RawEventStreamPublisher interface {
	Publish(ctx context.Context, event domain.RawStreamEvent) (domain.RawStreamPublishAck, error)
}

type RawEventStreamConsumer interface {
	Fetch(ctx context.Context, limit int) ([]RawEventStreamMessage, error)
}

type RawEventStreamStatsRepository interface {
	Stats(ctx context.Context) (domain.RawStreamStats, error)
}

type RawEventStreamMessage interface {
	ID() string
	Subject() string
	Data() []byte
	StreamSequence() uint64
	ConsumerSequence() uint64
	NumDelivered() uint64
	NumPending() uint64
	Timestamp() time.Time
	Ack(ctx context.Context) error
	Nak(ctx context.Context, delay time.Duration) error
	Term(ctx context.Context, reason string) error
}
