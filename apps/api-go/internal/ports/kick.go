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
