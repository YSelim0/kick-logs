package clickhouse

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type SubscriptionPeriodRepository struct {
	conn driver.Conn
}

func NewSubscriptionPeriodRepository(conn driver.Conn) *SubscriptionPeriodRepository {
	return &SubscriptionPeriodRepository{conn: conn}
}

func (r *SubscriptionPeriodRepository) InsertBatch(ctx context.Context, periods []domain.ChannelSubscriptionPeriod) error {
	if len(periods) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `INSERT INTO channel_subscription_periods (
		id, event_message_id, event_type, followed_channel_id, broadcaster_user_id,
		channel_slug, channel_display_name, subscriber_kick_user_id, subscriber_username,
		subscriber_slug, subscriber_profile_image_url, gifter_kick_user_id, gifter_username,
		gifter_slug, gifter_profile_image_url, is_gift, started_at, expires_at,
		raw_payload_json, ingested_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare subscription period insert: %w", err)
	}

	for _, p := range periods {
		isGift := uint8(0)
		if p.IsGift {
			isGift = 1
		}
		if err := batch.Append(
			p.ID,
			p.EventMessageID,
			p.EventType,
			p.FollowedChannelID,
			p.BroadcasterUserID,
			p.ChannelSlug,
			p.ChannelDisplayName,
			p.SubscriberKickUserID,
			p.SubscriberUsername,
			p.SubscriberSlug,
			p.SubscriberProfileImageURL,
			nullableInt64(p.GifterKickUserID),
			nullableString(p.GifterUsername),
			nullableString(p.GifterSlug),
			nullableString(p.GifterProfileImageURL),
			isGift,
			p.StartedAt,
			p.ExpiresAt,
			p.RawPayloadJSON,
			p.IngestedAt,
		); err != nil {
			return fmt.Errorf("append subscription period: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send subscription period batch: %w", err)
	}
	return nil
}

func (r *SubscriptionPeriodRepository) ActiveSummary(ctx context.Context, followedChannelID int64) (domain.ChannelSubscriptionSummary, error) {
	row := r.conn.QueryRow(ctx, `
		SELECT
			countDistinctIf(subscriber_kick_user_id, expires_at > now()) AS active_count,
			countDistinctIf(subscriber_kick_user_id, expires_at > now() AND is_gift = 1) AS active_gifted_count,
			max(started_at) AS latest_event_at
		FROM channel_subscription_periods FINAL
		WHERE followed_channel_id = ?`,
		followedChannelID,
	)

	var summary domain.ChannelSubscriptionSummary
	var activeCount uint64
	var activeGiftedCount uint64
	var latestEventAt time.Time
	if err := row.Scan(&activeCount, &activeGiftedCount, &latestEventAt); err != nil {
		return domain.ChannelSubscriptionSummary{}, fmt.Errorf("active subscription summary: %w", err)
	}
	summary.ActiveCount = uint64ToInt64(activeCount)
	summary.ActiveGiftedCount = uint64ToInt64(activeGiftedCount)
	summary.LatestEventAt = latestEventAt
	return summary, nil
}

func (r *SubscriptionPeriodRepository) ListActiveSubscribers(
	ctx context.Context,
	filter domain.ChannelSubscriberFilter,
) (domain.ChannelSubscriberPage, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	query := activeSubscribersQuery(filter.GiftOnly, true)
	rows, err := r.conn.Query(ctx, query, filter.FollowedChannelID, filter.Limit, filter.Offset)
	if err != nil {
		return domain.ChannelSubscriberPage{}, fmt.Errorf("query active subscribers: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ChannelSubscriber, 0, int(filter.Limit))
	var total uint64
	for rows.Next() {
		item, rowTotal, err := scanChannelSubscriber(rows, true)
		if err != nil {
			return domain.ChannelSubscriberPage{}, err
		}
		total = rowTotal
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.ChannelSubscriberPage{}, fmt.Errorf("iterate active subscribers: %w", err)
	}

	return domain.ChannelSubscriberPage{
		Items:  items,
		Count:  uint64ToInt64(total),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (r *SubscriptionPeriodRepository) ExportActiveSubscribers(
	ctx context.Context,
	followedChannelID int64,
	giftOnly bool,
) ([]domain.ChannelSubscriber, error) {
	query := activeSubscribersQuery(giftOnly, false)
	rows, err := r.conn.Query(ctx, query, followedChannelID)
	if err != nil {
		return nil, fmt.Errorf("query active subscriber export: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ChannelSubscriber, 0)
	for rows.Next() {
		item, _, err := scanChannelSubscriber(rows, false)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active subscriber export: %w", err)
	}
	return items, nil
}

func activeSubscribersQuery(giftOnly bool, paginated bool) string {
	giftClause := ""
	if giftOnly {
		giftClause = " AND is_gift = 1"
	}
	totalColumn := ""
	if paginated {
		totalColumn = ", count() OVER() AS total_count"
	}
	limitClause := ""
	if paginated {
		limitClause = " LIMIT ? OFFSET ?"
	}
	return fmt.Sprintf(`
		SELECT
			subscriber_kick_user_id,
			subscriber_username,
			subscriber_slug,
			subscriber_profile_image_url,
			is_gift,
			gifter_kick_user_id,
			gifter_username,
			gifter_slug,
			gifter_profile_image_url,
			started_at,
			expires_at%s
		FROM (
			SELECT
				subscriber_kick_user_id,
				subscriber_username,
				subscriber_slug,
				subscriber_profile_image_url,
				toUInt8(is_gift) AS is_gift,
				ifNull(gifter_kick_user_id, 0) AS gifter_kick_user_id,
				ifNull(gifter_username, '') AS gifter_username,
				ifNull(gifter_slug, '') AS gifter_slug,
				ifNull(gifter_profile_image_url, '') AS gifter_profile_image_url,
				started_at,
				expires_at
			FROM channel_subscription_periods FINAL
			WHERE followed_channel_id = ? AND expires_at > now()%s
			ORDER BY
				subscriber_kick_user_id ASC,
				expires_at DESC,
				started_at DESC,
				ingested_at DESC
			LIMIT 1 BY subscriber_kick_user_id
		)
		ORDER BY started_at DESC, subscriber_username ASC%s`,
		totalColumn,
		giftClause,
		limitClause,
	)
}

type channelSubscriberScanner interface {
	Scan(dest ...any) error
}

func scanChannelSubscriber(row channelSubscriberScanner, withTotal bool) (domain.ChannelSubscriber, uint64, error) {
	var item domain.ChannelSubscriber
	var isGift uint8
	var total uint64
	dest := []any{
		&item.SubscriberKickUserID,
		&item.Username,
		&item.Slug,
		&item.ProfileImageURL,
		&isGift,
		&item.GifterKickUserID,
		&item.GifterUsername,
		&item.GifterSlug,
		&item.GifterProfileImageURL,
		&item.StartedAt,
		&item.ExpiresAt,
	}
	if withTotal {
		dest = append(dest, &total)
	}
	if err := row.Scan(dest...); err != nil {
		return domain.ChannelSubscriber{}, 0, fmt.Errorf("scan active subscriber: %w", err)
	}
	item.IsGift = isGift == 1
	item.StartedAt = item.StartedAt.UTC()
	item.ExpiresAt = item.ExpiresAt.UTC()
	return item, total, nil
}

func uint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}
