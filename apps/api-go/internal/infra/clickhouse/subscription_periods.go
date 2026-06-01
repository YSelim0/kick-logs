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

func uint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}
