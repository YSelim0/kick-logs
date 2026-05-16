package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Source struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Source, error) {
	normalized := NormalizeDSN(dsn)
	if normalized == "" {
		return nil, fmt.Errorf("postgres source dsn is required")
	}
	db, err := sql.Open("pgx", normalized)
	if err != nil {
		return nil, fmt.Errorf("open postgres source: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres source: %w", err)
	}
	return &Source{db: db}, nil
}

func NormalizeDSN(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	trimmed = strings.Replace(trimmed, "postgresql+asyncpg://", "postgresql://", 1)
	trimmed = strings.Replace(trimmed, "postgres+asyncpg://", "postgres://", 1)
	return trimmed
}

func (source *Source) Close() error {
	return source.db.Close()
}

func (source *Source) Counts(ctx context.Context) (domain.MigrationCounts, error) {
	counts := domain.MigrationCounts{}
	queries := map[string]*int64{
		"SELECT COUNT(*) FROM users":                   &counts.AdminUsers,
		"SELECT COUNT(*) FROM channels":                &counts.FollowedChannels,
		"SELECT COUNT(*) FROM senders":                 &counts.SenderProfiles,
		"SELECT COUNT(*) FROM data_retention_settings": &counts.RetentionSettings,
		"SELECT COUNT(*) FROM worker_heartbeats":       &counts.WorkerHeartbeats,
		"SELECT COUNT(*) FROM chat_messages":           &counts.ChatMessages,
		"SELECT COUNT(*) FROM raw_kick_events":         &counts.RawEvents,
		`SELECT COALESCE(SUM(
			CASE
				WHEN attempts > 0 THEN attempts
				WHEN status IN ('processed', 'failed', 'processing') THEN 1
				ELSE 0
			END
		), 0) FROM raw_kick_events`: &counts.RawEventAttempts,
	}
	for query, target := range queries {
		if err := source.db.QueryRowContext(ctx, query).Scan(target); err != nil {
			return domain.MigrationCounts{}, fmt.Errorf("read postgres source counts: %w", err)
		}
	}
	return counts, nil
}

func (source *Source) AdminUsers(ctx context.Context, limit int, offset int) ([]domain.AdminUser, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT id, email, password_hash, role, is_active, created_at, updated_at
		 FROM users
		 ORDER BY id ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.AdminUser, 0)
	for rows.Next() {
		var user domain.AdminUser
		var role string
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&role,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source user: %w", err)
		}
		user.Role = domain.AdminRole(role)
		user.CreatedAt = user.CreatedAt.UTC()
		user.UpdatedAt = user.UpdatedAt.UTC()
		users = append(users, user)
	}
	return users, rows.Err()
}

func (source *Source) FollowedChannels(ctx context.Context, limit int, offset int) ([]domain.FollowedChannel, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT id, kick_channel_id, kick_chatroom_id, slug, display_name,
		        profile_image_url, banner_image_url, is_enabled, raw_payload, created_at, updated_at
		 FROM channels
		 ORDER BY id ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source channels: %w", err)
	}
	defer rows.Close()

	channels := make([]domain.FollowedChannel, 0)
	for rows.Next() {
		var channel domain.FollowedChannel
		var kickChannelID sql.NullInt64
		var kickChatroomID sql.NullInt64
		var profileImageURL sql.NullString
		var bannerImageURL sql.NullString
		var raw jsonValue
		if err := rows.Scan(
			&channel.ID,
			&kickChannelID,
			&kickChatroomID,
			&channel.Slug,
			&channel.DisplayName,
			&profileImageURL,
			&bannerImageURL,
			&channel.IsEnabled,
			&raw,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source channel: %w", err)
		}
		channel.KickChannelID = nullableInt64(kickChannelID)
		channel.KickChatroomID = nullableInt64(kickChatroomID)
		channel.ProfileImageURL = nullableString(profileImageURL)
		channel.BannerImageURL = nullableString(bannerImageURL)
		channel.RawPayloadJSON = raw.ObjectJSON()
		channel.LastResolvedAt = channel.UpdatedAt
		channel.CreatedAt = channel.CreatedAt.UTC()
		channel.UpdatedAt = channel.UpdatedAt.UTC()
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (source *Source) SenderProfiles(ctx context.Context, limit int, offset int) ([]domain.SenderProfile, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT id, kick_user_id, username, slug, profile_image_url, last_seen_color,
		        raw_profile_payload, created_at, updated_at
		 FROM senders
		 ORDER BY id ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source senders: %w", err)
	}
	defer rows.Close()

	senders := make([]domain.SenderProfile, 0)
	for rows.Next() {
		var sender domain.SenderProfile
		var profileImageURL sql.NullString
		var lastSeenColor sql.NullString
		var raw jsonValue
		if err := rows.Scan(
			&sender.ID,
			&sender.KickUserID,
			&sender.Username,
			&sender.Slug,
			&profileImageURL,
			&lastSeenColor,
			&raw,
			&sender.CreatedAt,
			&sender.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source sender: %w", err)
		}
		sender.ProfileImageURL = nullableString(profileImageURL)
		sender.LastSeenColor = nullableString(lastSeenColor)
		sender.RawProfilePayloadJSON = raw.ObjectJSON()
		sender.LastSeenAt = sender.UpdatedAt.UTC()
		sender.CreatedAt = sender.CreatedAt.UTC()
		sender.UpdatedAt = sender.UpdatedAt.UTC()
		senders = append(senders, sender)
	}
	return senders, rows.Err()
}

func (source *Source) RetentionSettings(ctx context.Context, limit int, offset int) ([]domain.RetentionSettings, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT id, message_retention_days, raw_event_retention_days, created_at, updated_at
		 FROM data_retention_settings
		 ORDER BY id ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source retention settings: %w", err)
	}
	defer rows.Close()

	settings := make([]domain.RetentionSettings, 0)
	for rows.Next() {
		var item domain.RetentionSettings
		var messageDays sql.NullInt64
		var rawDays sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&messageDays,
			&rawDays,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source retention settings: %w", err)
		}
		item.MessageRetentionDays = nullableInt(messageDays)
		item.RawEventRetentionDays = nullableInt(rawDays)
		item.CreatedAt = item.CreatedAt.UTC()
		item.UpdatedAt = item.UpdatedAt.UTC()
		settings = append(settings, item)
	}
	return settings, rows.Err()
}

func (source *Source) WorkerHeartbeats(ctx context.Context, limit int, offset int) ([]domain.ListenerHeartbeat, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT service_name, last_seen_at, metadata, created_at, updated_at
		 FROM worker_heartbeats
		 ORDER BY service_name ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source worker heartbeats: %w", err)
	}
	defer rows.Close()

	heartbeats := make([]domain.ListenerHeartbeat, 0)
	for rows.Next() {
		var heartbeat domain.ListenerHeartbeat
		var raw jsonValue
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(
			&heartbeat.ServiceName,
			&heartbeat.LastSeenAt,
			&raw,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source worker heartbeat: %w", err)
		}
		heartbeat.LastSeenAt = heartbeat.LastSeenAt.UTC()
		heartbeat.MetadataJSON = raw.ObjectJSON()
		heartbeats = append(heartbeats, heartbeat)
	}
	return heartbeats, rows.Err()
}

func (source *Source) ChatMessages(ctx context.Context, limit int, offset int) ([]domain.ChatMessage, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT
			m.id, m.kick_message_id, m.channel_id, m.sender_id, m.chatroom_id, m.content,
			m.message_type, m.sender_username_snapshot, m.sender_slug_snapshot,
			m.sender_color_snapshot, m.sender_badges, m.emotes, m.reply_metadata,
			m.thread_parent_id, m.raw_payload, m.message_created_at, m.ingested_at,
			c.kick_channel_id, c.kick_chatroom_id, c.slug, c.display_name,
			c.profile_image_url, c.banner_image_url,
			s.kick_user_id, s.username, s.slug, s.profile_image_url
		 FROM chat_messages AS m
		 JOIN channels AS c ON c.id = m.channel_id
		 JOIN senders AS s ON s.id = m.sender_id
		 ORDER BY m.id ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source chat messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.ChatMessage, 0)
	for rows.Next() {
		var message domain.ChatMessage
		var senderColor sql.NullString
		var senderBadges jsonValue
		var emotes jsonValue
		var replyMetadata jsonValue
		var threadParentID sql.NullString
		var rawPayload jsonValue
		var channelKickID sql.NullInt64
		var channelChatroomID sql.NullInt64
		var channelProfileImageURL sql.NullString
		var channelBannerImageURL sql.NullString
		var senderUsername string
		var senderSlug string
		var senderProfileImageURL sql.NullString
		if err := rows.Scan(
			&message.ID,
			&message.KickMessageID,
			&message.ChannelID,
			&message.SenderID,
			&message.ChannelChatroomID,
			&message.Content,
			&message.MessageType,
			&message.SenderUsername,
			&message.SenderSlug,
			&senderColor,
			&senderBadges,
			&emotes,
			&replyMetadata,
			&threadParentID,
			&rawPayload,
			&message.MessageCreatedAt,
			&message.IngestedAt,
			&channelKickID,
			&channelChatroomID,
			&message.ChannelSlug,
			&message.ChannelDisplayName,
			&channelProfileImageURL,
			&channelBannerImageURL,
			&message.SenderKickID,
			&senderUsername,
			&senderSlug,
			&senderProfileImageURL,
		); err != nil {
			return nil, fmt.Errorf("scan source chat message: %w", err)
		}
		if message.ChannelChatroomID == 0 {
			message.ChannelChatroomID = nullableInt64(channelChatroomID)
		}
		message.ChannelKickID = nullableInt64(channelKickID)
		message.ChannelProfileImageURL = nullableString(channelProfileImageURL)
		message.ChannelBannerImageURL = nullableString(channelBannerImageURL)
		message.ChannelPublicURL = kickPublicURL(message.ChannelSlug)
		message.SenderDisplayColor = nullableString(senderColor)
		if message.SenderUsername == "" {
			message.SenderUsername = senderUsername
		}
		if message.SenderSlug == "" {
			message.SenderSlug = senderSlug
		}
		message.SenderProfileImageURL = nullableString(senderProfileImageURL)
		message.SenderPublicURL = kickPublicURL(message.SenderSlug)
		message.SenderBadgesJSON = senderBadges.ArrayJSON()
		message.Emotes = emotes.ChatEmotes()
		message.ReplyMetadataJSON = replyMetadata.ObjectJSON()
		message.ThreadParentID = nullableString(threadParentID)
		message.RawPayloadJSON = rawPayload.ObjectJSON()
		applyReplySnapshot(&message, replyMetadata)
		message.MessageCreatedAt = message.MessageCreatedAt.UTC()
		message.IngestedAt = message.IngestedAt.UTC()
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (source *Source) RawEvents(ctx context.Context, limit int, offset int) ([]domain.RawKickEvent, error) {
	rows, err := source.db.QueryContext(
		ctx,
		`SELECT
			r.id, r.event_name, r.kick_message_id, r.chatroom_id, r.kick_channel_id,
			r.channel_id, COALESCE(c.slug, ''), r.payload, r.status, r.attempts,
			r.received_at, r.processing_started_at, r.processed_at, r.last_error,
			r.metadata
		 FROM raw_kick_events AS r
		 LEFT JOIN channels AS c ON c.id = r.channel_id
		 ORDER BY r.id ASC
		 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query source raw events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.RawKickEvent, 0)
	for rows.Next() {
		var numericID int64
		var event domain.RawKickEvent
		var kickMessageID sql.NullString
		var chatroomID sql.NullInt64
		var kickChannelID sql.NullInt64
		var channelID sql.NullInt64
		var payload jsonValue
		var attempts int
		var processingStartedAt sql.NullTime
		var processedAt sql.NullTime
		var lastError sql.NullString
		var metadata jsonValue
		if err := rows.Scan(
			&numericID,
			&event.EventName,
			&kickMessageID,
			&chatroomID,
			&kickChannelID,
			&channelID,
			&event.ChannelSlug,
			&payload,
			&event.Status,
			&attempts,
			&event.ReceivedAt,
			&processingStartedAt,
			&processedAt,
			&lastError,
			&metadata,
		); err != nil {
			return nil, fmt.Errorf("scan source raw event: %w", err)
		}
		event.ID = fmt.Sprintf("%d", numericID)
		event.EventType = "pusher"
		event.KickMessageID = nullableString(kickMessageID)
		event.ChatroomID = nullableInt64(chatroomID)
		event.ChannelID = nullableInt64(channelID)
		event.PayloadJSON = payload.ObjectJSON()
		event.MetadataJSON = metadata.ObjectJSON()
		if attempts > int(^uint16(0)) {
			event.Attempts = ^uint16(0)
		} else {
			event.Attempts = uint16(attempts)
		}
		event.ReceivedAt = event.ReceivedAt.UTC()
		event.ProcessingStartedAt = nullableTime(processingStartedAt)
		event.ProcessedAt = nullableTime(processedAt)
		event.ErrorMessage = nullableString(lastError)
		events = append(events, event)
	}
	return events, rows.Err()
}

type jsonValue struct {
	raw []byte
}

func (value *jsonValue) Scan(src any) error {
	switch typed := src.(type) {
	case nil:
		value.raw = nil
	case []byte:
		value.raw = append(value.raw[:0], typed...)
	case string:
		value.raw = append(value.raw[:0], typed...)
	default:
		return fmt.Errorf("unsupported json source type %T", src)
	}
	return nil
}

func (value jsonValue) ObjectJSON() string {
	return canonicalJSON(value.raw, "{}")
}

func (value jsonValue) ArrayJSON() string {
	return canonicalJSON(value.raw, "[]")
}

func (value jsonValue) ChatEmotes() []domain.ChatEmote {
	var items []map[string]any
	if len(value.raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(value.raw, &items); err != nil {
		return nil
	}
	emotes := make([]domain.ChatEmote, 0, len(items))
	for _, item := range items {
		id := cleanText(item["id"])
		if id == "" {
			id = cleanText(item["kick_emote_id"])
		}
		name := cleanText(item["name"])
		token := cleanText(item["token"])
		if id == "" || name == "" || token == "" {
			continue
		}
		imageURL := cleanText(item["image_url"])
		if imageURL == "" {
			imageURL = "https://files.kick.com/emotes/" + id + "/fullsize"
		}
		emotes = append(emotes, domain.ChatEmote{
			ID:       id,
			Name:     name,
			Token:    token,
			ImageURL: imageURL,
		})
	}
	return emotes
}

func canonicalJSON(raw []byte, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fallback
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(encoded)
}

func applyReplySnapshot(message *domain.ChatMessage, metadata jsonValue) {
	var value map[string]any
	if len(metadata.raw) == 0 || json.Unmarshal(metadata.raw, &value) != nil {
		return
	}
	if sender, ok := value["original_sender"].(map[string]any); ok {
		message.ReplyToSender = cleanText(sender["username"])
	}
	if originalMessage, ok := value["original_message"].(map[string]any); ok {
		message.ReplyToContent = cleanText(originalMessage["content"])
	}
	message.ReplyToMessageID = firstCleanText(value["message_ref"], value["message_id"])
}

func nullableString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableInt64(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func nullableTime(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func kickPublicURL(slug string) string {
	if strings.TrimSpace(slug) == "" {
		return ""
	}
	return "https://kick.com/" + strings.TrimSpace(slug)
}

func firstCleanText(values ...any) string {
	for _, value := range values {
		if text := cleanText(value); text != "" {
			return text
		}
	}
	return ""
}

func cleanText(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
