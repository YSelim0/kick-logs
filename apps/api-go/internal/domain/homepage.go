package domain

import "time"

// HomepageSnapshot is a disposable read model, never the source of message history.
type HomepageSnapshot struct {
	Version     int
	AsOf        time.Time
	Start       time.Time
	End         time.Time
	Overview    AnalyticsOverview
	Volume      []MessageVolumePoint
	TopChannels []TopChannelAnalytics
	TopSenders  []TopSenderAnalytics
	TopEmotes   []TopEmoteAnalytics
}
