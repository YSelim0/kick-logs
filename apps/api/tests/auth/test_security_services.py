from kick_logs.core.config import Settings
from kick_logs.infrastructure.auth import JwtTokenService, PasslibPasswordHasher


def test_passlib_password_hasher_hashes_and_verifies_password() -> None:
    hasher = PasslibPasswordHasher()

    password_hash = hasher.hash("admin123")

    assert password_hash != "admin123"
    assert hasher.verify("admin123", password_hash) is True
    assert hasher.verify("wrong", password_hash) is False


def test_jwt_token_service_round_trips_user_id() -> None:
    settings = Settings(jwt_secret_key="test-secret-with-at-least-32-bytes", jwt_expires_minutes=5)
    token_service = JwtTokenService(settings)

    token = token_service.create_access_token(user_id=42)

    assert token_service.get_user_id(token) == 42


def test_jwt_token_service_rejects_invalid_token() -> None:
    token_service = JwtTokenService(Settings(jwt_secret_key="test-secret-with-at-least-32-bytes"))

    assert token_service.get_user_id("not-a-token") is None
