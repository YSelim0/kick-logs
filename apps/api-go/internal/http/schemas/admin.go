package schemas

type AdminUserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type AuthResponse struct {
	User AdminUserResponse `json:"user"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateAdminUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChannelResponse struct {
	ID              int64   `json:"id"`
	KickChannelID   *int64  `json:"kick_channel_id"`
	KickChatroomID  *int64  `json:"kick_chatroom_id"`
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	ProfileImageURL *string `json:"profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url"`
	IsEnabled       bool    `json:"is_enabled"`
}

type AddChannelRequest struct {
	Slug string `json:"slug"`
}

type OperationsCountsResponse struct {
	Channels        int64 `json:"channels"`
	EnabledChannels int64 `json:"enabled_channels"`
	Senders         int64 `json:"senders"`
	Messages        int64 `json:"messages"`
	RawEvents       int64 `json:"raw_events"`
}

type OperationsStorageTableResponse struct {
	TableName  string `json:"table_name"`
	TotalBytes int64  `json:"total_bytes"`
}

type OperationsStorageResponse struct {
	DatabaseBytes int64                            `json:"database_bytes"`
	Tables        []OperationsStorageTableResponse `json:"tables"`
}

type OperationsTimestampsResponse struct {
	LatestMessageAt                 *string `json:"latest_message_at"`
	LatestRawEventReceivedAt        *string `json:"latest_raw_event_received_at"`
	LatestRawEventProcessedAt       *string `json:"latest_raw_event_processed_at"`
	OldestPendingRawEventReceivedAt *string `json:"oldest_pending_raw_event_received_at"`
}

type ListenerHeartbeatResponse struct {
	ServiceName          string  `json:"service_name"`
	LastSeenAt           *string `json:"last_seen_at"`
	IsFresh              bool    `json:"is_fresh"`
	StaleAfterSeconds    int     `json:"stale_after_seconds"`
	SecondsSinceLastSeen *int64  `json:"seconds_since_last_seen"`
}

type OperationsSummaryResponse struct {
	Counts               OperationsCountsResponse     `json:"counts"`
	RawEventStatusCounts map[string]int64             `json:"raw_event_status_counts"`
	Storage              OperationsStorageResponse    `json:"storage"`
	Timestamps           OperationsTimestampsResponse `json:"timestamps"`
	Listener             ListenerHeartbeatResponse    `json:"listener"`
	Ingestion            IngestionHealthResponse      `json:"ingestion"`
}

type IngestionHealthResponse struct {
	QueueDepth              int64  `json:"queue_depth"`
	OldestPendingAgeSeconds int64  `json:"oldest_pending_age_seconds"`
	WriteQueueDepth         int64  `json:"write_queue_depth"`
	WriteQueueHighWater     int64  `json:"write_queue_high_water_mark"`
	WriteDropCount          int64  `json:"write_drop_count"`
	WriteFlushCount         int64  `json:"write_flush_count"`
	LastFlushSize           int64  `json:"last_flush_size"`
	LastFlushMillis         int64  `json:"last_flush_millis"`
	ClickHouseFailures      int64  `json:"clickhouse_insert_failures"`
	QueueEnqueueFailures    int64  `json:"queue_enqueue_failures"`
	BreakerState            string `json:"breaker_state"`
	BreakerCurrentDelayMS   int64  `json:"breaker_current_delay_ms"`
}

type MessageEmoteResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Token    string `json:"token"`
	ImageURL string `json:"image_url"`
}

type MessageSenderResponse struct {
	ID              int64   `json:"id"`
	KickUserID      int64   `json:"kick_user_id"`
	Username        string  `json:"username"`
	Slug            string  `json:"slug"`
	ProfileImageURL *string `json:"profile_image_url"`
}

type MessageChannelResponse struct {
	ID              int64   `json:"id"`
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	ProfileImageURL *string `json:"profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url"`
}

type MessageResponse struct {
	ID                     int64                  `json:"id"`
	KickMessageID          string                 `json:"kick_message_id"`
	ChatroomID             int64                  `json:"chatroom_id"`
	Content                string                 `json:"content"`
	MessageType            string                 `json:"message_type"`
	SenderUsernameSnapshot string                 `json:"sender_username_snapshot"`
	SenderSlugSnapshot     string                 `json:"sender_slug_snapshot"`
	SenderColorSnapshot    *string                `json:"sender_color_snapshot"`
	SenderBadges           []map[string]any       `json:"sender_badges"`
	Emotes                 []MessageEmoteResponse `json:"emotes"`
	ReplyMetadata          map[string]any         `json:"reply_metadata"`
	ThreadParentID         *string                `json:"thread_parent_id"`
	MessageCreatedAt       string                 `json:"message_created_at"`
	IngestedAt             string                 `json:"ingested_at"`
	Sender                 MessageSenderResponse  `json:"sender"`
	Channel                MessageChannelResponse `json:"channel"`
}

type MessageSearchResponse struct {
	Items      []MessageResponse `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type MessageExportResponse struct {
	Items     []MessageResponse `json:"items"`
	Count     int               `json:"count"`
	MaxRows   int               `json:"max_rows"`
	Truncated bool              `json:"truncated"`
}
