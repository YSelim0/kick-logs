from collections.abc import Callable

from kick_logs.application.ports.password_hasher import PasswordHasher
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.core.config import Settings, get_settings
from kick_logs.domain.entities.user import User


async def seed_super_admin(
    unit_of_work_factory: Callable[[], UnitOfWork],
    password_hasher: PasswordHasher,
    settings: Settings | None = None,
) -> User:
    resolved_settings = settings or get_settings()

    async with unit_of_work_factory() as unit_of_work:
        existing_user = await unit_of_work.users.get_by_email(
            resolved_settings.default_super_admin_email
        )

        if existing_user is None:
            user = await unit_of_work.users.add(
                User.create_super_admin(
                    email=resolved_settings.default_super_admin_email,
                    password_hash=password_hasher.hash(
                        resolved_settings.default_super_admin_password
                    ),
                )
            )
            await unit_of_work.commit()
            return user

        changed = False
        if not existing_user.is_active:
            existing_user.activate()
            changed = True
        if not existing_user.role.can_manage_admins:
            existing_user.promote_to_super_admin()
            changed = True

        if changed:
            existing_user = await unit_of_work.users.update(existing_user)
            await unit_of_work.commit()

        return existing_user
