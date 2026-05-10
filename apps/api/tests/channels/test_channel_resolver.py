import pytest

from kick_logs.application.exceptions import ChannelResolutionError
from kick_logs.infrastructure.kick import KickWebChannelResolver


async def test_kick_channel_resolver_parses_channel_payload() -> None:
    async def fetch_json(slug: str) -> dict:
        return {
            "id": "123",
            "slug": slug,
            "chatroom": {"id": "456"},
            "user": {
                "username": "Hype",
                "profile_pic": "https://example.com/profile.png",
            },
            "banner_image": "https://example.com/banner.png",
        }

    resolved = await KickWebChannelResolver(fetch_json=fetch_json).resolve(" HYPE ")

    assert resolved.kick_channel_id == 123
    assert resolved.kick_chatroom_id == 456
    assert resolved.slug == "hype"
    assert resolved.display_name == "Hype"
    assert resolved.profile_image_url == "https://example.com/profile.png"
    assert resolved.banner_image_url == "https://example.com/banner.png"
    assert resolved.raw_payload["chatroom"]["id"] == "456"


async def test_kick_channel_resolver_falls_back_to_slug_display_name() -> None:
    async def fetch_json(slug: str) -> dict:
        return {"id": 123, "slug": slug, "chatroom": {"id": 456}}

    resolved = await KickWebChannelResolver(fetch_json=fetch_json).resolve("hype")

    assert resolved.display_name == "hype"


async def test_kick_channel_resolver_rejects_missing_chatroom() -> None:
    async def fetch_json(_slug: str) -> dict:
        return {"id": 123, "slug": "hype"}

    with pytest.raises(ChannelResolutionError):
        await KickWebChannelResolver(fetch_json=fetch_json).resolve("hype")


async def test_kick_channel_resolver_wraps_fetch_failures() -> None:
    async def fetch_json(_slug: str) -> dict:
        raise RuntimeError("network failure")

    with pytest.raises(ChannelResolutionError):
        await KickWebChannelResolver(fetch_json=fetch_json).resolve("hype")
