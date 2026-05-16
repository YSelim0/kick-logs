package datamanagement

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type Repository struct {
	sqliteDB   *sql.DB
	sqlitePath string
	clickHouse driver.Conn
}

func NewRepository(sqliteDB *sql.DB, sqlitePath string, clickHouse driver.Conn) *Repository {
	return &Repository{sqliteDB: sqliteDB, sqlitePath: sqlitePath, clickHouse: clickHouse}
}

func (repo *Repository) Summary(ctx context.Context) (domain.DataManagementSummary, error) {
	settings, err := repo.GetRetentionSettings(ctx)
	if err != nil {
		return domain.DataManagementSummary{}, err
	}
	counts, err := repo.counts(ctx)
	if err != nil {
		return domain.DataManagementSummary{}, err
	}
	tables, databaseBytes, err := repo.tableSizes(ctx)
	if err != nil {
		return domain.DataManagementSummary{}, err
	}
	return domain.DataManagementSummary{
		Counts:            counts,
		DatabaseBytes:     databaseBytes,
		Tables:            tables,
		RetentionSettings: settings,
	}, nil
}

func (repo *Repository) GetRetentionSettings(ctx context.Context) (domain.RetentionSettings, error) {
	if err := repo.ensureRetentionSettings(ctx); err != nil {
		return domain.RetentionSettings{}, err
	}
	row := repo.sqliteDB.QueryRowContext(
		ctx,
		`SELECT id, message_retention_days, raw_event_retention_days, created_at, updated_at
		 FROM retention_settings WHERE id = 1`,
	)
	return scanRetentionSettings(row)
}

func (repo *Repository) UpdateRetentionSettings(
	ctx context.Context,
	settings domain.RetentionSettings,
) (domain.RetentionSettings, error) {
	now := time.Now().UTC()
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now
	if _, err := repo.sqliteDB.ExecContext(
		ctx,
		`INSERT INTO retention_settings (
			id, message_retention_days, raw_event_retention_days, created_at, updated_at
		) VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			message_retention_days = excluded.message_retention_days,
			raw_event_retention_days = excluded.raw_event_retention_days,
			updated_at = excluded.updated_at`,
		nullableInt(settings.MessageRetentionDays),
		nullableInt(settings.RawEventRetentionDays),
		formatTime(settings.CreatedAt),
		formatTime(settings.UpdatedAt),
	); err != nil {
		return domain.RetentionSettings{}, fmt.Errorf("update retention settings: %w", err)
	}
	return repo.GetRetentionSettings(ctx)
}

func (repo *Repository) CountCleanup(
	ctx context.Context,
	criteria domain.DataCleanupCriteria,
) (domain.DataCleanupCounts, error) {
	if repo.clickHouse == nil {
		return domain.DataCleanupCounts{}, nil
	}
	switch criteria.Target {
	case domain.DataCleanupTargetOldMessages:
		if criteria.CutoffAt.IsZero() {
			return domain.DataCleanupCounts{}, nil
		}
		messages, err := repo.countClickHouse(ctx, oldMessagesWhere(), criteria.CutoffAt)
		return domain.DataCleanupCounts{Messages: messages}, err
	case domain.DataCleanupTargetOldRawEvents:
		if criteria.CutoffAt.IsZero() {
			return domain.DataCleanupCounts{}, nil
		}
		rawEvents, err := repo.countClickHouse(ctx, oldRawEventsWhere(), criteria.CutoffAt)
		return domain.DataCleanupCounts{RawEvents: rawEvents}, err
	case domain.DataCleanupTargetChannel:
		messages, err := repo.countClickHouse(ctx, channelMessagesWhere(), normalized(criteria.ChannelSlug), normalized(criteria.ChannelSlug))
		if err != nil {
			return domain.DataCleanupCounts{}, err
		}
		rawEvents, err := repo.countClickHouse(ctx, channelRawEventsWhere(), normalized(criteria.ChannelSlug))
		return domain.DataCleanupCounts{
			Messages:  messages,
			RawEvents: rawEvents,
		}, err
	case domain.DataCleanupTargetSender:
		normalizedSender := normalized(criteria.Sender)
		messages, err := repo.countClickHouse(ctx, senderMessagesWhere(), normalizedSender, normalizedSender)
		if err != nil {
			return domain.DataCleanupCounts{}, err
		}
		rawEvents, err := repo.countClickHouse(ctx, senderRawEventsWhere(), normalizedSender, normalizedSender)
		return domain.DataCleanupCounts{
			Messages:  messages,
			RawEvents: rawEvents,
		}, err
	default:
		return domain.DataCleanupCounts{}, fmt.Errorf("unsupported cleanup target %q", criteria.Target)
	}
}

func (repo *Repository) ExecuteCleanup(
	ctx context.Context,
	criteria domain.DataCleanupCriteria,
) (domain.DataCleanupCounts, error) {
	affected, err := repo.CountCleanup(ctx, criteria)
	if err != nil {
		return domain.DataCleanupCounts{}, err
	}
	if repo.clickHouse == nil || affected.Total() == 0 {
		return affected, nil
	}

	switch criteria.Target {
	case domain.DataCleanupTargetOldMessages:
		if !criteria.CutoffAt.IsZero() {
			err = repo.deleteWhere(ctx, "chat_messages", oldMessagesWhere(), criteria.CutoffAt)
		}
	case domain.DataCleanupTargetOldRawEvents:
		if !criteria.CutoffAt.IsZero() {
			err = repo.deleteRawAttemptsWhere(ctx, oldRawEventsWhere(), criteria.CutoffAt)
			if err == nil {
				err = repo.deleteWhere(ctx, "raw_kick_events", oldRawEventsWhere(), criteria.CutoffAt)
			}
		}
	case domain.DataCleanupTargetChannel:
		channel := normalized(criteria.ChannelSlug)
		err = repo.deleteRawAttemptsWhere(ctx, channelRawEventsWhere(), channel)
		if err == nil {
			err = repo.deleteWhere(ctx, "raw_kick_events", channelRawEventsWhere(), channel)
		}
		if err == nil {
			err = repo.deleteWhere(ctx, "chat_messages", channelMessagesWhere(), channel, channel)
		}
	case domain.DataCleanupTargetSender:
		sender := normalized(criteria.Sender)
		err = repo.deleteRawAttemptsWhere(ctx, senderRawEventsWhere(), sender, sender)
		if err == nil {
			err = repo.deleteWhere(ctx, "raw_kick_events", senderRawEventsWhere(), sender, sender)
		}
		if err == nil {
			err = repo.deleteWhere(ctx, "chat_messages", senderMessagesWhere(), sender, sender)
		}
	default:
		err = fmt.Errorf("unsupported cleanup target %q", criteria.Target)
	}
	if err != nil {
		return domain.DataCleanupCounts{}, err
	}
	return affected, nil
}

func (repo *Repository) counts(ctx context.Context) (domain.DataManagementCounts, error) {
	channels, err := countSQLiteRows(ctx, repo.sqliteDB, "followed_channels")
	if err != nil {
		return domain.DataManagementCounts{}, err
	}
	senders, err := countSQLiteRows(ctx, repo.sqliteDB, "sender_profiles")
	if err != nil {
		return domain.DataManagementCounts{}, err
	}

	counts := domain.DataManagementCounts{Channels: channels, Senders: senders}
	if repo.clickHouse == nil {
		return counts, nil
	}
	messages, err := repo.countClickHouse(ctx, "FROM chat_messages WHERE is_deleted = 0")
	if err != nil {
		return domain.DataManagementCounts{}, err
	}
	rawEvents, err := repo.countClickHouse(ctx, "FROM raw_kick_events")
	if err != nil {
		return domain.DataManagementCounts{}, err
	}
	counts.Messages = messages
	counts.RawEvents = rawEvents
	return counts, nil
}

func (repo *Repository) tableSizes(ctx context.Context) ([]domain.TableSize, int64, error) {
	tables := make([]domain.TableSize, 0)
	databaseBytes := int64(0)
	if stat, err := os.Stat(repo.sqlitePath); err == nil {
		databaseBytes += stat.Size()
		tables = append(tables, domain.TableSize{
			Name:        "_sqlite_database_file",
			BytesOnDisk: stat.Size(),
		})
	}
	for _, table := range []string{"admin_users", "followed_channels", "sender_profiles", "retention_settings", "worker_heartbeats", "data_migration_runs"} {
		rows, err := countSQLiteRows(ctx, repo.sqliteDB, table)
		if err != nil {
			return nil, 0, err
		}
		tables = append(tables, domain.TableSize{Name: table, Rows: rows})
	}

	if repo.clickHouse == nil {
		return tables, databaseBytes, nil
	}

	rows, err := repo.clickHouse.Query(
		ctx,
		`SELECT table, sum(rows), sum(bytes_on_disk)
		 FROM system.parts
		 WHERE active AND database = currentDatabase()
		 GROUP BY table
		 ORDER BY table ASC`,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query clickhouse table sizes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var table domain.TableSize
		var rowCount uint64
		var bytesOnDisk uint64
		if err := rows.Scan(&table.Name, &rowCount, &bytesOnDisk); err != nil {
			return nil, 0, fmt.Errorf("scan clickhouse table size: %w", err)
		}
		table.Rows = int64(rowCount)
		table.BytesOnDisk = int64(bytesOnDisk)
		databaseBytes += table.BytesOnDisk
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate clickhouse table sizes: %w", err)
	}
	return tables, databaseBytes, nil
}

func (repo *Repository) countClickHouse(ctx context.Context, fromAndWhere string, args ...any) (int64, error) {
	var count uint64
	if err := repo.clickHouse.QueryRow(ctx, "SELECT count() "+fromAndWhere, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count clickhouse rows: %w", err)
	}
	return int64(count), nil
}

func (repo *Repository) deleteWhere(ctx context.Context, table string, fromAndWhere string, args ...any) error {
	where := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(fromAndWhere), "FROM "+table+" WHERE"))
	if where == "" || where == fromAndWhere {
		return fmt.Errorf("invalid delete predicate for %s", table)
	}
	query := fmt.Sprintf("ALTER TABLE %s DELETE WHERE %s SETTINGS mutations_sync = 1", table, where)
	if err := repo.clickHouse.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete clickhouse %s rows: %w", table, err)
	}
	return nil
}

func (repo *Repository) deleteRawAttemptsWhere(ctx context.Context, rawEventFromAndWhere string, args ...any) error {
	subquery := "SELECT id " + rawEventFromAndWhere
	query := fmt.Sprintf(
		"ALTER TABLE raw_event_attempts DELETE WHERE raw_event_id IN (%s) SETTINGS mutations_sync = 1",
		subquery,
	)
	if err := repo.clickHouse.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete clickhouse raw event attempts: %w", err)
	}
	return nil
}

func (repo *Repository) ensureRetentionSettings(ctx context.Context) error {
	now := time.Now().UTC()
	if _, err := repo.sqliteDB.ExecContext(
		ctx,
		`INSERT INTO retention_settings (
			id, message_retention_days, raw_event_retention_days, created_at, updated_at
		) VALUES (1, NULL, NULL, ?, ?)
		ON CONFLICT(id) DO NOTHING`,
		formatTime(now),
		formatTime(now),
	); err != nil {
		return fmt.Errorf("ensure retention settings: %w", err)
	}
	return nil
}

type retentionScanner interface {
	Scan(dest ...any) error
}

func scanRetentionSettings(scanner retentionScanner) (domain.RetentionSettings, error) {
	var settings domain.RetentionSettings
	var messageDays sql.NullInt64
	var rawEventDays sql.NullInt64
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&settings.ID,
		&messageDays,
		&rawEventDays,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.RetentionSettings{}, err
	}
	settings.MessageRetentionDays = scanNullableInt(messageDays)
	settings.RawEventRetentionDays = scanNullableInt(rawEventDays)
	settings.CreatedAt = parseTime(createdAt)
	settings.UpdatedAt = parseTime(updatedAt)
	return settings, nil
}

func countSQLiteRows(ctx context.Context, db *sql.DB, table string) (int64, error) {
	var count int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count sqlite table %s: %w", table, err)
	}
	return count, nil
}

func oldMessagesWhere() string {
	return "FROM chat_messages WHERE message_created_at < ? AND is_deleted = 0"
}

func oldRawEventsWhere() string {
	return "FROM raw_kick_events WHERE received_at < ?"
}

func channelMessagesWhere() string {
	return "FROM chat_messages WHERE (channel_slug_lower = ? OR channel_display_name_lower = ?) AND is_deleted = 0"
}

func channelRawEventsWhere() string {
	return "FROM raw_kick_events WHERE lower(channel_slug) = ?"
}

func senderMessagesWhere() string {
	return "FROM chat_messages WHERE (sender_username_lower = ? OR sender_slug_lower = ?) AND is_deleted = 0"
}

func senderRawEventsWhere() string {
	return "FROM raw_kick_events WHERE lower(JSONExtractString(payload_json, 'sender', 'username')) = ? OR lower(JSONExtractString(payload_json, 'sender', 'slug')) = ?"
}

func normalized(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func nullableInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func scanNullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
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
