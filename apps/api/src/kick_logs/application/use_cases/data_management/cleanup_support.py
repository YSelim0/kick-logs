from datetime import UTC, datetime, timedelta

from kick_logs.application.dto.data_management import (
    CleanupTarget,
    DataCleanupCountsDTO,
    DataCleanupCriteriaDTO,
    DataCleanupPreviewDTO,
    DataCleanupRequestDTO,
    RetentionSettingsDTO,
)
from kick_logs.application.exceptions import InvalidDataCleanupRequestError

ALLOWED_RETENTION_DAYS = {30, 90}


def validate_retention_days(value: int | None) -> None:
    if value is not None and value not in ALLOWED_RETENTION_DAYS:
        allowed = ", ".join(str(days) for days in sorted(ALLOWED_RETENTION_DAYS))
        raise InvalidDataCleanupRequestError(f"Retention days must be one of: {allowed}.")


def build_cleanup_criteria(
    request: DataCleanupRequestDTO,
    settings: RetentionSettingsDTO,
) -> DataCleanupCriteriaDTO:
    if request.target == "old_messages":
        return _build_old_criteria(
            target=request.target,
            retention_days=settings.message_retention_days,
        )

    if request.target == "old_raw_events":
        return _build_old_criteria(
            target=request.target,
            retention_days=settings.raw_event_retention_days,
        )

    if request.target == "channel":
        channel_slug = _required_text(request.channel_slug, "Channel slug is required.")
        return DataCleanupCriteriaDTO(target=request.target, channel_slug=channel_slug)

    if request.target == "sender":
        sender = _required_text(request.sender, "Sender is required.")
        return DataCleanupCriteriaDTO(target=request.target, sender=sender)

    raise InvalidDataCleanupRequestError("Unsupported cleanup target.")


def build_preview(
    *,
    criteria: DataCleanupCriteriaDTO,
    affected_messages: int,
    affected_raw_events: int,
) -> DataCleanupPreviewDTO:
    reason = None
    can_execute = True
    if criteria.target in {"old_messages", "old_raw_events"} and criteria.cutoff_at is None:
        can_execute = False
        reason = "Retention is set to keep forever."

    return DataCleanupPreviewDTO(
        target=criteria.target,
        affected=DataCleanupCountsDTO(messages=affected_messages, raw_events=affected_raw_events),
        confirmation_text=build_confirmation_text(criteria),
        can_execute=can_execute,
        cutoff_at=criteria.cutoff_at,
        channel_slug=criteria.channel_slug,
        sender=criteria.sender,
        retention_days=criteria.retention_days,
        reason=reason,
    )


def build_confirmation_text(criteria: DataCleanupCriteriaDTO) -> str:
    if criteria.target == "old_messages":
        return "DELETE OLD MESSAGES"
    if criteria.target == "old_raw_events":
        return "DELETE OLD RAW EVENTS"
    if criteria.target == "channel":
        return f"DELETE CHANNEL {criteria.channel_slug}"
    return f"DELETE SENDER {criteria.sender}"


def ensure_preview_can_execute(preview: DataCleanupPreviewDTO) -> None:
    if not preview.can_execute:
        raise InvalidDataCleanupRequestError(preview.reason or "Cleanup cannot be executed.")


def _build_old_criteria(
    *,
    target: CleanupTarget,
    retention_days: int | None,
) -> DataCleanupCriteriaDTO:
    validate_retention_days(retention_days)
    if retention_days is None:
        return DataCleanupCriteriaDTO(target=target)

    return DataCleanupCriteriaDTO(
        target=target,
        retention_days=retention_days,
        cutoff_at=datetime.now(UTC) - timedelta(days=retention_days),
    )


def _required_text(value: str | None, message: str) -> str:
    normalized = value.strip() if value else ""
    if not normalized:
        raise InvalidDataCleanupRequestError(message)
    return normalized
