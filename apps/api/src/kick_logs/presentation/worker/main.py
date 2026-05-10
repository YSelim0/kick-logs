import asyncio

from kick_logs.core.config import get_settings
from kick_logs.core.logging import configure_logging
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork, create_session_factory
from kick_logs.infrastructure.kick import (
    KickEventParser,
    KickPusherClient,
    KickWebChannelResolver,
    KickWebSenderProfileResolver,
    ReconnectPolicy,
)
from kick_logs.presentation.worker.listener_service import ListenerService


async def run() -> None:
    settings = get_settings()
    configure_logging(settings.log_level)
    session_factory = create_session_factory()

    service = ListenerService(
        unit_of_work_factory=lambda: SqlAlchemyUnitOfWork(session_factory),
        channel_resolver=KickWebChannelResolver(),
        pusher_client=KickPusherClient(settings.kick_pusher_url),
        event_parser=KickEventParser(),
        sender_profile_resolver=KickWebSenderProfileResolver(),
        reconnect_policy=ReconnectPolicy(
            initial_delay_seconds=settings.listener_reconnect_initial_delay_seconds,
            max_delay_seconds=settings.listener_reconnect_max_delay_seconds,
            multiplier=settings.listener_reconnect_multiplier,
        ),
    )
    await service.run_forever()


def main() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    main()
