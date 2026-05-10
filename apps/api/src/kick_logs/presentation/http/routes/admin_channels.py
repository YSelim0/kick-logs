from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status

from kick_logs.application.exceptions import ChannelNotFoundError, ChannelResolutionError
from kick_logs.application.ports.kick_channel_resolver import KickChannelResolver
from kick_logs.application.use_cases.channels import (
    AddChannelUseCase,
    ListChannelsUseCase,
    RemoveChannelUseCase,
)
from kick_logs.domain.entities.user import User
from kick_logs.presentation.http.dependencies import (
    UnitOfWorkFactory,
    get_channel_resolver,
    get_unit_of_work_factory,
    require_admin,
)
from kick_logs.presentation.http.schemas.channels import AddChannelRequest, ChannelResponse

router = APIRouter(prefix="/admin/channels", tags=["admin-channels"])

AdminUserDep = Annotated[User, Depends(require_admin)]
UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
ChannelResolverDep = Annotated[KickChannelResolver, Depends(get_channel_resolver)]


@router.get("", response_model=list[ChannelResponse])
async def list_channels(
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> list[ChannelResponse]:
    channels = await ListChannelsUseCase(unit_of_work_factory).execute()
    return [ChannelResponse.from_dto(channel) for channel in channels]


@router.post("", response_model=ChannelResponse, status_code=status.HTTP_201_CREATED)
async def add_channel(
    payload: AddChannelRequest,
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
    channel_resolver: ChannelResolverDep,
) -> ChannelResponse:
    try:
        channel = await AddChannelUseCase(unit_of_work_factory, channel_resolver).execute(
            payload.slug
        )
    except ChannelResolutionError as exc:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="Kick channel could not be resolved.",
        ) from exc

    return ChannelResponse.from_dto(channel)


@router.delete("/{channel_id}", response_model=ChannelResponse)
async def remove_channel(
    channel_id: int,
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> ChannelResponse:
    try:
        channel = await RemoveChannelUseCase(unit_of_work_factory).execute(channel_id)
    except ChannelNotFoundError as exc:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Channel not found.",
        ) from exc

    return ChannelResponse.from_dto(channel)
