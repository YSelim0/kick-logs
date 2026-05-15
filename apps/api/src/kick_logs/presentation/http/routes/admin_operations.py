from typing import Annotated

from fastapi import APIRouter, Depends

from kick_logs.application.use_cases.operations import GetOperationsSummaryUseCase
from kick_logs.core.config import Settings
from kick_logs.domain.entities.user import User
from kick_logs.presentation.http.dependencies import (
    UnitOfWorkFactory,
    get_settings_dependency,
    get_unit_of_work_factory,
    require_admin,
)
from kick_logs.presentation.http.schemas.operations import OperationsSummaryResponse

router = APIRouter(prefix="/admin/operations", tags=["admin-operations"])

AdminUserDep = Annotated[User, Depends(require_admin)]
SettingsDep = Annotated[Settings, Depends(get_settings_dependency)]
UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]


@router.get("/summary", response_model=OperationsSummaryResponse)
async def get_operations_summary(
    _current_user: AdminUserDep,
    settings: SettingsDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> OperationsSummaryResponse:
    summary = await GetOperationsSummaryUseCase(
        unit_of_work_factory,
        heartbeat_stale_after_seconds=settings.listener_heartbeat_stale_after_seconds,
    ).execute()
    return OperationsSummaryResponse.from_dto(summary)
