package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	inserted, err := repo.UpsertChatMessages(ctx, []domain.ChatMessage{message})
	if err != nil {
		return false, err
	}
	return inserted == 1, nil
}

func (repo *DataMigrationRepository) UpsertChatMessages(ctx context.Context, messages []domain.ChatMessage) (int64, error) {
	if len(messages) == 0 {
		return 0, nil
	}
	keys := make([]string, 0, len(messages))
	for _, message := range messages {
		keys = append(keys, message.KickMessageID)
	}
	existing, err := repo.existingStrings(ctx, "chat_messages", "kick_message_id", keys, "is_deleted = 0")
	if err != nil {
		return 0, err
	}

	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO chat_messages (
		id, kick_message_id, channel_id, channel_kick_id, channel_chatroom_id, channel_slug,
		channel_slug_lower, channel_display_name, channel_display_name_lower,
		channel_profile_image_url, channel_banner_image_url, channel_public_url,
		sender_id, sender_kick_id, sender_username,
		sender_username_lower, sender_slug, sender_slug_lower, sender_display_color,
		sender_profile_image_url, sender_public_url, sender_badges_json, message_type, content, content_lower,
		emote_count, emote_ids, emote_names, emote_tokens, emote_image_urls,
		reply_to_sender, reply_to_sender_lower, reply_to_content, reply_to_message_id,
		thread_parent_id, reply_metadata_json, raw_payload_json, message_created_at, ingested_at
	)`)
	if err != nil {
		return 0, fmt.Errorf("prepare migrated chat message insert: %w", err)
	}

	var inserted int64
	for _, message := range messages {
		if _, ok := existing[message.KickMessageID]; ok {
			continue
		}
		if err := appendChatMessage(batch, message); err != nil {
			return inserted, err
		}
		inserted++
	}
	if inserted == 0 {
		return 0, nil
	}
	if err := batch.Send(); err != nil {
		return inserted, fmt.Errorf("send migrated chat message insert: %w", err)
	}
	return inserted, nil
}

func (repo *DataMigrationRepository) UpsertRawEvent(ctx context.Context, event domain.RawKickEvent) (bool, error) {
	inserted, err := repo.UpsertRawEvents(ctx, []domain.RawKickEvent{event})
	if err != nil {
		return false, err
	}
	return inserted == 1, nil
}

func (repo *DataMigrationRepository) UpsertRawEvents(ctx context.Context, events []domain.RawKickEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}
	keys := make([]string, 0, len(events))
	for _, event := range events {
		keys = append(keys, event.ID)
	}
	existing, err := repo.existingStrings(ctx, "raw_kick_events", "id", keys, "")
	if err != nil {
		return 0, err
	}

	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO raw_kick_events (
		id, channel_slug, event_type, event_name, kick_message_id, chatroom_id, channel_id,
		payload_json, metadata_json, status, received_at, processed_at, error_message
	)`)
	if err != nil {
		return 0, fmt.Errorf("prepare migrated raw event insert: %w", err)
	}

	var inserted int64
	for _, event := range events {
		if _, ok := existing[event.ID]; ok {
			continue
		}
		if err := appendRawEvent(batch, event); err != nil {
			return inserted, err
		}
		inserted++
	}
	if inserted == 0 {
		return 0, nil
	}
	if err := batch.Send(); err != nil {
		return inserted, fmt.Errorf("send migrated raw event insert: %w", err)
	}
	return inserted, nil
}

func (repo *DataMigrationRepository) UpsertRawEventAttempt(ctx context.Context, attempt domain.RawEventAttempt) (bool, error) {
	inserted, err := repo.UpsertRawEventAttempts(ctx, []domain.RawEventAttempt{attempt})
	if err != nil {
		return false, err
	}
	return inserted == 1, nil
}

func (repo *DataMigrationRepository) UpsertRawEventAttempts(ctx context.Context, attempts []domain.RawEventAttempt) (int64, error) {
	if len(attempts) == 0 {
		return 0, nil
	}
	keys := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		keys = append(keys, attempt.ID)
	}
	existing, err := repo.existingStrings(ctx, "raw_event_attempts", "id", keys, "")
	if err != nil {
		return 0, err
	}

	batch, err := repo.conn.PrepareBatch(ctx, `INSERT INTO raw_event_attempts (
		id, raw_event_id, attempt, status, error_message, started_at, finished_at
	)`)
	if err != nil {
		return 0, fmt.Errorf("prepare migrated raw event attempt insert: %w", err)
	}

	var inserted int64
	for _, attempt := range attempts {
		if _, ok := existing[attempt.ID]; ok {
			continue
		}
		if err := appendRawEventAttempt(batch, attempt); err != nil {
			return inserted, err
		}
		inserted++
	}
	if inserted == 0 {
		return 0, nil
	}
	if err := batch.Send(); err != nil {
		return inserted, fmt.Errorf("send migrated raw event attempt insert: %w", err)
	}
	return inserted, nil
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

func (repo *DataMigrationRepository) existingStrings(
	ctx context.Context,
	table string,
	column string,
	values []string,
	extraWhere string,
) (map[string]struct{}, error) {
	existing := make(map[string]struct{})
	values = compactStrings(values)
	if len(values) == 0 {
		return existing, nil
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s IN (%s)", column, table, column, placeholders(len(values)))
	if extraWhere != "" {
		query += " AND " + extraWhere
	}
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load existing migrated rows from %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		existing[value] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return existing, nil
}

func appendChatMessage(batch driver.Batch, message domain.ChatMessage) error {
	normalizeMessage(&message)
	emoteIDs, emoteNames, emoteTokens, emoteImageURLs := splitEmotes(message.Emotes)
	if err := batch.Append(
		message.ID,
		message.KickMessageID,
		nullableInt64(message.ChannelID),
		nullableInt64(message.ChannelKickID),
		nullableInt64(message.ChannelChatroomID),
		message.ChannelSlug,
		strings.ToLower(message.ChannelSlug),
		message.ChannelDisplayName,
		strings.ToLower(message.ChannelDisplayName),
		nullableString(message.ChannelProfileImageURL),
		nullableString(message.ChannelBannerImageURL),
		message.ChannelPublicURL,
		nullableInt64(message.SenderID),
		nullableInt64(message.SenderKickID),
		message.SenderUsername,
		strings.ToLower(message.SenderUsername),
		message.SenderSlug,
		strings.ToLower(message.SenderSlug),
		nullableString(message.SenderDisplayColor),
		nullableString(message.SenderProfileImageURL),
		message.SenderPublicURL,
		message.SenderBadgesJSON,
		message.MessageType,
		message.Content,
		strings.ToLower(message.Content),
		uint16(len(message.Emotes)),
		emoteIDs,
		emoteNames,
		emoteTokens,
		emoteImageURLs,
		nullableString(message.ReplyToSender),
		nullableString(strings.ToLower(message.ReplyToSender)),
		nullableString(message.ReplyToContent),
		nullableString(message.ReplyToMessageID),
		nullableString(message.ThreadParentID),
		message.ReplyMetadataJSON,
		message.RawPayloadJSON,
		message.MessageCreatedAt,
		message.IngestedAt,
	); err != nil {
		return fmt.Errorf("append migrated chat message insert: %w", err)
	}
	return nil
}

func appendRawEvent(batch driver.Batch, event domain.RawKickEvent) error {
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
		return fmt.Errorf("append migrated raw event insert: %w", err)
	}
	return nil
}

func appendRawEventAttempt(batch driver.Batch, attempt domain.RawEventAttempt) error {
	if attempt.Status == "" {
		attempt.Status = "started"
	}
	if attempt.StartedAt.IsZero() {
		attempt.StartedAt = time.Now().UTC()
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
		return fmt.Errorf("append migrated raw event attempt insert: %w", err)
	}
	return nil
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func compactStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	compacted := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		compacted = append(compacted, value)
	}
	return compacted
}
