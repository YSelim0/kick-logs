package kicksync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type Service struct {
	log       *slog.Logger
	channels  ports.FollowedChannelRepository
	eventSubs ports.KickEventSubscriptionRepository
	client    ports.KickEventSubscriptionClient
	events    []string
}

func NewService(
	log *slog.Logger,
	channels ports.FollowedChannelRepository,
	eventSubs ports.KickEventSubscriptionRepository,
	client ports.KickEventSubscriptionClient,
	events []string,
) *Service {
	return &Service{
		log:       log,
		channels:  channels,
		eventSubs: eventSubs,
		client:    client,
		events:    events,
	}
}

// SyncAll reconciles all enabled channels: resolves missing broadcaster IDs and
// ensures event subscriptions exist on Kick for each channel/event pair.
// Per-channel errors are logged and stored; the method never returns an error.
func (s *Service) SyncAll(ctx context.Context) {
	channels, err := s.channels.ListEnabled(ctx)
	if err != nil {
		s.log.Error("kicksync: list enabled channels failed", "error", err)
		return
	}
	for _, ch := range channels {
		if err := s.syncChannel(ctx, ch); err != nil {
			s.log.Warn("kicksync: sync channel failed", "channel", ch.Slug, "error", err)
		}
	}
}

// EnsureChannelSubscriptions creates any missing Kick event subscriptions for
// the given followed channel. Called after a channel is added.
func (s *Service) EnsureChannelSubscriptions(ctx context.Context, followedChannelID int64) error {
	ch, err := s.channels.GetByID(ctx, followedChannelID)
	if err != nil {
		return fmt.Errorf("get channel %d: %w", followedChannelID, err)
	}
	return s.syncChannel(ctx, ch)
}

// RemoveChannelSubscriptions deletes Kick event subscriptions and marks local
// registry entries as deleted. Called when a channel is disabled.
func (s *Service) RemoveChannelSubscriptions(ctx context.Context, followedChannelID int64) error {
	subs, err := s.eventSubs.ListByChannel(ctx, followedChannelID)
	if err != nil {
		return fmt.Errorf("list subscriptions for channel %d: %w", followedChannelID, err)
	}

	for _, sub := range subs {
		if sub.KickSubscriptionID == "" || sub.Status == domain.KickEventSubStatusDeleted {
			continue
		}
		if err := s.client.DeleteEventSubscription(ctx, sub.KickSubscriptionID); err != nil {
			s.log.Warn("kicksync: delete event subscription failed",
				"kick_sub_id", sub.KickSubscriptionID,
				"event_type", sub.EventType,
				"error", err,
			)
			_ = s.eventSubs.UpdateSyncError(context.Background(), sub.ID, err.Error())
		}
	}

	return s.eventSubs.DeleteByChannel(ctx, followedChannelID)
}

func (s *Service) syncChannel(ctx context.Context, ch domain.FollowedChannel) error {
	if ch.BroadcasterUserID == 0 {
		broadcasterID, err := s.client.ResolveBroadcasterUserID(ctx, ch.Slug)
		if err != nil {
			return fmt.Errorf("resolve broadcaster user id for %q: %w", ch.Slug, err)
		}
		ch.BroadcasterUserID = broadcasterID
		updated, err := s.channels.Upsert(ctx, ch)
		if err != nil {
			return fmt.Errorf("save broadcaster user id for %q: %w", ch.Slug, err)
		}
		ch = updated
		s.log.Info("kicksync: resolved broadcaster user id", "channel", ch.Slug, "broadcaster_user_id", ch.BroadcasterUserID)
	}

	existing, err := s.eventSubs.ListByChannel(ctx, ch.ID)
	if err != nil {
		return fmt.Errorf("list existing subscriptions for %q: %w", ch.Slug, err)
	}

	byType := make(map[string]domain.KickEventSubscription, len(existing))
	for _, sub := range existing {
		byType[sub.EventType] = sub
	}

	for _, eventType := range s.events {
		current, exists := byType[eventType]

		if exists && current.Status == domain.KickEventSubStatusActive && current.KickSubscriptionID != "" {
			continue
		}

		apiSub, createErr := s.client.CreateEventSubscription(ctx, ch.BroadcasterUserID, eventType)
		if createErr != nil {
			s.log.Warn("kicksync: create event subscription failed",
				"channel", ch.Slug,
				"event_type", eventType,
				"error", createErr,
			)
			if exists {
				_ = s.eventSubs.UpdateSyncError(ctx, current.ID, createErr.Error())
			} else {
				errSub := domain.KickEventSubscription{
					FollowedChannelID: ch.ID,
					BroadcasterUserID: ch.BroadcasterUserID,
					EventType:         eventType,
					EventVersion:      "v1",
					Method:            "webhook",
					Status:            domain.KickEventSubStatusError,
					LatestSyncError:   createErr.Error(),
				}
				if _, upsertErr := s.eventSubs.Upsert(ctx, errSub); upsertErr != nil {
					s.log.Warn("kicksync: save error subscription record failed", "error", upsertErr)
				}
			}
			continue
		}

		newSub := domain.KickEventSubscription{
			FollowedChannelID:  ch.ID,
			BroadcasterUserID:  ch.BroadcasterUserID,
			EventType:          eventType,
			EventVersion:       "v1",
			Method:             "webhook",
			KickSubscriptionID: apiSub.SubscriptionID,
			Status:             domain.KickEventSubStatusActive,
			SyncedAt:           time.Now().UTC(),
		}
		if exists {
			newSub.CreatedAt = current.CreatedAt
		}
		if _, upsertErr := s.eventSubs.Upsert(ctx, newSub); upsertErr != nil {
			s.log.Warn("kicksync: save subscription to registry failed",
				"channel", ch.Slug,
				"event_type", eventType,
				"error", upsertErr,
			)
		} else {
			s.log.Info("kicksync: event subscription created",
				"channel", ch.Slug,
				"event_type", eventType,
				"kick_sub_id", apiSub.SubscriptionID,
			)
		}
	}
	return nil
}
