from collections.abc import Callable
from typing import Any

from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.entities.raw_kick_event import RawKickEvent

UnitOfWorkFactory = Callable[[], UnitOfWork]


class StoreRawKickEventUseCase:
    def __init__(self, unit_of_work_factory: UnitOfWorkFactory) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(
        self,
        *,
        event_name: str,
        payload: dict[str, Any],
        pusher_channel: str | None,
    ) -> RawKickEvent:
        kick_message_id = self._clean_text(payload.get("id"))
        chatroom_id = self._read_int(payload.get("chatroom_id"))

        async with self._unit_of_work_factory() as unit_of_work:
            if kick_message_id is not None:
                existing_event = await unit_of_work.raw_events.get_by_kick_message_id(
                    kick_message_id
                )
                if existing_event is not None:
                    return existing_event

            channel = (
                await unit_of_work.channels.get_by_chatroom_id(chatroom_id)
                if chatroom_id is not None
                else None
            )
            raw_event = await unit_of_work.raw_events.add(
                RawKickEvent.pending(
                    event_name=event_name,
                    payload=payload,
                    kick_message_id=kick_message_id,
                    chatroom_id=chatroom_id,
                    kick_channel_id=channel.kick_channel_id if channel else None,
                    channel_id=channel.id if channel else None,
                    metadata={"pusher_channel": pusher_channel} if pusher_channel else {},
                )
            )
            await unit_of_work.commit()

        return raw_event

    def _read_int(self, value: Any) -> int | None:
        if value is None:
            return None
        try:
            parsed = int(value)
        except (TypeError, ValueError):
            return None
        return parsed if parsed > 0 else None

    def _clean_text(self, value: Any) -> str | None:
        if value is None:
            return None
        cleaned = str(value).strip()
        return cleaned or None
