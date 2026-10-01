package ports

import (
	"context"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

// ProfileSummaryRepository optionally combines the all-time overview and ranking.
// Profile filters use Sender/Channel, not the standalone rankings' free-text Query.
type ProfileSummaryRepository interface {
	OverviewAndTopSenders(context.Context, domain.AnalyticsFilter, uint64) (domain.AnalyticsOverview, []domain.TopSenderAnalytics, error)
	OverviewAndTopChannels(context.Context, domain.AnalyticsFilter, uint64) (domain.AnalyticsOverview, []domain.TopChannelAnalytics, error)
}
