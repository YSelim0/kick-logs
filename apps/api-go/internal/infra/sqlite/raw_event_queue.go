package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type RawEventQueueRepository struct {
	db *sql.DB
}

func NewRawEventQueueRepository(db *sql.DB) *RawEventQueueRepository {
	return &RawEventQueueRepository{db: db}
}

func (repo *RawEventQueueRepository) Enqueue(ctx context.Context, item domain.RawEventQueueItem) error {
	return repo.EnqueueBatch(ctx, []domain.RawEventQueueItem{item})
}

func (repo *RawEventQueueRepository) EnqueueBatch(ctx context.Context, items []domain.RawEventQueueItem) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin enqueue tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO raw_event_queue (
			raw_event_id, channel_id, chatroom_id, channel_slug, kick_message_id,
			status, attempts, claimed_by, claimed_at, enqueued_at, last_error, updated_at
		) VALUES (?, ?, ?, ?, ?, 'pending', 0, '', '', ?, '', ?)
		ON CONFLICT(raw_event_id) DO NOTHING;
	`)
	if err != nil {
		return fmt.Errorf("prepare enqueue: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, item := range items {
		if item.RawEventID == "" {
			return fmt.Errorf("raw event id is required")
		}
		enqueuedAt := item.EnqueuedAt
		if enqueuedAt.IsZero() {
			enqueuedAt = now
		}
		if _, err := stmt.ExecContext(
			ctx,
			item.RawEventID,
			item.ChannelID,
			item.ChatroomID,
			item.ChannelSlug,
			item.KickMessageID,
			formatTime(enqueuedAt),
			formatTime(now),
		); err != nil {
			return fmt.Errorf("enqueue raw event %s: %w", item.RawEventID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit enqueue: %w", err)
	}
	return nil
}

func (repo *RawEventQueueRepository) ListPending(
	ctx context.Context,
	limit uint64,
	maxAttempts uint16,
) ([]domain.RawEventQueueItem, error) {
	if limit == 0 {
		limit = 100
	}
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	rows, err := repo.db.QueryContext(
		ctx,
		`SELECT raw_event_id, channel_id, chatroom_id, channel_slug, kick_message_id,
				status, attempts, claimed_by, claimed_at, enqueued_at, last_error, updated_at
		 FROM raw_event_queue
		 WHERE status = 'pending' AND attempts < ?
		 ORDER BY enqueued_at ASC
		 LIMIT ?;`,
		maxAttempts,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending raw events: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RawEventQueueItem, 0)
	for rows.Next() {
		item, err := scanQueueItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending raw events: %w", err)
	}
	return items, nil
}

func (repo *RawEventQueueRepository) Claim(
	ctx context.Context,
	rawEventID string,
	workerID string,
) (bool, error) {
	if rawEventID == "" {
		return false, fmt.Errorf("raw event id is required")
	}
	if workerID == "" {
		return false, fmt.Errorf("worker id is required")
	}
	now := time.Now().UTC()
	result, err := repo.db.ExecContext(
		ctx,
		`UPDATE raw_event_queue
		 SET status = 'claimed',
			 claimed_by = ?,
			 claimed_at = ?,
			 updated_at = ?
		 WHERE raw_event_id = ? AND status = 'pending';`,
		workerID,
		formatTime(now),
		formatTime(now),
		rawEventID,
	)
	if err != nil {
		return false, fmt.Errorf("claim raw event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read claim result: %w", err)
	}
	return affected > 0, nil
}

func (repo *RawEventQueueRepository) Release(
	ctx context.Context,
	rawEventID string,
	workerID string,
) error {
	if rawEventID == "" {
		return fmt.Errorf("raw event id is required")
	}
	now := time.Now().UTC()
	_, err := repo.db.ExecContext(
		ctx,
		`UPDATE raw_event_queue
		 SET status = 'pending',
			 claimed_by = '',
			 claimed_at = '',
			 updated_at = ?
		 WHERE raw_event_id = ? AND status = 'claimed' AND (? = '' OR claimed_by = ?);`,
		formatTime(now),
		rawEventID,
		workerID,
		workerID,
	)
	if err != nil {
		return fmt.Errorf("release raw event: %w", err)
	}
	return nil
}

func (repo *RawEventQueueRepository) MarkProcessed(ctx context.Context, rawEventID string) error {
	if rawEventID == "" {
		return fmt.Errorf("raw event id is required")
	}
	_, err := repo.db.ExecContext(
		ctx,
		`DELETE FROM raw_event_queue WHERE raw_event_id = ?;`,
		rawEventID,
	)
	if err != nil {
		return fmt.Errorf("delete processed raw event queue row: %w", err)
	}
	return nil
}

func (repo *RawEventQueueRepository) MarkFailed(
	ctx context.Context,
	rawEventID string,
	errMessage string,
	maxAttempts uint16,
) error {
	if rawEventID == "" {
		return fmt.Errorf("raw event id is required")
	}
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	now := time.Now().UTC()
	_, err := repo.db.ExecContext(
		ctx,
		`UPDATE raw_event_queue
		 SET status = CASE
				WHEN attempts + 1 >= ? THEN 'failed'
				ELSE 'pending'
			 END,
			 attempts = attempts + 1,
			 claimed_by = '',
			 claimed_at = '',
			 last_error = ?,
			 updated_at = ?
		 WHERE raw_event_id = ? AND status NOT IN ('processed', 'failed');`,
		maxAttempts,
		errMessage,
		formatTime(now),
		rawEventID,
	)
	if err != nil {
		return fmt.Errorf("mark raw event failed: %w", err)
	}
	return nil
}

func (repo *RawEventQueueRepository) CountPending(ctx context.Context, maxAttempts uint16) (int64, error) {
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	var count int64
	if err := repo.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM raw_event_queue
		 WHERE status IN ('pending', 'claimed') AND attempts < ?;`,
		maxAttempts,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending raw events: %w", err)
	}
	return count, nil
}

func (repo *RawEventQueueRepository) OldestPendingAge(
	ctx context.Context,
	maxAttempts uint16,
) (time.Duration, error) {
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	var enqueuedAt sql.NullString
	if err := repo.db.QueryRowContext(
		ctx,
		`SELECT MIN(enqueued_at) FROM raw_event_queue
		 WHERE status IN ('pending', 'claimed') AND attempts < ?;`,
		maxAttempts,
	).Scan(&enqueuedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("oldest pending raw event: %w", err)
	}
	if !enqueuedAt.Valid || enqueuedAt.String == "" {
		return 0, nil
	}
	parsed := parseTime(enqueuedAt.String)
	if parsed.IsZero() {
		return 0, nil
	}
	age := time.Since(parsed)
	if age < 0 {
		return 0, nil
	}
	return age, nil
}

func (repo *RawEventQueueRepository) RecoverStaleClaims(
	ctx context.Context,
	olderThan time.Duration,
) (int64, error) {
	if olderThan <= 0 {
		olderThan = 5 * time.Minute
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := repo.db.ExecContext(
		ctx,
		`UPDATE raw_event_queue
		 SET status = 'pending',
			 claimed_by = '',
			 claimed_at = '',
			 updated_at = ?
		 WHERE status = 'claimed' AND claimed_at != '' AND claimed_at <= ?;`,
		formatTime(time.Now().UTC()),
		formatTime(cutoff),
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale claims: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read recover stale result: %w", err)
	}
	return affected, nil
}

func (repo *RawEventQueueRepository) GetByID(
	ctx context.Context,
	rawEventID string,
) (domain.RawEventQueueItem, error) {
	if rawEventID == "" {
		return domain.RawEventQueueItem{}, fmt.Errorf("raw event id is required")
	}
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT raw_event_id, channel_id, chatroom_id, channel_slug, kick_message_id,
				status, attempts, claimed_by, claimed_at, enqueued_at, last_error, updated_at
		 FROM raw_event_queue
		 WHERE raw_event_id = ?;`,
		rawEventID,
	)
	return scanQueueItem(row)
}

type queueRowScanner interface {
	Scan(dest ...any) error
}

func scanQueueItem(scanner queueRowScanner) (domain.RawEventQueueItem, error) {
	var item domain.RawEventQueueItem
	var claimedAt, enqueuedAt, updatedAt string
	if err := scanner.Scan(
		&item.RawEventID,
		&item.ChannelID,
		&item.ChatroomID,
		&item.ChannelSlug,
		&item.KickMessageID,
		&item.Status,
		&item.Attempts,
		&item.ClaimedBy,
		&claimedAt,
		&enqueuedAt,
		&item.LastError,
		&updatedAt,
	); err != nil {
		return domain.RawEventQueueItem{}, fmt.Errorf("scan raw event queue row: %w", err)
	}
	item.ClaimedAt = parseTime(claimedAt)
	item.EnqueuedAt = parseTime(enqueuedAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
}
