package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type RawEventRepository struct {
	conn driver.Conn
}

func NewRawEventRepository(conn driver.Conn) *RawEventRepository {
	return &RawEventRepository{conn: conn}
}

func (repo *RawEventRepository) InsertEvent(ctx context.Context, event domain.RawKickEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.PayloadJSON == "" {
		event.PayloadJSON = "{}"
	}
	if event.Status == "" {
		event.Status = "pending"
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now().UTC()
	}

	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO raw_kick_events (
		id, channel_slug, event_type, event_name, payload_json, status, received_at, processed_at, error_message
	)`)
	if err != nil {
		return fmt.Errorf("prepare raw event insert: %w", err)
	}
	if err := batch.Append(
		event.ID,
		event.ChannelSlug,
		event.EventType,
		event.EventName,
		event.PayloadJSON,
		event.Status,
		event.ReceivedAt.UTC(),
		nullableTime(event.ProcessedAt),
		nullableString(event.ErrorMessage),
	); err != nil {
		return fmt.Errorf("append raw event insert: %w", err)
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send raw event insert: %w", err)
	}
	return nil
}

func (repo *RawEventRepository) InsertAttempt(ctx context.Context, attempt domain.RawEventAttempt) error {
	if attempt.ID == "" {
		attempt.ID = uuid.NewString()
	}
	if attempt.Status == "" {
		attempt.Status = "started"
	}
	if attempt.StartedAt.IsZero() {
		attempt.StartedAt = time.Now().UTC()
	}

	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO raw_event_attempts (
		id, raw_event_id, attempt, status, error_message, started_at, finished_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare raw event attempt insert: %w", err)
	}
	if err := batch.Append(
		attempt.ID,
		attempt.RawEventID,
		attempt.Attempt,
		attempt.Status,
		nullableString(attempt.ErrorMessage),
		attempt.StartedAt.UTC(),
		nullableTime(attempt.FinishedAt),
	); err != nil {
		return fmt.Errorf("append raw event attempt insert: %w", err)
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send raw event attempt insert: %w", err)
	}
	return nil
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}
