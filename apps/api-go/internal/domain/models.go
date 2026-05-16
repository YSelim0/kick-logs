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

type RawEventAttempt struct {
	ID           string
	RawEventID   string
	Attempt      uint16
	Status       string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
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
