import pytest

from kick_logs.application.exceptions import SenderProfileResolutionError
from kick_logs.infrastructure.kick import KickWebSenderProfileResolver


async def test_sender_profile_resolver_extracts_profile_image() -> None:
    async def fetch_json(slug: str) -> dict:
        return {
            "slug": slug,
            "user": {
                "username": "Yavuz",
                "profile_pic": "https://example.com/avatar.png",
            },
        }

    profile = await KickWebSenderProfileResolver(fetch_json=fetch_json).resolve(" Yavuz ")

    assert profile.slug == "yavuz"
    assert profile.username == "Yavuz"
    assert profile.profile_image_url == "https://example.com/avatar.png"
    assert profile.raw_payload["user"]["username"] == "Yavuz"


async def test_sender_profile_resolver_wraps_fetch_failures() -> None:
    async def fetch_json(_slug: str) -> dict:
        raise RuntimeError("network failure")

    with pytest.raises(SenderProfileResolutionError):
        await KickWebSenderProfileResolver(fetch_json=fetch_json).resolve("yavuz")
