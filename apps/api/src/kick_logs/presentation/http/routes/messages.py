import csv
import json
from datetime import UTC, datetime
from io import StringIO
from typing import Annotated, Literal

from fastapi import APIRouter, Depends, HTTPException, Query, Response
from fastapi.responses import JSONResponse

from kick_logs.application.dto.messages import MessageExportDTO, MessageSearchItemDTO
from kick_logs.application.use_cases.messages import ExportMessagesUseCase, SearchMessagesUseCase
from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.pagination import CursorPagination, MessageCursor
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters
from kick_logs.presentation.http.dependencies import (
    SettingsDep,
    UnitOfWorkFactory,
    get_unit_of_work_factory,
)
from kick_logs.presentation.http.schemas.messages import (
    MessageExportResponse,
    MessageSearchResponse,
)

router = APIRouter(tags=["messages"])

UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
TextFilterQuery = Annotated[str | None, Query(max_length=160)]
ContentQuery = Annotated[str | None, Query(max_length=500)]
CursorQuery = Annotated[str | None, Query(max_length=200)]
LimitQuery = Annotated[int, Query(ge=1, le=100)]
ExportFormatQuery = Annotated[Literal["csv", "json"], Query(alias="format")]
ExportLimitQuery = Annotated[int | None, Query(ge=1)]

CSV_FIELDS = [
    "message_created_at",
    "kick_message_id",
    "channel_slug",
    "channel_display_name",
    "sender_username",
    "sender_slug",
    "message_type",
    "content",
    "emotes",
    "reply_to_sender",
    "reply_to_content",
    "thread_parent_id",
]


@router.get("/messages/export")
async def export_messages(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    settings: SettingsDep,
    sender: TextFilterQuery = None,
    channel: TextFilterQuery = None,
    q: ContentQuery = None,
    start: datetime | None = None,
    end: datetime | None = None,
    reply_only: bool = False,
    emote_only: bool = False,
    export_format: ExportFormatQuery = "json",
    limit: ExportLimitQuery = None,
) -> Response:
    try:
        export = await ExportMessagesUseCase(unit_of_work_factory).execute(
            _build_filters(
                sender=sender,
                channel=channel,
                q=q,
                start=start,
                end=end,
                reply_only=reply_only,
                emote_only=emote_only,
            ),
            max_rows=min(
                limit or settings.message_export_max_rows, settings.message_export_max_rows
            ),
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    if export_format == "csv":
        return Response(
            content=_message_export_to_csv(export),
            media_type="text/csv; charset=utf-8",
            headers={"Content-Disposition": 'attachment; filename="kick-logs-export.csv"'},
        )

    return JSONResponse(content=MessageExportResponse.from_dto(export).model_dump(mode="json"))


@router.get("/messages", response_model=MessageSearchResponse)
async def search_messages(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    sender: TextFilterQuery = None,
    channel: TextFilterQuery = None,
    q: ContentQuery = None,
    start: datetime | None = None,
    end: datetime | None = None,
    reply_only: bool = False,
    emote_only: bool = False,
    cursor: CursorQuery = None,
    limit: LimitQuery = 50,
) -> MessageSearchResponse:
    try:
        page = await SearchMessagesUseCase(unit_of_work_factory).execute(
            _build_filters(
                sender=sender,
                channel=channel,
                q=q,
                start=start,
                end=end,
                reply_only=reply_only,
                emote_only=emote_only,
            ),
            CursorPagination(limit=limit, cursor=_parse_cursor(cursor)),
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return MessageSearchResponse.from_dto(page)


def _build_filters(
    *,
    sender: str | None,
    channel: str | None,
    q: str | None,
    start: datetime | None,
    end: datetime | None,
    reply_only: bool,
    emote_only: bool,
) -> MessageSearchFilters:
    return MessageSearchFilters(
        sender=sender,
        channel=channel,
        q=q,
        start=start,
        end=end,
        reply_only=reply_only,
        emote_only=emote_only,
    )


def _message_export_to_csv(export: MessageExportDTO) -> str:
    stream = StringIO(newline="")
    writer = csv.DictWriter(stream, fieldnames=CSV_FIELDS)
    writer.writeheader()

    for item in export.items:
        writer.writerow({field: _safe_csv_value(value) for field, value in _csv_row(item).items()})

    return stream.getvalue()


def _csv_row(item: MessageSearchItemDTO) -> dict[str, str | None]:
    reply_metadata = item.message.reply_metadata
    original_sender = reply_metadata.get("original_sender", {})
    original_message = reply_metadata.get("original_message", {})

    return {
        "message_created_at": item.message.message_created_at.isoformat(),
        "kick_message_id": item.message.kick_message_id,
        "channel_slug": item.channel.slug,
        "channel_display_name": item.channel.display_name,
        "sender_username": item.sender.username,
        "sender_slug": item.sender.slug,
        "message_type": item.message.message_type,
        "content": item.message.content,
        "emotes": json.dumps(item.message.emotes, ensure_ascii=False),
        "reply_to_sender": original_sender.get("username")
        if isinstance(original_sender, dict)
        else None,
        "reply_to_content": original_message.get("content")
        if isinstance(original_message, dict)
        else None,
        "thread_parent_id": item.message.thread_parent_id,
    }


def _safe_csv_value(value: str | None) -> str:
    text = "" if value is None else str(value)
    return f"'{text}" if text.startswith(("=", "+", "-", "@")) else text


def _parse_cursor(value: str | None) -> MessageCursor | None:
    if value is None or not value.strip():
        return None

    try:
        timestamp_text, message_id_text = value.rsplit("|", 1)
        created_at = datetime.fromisoformat(timestamp_text.replace("Z", "+00:00"))
        message_id = int(message_id_text)
    except ValueError as exc:
        raise HTTPException(status_code=422, detail="Invalid cursor.") from exc

    if message_id < 1:
        raise HTTPException(status_code=422, detail="Invalid cursor.")

    if created_at.tzinfo is None:
        created_at = created_at.replace(tzinfo=UTC)

    return MessageCursor(message_created_at=created_at, message_id=message_id)
