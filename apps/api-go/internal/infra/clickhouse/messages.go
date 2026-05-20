package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type MessageRepository struct {
	conn driver.Conn
}

func NewMessageRepository(conn driver.Conn) *MessageRepository {
	return &MessageRepository{conn: conn}
}

func (repo *MessageRepository) Insert(ctx context.Context, message domain.ChatMessage) error {
	return repo.InsertMessagesBatch(ctx, []domain.ChatMessage{message})
}

func (repo *MessageRepository) InsertMessagesBatch(ctx context.Context, messages []domain.ChatMessage) error {
	if len(messages) == 0 {
		return nil
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
		return fmt.Errorf("prepare chat message insert: %w", err)
	}

	for index := range messages {
		message := messages[index]
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
			return fmt.Errorf("append chat message insert: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send chat message insert: %w", err)
	}
	return nil
}

func (repo *MessageRepository) SearchBasic(ctx context.Context, filter domain.MessageSearchFilter) ([]domain.ChatMessage, error) {
	return repo.Search(ctx, filter)
}

func (repo *MessageRepository) ExistsByKickMessageID(ctx context.Context, kickMessageID string) (bool, error) {
	var count uint64
	if err := repo.conn.QueryRow(
		ctx,
		"SELECT count() FROM chat_messages WHERE kick_message_id = ? AND is_deleted = 0",
		kickMessageID,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check chat message exists: %w", err)
	}
	return count > 0, nil
}

func (repo *MessageRepository) Search(ctx context.Context, filter domain.MessageSearchFilter) ([]domain.ChatMessage, error) {
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}

	where := []string{"is_deleted = 0"}
	args := make([]any, 0, 5)
	if filter.Sender != "" {
		sender := strings.ToLower(strings.TrimSpace(filter.Sender))
		where = append(where, "(sender_username_lower = ? OR sender_slug_lower = ?)")
		args = append(args, sender, sender)
	}
	if filter.Channel != "" {
		channel := strings.TrimSpace(filter.Channel)
		where = append(where, "(positionCaseInsensitive(channel_slug, ?) > 0 OR positionCaseInsensitive(channel_display_name, ?) > 0)")
		args = append(args, channel, channel)
	}
	if filter.Query != "" {
		where = append(where, "positionCaseInsensitive(content, ?) > 0")
		args = append(args, strings.TrimSpace(filter.Query))
	}
	if !filter.Start.IsZero() {
		where = append(where, "message_created_at >= ?")
		args = append(args, filter.Start.UTC())
	}
	if !filter.End.IsZero() {
		where = append(where, "message_created_at <= ?")
		args = append(args, filter.End.UTC())
	}
	if filter.ReplyOnly {
		where = append(where, "message_type = 'reply'")
	}
	if filter.EmoteOnly {
		where = append(where, "emote_count > 0")
	}
	if filter.Cursor != nil {
		where = append(where, "(message_created_at < ? OR (message_created_at = ? AND id < ?))")
		args = append(args, filter.Cursor.MessageCreatedAt.UTC(), filter.Cursor.MessageCreatedAt.UTC(), filter.Cursor.MessageID)
	}

	query := fmt.Sprintf(`SELECT
		id, kick_message_id, ifNull(channel_id, 0), ifNull(channel_kick_id, 0), ifNull(channel_chatroom_id, 0),
		channel_slug, channel_display_name, ifNull(channel_profile_image_url, ''),
		ifNull(channel_banner_image_url, ''), channel_public_url,
		ifNull(sender_id, 0), ifNull(sender_kick_id, 0), sender_username, sender_slug,
		ifNull(sender_display_color, ''), ifNull(sender_profile_image_url, ''),
		sender_public_url, sender_badges_json, message_type, content, emote_ids, emote_names, emote_tokens,
		emote_image_urls, ifNull(reply_to_sender, ''), ifNull(reply_to_content, ''),
		ifNull(reply_to_message_id, ''), ifNull(thread_parent_id, ''), reply_metadata_json,
		raw_payload_json, message_created_at, ingested_at
		FROM (
			SELECT
				id, kick_message_id, channel_id, channel_kick_id, channel_chatroom_id,
				channel_slug, channel_slug_lower, channel_display_name, channel_display_name_lower,
				channel_profile_image_url, channel_banner_image_url, channel_public_url,
				sender_id, sender_kick_id, sender_username, sender_username_lower, sender_slug,
				sender_slug_lower, sender_display_color, sender_profile_image_url, sender_public_url,
				sender_badges_json, message_type, content, emote_count, emote_ids, emote_names,
				emote_tokens, emote_image_urls, reply_to_sender, reply_to_content, reply_to_message_id,
				thread_parent_id, reply_metadata_json, raw_payload_json, message_created_at, ingested_at,
				is_deleted,
				row_number() OVER (
					PARTITION BY kick_message_id
					ORDER BY ingested_at DESC, message_created_at DESC, id DESC
				) AS message_rank
			FROM chat_messages
		)
		WHERE %s
			AND message_rank = 1
		ORDER BY message_created_at DESC, id DESC
		LIMIT ?`, strings.Join(where, " AND "))
	args = append(args, limit)

	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search chat messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.ChatMessage, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat messages: %w", err)
	}
	return messages, nil
}

func normalizeMessage(message *domain.ChatMessage) {
	if message.ChannelID == 0 {
		message.ChannelID = message.ChannelKickID
	}
	if message.SenderID == 0 {
		message.SenderID = message.SenderKickID
	}
	if message.MessageType == "" {
		message.MessageType = "message"
	}
	if message.SenderBadgesJSON == "" {
		message.SenderBadgesJSON = "[]"
	}
	if message.ReplyMetadataJSON == "" {
		message.ReplyMetadataJSON = "{}"
	}
	if message.RawPayloadJSON == "" {
		message.RawPayloadJSON = "{}"
	}
	if message.MessageCreatedAt.IsZero() {
		message.MessageCreatedAt = time.Now().UTC()
	}
	if message.IngestedAt.IsZero() {
		message.IngestedAt = time.Now().UTC()
	}
	message.MessageCreatedAt = message.MessageCreatedAt.UTC()
	message.IngestedAt = message.IngestedAt.UTC()
}

func splitEmotes(emotes []domain.ChatEmote) ([]string, []string, []string, []string) {
	ids := make([]string, 0, len(emotes))
	names := make([]string, 0, len(emotes))
	tokens := make([]string, 0, len(emotes))
	imageURLs := make([]string, 0, len(emotes))
	for _, emote := range emotes {
		ids = append(ids, emote.ID)
		names = append(names, emote.Name)
		tokens = append(tokens, emote.Token)
		imageURLs = append(imageURLs, emote.ImageURL)
	}
	return ids, names, tokens, imageURLs
}

func joinEmotes(ids []string, names []string, tokens []string, imageURLs []string) []domain.ChatEmote {
	count := len(ids)
	emotes := make([]domain.ChatEmote, 0, count)
	for index := 0; index < count; index++ {
		emote := domain.ChatEmote{ID: ids[index]}
		if index < len(names) {
			emote.Name = names[index]
		}
		if index < len(tokens) {
			emote.Token = tokens[index]
		}
		if index < len(imageURLs) {
			emote.ImageURL = imageURLs[index]
		}
		emotes = append(emotes, emote)
	}
	return emotes
}

type messageScanner interface {
	Scan(dest ...any) error
}

func scanMessage(scanner messageScanner) (domain.ChatMessage, error) {
	var message domain.ChatMessage
	var emoteIDs []string
	var emoteNames []string
	var emoteTokens []string
	var emoteImageURLs []string
	if err := scanner.Scan(
		&message.ID,
		&message.KickMessageID,
		&message.ChannelID,
		&message.ChannelKickID,
		&message.ChannelChatroomID,
		&message.ChannelSlug,
		&message.ChannelDisplayName,
		&message.ChannelProfileImageURL,
		&message.ChannelBannerImageURL,
		&message.ChannelPublicURL,
		&message.SenderID,
		&message.SenderKickID,
		&message.SenderUsername,
		&message.SenderSlug,
		&message.SenderDisplayColor,
		&message.SenderProfileImageURL,
		&message.SenderPublicURL,
		&message.SenderBadgesJSON,
		&message.MessageType,
		&message.Content,
		&emoteIDs,
		&emoteNames,
		&emoteTokens,
		&emoteImageURLs,
		&message.ReplyToSender,
		&message.ReplyToContent,
		&message.ReplyToMessageID,
		&message.ThreadParentID,
		&message.ReplyMetadataJSON,
		&message.RawPayloadJSON,
		&message.MessageCreatedAt,
		&message.IngestedAt,
	); err != nil {
		return domain.ChatMessage{}, fmt.Errorf("scan chat message: %w", err)
	}
	message.Emotes = joinEmotes(emoteIDs, emoteNames, emoteTokens, emoteImageURLs)
	return message, nil
}
