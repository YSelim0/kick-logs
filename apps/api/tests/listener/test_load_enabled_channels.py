from kick_logs.application.dto.channels import ResolvedKickChannelDTO
from kick_logs.application.exceptions import ChannelResolutionError
from kick_logs.application.use_cases.listener import LoadEnabledChannelsUseCase
from kick_logs.domain.entities import Channel


class FakeChannelRepository:
    def __init__(self, channels: list[Channel]) -> None:
        self.channels = channels
        self.updated_channels: list[Channel] = []

    async def list_enabled(self) -> list[Channel]:
        return [channel for channel in self.channels if channel.is_enabled]

    async def update(self, channel: Channel) -> Channel:
        self.updated_channels.append(channel)
        return channel


class FakeUnitOfWork:
    def __init__(self, channels: list[Channel]) -> None:
        self.channels = FakeChannelRepository(channels)
        self.committed = False

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, traceback) -> None:
        return None

    async def commit(self) -> None:
        self.committed = True

    async def rollback(self) -> None:
        return None


class FakeChannelResolver:
    async def resolve(self, slug: str) -> ResolvedKickChannelDTO:
        return ResolvedKickChannelDTO(
            kick_channel_id=100,
            kick_chatroom_id=200,
            slug=slug,
            display_name="Resolved",
            profile_image_url="https://example.com/profile.png",
            banner_image_url="https://example.com/banner.png",
            raw_payload={"id": 100, "chatroom": {"id": 200}},
        )


class FailingChannelResolver:
    async def resolve(self, _slug: str) -> ResolvedKickChannelDTO:
        raise ChannelResolutionError("failed")


async def test_load_enabled_channels_returns_ready_channels() -> None:
    unit_of_work = FakeUnitOfWork(
        [
            Channel(
                id=1,
                slug="hype",
                display_name="Hype",
                kick_channel_id=100,
                kick_chatroom_id=200,
            ),
            Channel(
                id=2,
                slug="disabled",
                display_name="Disabled",
                kick_channel_id=101,
                kick_chatroom_id=201,
                is_enabled=False,
            ),
        ]
    )

    result = await LoadEnabledChannelsUseCase(
        lambda: unit_of_work,
        FakeChannelResolver(),
    ).execute()

    assert [channel.slug for channel in result.channels] == ["hype"]
    assert result.channels[0].kick_chatroom_id == 200
    assert result.skipped == []
    assert unit_of_work.committed is True


async def test_load_enabled_channels_resolves_missing_kick_metadata() -> None:
    unit_of_work = FakeUnitOfWork([Channel(id=1, slug="hype", display_name="Hype")])

    result = await LoadEnabledChannelsUseCase(
        lambda: unit_of_work,
        FakeChannelResolver(),
    ).execute()

    assert len(result.channels) == 1
    assert result.channels[0].kick_channel_id == 100
    assert result.channels[0].kick_chatroom_id == 200
    assert unit_of_work.channels.updated_channels[0].display_name == "Resolved"


async def test_load_enabled_channels_skips_unresolvable_channels() -> None:
    unit_of_work = FakeUnitOfWork([Channel(id=1, slug="hype", display_name="Hype")])

    result = await LoadEnabledChannelsUseCase(
        lambda: unit_of_work,
        FailingChannelResolver(),
    ).execute()

    assert result.channels == []
    assert result.skipped[0].id == 1
    assert result.skipped[0].slug == "hype"
    assert result.skipped[0].reason == "Channel has missing Kick metadata."
