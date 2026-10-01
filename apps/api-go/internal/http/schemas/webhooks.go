package schemas

type ChannelSubscriptionSummaryResponse struct {
	ChannelSlug       string  `json:"channel_slug"`
	ActiveCount       int64   `json:"active_count"`
	ActiveGiftedCount int64   `json:"active_gifted_count"`
	LatestEventAt     *string `json:"latest_event_at,omitempty"`
}

type ChannelSubscriberResponse struct {
	SubscriberKickUserID  int64   `json:"subscriber_kick_user_id"`
	Username              string  `json:"username"`
	Slug                  string  `json:"slug"`
	ProfileImageURL       string  `json:"profile_image_url"`
	IsGift                bool    `json:"is_gift"`
	GifterKickUserID      *int64  `json:"gifter_kick_user_id,omitempty"`
	GifterUsername        *string `json:"gifter_username,omitempty"`
	GifterSlug            *string `json:"gifter_slug,omitempty"`
	GifterProfileImageURL *string `json:"gifter_profile_image_url,omitempty"`
	StartedAt             string  `json:"started_at"`
	ExpiresAt             string  `json:"expires_at"`
}

type ChannelSubscribersResponse struct {
	ChannelSlug string                      `json:"channel_slug"`
	GiftOnly    bool                        `json:"gift_only"`
	Count       int64                       `json:"count"`
	Limit       int64                       `json:"limit"`
	Offset      int64                       `json:"offset"`
	Items       []ChannelSubscriberResponse `json:"items"`
}

type ChannelSubscribersExportResponse struct {
	ChannelSlug string                      `json:"channel_slug"`
	GiftOnly    bool                        `json:"gift_only"`
	GeneratedAt string                      `json:"generated_at"`
	Count       int64                       `json:"count"`
	Items       []ChannelSubscriberResponse `json:"items"`
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
	FollowedChannelID int64            `json:"followed_channel_id"`
	Slug              string           `json:"slug"`
	BroadcasterUserID int64            `json:"broadcaster_user_id"`
	Subscriptions     []EventSubStatus `json:"subscriptions"`
}

type EventSubStatus struct {
	EventType          string  `json:"event_type"`
	KickSubscriptionID string  `json:"kick_subscription_id"`
	Status             string  `json:"status"`
	LatestSyncError    string  `json:"latest_sync_error,omitempty"`
	SyncedAt           *string `json:"synced_at,omitempty"`
}
