package ports

import (
	"context"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type AdminUserRepository interface {
	Upsert(ctx context.Context, user domain.AdminUser) (domain.AdminUser, error)
	GetByID(ctx context.Context, id int64) (domain.AdminUser, error)
	GetByEmail(ctx context.Context, email string) (domain.AdminUser, error)
	ListActive(ctx context.Context) ([]domain.AdminUser, error)
}

type FollowedChannelRepository interface {
	Upsert(ctx context.Context, channel domain.FollowedChannel) (domain.FollowedChannel, error)
	GetByID(ctx context.Context, id int64) (domain.FollowedChannel, error)
	GetBySlug(ctx context.Context, slug string) (domain.FollowedChannel, error)
	GetByChatroomID(ctx context.Context, kickChatroomID int64) (domain.FollowedChannel, error)
	GetByBroadcasterUserID(ctx context.Context, broadcasterUserID int64) (domain.FollowedChannel, error)
	List(ctx context.Context) ([]domain.FollowedChannel, error)
	ListEnabled(ctx context.Context) ([]domain.FollowedChannel, error)
	Disable(ctx context.Context, id int64) (domain.FollowedChannel, error)
}

type MessageRepository interface {
	Insert(ctx context.Context, message domain.ChatMessage) error
	InsertMessagesBatch(ctx context.Context, messages []domain.ChatMessage) error
	ExistsByKickMessageID(ctx context.Context, kickMessageID string) (bool, error)
	ExistingKickMessageIDs(ctx context.Context, kickMessageIDs []string) (map[string]bool, error)
	Search(ctx context.Context, filter domain.MessageSearchFilter) ([]domain.ChatMessage, error)
}

type AnalyticsRepository interface {
	Overview(ctx context.Context, filter domain.AnalyticsFilter) (domain.AnalyticsOverview, error)
	MessageVolume(
		ctx context.Context,
		filter domain.AnalyticsFilter,
		bucket domain.AnalyticsBucket,
	) ([]domain.MessageVolumePoint, error)
	TopSenders(
		ctx context.Context,
		filter domain.AnalyticsFilter,
		limit uint64,
	) ([]domain.TopSenderAnalytics, error)
	TopChannels(
		ctx context.Context,
		filter domain.AnalyticsFilter,
		limit uint64,
	) ([]domain.TopChannelAnalytics, error)
	TopEmotes(
		ctx context.Context,
		filter domain.AnalyticsFilter,
		limit uint64,
	) ([]domain.TopEmoteAnalytics, error)
	LatestMessages(
		ctx context.Context,
		filter domain.AnalyticsFilter,
		limit uint64,
	) ([]domain.ChatMessage, error)
}

type RawEventRepository interface {
	InsertEvent(ctx context.Context, event domain.RawKickEvent) error
	InsertEventsBatch(ctx context.Context, events []domain.RawKickEvent) error
	InsertAttempt(ctx context.Context, attempt domain.RawEventAttempt) error
	InsertAttemptsBatch(ctx context.Context, attempts []domain.RawEventAttempt) error
	ListUnprocessed(ctx context.Context, limit uint64, maxAttempts uint16) ([]domain.RawKickEvent, error)
	CountUnprocessed(ctx context.Context, maxAttempts uint16) (int64, error)
	AttemptCount(ctx context.Context, rawEventID string) (uint16, error)
	GetByID(ctx context.Context, rawEventID string) (domain.RawKickEvent, error)
	GetByIDs(ctx context.Context, rawEventIDs []string) (map[string]domain.RawKickEvent, error)
}

type RawEventClaimRepository interface {
	TryClaim(ctx context.Context, rawEventID string, workerID string, leaseDuration time.Duration) (bool, error)
	MarkCompleted(ctx context.Context, rawEventID string, workerID string) error
	Release(ctx context.Context, rawEventID string, workerID string) error
}

type RawEventQueueRepository interface {
	Enqueue(ctx context.Context, item domain.RawEventQueueItem) error
	EnqueueBatch(ctx context.Context, items []domain.RawEventQueueItem) error
	ListPending(ctx context.Context, limit uint64, maxAttempts uint16) ([]domain.RawEventQueueItem, error)
	Claim(ctx context.Context, rawEventID string, workerID string) (bool, error)
	Release(ctx context.Context, rawEventID string, workerID string) error
	MarkProcessed(ctx context.Context, rawEventID string) error
	MarkFailed(ctx context.Context, rawEventID string, errMessage string, maxAttempts uint16) error
	CountPending(ctx context.Context, maxAttempts uint16) (int64, error)
	OldestPendingAge(ctx context.Context, maxAttempts uint16) (time.Duration, error)
	RecoverStaleClaims(ctx context.Context, olderThan time.Duration) (int64, error)
	GetByID(ctx context.Context, rawEventID string) (domain.RawEventQueueItem, error)
}

type SenderProfileRepository interface {
	Upsert(ctx context.Context, sender domain.SenderProfile) (domain.SenderProfile, error)
	GetByKickUserID(ctx context.Context, kickUserID int64) (domain.SenderProfile, error)
	GetBySlug(ctx context.Context, slug string) (domain.SenderProfile, error)
}

type WorkerHeartbeatRepository interface {
	Upsert(ctx context.Context, heartbeat domain.ListenerHeartbeat) error
}

type StorageStatsRepository interface {
	TableSizes(ctx context.Context) ([]domain.TableSize, error)
}

type OperationsRepository interface {
	Summary(ctx context.Context) (domain.OperationsSummary, error)
	ListFailedEvents(ctx context.Context, limit int) ([]domain.FailedRawEvent, error)
	RetryFailedEvents(ctx context.Context) (int64, error)
	ClearFailedEvents(ctx context.Context) (int64, error)
}

type KickWebhookEventRepository interface {
	InsertIdempotent(ctx context.Context, event domain.KickWebhookEvent) error
	GetByMessageID(ctx context.Context, messageID string) (domain.KickWebhookEvent, error)
	ListPending(ctx context.Context, limit int, maxAttempts int) ([]domain.KickWebhookEvent, error)
	MarkProcessed(ctx context.Context, messageID string) error
	MarkFailed(ctx context.Context, messageID string, errMessage string, maxAttempts int) error
	MarkIgnored(ctx context.Context, messageID string, reason string) error
	PruneTerminalBefore(ctx context.Context, cutoff time.Time) (int64, error)
	CountByStatus(ctx context.Context) (map[string]int64, error)
	LatestReceivedAt(ctx context.Context) (time.Time, error)
}

type KickEventSubscriptionRepository interface {
	Upsert(ctx context.Context, sub domain.KickEventSubscription) (domain.KickEventSubscription, error)
	ListByChannel(ctx context.Context, followedChannelID int64) ([]domain.KickEventSubscription, error)
	DeleteByChannel(ctx context.Context, followedChannelID int64) error
	List(ctx context.Context) ([]domain.KickEventSubscription, error)
	UpdateSyncError(ctx context.Context, id int64, syncError string) error
}

type SubscriptionPeriodRepository interface {
	InsertBatch(ctx context.Context, periods []domain.ChannelSubscriptionPeriod) error
	ActiveSummary(ctx context.Context, followedChannelID int64) (domain.ChannelSubscriptionSummary, error)
}

type DataManagementRepository interface {
	Summary(ctx context.Context) (domain.DataManagementSummary, error)
	GetRetentionSettings(ctx context.Context) (domain.RetentionSettings, error)
	UpdateRetentionSettings(ctx context.Context, settings domain.RetentionSettings) (domain.RetentionSettings, error)
	CountCleanup(ctx context.Context, criteria domain.DataCleanupCriteria) (domain.DataCleanupCounts, error)
	ExecuteCleanup(ctx context.Context, criteria domain.DataCleanupCriteria) (domain.DataCleanupCounts, error)
}
