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

type KickEventSubscriptionClient interface {
	ResolveBroadcasterUserID(ctx context.Context, slug string) (int64, error)
	ListEventSubscriptions(ctx context.Context) ([]domain.KickAPIEventSub, error)
	CreateEventSubscription(ctx context.Context, broadcasterUserID int64, eventType string) (domain.KickAPIEventSub, error)
	DeleteEventSubscription(ctx context.Context, subscriptionID string) error
}
