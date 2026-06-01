package schemas

type ChannelSubscriptionSummaryResponse struct {
	ChannelSlug       string  `json:"channel_slug"`
	ActiveCount       int64   `json:"active_count"`
	ActiveGiftedCount int64   `json:"active_gifted_count"`
	LatestEventAt     *string `json:"latest_event_at,omitempty"`
}

type WebhookHealthResponse struct {
	ConfiguredEventTypes     []string            `json:"configured_event_types"`
	MissingClientCredentials bool                `json:"missing_client_credentials"`
	MissingWebhookPublicKey  bool                `json:"missing_webhook_public_key"`
	SyncEnabled              bool                `json:"webhook_sync_enabled"`
	LatestWebhookReceivedAt  *string             `json:"latest_webhook_received_at,omitempty"`
	InboxCounts              map[string]int64    `json:"inbox_counts"`
	Channels                 []ChannelSyncStatus `json:"channels"`
}

type ChannelSyncStatus struct {
	FollowedChannelID int64           `json:"followed_channel_id"`
	Slug              string          `json:"slug"`
	BroadcasterUserID int64           `json:"broadcaster_user_id"`
	Subscriptions     []EventSubStatus `json:"subscriptions"`
}

type EventSubStatus struct {
	EventType          string  `json:"event_type"`
	KickSubscriptionID string  `json:"kick_subscription_id"`
	Status             string  `json:"status"`
	LatestSyncError    string  `json:"latest_sync_error,omitempty"`
	SyncedAt           *string `json:"synced_at,omitempty"`
}
