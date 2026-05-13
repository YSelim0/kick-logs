from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Path

from kick_logs.application.exceptions import SenderNotFoundError
from kick_logs.application.use_cases.profiles import GetUserProfileUseCase
from kick_logs.presentation.http.dependencies import UnitOfWorkFactory, get_unit_of_work_factory
from kick_logs.presentation.http.schemas.user_profiles import UserProfileResponse

router = APIRouter(prefix="/users", tags=["user-profiles"])

UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
SenderSlugPath = Annotated[str, Path(min_length=1, max_length=160)]


@router.get("/{slug}/analytics", response_model=UserProfileResponse)
async def get_user_profile(
    slug: SenderSlugPath,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> UserProfileResponse:
    try:
        profile = await GetUserProfileUseCase(unit_of_work_factory).execute(slug)
    except SenderNotFoundError as exc:
        raise HTTPException(status_code=404, detail="Sender profile not found.") from exc

    return UserProfileResponse.from_dto(profile)
