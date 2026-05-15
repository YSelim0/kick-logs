from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True, slots=True)
class AnalyticsOverviewDTO:
    total_messages: int
    total_senders: int
    total_channels: int
    total_emote_usages: int
    first_message_at: datetime | None
    latest_message_at: datetime | None


@dataclass(frozen=True, slots=True)
class MessageVolumePointDTO:
    bucket_start: datetime
    message_count: int


@dataclass(frozen=True, slots=True)
class TopSenderDTO:
    sender_id: int
    kick_user_id: int
    username: str
    slug: str
    profile_image_url: str | None
    message_count: int
    first_message_at: datetime
    latest_message_at: datetime


@dataclass(frozen=True, slots=True)
class TopChannelDTO:
    channel_id: int
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None
    message_count: int
    first_message_at: datetime
    latest_message_at: datetime


@dataclass(frozen=True, slots=True)
class TopEmoteDTO:
    id: str
    name: str
    token: str
    image_url: str
    usage_count: int
    message_count: int
