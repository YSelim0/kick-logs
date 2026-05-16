package schemas

type AnalyticsOverviewResponse struct {
	TotalMessages    int64   `json:"total_messages"`
	TotalSenders     int64   `json:"total_senders"`
	TotalChannels    int64   `json:"total_channels"`
	TotalEmoteUsages int64   `json:"total_emote_usages"`
	FirstMessageAt   *string `json:"first_message_at"`
	LatestMessageAt  *string `json:"latest_message_at"`
}

type MessageVolumePointResponse struct {
	BucketStart  string `json:"bucket_start"`
	MessageCount int64  `json:"message_count"`
}

type MessageVolumeResponse struct {
	Items []MessageVolumePointResponse `json:"items"`
}

type TopSenderResponse struct {
	SenderID        int64   `json:"sender_id"`
	KickUserID      int64   `json:"kick_user_id"`
	Username        string  `json:"username"`
	Slug            string  `json:"slug"`
	ProfileImageURL *string `json:"profile_image_url"`
	MessageCount    int64   `json:"message_count"`
	FirstMessageAt  string  `json:"first_message_at"`
	LatestMessageAt string  `json:"latest_message_at"`
}

type TopSendersResponse struct {
	Items []TopSenderResponse `json:"items"`
}

type TopChannelResponse struct {
	ChannelID       int64   `json:"channel_id"`
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	ProfileImageURL *string `json:"profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url"`
	MessageCount    int64   `json:"message_count"`
	FirstMessageAt  string  `json:"first_message_at"`
	LatestMessageAt string  `json:"latest_message_at"`
}

type TopChannelsResponse struct {
	Items []TopChannelResponse `json:"items"`
}

type TopEmoteResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Token        string `json:"token"`
	ImageURL     string `json:"image_url"`
	UsageCount   int64  `json:"usage_count"`
	MessageCount int64  `json:"message_count"`
}

type TopEmotesResponse struct {
	Items []TopEmoteResponse `json:"items"`
}
