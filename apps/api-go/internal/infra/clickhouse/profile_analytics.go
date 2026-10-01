package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var _ ports.ProfileSummaryRepository = (*AnalyticsRepository)(nil)

// WITH TOTALS merges exact aggregate states across every group, before LIMIT.
// Keep raw nullable channel IDs separate from the ranking's null-to-zero identity.
const profileAggregateSQL = `count() AS message_count,
	min(message_created_at) AS first_message_at,
	max(message_created_at) AS latest_message_at,
	uniqExactIf(sender_identity, sender_present) AS total_senders,
	uniqExactIf(source_channel_id, isNotNull(source_channel_id)) AS total_channels,
	sum(emote_count) AS total_emote_usages`

func (repo *AnalyticsRepository) OverviewAndTopSenders(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) (domain.AnalyticsOverview, []domain.TopSenderAnalytics, error) {
	where, args := analyticsWhere(filter)
	query := fmt.Sprintf(`WITH argMax(
		tuple(ifNull(sender_id, 0), ifNull(sender_kick_id, 0), sender_username,
			sender_slug, ifNull(sender_profile_image_url, '')),
		tuple(message_created_at, ingested_at, id)
	) AS profile_metadata
	SELECT profile_metadata.1 AS profile_sender_id,
		profile_metadata.2 AS profile_kick_user_id,
		profile_metadata.3 AS profile_username,
		profile_metadata.4 AS profile_slug,
		profile_metadata.5 AS profile_image,
		%s
	FROM (
		SELECT %s AS sender_identity,
			(ifNull(sender_kick_id, 0) > 0 OR sender_slug_lower != '' OR sender_username_lower != '') AS sender_present,
			channel_id AS source_channel_id, emote_count,
			id, sender_id, sender_kick_id, sender_username, sender_slug,
			sender_profile_image_url, message_created_at, ingested_at
		FROM chat_messages FINAL WHERE %s
	)
	GROUP BY sender_identity WITH TOTALS
	ORDER BY message_count DESC, latest_message_at DESC, profile_slug ASC
	LIMIT ?`, profileAggregateSQL, senderIdentitySQL(), where)
	args = append(args, limitOrDefault(limit))
	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return domain.AnalyticsOverview{}, nil, fmt.Errorf("query profile overview and senders: %w", err)
	}
	defer rows.Close()

	senders := make([]domain.TopSenderAnalytics, 0)
	var sender domain.TopSenderAnalytics
	var aggregate profileAggregate
	dest := aggregate.destinations(&sender.SenderID, &sender.KickUserID, &sender.Username, &sender.Slug, &sender.ProfileImageURL)
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return domain.AnalyticsOverview{}, nil, fmt.Errorf("scan profile sender: %w", err)
		}
		if sender.KickUserID > 0 {
			sender.SenderID = sender.KickUserID
		}
		sender.MessageCount = int64(aggregate.messages)
		sender.FirstMessageAt = aggregate.first.UTC()
		sender.LatestMessageAt = aggregate.latest.UTC()
		senders = append(senders, sender)
	}
	if err := rows.Err(); err != nil {
		return domain.AnalyticsOverview{}, nil, fmt.Errorf("iterate profile senders: %w", err)
	}
	if err := rows.Totals(dest...); err != nil {
		return domain.AnalyticsOverview{}, nil, fmt.Errorf("read profile sender totals: %w", err)
	}
	return aggregate.overview(), senders, nil
}

func (repo *AnalyticsRepository) OverviewAndTopChannels(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) (domain.AnalyticsOverview, []domain.TopChannelAnalytics, error) {
	where, args := analyticsWhere(filter)
	query := fmt.Sprintf(`WITH argMax(
		tuple(channel_slug, channel_display_name, ifNull(channel_profile_image_url, ''),
			ifNull(channel_banner_image_url, '')),
		tuple(message_created_at, ingested_at, id)
	) AS profile_metadata
	SELECT group_channel_id,
		profile_metadata.1 AS profile_slug,
		profile_metadata.2 AS profile_display_name,
		profile_metadata.3 AS profile_image,
		profile_metadata.4 AS profile_banner,
		%s
	FROM (
		SELECT %s AS sender_identity,
			(ifNull(sender_kick_id, 0) > 0 OR sender_slug_lower != '' OR sender_username_lower != '') AS sender_present,
			channel_id AS source_channel_id, ifNull(channel_id, 0) AS group_channel_id,
			emote_count, id, channel_slug, channel_display_name,
			channel_profile_image_url, channel_banner_image_url, message_created_at, ingested_at
		FROM chat_messages FINAL WHERE %s
	)
	GROUP BY group_channel_id WITH TOTALS
	ORDER BY message_count DESC, latest_message_at DESC, profile_slug ASC
	LIMIT ?`, profileAggregateSQL, senderIdentitySQL(), where)
	args = append(args, limitOrDefault(limit))
	rows, err := repo.conn.Query(ctx, query, args...)
	if err != nil {
		return domain.AnalyticsOverview{}, nil, fmt.Errorf("query profile overview and channels: %w", err)
	}
	defer rows.Close()

	channels := make([]domain.TopChannelAnalytics, 0)
	var channel domain.TopChannelAnalytics
	var aggregate profileAggregate
	dest := aggregate.destinations(&channel.ChannelID, &channel.Slug, &channel.DisplayName, &channel.ProfileImageURL, &channel.BannerImageURL)
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return domain.AnalyticsOverview{}, nil, fmt.Errorf("scan profile channel: %w", err)
		}
		channel.MessageCount = int64(aggregate.messages)
		channel.FirstMessageAt = aggregate.first.UTC()
		channel.LatestMessageAt = aggregate.latest.UTC()
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		return domain.AnalyticsOverview{}, nil, fmt.Errorf("iterate profile channels: %w", err)
	}
	if err := rows.Totals(dest...); err != nil {
		return domain.AnalyticsOverview{}, nil, fmt.Errorf("read profile channel totals: %w", err)
	}
	return aggregate.overview(), channels, nil
}

type profileAggregate struct {
	messages uint64
	first    time.Time
	latest   time.Time
	senders  uint64
	channels uint64
	emotes   uint64
}

func (aggregate *profileAggregate) destinations(metadata ...any) []any {
	return append(metadata, &aggregate.messages, &aggregate.first, &aggregate.latest,
		&aggregate.senders, &aggregate.channels, &aggregate.emotes)
}

func (aggregate profileAggregate) overview() domain.AnalyticsOverview {
	result := domain.AnalyticsOverview{
		TotalMessages: int64(aggregate.messages), TotalSenders: int64(aggregate.senders),
		TotalChannels: int64(aggregate.channels), TotalEmoteUsages: int64(aggregate.emotes),
	}
	if aggregate.messages > 0 {
		result.FirstMessageAt, result.LatestMessageAt = aggregate.first.UTC(), aggregate.latest.UTC()
	}
	return result
}
