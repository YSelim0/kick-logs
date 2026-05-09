from kick_logs.core.config import Settings


def test_settings_import_with_defaults() -> None:
    settings = Settings()

    assert settings.app_name == "Kick Logs"
    assert settings.default_super_admin_email == "admin@kicklogs.local"
