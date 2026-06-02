package profiles

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrSenderNotFound  = errors.New("sender profile not found")
	ErrChannelNotFound = errors.New("channel profile not found")
)

type Service struct {
	analytics ports.AnalyticsRepository
	channels  ports.FollowedChannelRepository
	senders   ports.SenderProfileRepository
}

func NewService(
	analytics ports.AnalyticsRepository,
	channels ports.FollowedChannelRepository,
	senders ports.SenderProfileRepository,
) *Service {
	return &Service{
		analytics: analytics,
		channels:  channels,
		senders:   senders,
	}
}

func (service *Service) UserProfile(ctx context.Context, slug string) (domain.UserProfile, error) {
	sender, err := service.findSender(ctx, slug)
	if err != nil {
		return domain.UserProfile{}, err
	}
	if sender.KickUserID > 0 {
		sender.ID = sender.KickUserID
	}

	filter := domain.AnalyticsFilter{Sender: sender.Slug}
	overview, err := service.analytics.Overview(ctx, filter)
	if err != nil {
		return domain.UserProfile{}, err
	}
	volume, err := service.analytics.MessageVolume(ctx, filter, domain.AnalyticsBucketDay)
	if err != nil {
		return domain.UserProfile{}, err
	}
	topChannels, err := service.analytics.TopChannels(ctx, filter, 5)
	if err != nil {
		return domain.UserProfile{}, err
	}
	topEmotes, err := service.analytics.TopEmotes(ctx, filter, 5)
	if err != nil {
		return domain.UserProfile{}, err
	}
	latestMessages, err := service.analytics.LatestMessages(ctx, filter, 20)
	if err != nil {
		return domain.UserProfile{}, err
	}

	return domain.UserProfile{
		Sender:         sender,
		Overview:       overview,
		MessageVolume:  volume,
		TopChannels:    topChannels,
		TopEmotes:      topEmotes,
		LatestMessages: latestMessages,
	}, nil
}

func (service *Service) ChannelProfile(ctx context.Context, slug string) (domain.ChannelProfile, error) {
	channel, err := service.channels.GetBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ChannelProfile{}, ErrChannelNotFound
	}
	if err != nil {
		return domain.ChannelProfile{}, err
	}

	filter := domain.AnalyticsFilter{Channel: channel.Slug}
	overview, err := service.analytics.Overview(ctx, filter)
	if err != nil {
		return domain.ChannelProfile{}, err
	}
	volume, err := service.analytics.MessageVolume(ctx, filter, domain.AnalyticsBucketDay)
	if err != nil {
		return domain.ChannelProfile{}, err
	}
	topSenders, err := service.analytics.TopSenders(ctx, filter, 5)
	if err != nil {
		return domain.ChannelProfile{}, err
	}
	topEmotes, err := service.analytics.TopEmotes(ctx, filter, 5)
	if err != nil {
		return domain.ChannelProfile{}, err
	}
	latestMessages, err := service.analytics.LatestMessages(ctx, filter, 10)
	if err != nil {
		return domain.ChannelProfile{}, err
	}

	return domain.ChannelProfile{
		Channel:        channel,
		Overview:       overview,
		MessageVolume:  volume,
		TopSenders:     topSenders,
		TopEmotes:      topEmotes,
		LatestMessages: latestMessages,
	}, nil
}

func (service *Service) findSender(ctx context.Context, slug string) (domain.SenderProfile, error) {
	for _, term := range senderLookupTerms(slug) {
		sender, err := service.senders.GetBySlug(ctx, term)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return domain.SenderProfile{}, err
		}
		return sender, nil
	}
	return domain.SenderProfile{}, ErrSenderNotFound
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
		if !containsString(terms, term) {
			terms = append(terms, term)
		}
	}
	return terms
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
