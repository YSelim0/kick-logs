package schemas

type HomepageResponse struct {
	Status        string                       `json:"status"`
	AsOf          string                       `json:"as_of"`
	Start         string                       `json:"start"`
	End           string                       `json:"end"`
	Timezone      string                       `json:"timezone"`
	Stale         bool                         `json:"stale"`
	Overview      AnalyticsOverviewResponse    `json:"overview"`
	MessageVolume []MessageVolumePointResponse `json:"message_volume"`
	TopChannels   []TopChannelResponse         `json:"top_channels"`
	TopSenders    []TopSenderResponse          `json:"top_senders"`
	TopEmotes     []TopEmoteResponse           `json:"top_emotes"`
}

type HomepagePendingResponse struct {
	Status            string `json:"status"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}
