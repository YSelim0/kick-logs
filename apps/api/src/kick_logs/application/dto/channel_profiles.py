from dataclasses import dataclass

from kick_logs.application.dto.analytics import (
    AnalyticsOverviewDTO,
    MessageVolumePointDTO,
    TopEmoteDTO,
    TopSenderDTO,
)
from kick_logs.application.dto.messages import MessageSearchItemDTO


@dataclass(frozen=True, slots=True)
class ChannelProfileChannelDTO:
    id: int
    kick_channel_id: int | None
    kick_chatroom_id: int | None
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None
    is_enabled: bool


@dataclass(frozen=True, slots=True)
class ChannelProfileDTO:
    channel: ChannelProfileChannelDTO
    overview: AnalyticsOverviewDTO
    message_volume: list[MessageVolumePointDTO]
    top_senders: list[TopSenderDTO]
    top_emotes: list[TopEmoteDTO]
    latest_messages: list[MessageSearchItemDTO]
