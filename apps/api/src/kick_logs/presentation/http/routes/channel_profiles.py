from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Path

from kick_logs.application.exceptions import ChannelNotFoundError
from kick_logs.application.use_cases.profiles import GetChannelProfileUseCase
from kick_logs.presentation.http.dependencies import UnitOfWorkFactory, get_unit_of_work_factory
from kick_logs.presentation.http.schemas.channel_profiles import ChannelProfileResponse

router = APIRouter(prefix="/channels", tags=["channel-profiles"])

UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
ChannelSlugPath = Annotated[str, Path(min_length=1, max_length=160)]


@router.get("/{slug}/analytics", response_model=ChannelProfileResponse)
async def get_channel_profile(
    slug: ChannelSlugPath,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> ChannelProfileResponse:
    try:
        profile = await GetChannelProfileUseCase(unit_of_work_factory).execute(slug)
    except ChannelNotFoundError as exc:
        raise HTTPException(status_code=404, detail="Channel profile not found.") from exc

    return ChannelProfileResponse.from_dto(profile)
