package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type KickEventSubscriptionRepository struct {
	db *sql.DB
}

func NewKickEventSubscriptionRepository(db *sql.DB) *KickEventSubscriptionRepository {
	return &KickEventSubscriptionRepository{db: db}
}

func (r *KickEventSubscriptionRepository) Upsert(ctx context.Context, sub domain.KickEventSubscription) (domain.KickEventSubscription, error) {
	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO kick_event_subscriptions
			(followed_channel_id, broadcaster_user_id, event_type, event_version, method,
			 kick_subscription_id, status, latest_sync_error, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(followed_channel_id, event_type, event_version, method) DO UPDATE SET
			broadcaster_user_id   = excluded.broadcaster_user_id,
			kick_subscription_id  = excluded.kick_subscription_id,
			status                = excluded.status,
			latest_sync_error     = excluded.latest_sync_error,
			updated_at            = excluded.updated_at,
			synced_at             = excluded.synced_at`,
		sub.FollowedChannelID,
		sub.BroadcasterUserID,
		sub.EventType,
		sub.EventVersion,
		sub.Method,
		sub.KickSubscriptionID,
		sub.Status,
		sub.LatestSyncError,
		formatTime(sub.CreatedAt),
		formatTime(sub.UpdatedAt),
		formatTime(sub.SyncedAt),
	)
	if err != nil {
		return domain.KickEventSubscription{}, fmt.Errorf("upsert kick event subscription: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT id, followed_channel_id, broadcaster_user_id, event_type, event_version, method,
		       kick_subscription_id, status, latest_sync_error, created_at, updated_at, synced_at
		FROM kick_event_subscriptions
		WHERE followed_channel_id = ? AND event_type = ? AND event_version = ? AND method = ?`,
		sub.FollowedChannelID, sub.EventType, sub.EventVersion, sub.Method,
	)
	return scanKickEventSubscription(row)
}

func (r *KickEventSubscriptionRepository) ListByChannel(ctx context.Context, followedChannelID int64) ([]domain.KickEventSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, followed_channel_id, broadcaster_user_id, event_type, event_version, method,
		       kick_subscription_id, status, latest_sync_error, created_at, updated_at, synced_at
		FROM kick_event_subscriptions
		WHERE followed_channel_id = ?
		ORDER BY event_type ASC`, followedChannelID)
	if err != nil {
		return nil, fmt.Errorf("list kick event subscriptions by channel: %w", err)
	}
	defer rows.Close()
	return scanKickEventSubscriptions(rows)
}

func (r *KickEventSubscriptionRepository) DeleteByChannel(ctx context.Context, followedChannelID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE kick_event_subscriptions
		SET status = 'deleted', updated_at = ?
		WHERE followed_channel_id = ? AND status != 'deleted'`,
		formatTime(time.Now().UTC()), followedChannelID)
	if err != nil {
		return fmt.Errorf("delete kick event subscriptions by channel: %w", err)
	}
	return nil
}

func (r *KickEventSubscriptionRepository) List(ctx context.Context) ([]domain.KickEventSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, followed_channel_id, broadcaster_user_id, event_type, event_version, method,
		       kick_subscription_id, status, latest_sync_error, created_at, updated_at, synced_at
		FROM kick_event_subscriptions
		ORDER BY followed_channel_id ASC, event_type ASC`)
	if err != nil {
		return nil, fmt.Errorf("list kick event subscriptions: %w", err)
	}
	defer rows.Close()
	return scanKickEventSubscriptions(rows)
}

func (r *KickEventSubscriptionRepository) UpdateSyncError(ctx context.Context, id int64, syncError string) error {
	status := domain.KickEventSubStatusActive
	if syncError != "" {
		status = domain.KickEventSubStatusError
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE kick_event_subscriptions
		SET latest_sync_error = ?, status = ?, updated_at = ?
		WHERE id = ?`,
		syncError, status, formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("update kick event subscription sync error: %w", err)
	}
	return nil
}

type kickEventSubscriptionScanner interface {
	Scan(dest ...any) error
}

func scanKickEventSubscription(scanner kickEventSubscriptionScanner) (domain.KickEventSubscription, error) {
	var s domain.KickEventSubscription
	var createdAt, updatedAt, syncedAt string
	if err := scanner.Scan(
		&s.ID, &s.FollowedChannelID, &s.BroadcasterUserID, &s.EventType, &s.EventVersion, &s.Method,
		&s.KickSubscriptionID, &s.Status, &s.LatestSyncError, &createdAt, &updatedAt, &syncedAt,
	); err != nil {
		return domain.KickEventSubscription{}, err
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	s.SyncedAt = parseTime(syncedAt)
	return s, nil
}

func scanKickEventSubscriptions(rows *sql.Rows) ([]domain.KickEventSubscription, error) {
	var subs []domain.KickEventSubscription
	for rows.Next() {
		s, err := scanKickEventSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
