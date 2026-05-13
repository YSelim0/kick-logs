from datetime import datetime

from pydantic import BaseModel

from kick_logs.application.dto.analytics import (
    AnalyticsOverviewDTO,
    MessageVolumePointDTO,
    TopChannelDTO,
    TopEmoteDTO,
    TopSenderDTO,
)


class AnalyticsOverviewResponse(BaseModel):
    total_messages: int
    total_senders: int
    total_channels: int
    total_emote_usages: int
    first_message_at: datetime | None
    latest_message_at: datetime | None

    @classmethod
    def from_dto(cls, overview: AnalyticsOverviewDTO) -> "AnalyticsOverviewResponse":
        return cls(
            total_messages=overview.total_messages,
            total_senders=overview.total_senders,
            total_channels=overview.total_channels,
            total_emote_usages=overview.total_emote_usages,
            first_message_at=overview.first_message_at,
            latest_message_at=overview.latest_message_at,
        )


class MessageVolumePointResponse(BaseModel):
    bucket_start: datetime
    message_count: int

    @classmethod
    def from_dto(cls, point: MessageVolumePointDTO) -> "MessageVolumePointResponse":
        return cls(bucket_start=point.bucket_start, message_count=point.message_count)


class MessageVolumeResponse(BaseModel):
    items: list[MessageVolumePointResponse]

    @classmethod
    def from_dto(cls, points: list[MessageVolumePointDTO]) -> "MessageVolumeResponse":
        return cls(items=[MessageVolumePointResponse.from_dto(point) for point in points])


class TopSenderResponse(BaseModel):
    sender_id: int
    kick_user_id: int
    username: str
    slug: str
    profile_image_url: str | None
    message_count: int
    first_message_at: datetime
    latest_message_at: datetime

    @classmethod
    def from_dto(cls, sender: TopSenderDTO) -> "TopSenderResponse":
        return cls(
            sender_id=sender.sender_id,
            kick_user_id=sender.kick_user_id,
            username=sender.username,
            slug=sender.slug,
            profile_image_url=sender.profile_image_url,
            message_count=sender.message_count,
            first_message_at=sender.first_message_at,
            latest_message_at=sender.latest_message_at,
        )


class TopSendersResponse(BaseModel):
    items: list[TopSenderResponse]

    @classmethod
    def from_dto(cls, senders: list[TopSenderDTO]) -> "TopSendersResponse":
        return cls(items=[TopSenderResponse.from_dto(sender) for sender in senders])


class TopChannelResponse(BaseModel):
    channel_id: int
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None
    message_count: int
    first_message_at: datetime
    latest_message_at: datetime

    @classmethod
    def from_dto(cls, channel: TopChannelDTO) -> "TopChannelResponse":
        return cls(
            channel_id=channel.channel_id,
            slug=channel.slug,
            display_name=channel.display_name,
            profile_image_url=channel.profile_image_url,
            banner_image_url=channel.banner_image_url,
            message_count=channel.message_count,
            first_message_at=channel.first_message_at,
            latest_message_at=channel.latest_message_at,
        )


class TopChannelsResponse(BaseModel):
    items: list[TopChannelResponse]

    @classmethod
    def from_dto(cls, channels: list[TopChannelDTO]) -> "TopChannelsResponse":
        return cls(items=[TopChannelResponse.from_dto(channel) for channel in channels])


class TopEmoteResponse(BaseModel):
    id: str
    name: str
    token: str
    image_url: str
    usage_count: int
    message_count: int

    @classmethod
    def from_dto(cls, emote: TopEmoteDTO) -> "TopEmoteResponse":
        return cls(
            id=emote.id,
            name=emote.name,
            token=emote.token,
            image_url=emote.image_url,
            usage_count=emote.usage_count,
            message_count=emote.message_count,
        )


class TopEmotesResponse(BaseModel):
    items: list[TopEmoteResponse]

    @classmethod
    def from_dto(cls, emotes: list[TopEmoteDTO]) -> "TopEmotesResponse":
        return cls(items=[TopEmoteResponse.from_dto(emote) for emote in emotes])
