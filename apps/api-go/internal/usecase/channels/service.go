package channels

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrChannelNotFound   = errors.New("channel not found")
	ErrChannelResolution = errors.New("channel resolution failed")
	ErrValidation        = errors.New("validation failed")
)

type Service struct {
	channels ports.FollowedChannelRepository
	resolver ports.KickChannelResolver
}

func NewService(
	channels ports.FollowedChannelRepository,
	resolver ports.KickChannelResolver,
) *Service {
	return &Service{channels: channels, resolver: resolver}
}

func (service *Service) List(ctx context.Context) ([]domain.FollowedChannel, error) {
	return service.channels.List(ctx)
}

func (service *Service) Add(ctx context.Context, slug string) (domain.FollowedChannel, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" || len(slug) > 120 {
		return domain.FollowedChannel{}, fmt.Errorf("%w: invalid channel slug", ErrValidation)
	}

	channel, err := service.resolver.ResolveChannel(ctx, slug)
	if err != nil {
		return domain.FollowedChannel{}, fmt.Errorf("%w: %v", ErrChannelResolution, err)
	}
	channel.IsEnabled = true
	channel.LastResolvedAt = time.Now().UTC()
	return service.channels.Upsert(ctx, channel)
}

func (service *Service) GetBySlug(ctx context.Context, slug string) (domain.FollowedChannel, error) {
	slug = strings.TrimSpace(slug)
	ch, err := service.channels.GetBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.FollowedChannel{}, ErrChannelNotFound
	}
	return ch, err
}

func (service *Service) Disable(ctx context.Context, id int64) (domain.FollowedChannel, error) {
	channel, err := service.channels.Disable(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.FollowedChannel{}, ErrChannelNotFound
	}
	if err != nil {
		return domain.FollowedChannel{}, err
	}
	return channel, nil
}
