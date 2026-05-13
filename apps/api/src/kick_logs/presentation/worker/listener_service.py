import asyncio
import logging
import time
from collections.abc import Awaitable, Callable

from kick_logs.application.ports.kick_channel_resolver import KickChannelResolver
from kick_logs.application.ports.pusher_client import PusherClient
from kick_logs.application.ports.sender_profile_resolver import SenderProfileResolver
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.listener import (
    LoadEnabledChannelsUseCase,
    ProcessRawKickEventsUseCase,
    RecordWorkerHeartbeatUseCase,
    StoreRawKickEventUseCase,
)
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
        raw_event_worker_count: int = 4,
        raw_event_batch_size: int = 100,
        raw_event_processing_timeout_seconds: int = 300,
        raw_event_max_attempts: int = 5,
        raw_event_worker_idle_delay_seconds: float = 0.25,
        channel_resync_interval_seconds: float = 60.0,
        heartbeat_interval_seconds: float = 15.0,
        heartbeat_service_name: str = "listener",
        sleep: SleepCallable = asyncio.sleep,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._channel_resolver = channel_resolver
        self._pusher_client = pusher_client
        self._event_parser = event_parser
        self._sender_profile_resolver = sender_profile_resolver
        self._reconnect_policy = reconnect_policy
        self._raw_event_worker_count = max(0, raw_event_worker_count)
        self._raw_event_batch_size = max(1, raw_event_batch_size)
        self._raw_event_processing_timeout_seconds = max(1, raw_event_processing_timeout_seconds)
        self._raw_event_max_attempts = max(1, raw_event_max_attempts)
        self._raw_event_worker_idle_delay_seconds = max(0.01, raw_event_worker_idle_delay_seconds)
        self._channel_resync_interval_seconds = max(0.01, channel_resync_interval_seconds)
        self._heartbeat_interval_seconds = max(0.01, heartbeat_interval_seconds)
        self._heartbeat_service_name = heartbeat_service_name
        self._sleep = sleep

    async def run_forever(self) -> None:
        worker_tasks = self._start_raw_event_workers()
        worker_tasks.append(
            asyncio.create_task(
                self._record_heartbeat_forever(),
                name=f"{self._heartbeat_service_name}-heartbeat",
            )
        )
        attempt = 1
        try:
            while True:
                try:
                    stored_count = await self.run_once()
                    attempt = 1
                    delay = self._reconnect_policy.delay_for_attempt(attempt)
                    if stored_count == 0:
                        logger.info("Kick listener idle; checking channels again in %.2fs.", delay)
                    else:
                        logger.warning(
                            "Kick listener stream ended after storing %d raw events; "
                            "reconnecting in %.2fs.",
                            stored_count,
                            delay,
                        )
                except asyncio.CancelledError:
                    raise
                except Exception:
                    delay = self._reconnect_policy.delay_for_attempt(attempt)
                    logger.exception("Kick listener failed; reconnecting in %.2fs.", delay)
                    attempt += 1

                await self._sleep(delay)
        finally:
            for task in worker_tasks:
                task.cancel()
            await asyncio.gather(*worker_tasks, return_exceptions=True)

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

        logger.info(
            "Subscribing to %d enabled Kick channels; next resync in %.2fs.",
            len(channel_result.channels),
            self._channel_resync_interval_seconds,
        )

        stored_count = 0
        raw_events = self._pusher_client.listen(channel_result.channels)
        resync_at = time.monotonic() + self._channel_resync_interval_seconds
        try:
            while True:
                timeout = resync_at - time.monotonic()
                if timeout <= 0:
                    logger.info("Kick listener channel resync interval elapsed.")
                    break

                try:
                    raw_event = await asyncio.wait_for(anext(raw_events), timeout=timeout)
                except TimeoutError:
                    logger.info("Kick listener channel resync interval elapsed.")
                    break
                except StopAsyncIteration:
                    break

                event = self._event_parser.parse(raw_event)
                if event is None:
                    logger.debug("Ignoring non-chat or malformed Kick event.")
                    continue

                raw_event_entity = await StoreRawKickEventUseCase(
                    self._unit_of_work_factory
                ).execute(
                    event_name=event.event,
                    payload=event.payload,
                    pusher_channel=event.channel,
                )
                stored_count += 1
                logger.info(
                    "Stored raw Kick chat event id=%s kick_message_id=%s chatroom_id=%s",
                    raw_event_entity.id,
                    raw_event_entity.kick_message_id,
                    raw_event_entity.chatroom_id,
                )
        finally:
            close = getattr(raw_events, "aclose", None)
            if close is not None:
                await close()

        return stored_count

    def _start_raw_event_workers(self) -> list[asyncio.Task[None]]:
        return [
            asyncio.create_task(
                self._process_raw_events_forever(worker_id),
                name=f"raw-kick-event-worker-{worker_id}",
            )
            for worker_id in range(1, self._raw_event_worker_count + 1)
        ]

    async def _process_raw_events_forever(self, worker_id: int) -> None:
        processor = ProcessRawKickEventsUseCase(self._unit_of_work_factory)

        while True:
            try:
                result = await processor.execute_once(
                    limit=self._raw_event_batch_size,
                    processing_timeout_seconds=self._raw_event_processing_timeout_seconds,
                    max_attempts=self._raw_event_max_attempts,
                )
            except asyncio.CancelledError:
                raise
            except Exception:
                logger.exception("Raw Kick event worker %d failed.", worker_id)
                await self._sleep(self._raw_event_worker_idle_delay_seconds)
                continue

            if result.claimed == 0:
                await self._sleep(self._raw_event_worker_idle_delay_seconds)
                continue

            logger.info(
                "Raw Kick event worker %d processed batch claimed=%d processed=%d failed=%d "
                "pending=%d.",
                worker_id,
                result.claimed,
                result.processed,
                result.failed,
                result.pending_count,
            )

    async def _record_heartbeat_forever(self) -> None:
        recorder = RecordWorkerHeartbeatUseCase(self._unit_of_work_factory)

        while True:
            try:
                await recorder.execute(
                    service_name=self._heartbeat_service_name,
                    metadata={
                        "raw_event_worker_count": self._raw_event_worker_count,
                        "raw_event_batch_size": self._raw_event_batch_size,
                        "channel_resync_interval_seconds": self._channel_resync_interval_seconds,
                    },
                )
            except asyncio.CancelledError:
                raise
            except Exception:
                logger.exception("Failed to record Kick listener heartbeat.")

            await self._sleep(self._heartbeat_interval_seconds)
