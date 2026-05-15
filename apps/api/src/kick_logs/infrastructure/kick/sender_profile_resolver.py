from collections.abc import Awaitable, Callable
from typing import Any

from curl_cffi.requests import AsyncSession

from kick_logs.application.dto.senders import ResolvedSenderProfileDTO
from kick_logs.application.exceptions import SenderProfileResolutionError
from kick_logs.domain.value_objects.sender_slug import normalize_kick_profile_slug

FetchJson = Callable[[str], Awaitable[dict[str, Any]]]


class KickWebSenderProfileResolver:
    def __init__(
        self,
        fetch_json: FetchJson | None = None,
        base_url: str = "https://kick.com/api/v2/channels",
    ) -> None:
        self._fetch_json = fetch_json or self._fetch_profile_payload
        self._base_url = base_url.rstrip("/")

    async def resolve(self, slug: str) -> ResolvedSenderProfileDTO:
        normalized_slug = normalize_kick_profile_slug(slug)
        if not normalized_slug:
            raise SenderProfileResolutionError("Sender slug is required.")

        try:
            payload = await self._fetch_json(normalized_slug)
            return self._parse_payload(payload, normalized_slug)
        except SenderProfileResolutionError:
            raise
        except Exception as exc:
            raise SenderProfileResolutionError(
                "Kick sender profile could not be resolved."
            ) from exc

    async def _fetch_profile_payload(self, slug: str) -> dict[str, Any]:
        async with AsyncSession(impersonate="chrome124") as session:
            response = await session.get(f"{self._base_url}/{slug}", timeout=15)
            if response.status_code >= 400:
                raise SenderProfileResolutionError("Kick profile endpoint returned an error.")
            payload = response.json()

        if not isinstance(payload, dict):
            raise SenderProfileResolutionError("Kick profile endpoint returned an invalid payload.")
        return payload

    def _parse_payload(
        self,
        payload: dict[str, Any],
        fallback_slug: str,
    ) -> ResolvedSenderProfileDTO:
        user = payload.get("user")
        user_payload = user if isinstance(user, dict) else {}

        return ResolvedSenderProfileDTO(
            slug=normalize_kick_profile_slug(str(payload.get("slug") or fallback_slug))
            or fallback_slug,
            username=self._first_non_empty_string([user_payload.get("username")]),
            profile_image_url=self._first_non_empty_string(
                [
                    user_payload.get("profile_pic"),
                    user_payload.get("profilepic"),
                    user_payload.get("profile_image_url"),
                    payload.get("profile_pic"),
                    payload.get("profilepic"),
                    payload.get("profile_image_url"),
                ]
            ),
            raw_payload=payload,
        )

    def _first_non_empty_string(self, values: list[Any]) -> str | None:
        for value in values:
            if isinstance(value, str) and value.strip():
                return value.strip()
        return None
