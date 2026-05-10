from dataclasses import dataclass, field
from typing import Any

from kick_logs.domain.entities.channel import Channel


@dataclass(frozen=True, slots=True)
class ResolvedKickChannelDTO:
    kick_channel_id: int
    kick_chatroom_id: int
    slug: str
    display_name: str
    profile_image_url: str | None = None
    banner_image_url: str | None = None
    raw_payload: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True, slots=True)
class ChannelDTO:
    id: int
    kick_channel_id: int | None
    kick_chatroom_id: int | None
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None
    is_enabled: bool


def channel_to_dto(channel: Channel) -> ChannelDTO:
    if channel.id is None:
        raise ValueError("Channel id is required for API responses.")
    return ChannelDTO(
        id=channel.id,
        kick_channel_id=channel.kick_channel_id,
        kick_chatroom_id=channel.kick_chatroom_id,
        slug=channel.slug,
        display_name=channel.display_name,
        profile_image_url=channel.profile_image_url,
        banner_image_url=channel.banner_image_url,
        is_enabled=channel.is_enabled,
    )
