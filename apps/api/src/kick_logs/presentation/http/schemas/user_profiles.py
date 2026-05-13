from pydantic import BaseModel

from kick_logs.application.dto.user_profiles import UserProfileDTO, UserProfileSenderDTO
from kick_logs.presentation.http.schemas.analytics import (
    AnalyticsOverviewResponse,
    MessageVolumePointResponse,
    TopChannelResponse,
    TopEmoteResponse,
)
from kick_logs.presentation.http.schemas.messages import MessageResponse


class UserProfileSenderResponse(BaseModel):
    id: int
    kick_user_id: int
    username: str
    slug: str
    profile_image_url: str | None

    @classmethod
    def from_dto(cls, sender: UserProfileSenderDTO) -> "UserProfileSenderResponse":
        return cls(
            id=sender.id,
            kick_user_id=sender.kick_user_id,
            username=sender.username,
            slug=sender.slug,
            profile_image_url=sender.profile_image_url,
        )


class UserProfileResponse(BaseModel):
    sender: UserProfileSenderResponse
    overview: AnalyticsOverviewResponse
    message_volume: list[MessageVolumePointResponse]
    top_channels: list[TopChannelResponse]
    top_emotes: list[TopEmoteResponse]
    latest_messages: list[MessageResponse]

    @classmethod
    def from_dto(cls, profile: UserProfileDTO) -> "UserProfileResponse":
        return cls(
            sender=UserProfileSenderResponse.from_dto(profile.sender),
            overview=AnalyticsOverviewResponse.from_dto(profile.overview),
            message_volume=[
                MessageVolumePointResponse.from_dto(point) for point in profile.message_volume
            ],
            top_channels=[TopChannelResponse.from_dto(channel) for channel in profile.top_channels],
            top_emotes=[TopEmoteResponse.from_dto(emote) for emote in profile.top_emotes],
            latest_messages=[
                MessageResponse.from_dto(message) for message in profile.latest_messages
            ],
        )
