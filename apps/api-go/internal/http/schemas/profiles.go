package schemas

type UserProfileSenderResponse struct {
	ID              int64   `json:"id"`
	KickUserID      int64   `json:"kick_user_id"`
	Username        string  `json:"username"`
	Slug            string  `json:"slug"`
	ProfileImageURL *string `json:"profile_image_url"`
}

type UserProfileResponse struct {
	Sender         UserProfileSenderResponse    `json:"sender"`
	Overview       AnalyticsOverviewResponse    `json:"overview"`
	MessageVolume  []MessageVolumePointResponse `json:"message_volume"`
	TopChannels    []TopChannelResponse         `json:"top_channels"`
	TopEmotes      []TopEmoteResponse           `json:"top_emotes"`
	LatestMessages []MessageResponse            `json:"latest_messages"`
}

type ChannelProfileChannelResponse struct {
	ID              int64   `json:"id"`
	KickChannelID   *int64  `json:"kick_channel_id"`
	KickChatroomID  *int64  `json:"kick_chatroom_id"`
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	ProfileImageURL *string `json:"profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url"`
	IsEnabled       bool    `json:"is_enabled"`
}

type ChannelProfileResponse struct {
	Channel        ChannelProfileChannelResponse `json:"channel"`
	Overview       AnalyticsOverviewResponse     `json:"overview"`
	MessageVolume  []MessageVolumePointResponse  `json:"message_volume"`
	TopSenders     []TopSenderResponse           `json:"top_senders"`
	TopEmotes      []TopEmoteResponse            `json:"top_emotes"`
	LatestMessages []MessageResponse             `json:"latest_messages"`
}
