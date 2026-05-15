from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status

from kick_logs.application.exceptions import (
    CleanupConfirmationError,
    InvalidDataCleanupRequestError,
)
from kick_logs.application.use_cases.data_management import (
    ConfirmDataCleanupUseCase,
    GetDataManagementSummaryUseCase,
    PreviewDataCleanupUseCase,
    UpdateRetentionSettingsUseCase,
)
from kick_logs.domain.entities.user import User
from kick_logs.presentation.http.dependencies import (
    UnitOfWorkFactory,
    get_unit_of_work_factory,
    require_admin,
)
from kick_logs.presentation.http.schemas.data_management import (
    DataCleanupConfirmRequest,
    DataCleanupPreviewResponse,
    DataCleanupRequest,
    DataCleanupResultResponse,
    DataManagementSummaryResponse,
    RetentionSettingsRequest,
    RetentionSettingsResponse,
)

router = APIRouter(prefix="/admin/data-management", tags=["admin-data-management"])

AdminUserDep = Annotated[User, Depends(require_admin)]
UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]


@router.get("/summary", response_model=DataManagementSummaryResponse)
async def get_data_management_summary(
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> DataManagementSummaryResponse:
    summary = await GetDataManagementSummaryUseCase(unit_of_work_factory).execute()
    return DataManagementSummaryResponse.from_dto(summary)


@router.put("/retention-settings", response_model=RetentionSettingsResponse)
async def update_retention_settings(
    request: RetentionSettingsRequest,
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> RetentionSettingsResponse:
    try:
        settings = await UpdateRetentionSettingsUseCase(unit_of_work_factory).execute(
            request.to_dto()
        )
    except InvalidDataCleanupRequestError as exc:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(exc)
        ) from exc
    return RetentionSettingsResponse.from_dto(settings)


@router.post("/cleanup/preview", response_model=DataCleanupPreviewResponse)
async def preview_data_cleanup(
    request: DataCleanupRequest,
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> DataCleanupPreviewResponse:
    try:
        preview = await PreviewDataCleanupUseCase(unit_of_work_factory).execute(request.to_dto())
    except InvalidDataCleanupRequestError as exc:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(exc)
        ) from exc
    return DataCleanupPreviewResponse.from_dto(preview)


@router.post("/cleanup/confirm", response_model=DataCleanupResultResponse)
async def confirm_data_cleanup(
    request: DataCleanupConfirmRequest,
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> DataCleanupResultResponse:
    try:
        result = await ConfirmDataCleanupUseCase(unit_of_work_factory).execute(
            request.to_dto(),
            confirmation_text=request.confirmation_text,
        )
    except InvalidDataCleanupRequestError as exc:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(exc)
        ) from exc
    except CleanupConfirmationError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc
    return DataCleanupResultResponse.from_dto(result)
