package ports

import (
	"context"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type KickChannelResolver interface {
	ResolveChannel(ctx context.Context, slug string) (domain.FollowedChannel, error)
}
