package operations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type Repository struct {
	sqliteDB           *sql.DB
	sqlitePath         string
	clickHouse         driver.Conn
	listenerStaleAfter int
}

func NewRepository(
	sqliteDB *sql.DB,
	sqlitePath string,
	clickHouse driver.Conn,
	listenerStaleAfter int,
) *Repository {
	return &Repository{
		sqliteDB:           sqliteDB,
		sqlitePath:         sqlitePath,
		clickHouse:         clickHouse,
		listenerStaleAfter: listenerStaleAfter,
	}
}

func (repo *Repository) Summary(ctx context.Context) (domain.OperationsSummary, error) {
	summary := domain.OperationsSummary{
		RawEventStatusCounts: map[string]int64{},
		StorageTables:        []domain.TableSize{},
		Listener: domain.ListenerHeartbeat{
			ServiceName:       "listener",
			StaleAfterSeconds: repo.listenerStaleAfter,
			IsFresh:           false,
		},
	}

	if err := repo.fillSQLiteSummary(ctx, &summary); err != nil {
		return domain.OperationsSummary{}, err
	}
	if repo.clickHouse != nil {
		if err := repo.fillClickHouseSummary(ctx, &summary); err != nil {
			return domain.OperationsSummary{}, err
		}
	}
	return summary, nil
}

func (repo *Repository) fillSQLiteSummary(ctx context.Context, summary *domain.OperationsSummary) error {
	channels, err := countSQLiteRows(ctx, repo.sqliteDB, "followed_channels", "")
	if err != nil {
		return err
	}
	enabledChannels, err := countSQLiteRows(ctx, repo.sqliteDB, "followed_channels", "WHERE is_enabled = 1")
	if err != nil {
		return err
	}
	senders, err := countSQLiteRows(ctx, repo.sqliteDB, "sender_profiles", "")
	if err != nil {
		return err
	}

	summary.Counts.Channels = channels
	summary.Counts.EnabledChannels = enabledChannels
	summary.Counts.Senders = senders

	if stat, err := os.Stat(repo.sqlitePath); err == nil {
		summary.StorageDatabaseBytes += stat.Size()
		summary.StorageTables = append(summary.StorageTables, domain.TableSize{
			Name:        "_sqlite_database_file",
			Rows:        0,
			BytesOnDisk: stat.Size(),
		})
	}

	var lastSeenAt string
	var metadataJSON string
	err = repo.sqliteDB.QueryRowContext(
		ctx,
		`SELECT last_seen_at, metadata_json FROM worker_heartbeats WHERE service_name = 'listener'`,
	).Scan(&lastSeenAt, &metadataJSON)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read listener heartbeat: %w", err)
	}
	if err == nil {
		parsed := parseTime(lastSeenAt)
		summary.Listener.LastSeenAt = parsed
		summary.Listener.MetadataJSON = metadataJSON
		if !parsed.IsZero() {
			seconds := int64(time.Since(parsed).Seconds())
			summary.Listener.SecondsSinceLastSeen = seconds
			summary.Listener.IsFresh = seconds <= int64(repo.listenerStaleAfter)
		}
		if metadataJSON != "" {
			applyIngestionMetadata(metadataJSON, &summary.Ingestion)
		}
	}

	queueDepth, err := repo.queueDepth(ctx)
	if err != nil {
		return err
	}
	summary.Ingestion.QueueDepth = queueDepth

	oldestAge, err := repo.oldestPendingAgeSeconds(ctx)
	if err != nil {
		return err
	}
	summary.Ingestion.OldestPendingAgeSeconds = oldestAge

	return nil
}

func (repo *Repository) queueDepth(ctx context.Context) (int64, error) {
	var count int64
	if err := repo.sqliteDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM raw_event_queue WHERE status IN ('pending', 'claimed')`,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count raw_event_queue: %w", err)
	}
	return count, nil
}

func (repo *Repository) oldestPendingAgeSeconds(ctx context.Context) (int64, error) {
	var enqueuedAt sql.NullString
	if err := repo.sqliteDB.QueryRowContext(
		ctx,
		`SELECT MIN(enqueued_at) FROM raw_event_queue WHERE status IN ('pending', 'claimed')`,
	).Scan(&enqueuedAt); err != nil {
		return 0, fmt.Errorf("query oldest pending: %w", err)
	}
	if !enqueuedAt.Valid || enqueuedAt.String == "" {
		return 0, nil
	}
	parsed := parseTime(enqueuedAt.String)
	if parsed.IsZero() {
		return 0, nil
	}
	age := int64(time.Since(parsed).Seconds())
	if age < 0 {
		return 0, nil
	}
	return age, nil
}

func applyIngestionMetadata(metadataJSON string, ingestion *domain.IngestionHealth) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &raw); err != nil {
		return
	}
	ingestion.WriteQueueDepth = readInt64(raw, "write_queue_depth")
	ingestion.WriteQueueHighWater = readInt64(raw, "write_queue_high_water_mark")
	ingestion.WriteDropCount = readInt64(raw, "write_drop_count")
	ingestion.WriteFlushCount = readInt64(raw, "write_flush_count")
	ingestion.LastFlushSize = readInt64(raw, "last_flush_size")
	ingestion.LastFlushMillis = readInt64(raw, "last_flush_millis")
	ingestion.ClickHouseFailures = readInt64(raw, "clickhouse_insert_failures")
	ingestion.QueueEnqueueFailures = readInt64(raw, "queue_enqueue_failures")
	if state, ok := raw["breaker_state"].(string); ok {
		ingestion.BreakerState = state
	}
	ingestion.BreakerCurrentDelayMS = readInt64(raw, "breaker_current_delay_ms")
}

func readInt64(raw map[string]any, key string) int64 {
	value, ok := raw[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	}
	return 0
}

func (repo *Repository) fillClickHouseSummary(ctx context.Context, summary *domain.OperationsSummary) error {
	var messages uint64
	if err := repo.clickHouse.QueryRow(ctx, "SELECT count() FROM chat_messages WHERE is_deleted = 0").Scan(&messages); err != nil {
		return fmt.Errorf("count clickhouse messages: %w", err)
	}
	summary.Counts.Messages = int64(messages)

	var rawEvents uint64
	if err := repo.clickHouse.QueryRow(ctx, "SELECT count() FROM raw_kick_events").Scan(&rawEvents); err != nil {
		return fmt.Errorf("count clickhouse raw events: %w", err)
	}
	summary.Counts.RawEvents = int64(rawEvents)

	processed, err := countClickHouseRows(
		ctx,
		repo.clickHouse,
		`SELECT uniqExact(raw_event_id) FROM raw_event_attempts WHERE status = 'processed'`,
	)
	if err != nil {
		return err
	}
	failed, err := countClickHouseRows(
		ctx,
		repo.clickHouse,
		`SELECT uniqExact(raw_event_id)
		 FROM raw_event_attempts
		 WHERE status = 'failed'
		   AND raw_event_id NOT IN (
			SELECT raw_event_id FROM raw_event_attempts WHERE status = 'processed'
		   )`,
	)
	if err != nil {
		return err
	}
	pending := int64(rawEvents) - processed - failed
	if pending < 0 {
		pending = 0
	}
	summary.RawEventStatusCounts["pending"] = pending
	summary.RawEventStatusCounts["processed"] = processed
	summary.RawEventStatusCounts["failed"] = failed

	sizeRows, err := repo.clickHouse.Query(
		ctx,
		`SELECT table, sum(rows), sum(bytes_on_disk)
		 FROM system.parts
		 WHERE active AND database = currentDatabase()
		 GROUP BY table
		 ORDER BY table ASC`,
	)
	if err != nil {
		return fmt.Errorf("query clickhouse table sizes: %w", err)
	}
	defer sizeRows.Close()
	for sizeRows.Next() {
		var table string
		var rows uint64
		var bytes uint64
		if err := sizeRows.Scan(&table, &rows, &bytes); err != nil {
			return fmt.Errorf("scan clickhouse table size: %w", err)
		}
		summary.StorageDatabaseBytes += int64(bytes)
		summary.StorageTables = append(summary.StorageTables, domain.TableSize{
			Name:        table,
			Rows:        int64(rows),
			BytesOnDisk: int64(bytes),
		})
	}
	if err := sizeRows.Err(); err != nil {
		return fmt.Errorf("iterate clickhouse table sizes: %w", err)
	}

	if err := scanNullableTime(ctx, repo.clickHouse, "SELECT max(message_created_at) FROM chat_messages WHERE is_deleted = 0", &summary.Timestamps.LatestMessageAt); err != nil {
		return err
	}
	if err := scanNullableTime(ctx, repo.clickHouse, "SELECT max(received_at) FROM raw_kick_events", &summary.Timestamps.LatestRawEventReceivedAt); err != nil {
		return err
	}
	if err := scanNullableTime(ctx, repo.clickHouse, "SELECT max(finished_at) FROM raw_event_attempts WHERE status = 'processed' AND finished_at IS NOT NULL", &summary.Timestamps.LatestRawEventProcessedAt); err != nil {
		return err
	}
	if err := scanNullableTime(
		ctx,
		repo.clickHouse,
		`SELECT min(received_at)
		 FROM raw_kick_events
		 WHERE id NOT IN (
			SELECT raw_event_id FROM raw_event_attempts WHERE status = 'processed'
		 )
		   AND id NOT IN (
			SELECT raw_event_id FROM raw_event_attempts WHERE status = 'failed'
		 )`,
		&summary.Timestamps.OldestPendingRawEventReceivedAt,
	); err != nil {
		return err
	}
	return nil
}

func countSQLiteRows(ctx context.Context, db *sql.DB, table string, suffix string) (int64, error) {
	var count int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s %s", table, suffix)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count sqlite table %s: %w", table, err)
	}
	return count, nil
}

func countClickHouseRows(ctx context.Context, conn driver.Conn, query string) (int64, error) {
	var count uint64
	if err := conn.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count clickhouse rows: %w", err)
	}
	return int64(count), nil
}

func scanNullableTime(ctx context.Context, conn driver.Conn, query string, target *time.Time) error {
	var value *time.Time
	if err := conn.QueryRow(ctx, query).Scan(&value); err != nil {
		return fmt.Errorf("scan clickhouse timestamp: %w", err)
	}
	if value != nil && value.Unix() > 0 {
		*target = value.UTC()
	}
	return nil
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
