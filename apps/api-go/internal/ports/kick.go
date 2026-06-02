package ports

import (
	"context"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type KickChannelResolver interface {
	ResolveChannel(ctx context.Context, slug string) (domain.FollowedChannel, error)
}

type KickSenderProfileResolver interface {
	ResolveSender(ctx context.Context, slug string) (domain.SenderProfile, error)
}

type PusherClient interface {
	Listen(ctx context.Context, channels []domain.ListenerChannel, handle func(string) error) error
}

type KickRecentMessagesClient interface {
	FetchRecentMessages(ctx context.Context, channel domain.FollowedChannel) ([]domain.RawChatEventEnvelope, error)
}

// KickWebhookVerifier verifies the RSA-SHA256 signature on incoming Kick webhook requests.
type KickWebhookVerifier interface {
	Verify(messageID, timestamp string, body []byte, signature string) error
}

type KickEventSubscriptionClient interface {
	ResolveBroadcasterUserID(ctx context.Context, slug string) (int64, error)
	ListEventSubscriptions(ctx context.Context) ([]domain.KickAPIEventSub, error)
	CreateEventSubscriptions(ctx context.Context, broadcasterUserID int64, eventTypes []string) ([]domain.KickAPIEventSub, error)
	DeleteEventSubscription(ctx context.Context, subscriptionID string) error
}
