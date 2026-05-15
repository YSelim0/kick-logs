from pydantic import BaseModel

from kick_logs.application.dto.channel_profiles import (
    ChannelProfileChannelDTO,
    ChannelProfileDTO,
)
from kick_logs.presentation.http.schemas.analytics import (
    AnalyticsOverviewResponse,
    MessageVolumePointResponse,
    TopEmoteResponse,
    TopSenderResponse,
)
from kick_logs.presentation.http.schemas.messages import MessageResponse


class ChannelProfileChannelResponse(BaseModel):
    id: int
    kick_channel_id: int | None
    kick_chatroom_id: int | None
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None
    is_enabled: bool

    @classmethod
    def from_dto(cls, channel: ChannelProfileChannelDTO) -> "ChannelProfileChannelResponse":
        return cls(
            id=channel.id,
            kick_channel_id=channel.kick_channel_id,
            kick_chatroom_id=channel.kick_chatroom_id,
            slug=channel.slug,
            display_name=channel.display_name,
            profile_image_url=channel.profile_image_url,
            banner_image_url=channel.banner_image_url,
            is_enabled=channel.is_enabled,
        )


class ChannelProfileResponse(BaseModel):
    channel: ChannelProfileChannelResponse
    overview: AnalyticsOverviewResponse
    message_volume: list[MessageVolumePointResponse]
    top_senders: list[TopSenderResponse]
    top_emotes: list[TopEmoteResponse]
    latest_messages: list[MessageResponse]

    @classmethod
    def from_dto(cls, profile: ChannelProfileDTO) -> "ChannelProfileResponse":
        return cls(
            channel=ChannelProfileChannelResponse.from_dto(profile.channel),
            overview=AnalyticsOverviewResponse.from_dto(profile.overview),
            message_volume=[
                MessageVolumePointResponse.from_dto(point) for point in profile.message_volume
            ],
            top_senders=[TopSenderResponse.from_dto(sender) for sender in profile.top_senders],
            top_emotes=[TopEmoteResponse.from_dto(emote) for emote in profile.top_emotes],
            latest_messages=[
                MessageResponse.from_dto(message) for message in profile.latest_messages
            ],
        )
