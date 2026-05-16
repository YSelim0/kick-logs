package analytics

import (
	"context"
	"errors"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var ErrInvalidRange = errors.New("invalid analytics date range")

type Service struct {
	repository ports.AnalyticsRepository
}

func NewService(repository ports.AnalyticsRepository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Overview(
	ctx context.Context,
	filter domain.AnalyticsFilter,
) (domain.AnalyticsOverview, error) {
	if err := validateFilter(filter); err != nil {
		return domain.AnalyticsOverview{}, err
	}
	return service.repository.Overview(ctx, filter)
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
	return service.repository.MessageVolume(ctx, filter, bucket)
}

func (service *Service) TopSenders(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopSenderAnalytics, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	return service.repository.TopSenders(ctx, filter, normalizeLimit(limit))
}

func (service *Service) TopChannels(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopChannelAnalytics, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	return service.repository.TopChannels(ctx, filter, normalizeLimit(limit))
}

func (service *Service) TopEmotes(
	ctx context.Context,
	filter domain.AnalyticsFilter,
	limit uint64,
) ([]domain.TopEmoteAnalytics, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	return service.repository.TopEmotes(ctx, filter, normalizeLimit(limit))
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
