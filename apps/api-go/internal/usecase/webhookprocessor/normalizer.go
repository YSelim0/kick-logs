package webhookprocessor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

const (
	EventTypeNew     = "channel.subscription.new"
	EventTypeRenewal = "channel.subscription.renewal"
	EventTypeGifts   = "channel.subscription.gifts"
)

// ExtractBroadcasterUserID pulls broadcaster.user_id from any subscription payload.
func ExtractBroadcasterUserID(rawJSON string) (int64, error) {
	var env broadcasterEnvelope
	if err := json.Unmarshal([]byte(rawJSON), &env); err != nil {
		return 0, fmt.Errorf("parse broadcaster envelope: %w", err)
	}
	if env.Broadcaster.UserID == 0 {
		return 0, fmt.Errorf("broadcaster.user_id is missing or zero")
	}
	return env.Broadcaster.UserID, nil
}

// NormalizeEvent parses a raw webhook event and returns the subscription periods
// it produces. Returns (nil, ErrIgnored) for valid but unsupported event types.
func NormalizeEvent(event domain.KickWebhookEvent, ch domain.FollowedChannel) ([]domain.ChannelSubscriptionPeriod, error) {
	now := time.Now().UTC()

	switch event.EventType {
	case EventTypeNew, EventTypeRenewal:
		period, err := normalizeSubscription(event, ch, now)
		if err != nil {
			return nil, err
		}
		return []domain.ChannelSubscriptionPeriod{period}, nil

	case EventTypeGifts:
		return normalizeGifts(event, ch, now)

	default:
		return nil, &ErrIgnored{Reason: fmt.Sprintf("unsupported event type: %s", event.EventType)}
	}
}

// ErrIgnored signals the event should be marked ignored, not failed.
type ErrIgnored struct {
	Reason string
}

func (e *ErrIgnored) Error() string { return e.Reason }

func normalizeSubscription(event domain.KickWebhookEvent, ch domain.FollowedChannel, now time.Time) (domain.ChannelSubscriptionPeriod, error) {
	var p subscriptionPayload
	if err := json.Unmarshal([]byte(event.RawPayloadJSON), &p); err != nil {
		return domain.ChannelSubscriptionPeriod{}, fmt.Errorf("parse subscription payload: %w", err)
	}

	createdAt := parseTime(p.CreatedAt, now)
	expiresAt := resolveExpiry(createdAt, p.ExpiresAt)

	return domain.ChannelSubscriptionPeriod{
		ID:                        event.MessageID,
		EventMessageID:            event.MessageID,
		EventType:                 event.EventType,
		FollowedChannelID:         ch.ID,
		BroadcasterUserID:         ch.BroadcasterUserID,
		ChannelSlug:               ch.Slug,
		ChannelDisplayName:        ch.DisplayName,
		SubscriberKickUserID:      p.Subscriber.UserID,
		SubscriberUsername:        p.Subscriber.Username,
		SubscriberSlug:            p.Subscriber.ChannelSlug,
		SubscriberProfileImageURL: p.Subscriber.ProfilePicture,
		IsGift:                    false,
		StartedAt:                 createdAt,
		ExpiresAt:                 expiresAt,
		RawPayloadJSON:            event.RawPayloadJSON,
		IngestedAt:                now,
	}, nil
}

func normalizeGifts(event domain.KickWebhookEvent, ch domain.FollowedChannel, now time.Time) ([]domain.ChannelSubscriptionPeriod, error) {
	var p giftPayload
	if err := json.Unmarshal([]byte(event.RawPayloadJSON), &p); err != nil {
		return nil, fmt.Errorf("parse gift payload: %w", err)
	}

	recipients := p.Recipients
	if len(recipients) == 0 {
		recipients = p.Giftees
	}
	if len(recipients) == 0 {
		return nil, &ErrIgnored{Reason: "gift event has no recipients"}
	}

	createdAt := parseTime(p.CreatedAt, now)
	expiresAt := resolveExpiry(createdAt, p.ExpiresAt)

	var gifterUserID int64
	var gifterUsername, gifterSlug, gifterProfilePic string
	if p.Gifter != nil && !p.Gifter.IsAnonymous {
		gifterUserID = p.Gifter.UserID
		gifterUsername = p.Gifter.Username
		gifterSlug = p.Gifter.ChannelSlug
		gifterProfilePic = p.Gifter.ProfilePicture
	}

	periods := make([]domain.ChannelSubscriptionPeriod, 0, len(recipients))
	for _, r := range recipients {
		periods = append(periods, domain.ChannelSubscriptionPeriod{
			ID:                        giftPeriodID(event.MessageID, r.UserID),
			EventMessageID:            event.MessageID,
			EventType:                 event.EventType,
			FollowedChannelID:         ch.ID,
			BroadcasterUserID:         ch.BroadcasterUserID,
			ChannelSlug:               ch.Slug,
			ChannelDisplayName:        ch.DisplayName,
			SubscriberKickUserID:      r.UserID,
			SubscriberUsername:        r.Username,
			SubscriberSlug:            r.ChannelSlug,
			SubscriberProfileImageURL: r.ProfilePicture,
			GifterKickUserID:          gifterUserID,
			GifterUsername:            gifterUsername,
			GifterSlug:                gifterSlug,
			GifterProfileImageURL:     gifterProfilePic,
			IsGift:                    true,
			StartedAt:                 createdAt,
			ExpiresAt:                 expiresAt,
			RawPayloadJSON:            event.RawPayloadJSON,
			IngestedAt:                now,
		})
	}
	return periods, nil
}

func giftPeriodID(messageID string, gifteeUserID int64) string {
	return messageID + "_" + strconv.FormatInt(gifteeUserID, 10)
}

func parseTime(s string, fallback time.Time) time.Time {
	if s == "" {
		return fallback
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return fallback
}

func resolveExpiry(createdAt time.Time, expiresAtStr string) time.Time {
	if expiresAtStr != "" {
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
			if t, err := time.Parse(layout, expiresAtStr); err == nil {
				return t.UTC()
			}
		}
	}
	return createdAt.Add(30 * 24 * time.Hour)
}
