package operations

import (
	"context"
	"database/sql"
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
	err = repo.sqliteDB.QueryRowContext(
		ctx,
		`SELECT last_seen_at FROM worker_heartbeats WHERE service_name = 'listener'`,
	).Scan(&lastSeenAt)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read listener heartbeat: %w", err)
	}
	if err == nil {
		parsed := parseTime(lastSeenAt)
		summary.Listener.LastSeenAt = parsed
		if !parsed.IsZero() {
			seconds := int64(time.Since(parsed).Seconds())
			summary.Listener.SecondsSinceLastSeen = seconds
			summary.Listener.IsFresh = seconds <= int64(repo.listenerStaleAfter)
		}
	}
	return nil
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

	statusRows, err := repo.clickHouse.Query(
		ctx,
		`SELECT status, count()
		 FROM raw_kick_events
		 GROUP BY status`,
	)
	if err != nil {
		return fmt.Errorf("query clickhouse raw event statuses: %w", err)
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count uint64
		if err := statusRows.Scan(&status, &count); err != nil {
			return fmt.Errorf("scan clickhouse raw event status: %w", err)
		}
		summary.RawEventStatusCounts[status] = int64(count)
	}
	if err := statusRows.Err(); err != nil {
		return fmt.Errorf("iterate clickhouse raw event statuses: %w", err)
	}

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
	if err := scanNullableTime(ctx, repo.clickHouse, "SELECT max(processed_at) FROM raw_kick_events WHERE processed_at IS NOT NULL", &summary.Timestamps.LatestRawEventProcessedAt); err != nil {
		return err
	}
	if err := scanNullableTime(ctx, repo.clickHouse, "SELECT min(received_at) FROM raw_kick_events WHERE status = 'pending'", &summary.Timestamps.OldestPendingRawEventReceivedAt); err != nil {
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

func scanNullableTime(ctx context.Context, conn driver.Conn, query string, target *time.Time) error {
	var value *time.Time
	if err := conn.QueryRow(ctx, query).Scan(&value); err != nil {
		return fmt.Errorf("scan clickhouse timestamp: %w", err)
	}
	if value != nil {
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
