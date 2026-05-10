from datetime import UTC, datetime, timedelta

import jwt
from jwt import InvalidTokenError

from kick_logs.core.config import Settings, get_settings


class JwtTokenService:
    def __init__(self, settings: Settings | None = None) -> None:
        self._settings = settings or get_settings()

    def create_access_token(self, user_id: int) -> str:
        now = datetime.now(UTC)
        payload = {
            "sub": str(user_id),
            "iat": now,
            "exp": now + timedelta(minutes=self._settings.jwt_expires_minutes),
        }
        return jwt.encode(
            payload,
            self._settings.jwt_secret_key,
            algorithm=self._settings.jwt_algorithm,
        )

    def get_user_id(self, token: str) -> int | None:
        try:
            payload = jwt.decode(
                token,
                self._settings.jwt_secret_key,
                algorithms=[self._settings.jwt_algorithm],
            )
            subject = payload.get("sub")
            if subject is None:
                return None
            return int(subject)
        except (InvalidTokenError, TypeError, ValueError):
            return None
