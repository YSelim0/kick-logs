package webhookprocessor

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

const defaultTickInterval = 5 * time.Second

type Service struct {
	log          *slog.Logger
	inbox        ports.KickWebhookEventRepository
	channels     ports.FollowedChannelRepository
	periods      ports.SubscriptionPeriodRepository
	batchSize    int
	maxAttempts  int
	tickInterval time.Duration
}

func NewService(
	log *slog.Logger,
	inbox ports.KickWebhookEventRepository,
	channels ports.FollowedChannelRepository,
	periods ports.SubscriptionPeriodRepository,
	batchSize int,
	maxAttempts int,
) *Service {
	return &Service{
		log:          log,
		inbox:        inbox,
		channels:     channels,
		periods:      periods,
		batchSize:    batchSize,
		maxAttempts:  maxAttempts,
		tickInterval: defaultTickInterval,
	}
}

// Start runs the processor loop in a background goroutine until ctx is cancelled.
func (s *Service) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Service) run(ctx context.Context) {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processBatch(ctx)
		}
	}
}

// ProcessBatchOnce runs one processing tick synchronously. Exported for tests.
func (s *Service) ProcessBatchOnce(ctx context.Context) {
	s.processBatch(ctx)
}

func (s *Service) processBatch(ctx context.Context) {
	events, err := s.inbox.ListPending(ctx, s.batchSize, s.maxAttempts)
	if err != nil {
		s.log.Error("webhook processor: list pending failed", "error", err)
		return
	}
	for i := range events {
		s.processOne(ctx, events[i])
	}
}

func (s *Service) processOne(ctx context.Context, event domain.KickWebhookEvent) {
	broadcasterID, err := ExtractBroadcasterUserID(event.RawPayloadJSON)
	if err != nil {
		s.log.Warn("webhook processor: extract broadcaster user id failed",
			"message_id", event.MessageID,
			"error", err,
		)
		_ = s.inbox.MarkFailed(ctx, event.MessageID, err.Error(), s.maxAttempts)
		return
	}

	ch, err := s.channels.GetByBroadcasterUserID(ctx, broadcasterID)
	if errors.Is(err, sql.ErrNoRows) {
		reason := "broadcaster not followed"
		s.log.Debug("webhook processor: ignoring event for unknown broadcaster",
			"message_id", event.MessageID,
			"broadcaster_user_id", broadcasterID,
		)
		_ = s.inbox.MarkIgnored(ctx, event.MessageID, reason)
		return
	}
	if err != nil {
		s.log.Warn("webhook processor: lookup followed channel failed",
			"message_id", event.MessageID,
			"broadcaster_user_id", broadcasterID,
			"error", err,
		)
		_ = s.inbox.MarkFailed(ctx, event.MessageID, err.Error(), s.maxAttempts)
		return
	}
	if !ch.IsEnabled {
		reason := "channel disabled"
		s.log.Debug("webhook processor: ignoring event for disabled channel",
			"message_id", event.MessageID,
			"broadcaster_user_id", broadcasterID,
			"channel", ch.Slug,
		)
		_ = s.inbox.MarkIgnored(ctx, event.MessageID, reason)
		return
	}

	normalizedPeriods, err := NormalizeEvent(event, ch)
	if err != nil {
		var ignored *ErrIgnored
		if errors.As(err, &ignored) {
			s.log.Debug("webhook processor: ignoring event",
				"message_id", event.MessageID,
				"reason", ignored.Reason,
			)
			_ = s.inbox.MarkIgnored(ctx, event.MessageID, ignored.Reason)
			return
		}
		s.log.Warn("webhook processor: normalize event failed",
			"message_id", event.MessageID,
			"error", err,
		)
		_ = s.inbox.MarkFailed(ctx, event.MessageID, err.Error(), s.maxAttempts)
		return
	}

	if err := s.periods.InsertBatch(ctx, normalizedPeriods); err != nil {
		s.log.Warn("webhook processor: insert subscription periods failed",
			"message_id", event.MessageID,
			"error", err,
		)
		_ = s.inbox.MarkFailed(ctx, event.MessageID, err.Error(), s.maxAttempts)
		return
	}

	if err := s.inbox.MarkProcessed(ctx, event.MessageID); err != nil {
		s.log.Warn("webhook processor: mark processed failed",
			"message_id", event.MessageID,
			"error", err,
		)
	}
}
