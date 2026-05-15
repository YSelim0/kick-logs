from dataclasses import dataclass

from kick_logs.application.dto.analytics import (
    AnalyticsOverviewDTO,
    MessageVolumePointDTO,
    TopChannelDTO,
    TopEmoteDTO,
)
from kick_logs.application.dto.messages import MessageSearchItemDTO


@dataclass(frozen=True, slots=True)
class UserProfileSenderDTO:
    id: int
    kick_user_id: int
    username: str
    slug: str
    profile_image_url: str | None


@dataclass(frozen=True, slots=True)
class UserProfileDTO:
    sender: UserProfileSenderDTO
    overview: AnalyticsOverviewDTO
    message_volume: list[MessageVolumePointDTO]
    top_channels: list[TopChannelDTO]
    top_emotes: list[TopEmoteDTO]
    latest_messages: list[MessageSearchItemDTO]
