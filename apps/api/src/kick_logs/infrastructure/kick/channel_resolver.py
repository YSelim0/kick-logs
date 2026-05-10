from collections.abc import Awaitable, Callable
from typing import Any

from curl_cffi.requests import AsyncSession

from kick_logs.application.dto.channels import ResolvedKickChannelDTO
from kick_logs.application.exceptions import ChannelResolutionError

FetchJson = Callable[[str], Awaitable[dict[str, Any]]]


class KickWebChannelResolver:
    def __init__(
        self,
        fetch_json: FetchJson | None = None,
        base_url: str = "https://kick.com/api/v2/channels",
    ) -> None:
        self._fetch_json = fetch_json or self._fetch_channel_payload
        self._base_url = base_url.rstrip("/")

    async def resolve(self, slug: str) -> ResolvedKickChannelDTO:
        normalized_slug = slug.strip().lower()
        if not normalized_slug:
            raise ChannelResolutionError("Channel slug is required.")

        try:
            payload = await self._fetch_json(normalized_slug)
            return self._parse_payload(payload, normalized_slug)
        except ChannelResolutionError:
            raise
        except Exception as exc:
            raise ChannelResolutionError("Kick channel metadata could not be resolved.") from exc

    async def _fetch_channel_payload(self, slug: str) -> dict[str, Any]:
        async with AsyncSession(impersonate="chrome124") as session:
            response = await session.get(f"{self._base_url}/{slug}", timeout=15)
            if response.status_code >= 400:
                raise ChannelResolutionError("Kick channel endpoint returned an error.")
            payload = response.json()

        if not isinstance(payload, dict):
            raise ChannelResolutionError("Kick channel endpoint returned an invalid payload.")
        return payload

    def _parse_payload(
        self,
        payload: dict[str, Any],
        fallback_slug: str,
    ) -> ResolvedKickChannelDTO:
        kick_channel_id = self._read_int(payload, "id")
        chatroom = payload.get("chatroom")
        if not isinstance(chatroom, dict):
            raise ChannelResolutionError("Kick channel payload has no chatroom metadata.")

        kick_chatroom_id = self._read_int(chatroom, "id")
        slug = str(payload.get("slug") or fallback_slug).strip().lower()
        display_name = self._read_display_name(payload, slug)

        return ResolvedKickChannelDTO(
            kick_channel_id=kick_channel_id,
            kick_chatroom_id=kick_chatroom_id,
            slug=slug,
            display_name=display_name,
            profile_image_url=self._read_profile_image_url(payload),
            banner_image_url=self._read_banner_image_url(payload),
            raw_payload=payload,
        )

    def _read_int(self, payload: dict[str, Any], key: str) -> int:
        value = payload.get(key)
        if value is None:
            raise ChannelResolutionError(f"Kick channel payload missing `{key}`.")
        try:
            return int(value)
        except (TypeError, ValueError) as exc:
            raise ChannelResolutionError(f"Kick channel payload has invalid `{key}`.") from exc

    def _read_display_name(self, payload: dict[str, Any], slug: str) -> str:
        user = payload.get("user")
        if isinstance(user, dict):
            username = user.get("username")
            if isinstance(username, str) and username.strip():
                return username.strip()

        for key in ("display_name", "name", "slug"):
            value = payload.get(key)
            if isinstance(value, str) and value.strip():
                return value.strip()

        return slug

    def _read_profile_image_url(self, payload: dict[str, Any]) -> str | None:
        user = payload.get("user")
        candidates: list[Any] = [
            payload.get("profilepic"),
            payload.get("profile_pic"),
            payload.get("profile_image_url"),
        ]
        if isinstance(user, dict):
            candidates.extend(
                [
                    user.get("profilepic"),
                    user.get("profile_pic"),
                    user.get("profile_image_url"),
                ]
            )
        return self._first_non_empty_string(candidates)

    def _read_banner_image_url(self, payload: dict[str, Any]) -> str | None:
        return self._first_non_empty_string(
            [
                payload.get("banner_image"),
                payload.get("banner_image_url"),
                payload.get("banner"),
            ]
        )

    def _first_non_empty_string(self, values: list[Any]) -> str | None:
        for value in values:
            if isinstance(value, str) and value.strip():
                return value.strip()
        return None
