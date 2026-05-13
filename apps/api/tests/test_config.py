from kick_logs.core.config import Settings


def test_settings_import_with_defaults() -> None:
    settings = Settings()

    assert settings.app_name == "Kick Logs"
    assert settings.default_super_admin_email == "admin@kicklogs.local"
    assert settings.listener_heartbeat_interval_seconds == 15.0
    assert settings.listener_heartbeat_stale_after_seconds == 45
    assert settings.message_export_max_rows == 1000
