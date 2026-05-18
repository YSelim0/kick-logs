package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type RawEventClaimRepository struct {
	db *sql.DB
}

func NewRawEventClaimRepository(db *sql.DB) *RawEventClaimRepository {
	return &RawEventClaimRepository{db: db}
}

func (repo *RawEventClaimRepository) TryClaim(
	ctx context.Context,
	rawEventID string,
	workerID string,
	leaseDuration time.Duration,
) (bool, error) {
	if rawEventID == "" {
		return false, fmt.Errorf("raw event id is required")
	}
	if workerID == "" {
		return false, fmt.Errorf("raw event claim worker id is required")
	}
	if leaseDuration <= 0 {
		leaseDuration = 5 * time.Minute
	}

	now := time.Now().UTC()
	leaseExpiresAt := now.Add(leaseDuration)
	result, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO raw_event_claims (
			raw_event_id, worker_id, status, lease_expires_at, claimed_at, completed_at, updated_at
		) VALUES (?, ?, 'claimed', ?, ?, '', ?)
		ON CONFLICT(raw_event_id) DO UPDATE SET
			worker_id = excluded.worker_id,
			status = 'claimed',
			lease_expires_at = excluded.lease_expires_at,
			claimed_at = excluded.claimed_at,
			completed_at = '',
			updated_at = excluded.updated_at
		WHERE raw_event_claims.status != 'completed'
			AND (
				raw_event_claims.status = 'released'
				OR raw_event_claims.lease_expires_at <= ?
			);`,
		rawEventID,
		workerID,
		formatTime(leaseExpiresAt),
		formatTime(now),
		formatTime(now),
		formatTime(now),
	)
	if err != nil {
		return false, fmt.Errorf("claim raw event: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read raw event claim result: %w", err)
	}
	return rowsAffected > 0, nil
}

func (repo *RawEventClaimRepository) MarkCompleted(ctx context.Context, rawEventID string, workerID string) error {
	if rawEventID == "" {
		return fmt.Errorf("raw event id is required")
	}
	if workerID == "" {
		return fmt.Errorf("raw event claim worker id is required")
	}

	now := time.Now().UTC()
	if _, err := repo.db.ExecContext(
		ctx,
		`UPDATE raw_event_claims
		 SET status = 'completed',
			lease_expires_at = '',
			completed_at = ?,
			updated_at = ?
		 WHERE raw_event_id = ?
			AND worker_id = ?
			AND status = 'claimed';`,
		formatTime(now),
		formatTime(now),
		rawEventID,
		workerID,
	); err != nil {
		return fmt.Errorf("complete raw event claim: %w", err)
	}
	return nil
}

func (repo *RawEventClaimRepository) Release(ctx context.Context, rawEventID string, workerID string) error {
	if rawEventID == "" {
		return fmt.Errorf("raw event id is required")
	}
	if workerID == "" {
		return fmt.Errorf("raw event claim worker id is required")
	}

	now := time.Now().UTC()
	if _, err := repo.db.ExecContext(
		ctx,
		`UPDATE raw_event_claims
		 SET status = 'released',
			lease_expires_at = '',
			updated_at = ?
		 WHERE raw_event_id = ?
			AND worker_id = ?
			AND status = 'claimed';`,
		formatTime(now),
		rawEventID,
		workerID,
	); err != nil {
		return fmt.Errorf("release raw event claim: %w", err)
	}
	return nil
}
