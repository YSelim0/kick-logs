package analytics

import (
	"context"
	"errors"
	"strconv"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var ErrInvalidRange = errors.New("invalid analytics date range")

type Service struct {
	repository ports.AnalyticsRepository
	cache      *analyticsCache
}

func NewService(repository ports.AnalyticsRepository) *Service {
	return &Service{
		repository: repository,
		cache:      newAnalyticsCache(),
	}
}

func (service *Service) Overview(
	ctx context.Context,
	filter domain.AnalyticsFilter,
) (domain.AnalyticsOverview, error) {
	if err := validateFilter(filter); err != nil {
		return domain.AnalyticsOverview{}, err
	}
	if !globalAnalyticsCacheable(filter) {
		return service.repository.Overview(ctx, filter)
	}
	key := "overview:" + analyticsFilterCacheKey(filter)
	return cachedAnalyticsValue(ctx, service.cache, key, func(ctx context.Context) (domain.AnalyticsOverview, error) {
		return service.repository.Overview(ctx, filter)
	})
}

func (service *Service) MessageVolume(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	bucket domain.AnalyticsBucket,
) ([]domain.MessageVolumePoint, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	if bucket == "" {
		bucket = domain.AnalyticsBucketDay
	}
	if bucket != domain.AnalyticsBucketHour && bucket != domain.AnalyticsBucketDay {
		return nil, errors.New("invalid analytics bucket")
	}
	if !globalAnalyticsCacheable(filter) {
		return service.repository.MessageVolume(ctx, filter, bucket)
	}
	key := "message-volume:" + string(bucket) + ":" + analyticsFilterCacheKey(filter)
	return cachedAnalyticsValue(ctx, service.cache, key, func(ctx context.Context) ([]domain.MessageVolumePoint, error) {
		return service.repository.MessageVolume(ctx, filter, bucket)
	})
}

func (service *Service) TopSenders(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopSenderAnalytics, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	normalizedLimit := normalizeLimit(limit)
	if !globalAnalyticsCacheable(filter) {
		return service.repository.TopSenders(ctx, filter, normalizedLimit)
	}
	key := "top-senders:" + analyticsFilterCacheKey(filter) + ":limit=" + limitCacheKey(normalizedLimit)
	return cachedAnalyticsValue(ctx, service.cache, key, func(ctx context.Context) ([]domain.TopSenderAnalytics, error) {
		return service.repository.TopSenders(ctx, filter, normalizedLimit)
	})
}

func (service *Service) TopChannels(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopChannelAnalytics, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	normalizedLimit := normalizeLimit(limit)
	if !globalAnalyticsCacheable(filter) {
		return service.repository.TopChannels(ctx, filter, normalizedLimit)
	}
	key := "top-channels:" + analyticsFilterCacheKey(filter) + ":limit=" + limitCacheKey(normalizedLimit)
	return cachedAnalyticsValue(ctx, service.cache, key, func(ctx context.Context) ([]domain.TopChannelAnalytics, error) {
		return service.repository.TopChannels(ctx, filter, normalizedLimit)
	})
}

func (service *Service) TopEmotes(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopEmoteAnalytics, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	normalizedLimit := normalizeLimit(limit)
	if !globalAnalyticsCacheable(filter) {
		return service.repository.TopEmotes(ctx, filter, normalizedLimit)
	}
	key := "top-emotes:" + analyticsFilterCacheKey(filter) + ":limit=" + limitCacheKey(normalizedLimit)
	return cachedAnalyticsValue(ctx, service.cache, key, func(ctx context.Context) ([]domain.TopEmoteAnalytics, error) {
		return service.repository.TopEmotes(ctx, filter, normalizedLimit)
	})
}

func validateFilter(filter domain.AnalyticsFilter) error {
	if !filter.Start.IsZero() && !filter.End.IsZero() && filter.Start.After(filter.End) {
		return ErrInvalidRange
	}
	return nil
}

func normalizeLimit(limit uint64) uint64 {
	if limit == 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func limitCacheKey(limit uint64) string {
	return strconv.FormatUint(limit, 10)
}
