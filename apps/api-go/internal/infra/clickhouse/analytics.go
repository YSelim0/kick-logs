package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type AnalyticsRepository struct {
	conn driver.Conn
}

func NewAnalyticsRepository(conn driver.Conn) *AnalyticsRepository {
	return &AnalyticsRepository{conn: conn}
}

func (repo *AnalyticsRepository) Overview(
	ctx context.Context,
	filter domain.AnalyticsFilter,
) (domain.AnalyticsOverview, error) {
	where, args := analyticsWhere(filter)
	query := fmt.Sprintf(`SELECT
		count(),
		uniqExactIf(
			%s,
			ifNull(sender_kick_id, 0) > 0 OR sender_slug_lower != '' OR sender_username_lower != ''
		),
		uniqExactIf(channel_id, isNotNull(channel_id)),
		sum(emote_count),
		min(message_created_at),
		max(message_created_at)
		FROM chat_messages FINAL
		WHERE %s`, senderIdentitySQL(), where)

	var totalMessages uint64
	var totalSenders uint64
	var totalChannels uint64
	var totalEmoteUsages uint64
	var firstMessageAt time.Time
	var latestMessageAt time.Time
	if err := repo.conn.QueryRow(ctx, query, args...).Scan(
		&totalMessages,
		&totalSenders,
		&totalChannels,
		&totalEmoteUsages,
		&firstMessageAt,
		&latestMessageAt,
	); err != nil {
		return domain.AnalyticsOverview{}, fmt.Errorf("query analytics overview: %w", err)
	}

	overview := domain.AnalyticsOverview{
		TotalMessages:    int64(totalMessages),
		TotalSenders:     int64(totalSenders),
		TotalChannels:    int64(totalChannels),
		TotalEmoteUsages: int64(totalEmoteUsages),
	}
	if totalMessages > 0 {
		overview.FirstMessageAt = firstMessageAt.UTC()
		overview.LatestMessageAt = latestMessageAt.UTC()
	}
	return overview, nil
}

func (repo *AnalyticsRepository) MessageVolume(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	bucket domain.AnalyticsBucket,
) ([]domain.MessageVolumePoint, error) {
	where, args := analyticsWhere(filter)
	bucketFunction := "toStartOfDay"
	if bucket == domain.AnalyticsBucketHour {
		bucketFunction = "toStartOfHour"
	}
	query := fmt.Sprintf(`SELECT
		%s(message_created_at) AS bucket_start,
		count() AS message_count
		FROM chat_messages FINAL
		WHERE %s
		GROUP BY bucket_start
		ORDER BY bucket_start ASC`, bucketFunction, where)

	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query message volume: %w", err)
	}
	defer rows.Close()

	points := make([]domain.MessageVolumePoint, 0)
	for rows.Next() {
		var point domain.MessageVolumePoint
		var messageCount uint64
		if err := rows.Scan(&point.BucketStart, &messageCount); err != nil {
			return nil, fmt.Errorf("scan message volume: %w", err)
		}
		point.BucketStart = point.BucketStart.UTC()
		point.MessageCount = int64(messageCount)
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message volume: %w", err)
	}
	return points, nil
}

func (repo *AnalyticsRepository) TopSenders(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopSenderAnalytics, error) {
	where, args := topSendersWhere(filter)
	query := fmt.Sprintf(`SELECT
		argMax(ifNull(sender_id, 0), tuple(message_created_at, ingested_at, id)) AS sender_id,
		argMax(ifNull(sender_kick_id, 0), tuple(message_created_at, ingested_at, id)) AS kick_user_id,
		argMax(sender_username, tuple(message_created_at, ingested_at, id)) AS username,
		argMax(sender_slug, tuple(message_created_at, ingested_at, id)) AS slug,
		argMax(ifNull(sender_profile_image_url, ''), tuple(message_created_at, ingested_at, id)) AS profile_image_url,
		count() AS message_count,
		min(message_created_at) AS first_message_at,
		max(message_created_at) AS latest_message_at
		FROM (
			SELECT
				%s AS sender_identity,
				id,
				kick_message_id,
				sender_id,
				sender_kick_id,
				sender_username,
				sender_slug,
				sender_profile_image_url,
				message_created_at,
				ingested_at
			FROM chat_messages FINAL
			WHERE %s
		)
		GROUP BY sender_identity
		ORDER BY message_count DESC, latest_message_at DESC, slug ASC
		LIMIT ?`, senderIdentitySQL(), where)
	args = append(args, limitOrDefault(limit))

	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query top senders: %w", err)
	}
	defer rows.Close()

	senders := make([]domain.TopSenderAnalytics, 0)
	for rows.Next() {
		var sender domain.TopSenderAnalytics
		var messageCount uint64
		if err := rows.Scan(
			&sender.SenderID,
			&sender.KickUserID,
			&sender.Username,
			&sender.Slug,
			&sender.ProfileImageURL,
			&messageCount,
			&sender.FirstMessageAt,
			&sender.LatestMessageAt,
		); err != nil {
			return nil, fmt.Errorf("scan top sender: %w", err)
		}
		if sender.KickUserID > 0 {
			sender.SenderID = sender.KickUserID
		}
		sender.MessageCount = int64(messageCount)
		sender.FirstMessageAt = sender.FirstMessageAt.UTC()
		sender.LatestMessageAt = sender.LatestMessageAt.UTC()
		senders = append(senders, sender)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top senders: %w", err)
	}
	return senders, nil
}

func senderIdentitySQL() string {
	return `multiIf(
			ifNull(sender_kick_id, 0) > 0,
			concat('kick:', toString(ifNull(sender_kick_id, 0))),
			sender_slug_lower != '',
			concat('slug:', sender_slug_lower),
			concat('username:', sender_username_lower)
		)`
}

func messageRankSQL() string {
	return `row_number() OVER (
			PARTITION BY kick_message_id
			ORDER BY ingested_at DESC, message_created_at DESC, id DESC
		) AS message_rank`
}

func (repo *AnalyticsRepository) TopChannels(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopChannelAnalytics, error) {
	where, args := topChannelsWhere(filter)
	query := fmt.Sprintf(`SELECT
		channel_id,
		argMax(channel_slug, tuple(message_created_at, ingested_at, id)) AS slug,
		argMax(channel_display_name, tuple(message_created_at, ingested_at, id)) AS display_name,
		argMax(ifNull(channel_profile_image_url, ''), tuple(message_created_at, ingested_at, id)) AS profile_image_url,
		argMax(ifNull(channel_banner_image_url, ''), tuple(message_created_at, ingested_at, id)) AS banner_image_url,
		count() AS message_count,
		min(message_created_at) AS first_message_at,
		max(message_created_at) AS latest_message_at
		FROM (
			SELECT
				id,
				kick_message_id,
				ifNull(channel_id, 0) AS channel_id,
				channel_slug,
				channel_display_name,
				channel_profile_image_url,
				channel_banner_image_url,
				message_created_at,
				ingested_at
			FROM chat_messages FINAL
			WHERE %s
		)
		GROUP BY channel_id
		ORDER BY message_count DESC, latest_message_at DESC, slug ASC
		LIMIT ?`, where)
	args = append(args, limitOrDefault(limit))

	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query top channels: %w", err)
	}
	defer rows.Close()

	channels := make([]domain.TopChannelAnalytics, 0)
	for rows.Next() {
		var channel domain.TopChannelAnalytics
		var messageCount uint64
		if err := rows.Scan(
			&channel.ChannelID,
			&channel.Slug,
			&channel.DisplayName,
			&channel.ProfileImageURL,
			&channel.BannerImageURL,
			&messageCount,
			&channel.FirstMessageAt,
			&channel.LatestMessageAt,
		); err != nil {
			return nil, fmt.Errorf("scan top channel: %w", err)
		}
		channel.MessageCount = int64(messageCount)
		channel.FirstMessageAt = channel.FirstMessageAt.UTC()
		channel.LatestMessageAt = channel.LatestMessageAt.UTC()
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top channels: %w", err)
	}
	return channels, nil
}

func (repo *AnalyticsRepository) TopEmotes(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopEmoteAnalytics, error) {
	where, args := analyticsWhere(filter)
	query := fmt.Sprintf(`SELECT
		emote_id,
		emote_name,
		emote_token,
		emote_image_url,
		count() AS usage_count,
		uniqExact(kick_message_id) AS message_count
		FROM (
			SELECT
				kick_message_id,
				emote_ids AS latest_emote_ids,
				emote_names AS latest_emote_names,
				emote_tokens AS latest_emote_tokens,
				emote_image_urls AS latest_emote_image_urls
			FROM chat_messages FINAL
			WHERE %s
		)
		ARRAY JOIN
			latest_emote_ids AS emote_id,
			latest_emote_names AS emote_name,
			latest_emote_tokens AS emote_token,
			latest_emote_image_urls AS emote_image_url
		GROUP BY emote_id, emote_name, emote_token, emote_image_url
		ORDER BY usage_count DESC, message_count DESC, emote_name ASC
		LIMIT ?`, where)
	args = append(args, limitOrDefault(limit))

	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query top emotes: %w", err)
	}
	defer rows.Close()

	emotes := make([]domain.TopEmoteAnalytics, 0)
	for rows.Next() {
		var emote domain.TopEmoteAnalytics
		var usageCount uint64
		var messageCount uint64
		if err := rows.Scan(
			&emote.ID,
			&emote.Name,
			&emote.Token,
			&emote.ImageURL,
			&usageCount,
			&messageCount,
		); err != nil {
			return nil, fmt.Errorf("scan top emote: %w", err)
		}
		emote.UsageCount = int64(usageCount)
		emote.MessageCount = int64(messageCount)
		emotes = append(emotes, emote)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top emotes: %w", err)
	}
	return emotes, nil
}

func (repo *AnalyticsRepository) LatestMessages(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.ChatMessage, error) {
	limit = limitOrDefault(limit)
	where, args := analyticsWhere(filter)
	candidateLimit := latestMessagesCandidateLimit(limit)
	idQuery := fmt.Sprintf(`SELECT id
		FROM (
			SELECT
				id,
				kick_message_id,
				message_created_at,
				ingested_at,
				%s
			FROM (
				SELECT
					id,
					kick_message_id,
					message_created_at,
					ingested_at
				FROM chat_messages
				WHERE %s
				ORDER BY message_created_at DESC, id DESC
				LIMIT ?
			)
		)
		WHERE message_rank = 1
		ORDER BY message_created_at DESC, id DESC
		LIMIT ?`, messageRankSQL(), where)
	idArgs := append([]any{}, args...)
	idArgs = append(idArgs, candidateLimit, limit)

	idRows, err := repo.conn.Query(ctx, idQuery, idArgs...)
	if err != nil {
		return nil, fmt.Errorf("query latest analytics message ids: %w", err)
	}
	defer idRows.Close()

	messageIDs := make([]int64, 0, int(limit))
	for idRows.Next() {
		var id int64
		if err := idRows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan latest analytics message id: %w", err)
		}
		messageIDs = append(messageIDs, id)
	}
	if err := idRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest analytics message ids: %w", err)
	}
	if len(messageIDs) == 0 {
		return []domain.ChatMessage{}, nil
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
				channel_slug, channel_display_name, channel_profile_image_url,
				channel_banner_image_url, channel_public_url,
				sender_id, sender_kick_id, sender_username, sender_slug,
				sender_display_color, sender_profile_image_url, sender_public_url,
				sender_badges_json, message_type, content, emote_ids, emote_names, emote_tokens,
				emote_image_urls, reply_to_sender, reply_to_content, reply_to_message_id,
				thread_parent_id, reply_metadata_json, raw_payload_json, message_created_at, ingested_at,
				is_deleted,
				%s
			FROM chat_messages
			WHERE id IN (?)
		)
		WHERE is_deleted = 0 AND message_rank = 1
		ORDER BY message_created_at DESC, id DESC
		LIMIT ?`, messageRankSQL())

	rows, err := repo.conn.Query(ctx, query, messageIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("query latest analytics messages: %w", err)
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
		return nil, fmt.Errorf("iterate latest analytics messages: %w", err)
	}
	return messages, nil
}

func latestMessagesCandidateLimit(limit uint64) uint64 {
	candidateLimit := limit * 100
	if candidateLimit < 1000 {
		return 1000
	}
	if candidateLimit > 5000 {
		return 5000
	}
	return candidateLimit
}

func analyticsWhere(filter domain.AnalyticsFilter) (string, []any) {
	where := []string{"is_deleted = 0"}
	args := make([]any, 0, 8)

	if !filter.Start.IsZero() {
		where = append(where, "message_created_at >= ?")
		args = append(args, filter.Start.UTC())
	}
	if !filter.End.IsZero() {
		where = append(where, "message_created_at <= ?")
		args = append(args, filter.End.UTC())
	}
	if filter.Channel != "" {
		channel := strings.ToLower(strings.TrimSpace(filter.Channel))
		where = append(where, "(channel_slug_lower = ? OR channel_display_name_lower = ?)")
		args = append(args, channel, channel)
	}
	if filter.Sender != "" {
		terms := senderLookupTerms(filter.Sender)
		if len(terms) > 0 {
			placeholders := strings.TrimRight(strings.Repeat("?,", len(terms)), ",")
			where = append(
				where,
				fmt.Sprintf("(sender_username_lower IN (%s) OR sender_slug_lower IN (%s))", placeholders, placeholders),
			)
			for _, term := range terms {
				args = append(args, term)
			}
			for _, term := range terms {
				args = append(args, term)
			}
		}
	}
	return strings.Join(where, " AND "), args
}

// topSendersWhere builds a WHERE clause for top-senders queries, additionally
// applying a free-text LIKE search on sender username/slug when filter.Query is set.
func topSendersWhere(filter domain.AnalyticsFilter) (string, []any) {
	clause, args := analyticsWhere(filter)
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		clause += " AND (sender_username_lower LIKE ? OR sender_slug_lower LIKE ?)"
		args = append(args, like, like)
	}
	return clause, args
}

// topChannelsWhere builds a WHERE clause for top-channels queries, additionally
// applying a free-text LIKE search on channel slug/display name when filter.Query is set.
func topChannelsWhere(filter domain.AnalyticsFilter) (string, []any) {
	clause, args := analyticsWhere(filter)
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		clause += " AND (channel_slug_lower LIKE ? OR channel_display_name_lower LIKE ?)"
		args = append(args, like, like)
	}
	return clause, args
}

func senderLookupTerms(value string) []string {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	if cleaned == "" {
		return nil
	}
	terms := make([]string, 0, 3)
	for _, term := range []string{
		cleaned,
		strings.ReplaceAll(cleaned, "_", "-"),
		strings.ReplaceAll(cleaned, "-", "_"),
	} {
		if !hasTerm(terms, term) {
			terms = append(terms, term)
		}
	}
	return terms
}

func hasTerm(terms []string, candidate string) bool {
	for _, term := range terms {
		if term == candidate {
			return true
		}
	}
	return false
}

func limitOrDefault(limit uint64) uint64 {
	if limit == 0 {
		return 10
	}
	return limit
}
