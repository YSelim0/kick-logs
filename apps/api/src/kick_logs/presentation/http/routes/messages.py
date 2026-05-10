from datetime import UTC, datetime
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Query

from kick_logs.application.use_cases.messages import SearchMessagesUseCase
from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.pagination import CursorPagination, MessageCursor
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters
from kick_logs.presentation.http.dependencies import UnitOfWorkFactory, get_unit_of_work_factory
from kick_logs.presentation.http.schemas.messages import MessageSearchResponse

router = APIRouter(tags=["messages"])

UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
TextFilterQuery = Annotated[str | None, Query(max_length=160)]
ContentQuery = Annotated[str | None, Query(max_length=500)]
CursorQuery = Annotated[str | None, Query(max_length=200)]
LimitQuery = Annotated[int, Query(ge=1, le=100)]


@router.get("/messages", response_model=MessageSearchResponse)
async def search_messages(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    sender: TextFilterQuery = None,
    channel: TextFilterQuery = None,
    q: ContentQuery = None,
    start: datetime | None = None,
    end: datetime | None = None,
    cursor: CursorQuery = None,
    limit: LimitQuery = 50,
) -> MessageSearchResponse:
    try:
        page = await SearchMessagesUseCase(unit_of_work_factory).execute(
            MessageSearchFilters(sender=sender, channel=channel, q=q, start=start, end=end),
            CursorPagination(limit=limit, cursor=_parse_cursor(cursor)),
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return MessageSearchResponse.from_dto(page)


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
