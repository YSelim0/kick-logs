import asyncio
import copy
import logging
from collections.abc import Awaitable, Callable
from typing import Any

from kick_logs.application.exceptions import ApplicationError, SenderProfileResolutionError
from kick_logs.application.ports.kick_channel_resolver import KickChannelResolver
from kick_logs.application.ports.pusher_client import PusherClient
from kick_logs.application.ports.sender_profile_resolver import SenderProfileResolver
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.listener import LoadEnabledChannelsUseCase
from kick_logs.application.use_cases.messages import IngestMessageUseCase
from kick_logs.infrastructure.kick import KickEventParser, ReconnectPolicy

SleepCallable = Callable[[float], Awaitable[None]]
UnitOfWorkFactory = Callable[[], UnitOfWork]

logger = logging.getLogger(__name__)


class ListenerService:
    def __init__(
        self,
        unit_of_work_factory: UnitOfWorkFactory,
        channel_resolver: KickChannelResolver,
        pusher_client: PusherClient,
        event_parser: KickEventParser,
        sender_profile_resolver: SenderProfileResolver,
        reconnect_policy: ReconnectPolicy,
        sleep: SleepCallable = asyncio.sleep,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._channel_resolver = channel_resolver
        self._pusher_client = pusher_client
        self._event_parser = event_parser
        self._sender_profile_resolver = sender_profile_resolver
        self._reconnect_policy = reconnect_policy
        self._sleep = sleep

    async def run_forever(self) -> None:
        attempt = 1
        while True:
            try:
                await self.run_once()
                attempt = 1
                delay = self._reconnect_policy.delay_for_attempt(attempt)
                logger.warning("Kick listener stream ended; reconnecting in %.2fs.", delay)
            except asyncio.CancelledError:
                raise
            except Exception:
                delay = self._reconnect_policy.delay_for_attempt(attempt)
                logger.exception("Kick listener failed; reconnecting in %.2fs.", delay)
                attempt += 1

            await self._sleep(delay)

    async def run_once(self) -> int:
        channel_result = await LoadEnabledChannelsUseCase(
            self._unit_of_work_factory,
            self._channel_resolver,
        ).execute()

        for skipped_channel in channel_result.skipped:
            logger.warning(
                "Skipping Kick channel slug=%s id=%s reason=%s",
                skipped_channel.slug,
                skipped_channel.id,
                skipped_channel.reason,
            )

        if not channel_result.channels:
            logger.info("No enabled Kick channels are ready for listener subscription.")
            return 0

        logger.info("Subscribing to %d enabled Kick channels.", len(channel_result.channels))

        ingested_count = 0
        async for raw_event in self._pusher_client.listen(channel_result.channels):
            event = self._event_parser.parse(raw_event)
            if event is None:
                logger.debug("Ignoring non-chat or malformed Kick event.")
                continue

            payload = await self._enrich_sender_profile(event.payload)
            try:
                message = await IngestMessageUseCase(self._unit_of_work_factory).execute(payload)
            except ApplicationError:
                logger.exception("Kick chat message could not be ingested.")
                continue

            ingested_count += 1
            logger.info(
                "Ingested Kick chat message id=%s chatroom_id=%s",
                message.kick_message_id,
                message.chatroom_id,
            )

        return ingested_count

    async def _enrich_sender_profile(self, payload: dict[str, Any]) -> dict[str, Any]:
        enriched_payload = copy.deepcopy(payload)
        sender = enriched_payload.get("sender")
        if not isinstance(sender, dict):
            return enriched_payload

        if self._sender_has_profile_image(sender):
            return enriched_payload

        slug = self._clean_text(sender.get("slug")) or self._clean_text(sender.get("username"))
        if slug is None:
            return enriched_payload

        try:
            profile = await self._sender_profile_resolver.resolve(slug)
        except SenderProfileResolutionError:
            logger.warning("Kick sender profile could not be resolved for slug=%s.", slug)
            return enriched_payload

        if profile.profile_image_url:
            sender["profile_pic"] = profile.profile_image_url
            sender["resolved_profile_payload"] = profile.raw_payload

        return enriched_payload

    def _sender_has_profile_image(self, sender: dict[str, Any]) -> bool:
        return any(
            self._clean_text(sender.get(key)) is not None
            for key in ("profile_image_url", "profile_pic", "profilepic")
        )

    def _clean_text(self, value: Any) -> str | None:
        if value is None:
            return None
        cleaned = str(value).strip()
        return cleaned or None
