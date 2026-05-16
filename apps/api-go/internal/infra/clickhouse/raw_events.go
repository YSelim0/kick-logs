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
	if event.MetadataJSON == "" {
		event.MetadataJSON = "{}"
	}
	if event.Status == "" {
		event.Status = "pending"
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now().UTC()
	}

	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO raw_kick_events (
		id, channel_slug, event_type, event_name, kick_message_id, chatroom_id, channel_id,
		payload_json, metadata_json, status, received_at, processed_at, error_message
	)`)
	if err != nil {
		return fmt.Errorf("prepare raw event insert: %w", err)
	}
	if err := batch.Append(
		event.ID,
		event.ChannelSlug,
		event.EventType,
		event.EventName,
		nullableString(event.KickMessageID),
		nullableInt64(event.ChatroomID),
		nullableInt64(event.ChannelID),
		event.PayloadJSON,
		event.MetadataJSON,
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

func (repo *RawEventRepository) ListUnprocessed(
	ctx context.Context,
	limit uint64,
	maxAttempts uint16,
) ([]domain.RawKickEvent, error) {
	if limit == 0 {
		limit = 100
	}
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	rows, err := repo.conn.Query(
		ctx,
		`SELECT
			e.id, e.channel_slug, e.event_type, e.event_name, ifNull(e.kick_message_id, ''),
			ifNull(e.chatroom_id, 0), ifNull(e.channel_id, 0), e.payload_json, e.metadata_json, e.status,
			ifNull(a.attempts, 0), e.received_at, e.processed_at, e.error_message
		 FROM raw_kick_events AS e
		 LEFT JOIN (
			SELECT raw_event_id, max(attempt) AS attempts
			FROM raw_event_attempts
			GROUP BY raw_event_id
		 ) AS a ON a.raw_event_id = e.id
		 LEFT JOIN (
			SELECT raw_event_id
			FROM raw_event_attempts
			WHERE status = 'processed'
			GROUP BY raw_event_id
		 ) AS processed ON processed.raw_event_id = e.id
		 WHERE processed.raw_event_id = '' AND ifNull(a.attempts, 0) < ?
		 ORDER BY e.received_at ASC
		 LIMIT ?`,
		maxAttempts,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list unprocessed raw events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.RawKickEvent, 0)
	for rows.Next() {
		event, err := scanRawKickEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unprocessed raw events: %w", err)
	}
	return events, nil
}

func (repo *RawEventRepository) CountUnprocessed(ctx context.Context, maxAttempts uint16) (int64, error) {
	var count uint64
	if err := repo.conn.QueryRow(
		ctx,
		`SELECT count()
		 FROM (
			SELECT e.id, ifNull(a.attempts, 0) AS attempts
			FROM raw_kick_events AS e
			LEFT JOIN (
				SELECT raw_event_id, max(attempt) AS attempts
				FROM raw_event_attempts
				GROUP BY raw_event_id
			) AS a ON a.raw_event_id = e.id
			LEFT JOIN (
				SELECT raw_event_id
				FROM raw_event_attempts
				WHERE status = 'processed'
				GROUP BY raw_event_id
			) AS processed ON processed.raw_event_id = e.id
			WHERE processed.raw_event_id = '' AND attempts < ?
		 )`,
		maxAttempts,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unprocessed raw events: %w", err)
	}
	return int64(count), nil
}

func (repo *RawEventRepository) AttemptCount(ctx context.Context, rawEventID string) (uint16, error) {
	var count uint64
	if err := repo.conn.QueryRow(
		ctx,
		"SELECT count() FROM raw_event_attempts WHERE raw_event_id = ?",
		rawEventID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count raw event attempts: %w", err)
	}
	if count > uint64(^uint16(0)) {
		return ^uint16(0), nil
	}
	return uint16(count), nil
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

type rawEventScanner interface {
	Scan(dest ...any) error
}

func scanRawKickEvent(scanner rawEventScanner) (domain.RawKickEvent, error) {
	var event domain.RawKickEvent
	var processedAt *time.Time
	var errorMessage *string
	if err := scanner.Scan(
		&event.ID,
		&event.ChannelSlug,
		&event.EventType,
		&event.EventName,
		&event.KickMessageID,
		&event.ChatroomID,
		&event.ChannelID,
		&event.PayloadJSON,
		&event.MetadataJSON,
		&event.Status,
		&event.Attempts,
		&event.ReceivedAt,
		&processedAt,
		&errorMessage,
	); err != nil {
		return domain.RawKickEvent{}, fmt.Errorf("scan raw event: %w", err)
	}
	if processedAt != nil {
		event.ProcessedAt = processedAt.UTC()
	}
	if errorMessage != nil {
		event.ErrorMessage = *errorMessage
	}
	event.ReceivedAt = event.ReceivedAt.UTC()
	return event, nil
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}
