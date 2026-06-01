package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type KickWebhookEventRepository struct {
	db *sql.DB
}

func NewKickWebhookEventRepository(db *sql.DB) *KickWebhookEventRepository {
	return &KickWebhookEventRepository{db: db}
}

func (r *KickWebhookEventRepository) InsertIdempotent(ctx context.Context, event domain.KickWebhookEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO kick_webhook_events
			(message_id, subscription_id, event_type, event_version, raw_payload_json,
			 status, attempts, received_at, processed_at, error_message)
		VALUES (?, ?, ?, ?, ?, 'pending', 0, ?, '', '')`,
		event.MessageID,
		event.SubscriptionID,
		event.EventType,
		event.EventVersion,
		event.RawPayloadJSON,
		formatTime(event.ReceivedAt),
	)
	if err != nil {
		return fmt.Errorf("insert webhook event: %w", err)
	}
	return nil
}

func (r *KickWebhookEventRepository) GetByMessageID(ctx context.Context, messageID string) (domain.KickWebhookEvent, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT message_id, subscription_id, event_type, event_version, raw_payload_json,
		       status, attempts, received_at, processed_at, error_message
		FROM kick_webhook_events
		WHERE message_id = ?`, messageID)
	return scanKickWebhookEvent(row)
}

func (r *KickWebhookEventRepository) ListPending(ctx context.Context, limit int, maxAttempts int) ([]domain.KickWebhookEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT message_id, subscription_id, event_type, event_version, raw_payload_json,
		       status, attempts, received_at, processed_at, error_message
		FROM kick_webhook_events
		WHERE status = 'pending' AND attempts < ?
		ORDER BY received_at ASC
		LIMIT ?`, maxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending webhook events: %w", err)
	}
	defer rows.Close()
	return scanKickWebhookEvents(rows)
}

func (r *KickWebhookEventRepository) MarkProcessed(ctx context.Context, messageID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE kick_webhook_events
		SET status = 'processed', processed_at = ?
		WHERE message_id = ?`,
		formatTime(time.Now().UTC()), messageID)
	if err != nil {
		return fmt.Errorf("mark webhook event processed: %w", err)
	}
	return nil
}

func (r *KickWebhookEventRepository) MarkFailed(ctx context.Context, messageID string, errMessage string, maxAttempts int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE kick_webhook_events
		SET attempts = attempts + 1,
		    error_message = ?,
		    status = CASE WHEN attempts + 1 >= ? THEN 'failed' ELSE 'pending' END
		WHERE message_id = ?`,
		errMessage, maxAttempts, messageID)
	if err != nil {
		return fmt.Errorf("mark webhook event failed: %w", err)
	}
	return nil
}

func (r *KickWebhookEventRepository) MarkIgnored(ctx context.Context, messageID string, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE kick_webhook_events
		SET status = 'ignored', error_message = ?, processed_at = ?
		WHERE message_id = ?`,
		reason, formatTime(time.Now().UTC()), messageID)
	if err != nil {
		return fmt.Errorf("mark webhook event ignored: %w", err)
	}
	return nil
}

func (r *KickWebhookEventRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM kick_webhook_events GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("count webhook events by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan webhook event count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *KickWebhookEventRepository) LatestReceivedAt(ctx context.Context) (time.Time, error) {
	row := r.db.QueryRowContext(ctx, `SELECT MAX(received_at) FROM kick_webhook_events`)
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		return time.Time{}, fmt.Errorf("latest webhook received_at: %w", err)
	}
	if !raw.Valid {
		return time.Time{}, nil
	}
	return parseTime(raw.String), nil
}

type kickWebhookEventScanner interface {
	Scan(dest ...any) error
}

func scanKickWebhookEvent(scanner kickWebhookEventScanner) (domain.KickWebhookEvent, error) {
	var e domain.KickWebhookEvent
	var receivedAt, processedAt string
	if err := scanner.Scan(
		&e.MessageID, &e.SubscriptionID, &e.EventType, &e.EventVersion, &e.RawPayloadJSON,
		&e.Status, &e.Attempts, &receivedAt, &processedAt, &e.ErrorMessage,
	); err != nil {
		return domain.KickWebhookEvent{}, err
	}
	e.ReceivedAt = parseTime(receivedAt)
	e.ProcessedAt = parseTime(processedAt)
	return e, nil
}

func scanKickWebhookEvents(rows *sql.Rows) ([]domain.KickWebhookEvent, error) {
	var events []domain.KickWebhookEvent
	for rows.Next() {
		e, err := scanKickWebhookEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
