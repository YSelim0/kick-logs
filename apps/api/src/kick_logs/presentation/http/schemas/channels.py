from pydantic import BaseModel, Field

from kick_logs.application.dto.channels import ChannelDTO


class ChannelResponse(BaseModel):
    id: int
    kick_channel_id: int | None
    kick_chatroom_id: int | None
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None
    is_enabled: bool

    @classmethod
    def from_dto(cls, channel: ChannelDTO) -> "ChannelResponse":
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


class AddChannelRequest(BaseModel):
    slug: str = Field(min_length=1, max_length=120)
