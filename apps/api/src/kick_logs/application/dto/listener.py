from dataclasses import dataclass

from kick_logs.domain.entities import Channel


@dataclass(frozen=True, slots=True)
class ListenerChannelDTO:
    id: int
    kick_channel_id: int
    kick_chatroom_id: int
    slug: str
    display_name: str


@dataclass(frozen=True, slots=True)
class SkippedListenerChannelDTO:
    id: int | None
    slug: str
    reason: str


@dataclass(frozen=True, slots=True)
class LoadEnabledChannelsResultDTO:
    channels: list[ListenerChannelDTO]
    skipped: list[SkippedListenerChannelDTO]


def channel_to_listener_dto(channel: Channel) -> ListenerChannelDTO:
    if channel.id is None:
        raise ValueError("Channel id is required for listener subscriptions.")
    if channel.kick_channel_id is None:
        raise ValueError("Kick channel id is required for listener subscriptions.")
    if channel.kick_chatroom_id is None:
        raise ValueError("Kick chatroom id is required for listener subscriptions.")

    return ListenerChannelDTO(
        id=channel.id,
        kick_channel_id=channel.kick_channel_id,
        kick_chatroom_id=channel.kick_chatroom_id,
        slug=channel.slug,
        display_name=channel.display_name,
    )
