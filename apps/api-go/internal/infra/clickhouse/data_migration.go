package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type DataMigrationRepository struct {
	conn      driver.Conn
	messages  *MessageRepository
	rawEvents *RawEventRepository
}

func NewDataMigrationRepository(conn driver.Conn) *DataMigrationRepository {
	return &DataMigrationRepository{
		conn:      conn,
		messages:  NewMessageRepository(conn),
		rawEvents: NewRawEventRepository(conn),
	}
}

func (repo *DataMigrationRepository) UpsertChatMessage(ctx context.Context, message domain.ChatMessage) (bool, error) {
	exists, err := repo.messages.ExistsByKickMessageID(ctx, message.KickMessageID)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := repo.messages.Insert(ctx, message); err != nil {
		return false, err
	}
	return true, nil
}

func (repo *DataMigrationRepository) UpsertRawEvent(ctx context.Context, event domain.RawKickEvent) (bool, error) {
	exists, err := repo.rawEventExists(ctx, event.ID)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := repo.rawEvents.InsertEvent(ctx, event); err != nil {
		return false, err
	}
	return true, nil
}

func (repo *DataMigrationRepository) UpsertRawEventAttempt(ctx context.Context, attempt domain.RawEventAttempt) (bool, error) {
	exists, err := repo.rawAttemptExists(ctx, attempt.ID)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := repo.rawEvents.InsertAttempt(ctx, attempt); err != nil {
		return false, err
	}
	return true, nil
}

func (repo *DataMigrationRepository) DataCounts(ctx context.Context) (domain.MigrationCounts, error) {
	counts := domain.MigrationCounts{}
	var chatMessages uint64
	var rawEvents uint64
	var rawEventAttempts uint64
	if err := repo.conn.QueryRow(ctx, "SELECT count() FROM chat_messages WHERE is_deleted = 0").Scan(&chatMessages); err != nil {
		return domain.MigrationCounts{}, fmt.Errorf("count clickhouse chat_messages: %w", err)
	}
	if err := repo.conn.QueryRow(ctx, "SELECT count() FROM raw_kick_events").Scan(&rawEvents); err != nil {
		return domain.MigrationCounts{}, fmt.Errorf("count clickhouse raw_kick_events: %w", err)
	}
	if err := repo.conn.QueryRow(ctx, "SELECT count() FROM raw_event_attempts").Scan(&rawEventAttempts); err != nil {
		return domain.MigrationCounts{}, fmt.Errorf("count clickhouse raw_event_attempts: %w", err)
	}
	counts.ChatMessages = int64(chatMessages)
	counts.RawEvents = int64(rawEvents)
	counts.RawEventAttempts = int64(rawEventAttempts)
	return counts, nil
}

func (repo *DataMigrationRepository) FindChatMessage(ctx context.Context, id int64, kickMessageID string) (domain.ChatMessage, error) {
	rows, err := repo.conn.Query(
		ctx,
		`SELECT
			id, kick_message_id, ifNull(channel_id, 0), ifNull(channel_kick_id, 0), ifNull(channel_chatroom_id, 0),
			channel_slug, channel_display_name, ifNull(channel_profile_image_url, ''),
			ifNull(channel_banner_image_url, ''), channel_public_url,
			ifNull(sender_id, 0), ifNull(sender_kick_id, 0), sender_username, sender_slug,
			ifNull(sender_display_color, ''), ifNull(sender_profile_image_url, ''),
			sender_public_url, sender_badges_json, message_type, content, emote_ids, emote_names, emote_tokens,
			emote_image_urls, ifNull(reply_to_sender, ''), ifNull(reply_to_content, ''),
			ifNull(reply_to_message_id, ''), ifNull(thread_parent_id, ''), reply_metadata_json,
			raw_payload_json, message_created_at, ingested_at
		 FROM chat_messages
		 WHERE id = ? AND kick_message_id = ? AND is_deleted = 0
		 ORDER BY ingested_at DESC
		 LIMIT 1`,
		id,
		kickMessageID,
	)
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("find migrated chat message: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return domain.ChatMessage{}, fmt.Errorf("migrated chat message id=%d kick_message_id=%q not found", id, kickMessageID)
	}
	message, err := scanMessage(rows)
	if err != nil {
		return domain.ChatMessage{}, err
	}
	return message, rows.Err()
}

func (repo *DataMigrationRepository) FindRawEvent(ctx context.Context, id string) (domain.RawKickEvent, error) {
	rows, err := repo.conn.Query(
		ctx,
		`SELECT
			id, channel_slug, event_type, event_name, ifNull(kick_message_id, ''),
			ifNull(chatroom_id, 0), ifNull(channel_id, 0), payload_json, metadata_json,
			status, toUInt16(0), received_at, processed_at, error_message
		 FROM raw_kick_events
		 WHERE id = ?
		 LIMIT 1`,
		id,
	)
	if err != nil {
		return domain.RawKickEvent{}, fmt.Errorf("find migrated raw event: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return domain.RawKickEvent{}, fmt.Errorf("migrated raw event id=%q not found", id)
	}
	event, err := scanRawKickEvent(rows)
	if err != nil {
		return domain.RawKickEvent{}, err
	}
	return event, rows.Err()
}

func (repo *DataMigrationRepository) rawEventExists(ctx context.Context, id string) (bool, error) {
	var count uint64
	if err := repo.conn.QueryRow(ctx, "SELECT count() FROM raw_kick_events WHERE id = ?", id).Scan(&count); err != nil {
		return false, fmt.Errorf("check raw event exists: %w", err)
	}
	return count > 0, nil
}

func (repo *DataMigrationRepository) rawAttemptExists(ctx context.Context, id string) (bool, error) {
	var count uint64
	if err := repo.conn.QueryRow(ctx, "SELECT count() FROM raw_event_attempts WHERE id = ?", id).Scan(&count); err != nil {
		return false, fmt.Errorf("check raw event attempt exists: %w", err)
	}
	return count > 0, nil
}
