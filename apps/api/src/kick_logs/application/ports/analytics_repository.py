from typing import Protocol

from kick_logs.application.dto.analytics import (
    AnalyticsOverviewDTO,
    MessageVolumePointDTO,
    TopChannelDTO,
    TopEmoteDTO,
    TopSenderDTO,
)
from kick_logs.domain.value_objects.analytics_filters import AnalyticsBucket, AnalyticsFilters


class AnalyticsRepository(Protocol):
    async def get_overview(self, filters: AnalyticsFilters) -> AnalyticsOverviewDTO: ...

    async def get_message_volume(
        self,
        filters: AnalyticsFilters,
        bucket: AnalyticsBucket,
    ) -> list[MessageVolumePointDTO]: ...

    async def get_top_senders(
        self,
        filters: AnalyticsFilters,
        limit: int,
    ) -> list[TopSenderDTO]: ...

    async def get_top_channels(
        self,
        filters: AnalyticsFilters,
        limit: int,
    ) -> list[TopChannelDTO]: ...

    async def get_top_emotes(
        self,
        filters: AnalyticsFilters,
        limit: int,
    ) -> list[TopEmoteDTO]: ...
