package domain

import "time"

type AdminRole string

const (
	AdminRoleAdmin      AdminRole = "admin"
	AdminRoleSuperAdmin AdminRole = "super_admin"
)

type AdminUser struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         AdminRole
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type FollowedChannel struct {
	ID                int64
	KickChannelID     int64
	KickChatroomID    int64
	BroadcasterUserID int64
	Slug              string
	DisplayName       string
	ProfileImageURL   string
	BannerImageURL    string
	IsEnabled         bool
	RawPayloadJSON    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	LastResolvedAt    time.Time
	LastMessageAt     time.Time
	LastListenerError string
}

type SenderProfile struct {
	ID                    int64
	KickUserID            int64
	Username              string
	Slug                  string
	ProfileImageURL       string
	LastSeenColor         string
	RawProfilePayloadJSON string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastSeenAt            time.Time
}

type ListenerChannel struct {
	ID             int64
	KickChannelID  int64
	KickChatroomID int64
	Slug           string
	DisplayName    string
}

type ChatEmote struct {
	ID       string
	Name     string
	Token    string
	ImageURL string
}

type ChatMessage struct {
	ID                     int64
	KickMessageID          string
	ChannelID              int64
	ChannelKickID          int64
	ChannelChatroomID      int64
	ChannelSlug            string
	ChannelDisplayName     string
	ChannelProfileImageURL string
	ChannelBannerImageURL  string
	ChannelPublicURL       string
	SenderID               int64
	SenderKickID           int64
	SenderUsername         string
	SenderSlug             string
	SenderDisplayColor     string
	SenderProfileImageURL  string
	SenderPublicURL        string
	SenderBadgesJSON       string
	MessageType            string
	Content                string
	Emotes                 []ChatEmote
	ReplyToSender          string
	ReplyToContent         string
	ReplyToMessageID       string
	ThreadParentID         string
	ReplyMetadataJSON      string
	RawPayloadJSON         string
	MessageCreatedAt       time.Time
	IngestedAt             time.Time
}

type MessageCursor struct {
	MessageCreatedAt time.Time
	MessageID        int64
}

type MessageSearchFilter struct {
	Sender    string
	Channel   string
	Query     string
	Start     time.Time
	End       time.Time
	ReplyOnly bool
	EmoteOnly bool
	Cursor    *MessageCursor
	Limit     uint64
}

type AnalyticsBucket string

const (
	AnalyticsBucketHour AnalyticsBucket = "hour"
	AnalyticsBucketDay  AnalyticsBucket = "day"
)

type AnalyticsFilter struct {
	Start   time.Time
	End     time.Time
	Channel string
	Sender  string
	Query   string
}

type AnalyticsOverview struct {
	TotalMessages    int64
	TotalSenders     int64
	TotalChannels    int64
	TotalEmoteUsages int64
	FirstMessageAt   time.Time
	LatestMessageAt  time.Time
}

type MessageVolumePoint struct {
	BucketStart  time.Time
	MessageCount int64
}

type TopSenderAnalytics struct {
	SenderID        int64
	KickUserID      int64
	Username        string
	Slug            string
	ProfileImageURL string
	MessageCount    int64
	FirstMessageAt  time.Time
	LatestMessageAt time.Time
}

type TopChannelAnalytics struct {
	ChannelID       int64
	Slug            string
	DisplayName     string
	ProfileImageURL string
	BannerImageURL  string
	MessageCount    int64
	FirstMessageAt  time.Time
	LatestMessageAt time.Time
}

type TopEmoteAnalytics struct {
	ID           string
	Name         string
	Token        string
	ImageURL     string
	UsageCount   int64
	MessageCount int64
}

type UserProfile struct {
	Sender         SenderProfile
	Overview       AnalyticsOverview
	MessageVolume  []MessageVolumePoint
	TopChannels    []TopChannelAnalytics
	TopEmotes      []TopEmoteAnalytics
	LatestMessages []ChatMessage
}

type ChannelProfile struct {
	Channel        FollowedChannel
	Overview       AnalyticsOverview
	MessageVolume  []MessageVolumePoint
	TopSenders     []TopSenderAnalytics
	TopEmotes      []TopEmoteAnalytics
	LatestMessages []ChatMessage
}

type RawKickEvent struct {
	ID                  string
	ChannelSlug         string
	EventType           string
	EventName           string
	KickMessageID       string
	ChatroomID          int64
	ChannelID           int64
	PayloadJSON         string
	MetadataJSON        string
	Status              string
	Attempts            uint16
	ReceivedAt          time.Time
	ProcessedAt         time.Time
	ProcessingStartedAt time.Time
	ErrorMessage        string
}

type RawStreamEvent struct {
	ID      string
	Subject string
	Payload []byte
	Headers map[string]string
}

type RawStreamPublishAck struct {
	Stream    string
	Sequence  uint64
	Duplicate bool
}

type RawStreamStats struct {
	StreamName               string
	ConsumerName             string
	Messages                 int64
	Bytes                    int64
	ConsumerPending          int64
	ConsumerAckPending       int64
	ConsumerRedelivered      int64
	OldestPendingAgeSeconds  int64
	LatestMessageAgeSeconds  int64
	LatestConsumerUpdateTime time.Time
}

type RawChatEventEnvelope struct {
	RawEventID        string    `json:"raw_event_id"`
	KickMessageID     string    `json:"kick_message_id"`
	EventName         string    `json:"event_name"`
	PusherChannel     string    `json:"pusher_channel"`
	FollowedChannelID int64     `json:"followed_channel_id"`
	ChannelSlug       string    `json:"channel_slug"`
	KickChannelID     int64     `json:"kick_channel_id"`
	KickChatroomID    int64     `json:"kick_chatroom_id"`
	ReceivedAt        time.Time `json:"received_at"`
	PayloadJSON       string    `json:"payload_json"`
	RawPusherJSON     string    `json:"raw_pusher_json"`
}

type RawEventAttempt struct {
	ID           string
	RawEventID   string
	Attempt      uint16
	Status       string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

type RawEventClaim struct {
	RawEventID     string
	WorkerID       string
	Status         string
	LeaseExpiresAt time.Time
	ClaimedAt      time.Time
	CompletedAt    time.Time
	UpdatedAt      time.Time
}

const (
	RawEventQueueStatusPending   = "pending"
	RawEventQueueStatusClaimed   = "claimed"
	RawEventQueueStatusProcessed = "processed"
	RawEventQueueStatusFailed    = "failed"
)

type RawEventQueueItem struct {
	RawEventID    string
	ChannelID     int64
	ChatroomID    int64
	ChannelSlug   string
	KickMessageID string
	Status        string
	Attempts      uint16
	ClaimedBy     string
	ClaimedAt     time.Time
	EnqueuedAt    time.Time
	LastError     string
	UpdatedAt     time.Time
}

type TableSize struct {
	Name        string
	Rows        int64
	BytesOnDisk int64
}

type OperationsCounts struct {
	Channels        int64
	EnabledChannels int64
	Senders         int64
	Messages        int64
	RawEvents       int64
}

type OperationsTimestamps struct {
	LatestMessageAt                 time.Time
	LatestRawEventReceivedAt        time.Time
	LatestRawEventProcessedAt       time.Time
	OldestPendingRawEventReceivedAt time.Time
}

type ListenerHeartbeat struct {
	ServiceName          string
	LastSeenAt           time.Time
	MetadataJSON         string
	IsFresh              bool
	StaleAfterSeconds    int
	SecondsSinceLastSeen int64
}

type OperationsSummary struct {
	Counts               OperationsCounts
	RawEventStatusCounts map[string]int64
	StorageDatabaseBytes int64
	StorageTables        []TableSize
	Timestamps           OperationsTimestamps
	Listener             ListenerHeartbeat
	Processor            ListenerHeartbeat
	Ingestion            IngestionHealth
}

type IngestionHealth struct {
	QueueDepth                     int64
	OldestPendingAgeSeconds        int64
	LegacyQueueDepth               int64
	LegacyOldestPendingAgeSeconds  int64
	CapturedRawEvents              int64
	RecentMessagePollCaptured      int64
	RecentMessagePollErrors        int64
	StreamMessages                 int64
	StreamBytes                    int64
	StreamConsumerPending          int64
	StreamConsumerAckPending       int64
	StreamConsumerRedelivered      int64
	StreamOldestPendingAgeSeconds  int64
	StreamLatestMessageAgeSeconds  int64
	StreamLatestConsumerUpdateTime time.Time
	StreamError                    string
	WriteQueueDepth                int64
	WriteQueueHighWater            int64
	WriteDropCount                 int64
	WriteFlushCount                int64
	LastFlushSize                  int64
	LastFlushMillis                int64
	ClickHouseFailures             int64
	QueueEnqueueFailures           int64
	BreakerState                   string
	BreakerCurrentDelayMS          int64
}

type FailedRawEvent struct {
	RawEventID   string
	ChannelSlug  string
	ErrorMessage string
	Attempts     uint16
	ReceivedAt   time.Time
	FailedAt     time.Time
}

type RetentionSettings struct {
	ID                    int64
	MessageRetentionDays  *int
	RawEventRetentionDays *int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MigrationCounts struct {
	AdminUsers        int64 `json:"admin_users"`
	FollowedChannels  int64 `json:"followed_channels"`
	SenderProfiles    int64 `json:"sender_profiles"`
	RetentionSettings int64 `json:"retention_settings"`
	WorkerHeartbeats  int64 `json:"worker_heartbeats"`
	ChatMessages      int64 `json:"chat_messages"`
	RawEvents         int64 `json:"raw_events"`
	RawEventAttempts  int64 `json:"raw_event_attempts"`
}

type DataMigrationRun struct {
	RunID                 string
	Name                  string
	Mode                  string
	Status                string
	SourceCountsJSON      string
	DestinationCountsJSON string
	ValidationJSON        string
	ErrorMessage          string
	StartedAt             time.Time
	FinishedAt            time.Time
}

type DataManagementCounts struct {
	Channels  int64
	Senders   int64
	Messages  int64
	RawEvents int64
}

type DataManagementSummary struct {
	Counts            DataManagementCounts
	DatabaseBytes     int64
	Tables            []TableSize
	RetentionSettings RetentionSettings
}

type DataCleanupTarget string

const (
	DataCleanupTargetOldMessages  DataCleanupTarget = "old_messages"
	DataCleanupTargetOldRawEvents DataCleanupTarget = "old_raw_events"
	DataCleanupTargetChannel      DataCleanupTarget = "channel"
	DataCleanupTargetSender       DataCleanupTarget = "sender"
)

type DataCleanupRequest struct {
	Target      DataCleanupTarget
	ChannelSlug string
	Sender      string
}

type DataCleanupCriteria struct {
	Target        DataCleanupTarget
	CutoffAt      time.Time
	ChannelSlug   string
	Sender        string
	RetentionDays *int
}

type DataCleanupCounts struct {
	Messages  int64
	RawEvents int64
}

func (counts DataCleanupCounts) Total() int64 {
	return counts.Messages + counts.RawEvents
}

type DataCleanupPreview struct {
	Target           DataCleanupTarget
	Affected         DataCleanupCounts
	ConfirmationText string
	CanExecute       bool
	CutoffAt         time.Time
	ChannelSlug      string
	Sender           string
	RetentionDays    *int
	Reason           string
}

type DataCleanupResult struct {
	Target           DataCleanupTarget
	Deleted          DataCleanupCounts
	ConfirmationText string
	CutoffAt         time.Time
	ChannelSlug      string
	Sender           string
	RetentionDays    *int
}

const (
	WebhookEventStatusPending   = "pending"
	WebhookEventStatusProcessed = "processed"
	WebhookEventStatusFailed    = "failed"
	WebhookEventStatusIgnored   = "ignored"
)

type KickWebhookEvent struct {
	MessageID      string
	SubscriptionID string
	EventType      string
	EventVersion   string
	RawPayloadJSON string
	Status         string
	Attempts       int
	ReceivedAt     time.Time
	ProcessedAt    time.Time
	ErrorMessage   string
}

const (
	KickEventSubStatusActive  = "active"
	KickEventSubStatusDeleted = "deleted"
	KickEventSubStatusError   = "error"
)

type KickEventSubscription struct {
	ID                 int64
	FollowedChannelID  int64
	BroadcasterUserID  int64
	EventType          string
	EventVersion       string
	Method             string
	KickSubscriptionID string
	Status             string
	LatestSyncError    string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SyncedAt           time.Time
}

type KickAPIEventSub struct {
	SubscriptionID    string
	BroadcasterUserID int64
	EventType         string
	Method            string
	CreatedAt         time.Time
}

type ChannelSubscriptionPeriod struct {
	ID                        string
	EventMessageID            string
	EventType                 string
	FollowedChannelID         int64
	BroadcasterUserID         int64
	ChannelSlug               string
	ChannelDisplayName        string
	SubscriberKickUserID      int64
	SubscriberUsername        string
	SubscriberSlug            string
	SubscriberProfileImageURL string
	GifterKickUserID          int64
	GifterUsername            string
	GifterSlug                string
	GifterProfileImageURL     string
	IsGift                    bool
	StartedAt                 time.Time
	ExpiresAt                 time.Time
	RawPayloadJSON            string
	IngestedAt                time.Time
}

type ChannelSubscriptionSummary struct {
	ChannelSlug       string
	ActiveCount       int64
	ActiveGiftedCount int64
	LatestEventAt     time.Time
}
