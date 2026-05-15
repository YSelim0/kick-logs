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

type ChatEmote struct {
	ID       string
	Name     string
	Token    string
	ImageURL string
}

type ChatMessage struct {
	ID                     int64
	KickMessageID          string
	ChannelKickID          int64
	ChannelChatroomID      int64
	ChannelSlug            string
	ChannelDisplayName     string
	ChannelProfileImageURL string
	ChannelPublicURL       string
	SenderKickID           int64
	SenderUsername         string
	SenderSlug             string
	SenderDisplayColor     string
	SenderProfileImageURL  string
	SenderPublicURL        string
	MessageType            string
	Content                string
	Emotes                 []ChatEmote
	ReplyToSender          string
	ReplyToContent         string
	ReplyToMessageID       string
	ThreadParentID         string
	RawPayloadJSON         string
	MessageCreatedAt       time.Time
	IngestedAt             time.Time
}

type MessageSearchFilter struct {
	Sender  string
	Channel string
	Query   string
	Limit   uint64
}

type RawKickEvent struct {
	ID           string
	ChannelSlug  string
	EventType    string
	EventName    string
	PayloadJSON  string
	Status       string
	ReceivedAt   time.Time
	ProcessedAt  time.Time
	ErrorMessage string
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
