package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type WorkerHeartbeatRepository struct {
	db *sql.DB
}

func NewWorkerHeartbeatRepository(db *sql.DB) *WorkerHeartbeatRepository {
	return &WorkerHeartbeatRepository{db: db}
}

func (repo *WorkerHeartbeatRepository) Upsert(ctx context.Context, heartbeat domain.ListenerHeartbeat) error {
	if heartbeat.ServiceName == "" {
		return fmt.Errorf("heartbeat service name is required")
	}
	if heartbeat.MetadataJSON == "" {
		heartbeat.MetadataJSON = "{}"
	}
	now := time.Now().UTC()
	if heartbeat.LastSeenAt.IsZero() {
		heartbeat.LastSeenAt = now
	}

	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO worker_heartbeats (
			service_name, last_seen_at, metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(service_name) DO UPDATE SET
			last_seen_at = excluded.last_seen_at,
			metadata_json = excluded.metadata_json,
			updated_at = excluded.updated_at`,
		heartbeat.ServiceName,
		formatTime(heartbeat.LastSeenAt),
		heartbeat.MetadataJSON,
		formatTime(now),
		formatTime(now),
	)
	if err != nil {
		return fmt.Errorf("upsert worker heartbeat: %w", err)
	}
	return nil
}
