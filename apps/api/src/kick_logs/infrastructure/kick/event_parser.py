import json
from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True, slots=True)
class KickChatMessageEvent:
    event: str
    channel: str | None
    payload: dict[str, Any]


class KickEventParser:
    chat_message_event = r"App\Events\ChatMessageEvent"

    def parse(self, raw_event: str | dict[str, Any]) -> KickChatMessageEvent | None:
        envelope = self._read_json_object(raw_event)
        if envelope is None:
            return None

        if envelope.get("event") != self.chat_message_event:
            return None

        payload = self._read_json_object(envelope.get("data"))
        if payload is None or not self._has_required_message_fields(payload):
            return None

        return KickChatMessageEvent(
            event=str(envelope["event"]),
            channel=self._clean_text(envelope.get("channel")),
            payload=payload,
        )

    def _read_json_object(self, value: str | dict[str, Any] | Any) -> dict[str, Any] | None:
        if isinstance(value, dict):
            return value
        if not isinstance(value, str) or not value.strip():
            return None
        try:
            parsed = json.loads(value)
        except json.JSONDecodeError:
            return None
        return parsed if isinstance(parsed, dict) else None

    def _has_required_message_fields(self, payload: dict[str, Any]) -> bool:
        sender = payload.get("sender")
        return (
            self._clean_text(payload.get("id")) is not None
            and payload.get("chatroom_id") is not None
            and "content" in payload
            and isinstance(sender, dict)
            and self._clean_text(sender.get("id")) is not None
            and self._clean_text(sender.get("username")) is not None
        )

    def _clean_text(self, value: Any) -> str | None:
        if value is None:
            return None
        cleaned = str(value).strip()
        return cleaned or None
