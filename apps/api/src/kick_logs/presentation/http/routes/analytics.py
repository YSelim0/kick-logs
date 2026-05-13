from datetime import datetime
from typing import Annotated, Literal

from fastapi import APIRouter, Depends, HTTPException, Query

from kick_logs.application.use_cases.analytics import (
    GetAnalyticsOverviewUseCase,
    GetMessageVolumeUseCase,
    GetTopChannelsUseCase,
    GetTopEmotesUseCase,
    GetTopSendersUseCase,
)
from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.analytics_filters import AnalyticsFilters
from kick_logs.presentation.http.dependencies import UnitOfWorkFactory, get_unit_of_work_factory
from kick_logs.presentation.http.schemas.analytics import (
    AnalyticsOverviewResponse,
    MessageVolumeResponse,
    TopChannelsResponse,
    TopEmotesResponse,
    TopSendersResponse,
)

router = APIRouter(prefix="/analytics", tags=["analytics"])

UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
TextFilterQuery = Annotated[str | None, Query(max_length=160)]
LimitQuery = Annotated[int, Query(ge=1, le=100)]
BucketQuery = Annotated[Literal["hour", "day"], Query()]


@router.get("/overview", response_model=AnalyticsOverviewResponse)
async def get_analytics_overview(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    start: datetime | None = None,
    end: datetime | None = None,
    channel: TextFilterQuery = None,
    sender: TextFilterQuery = None,
) -> AnalyticsOverviewResponse:
    try:
        overview = await GetAnalyticsOverviewUseCase(unit_of_work_factory).execute(
            _build_filters(start=start, end=end, channel=channel, sender=sender)
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return AnalyticsOverviewResponse.from_dto(overview)


@router.get("/message-volume", response_model=MessageVolumeResponse)
async def get_message_volume(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    start: datetime | None = None,
    end: datetime | None = None,
    channel: TextFilterQuery = None,
    sender: TextFilterQuery = None,
    bucket: BucketQuery = "day",
) -> MessageVolumeResponse:
    try:
        points = await GetMessageVolumeUseCase(unit_of_work_factory).execute(
            _build_filters(start=start, end=end, channel=channel, sender=sender),
            bucket,
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return MessageVolumeResponse.from_dto(points)


@router.get("/top-senders", response_model=TopSendersResponse)
async def get_top_senders(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    start: datetime | None = None,
    end: datetime | None = None,
    channel: TextFilterQuery = None,
    sender: TextFilterQuery = None,
    limit: LimitQuery = 10,
) -> TopSendersResponse:
    try:
        senders = await GetTopSendersUseCase(unit_of_work_factory).execute(
            _build_filters(start=start, end=end, channel=channel, sender=sender),
            limit,
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return TopSendersResponse.from_dto(senders)


@router.get("/top-channels", response_model=TopChannelsResponse)
async def get_top_channels(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    start: datetime | None = None,
    end: datetime | None = None,
    channel: TextFilterQuery = None,
    sender: TextFilterQuery = None,
    limit: LimitQuery = 10,
) -> TopChannelsResponse:
    try:
        channels = await GetTopChannelsUseCase(unit_of_work_factory).execute(
            _build_filters(start=start, end=end, channel=channel, sender=sender),
            limit,
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return TopChannelsResponse.from_dto(channels)


@router.get("/top-emotes", response_model=TopEmotesResponse)
async def get_top_emotes(
    unit_of_work_factory: UnitOfWorkFactoryDep,
    start: datetime | None = None,
    end: datetime | None = None,
    channel: TextFilterQuery = None,
    sender: TextFilterQuery = None,
    limit: LimitQuery = 10,
) -> TopEmotesResponse:
    try:
        emotes = await GetTopEmotesUseCase(unit_of_work_factory).execute(
            _build_filters(start=start, end=end, channel=channel, sender=sender),
            limit,
        )
    except DomainError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    return TopEmotesResponse.from_dto(emotes)


def _build_filters(
    *,
    start: datetime | None,
    end: datetime | None,
    channel: str | None,
    sender: str | None,
) -> AnalyticsFilters:
    return AnalyticsFilters(start=start, end=end, channel=channel, sender=sender)
