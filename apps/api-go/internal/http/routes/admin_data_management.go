package routes

import (
	"errors"
	"net/http"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	datamanagementusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/data_management"
)

func RegisterAdminDataManagementRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/data-management/summary", func(response http.ResponseWriter, request *http.Request) {
		dataManagementSummary(response, request, deps)
	})
	mux.HandleFunc("PUT /admin/data-management/retention-settings", func(response http.ResponseWriter, request *http.Request) {
		updateRetentionSettings(response, request, deps)
	})
	mux.HandleFunc("POST /admin/data-management/cleanup/preview", func(response http.ResponseWriter, request *http.Request) {
		previewDataCleanup(response, request, deps)
	})
	mux.HandleFunc("POST /admin/data-management/cleanup/confirm", func(response http.ResponseWriter, request *http.Request) {
		confirmDataCleanup(response, request, deps)
	})
}

func dataManagementSummary(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Data == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	summary, err := deps.Data.Summary(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	writeJSON(response, http.StatusOK, dataManagementSummaryResponse(summary))
}

func updateRetentionSettings(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Data == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	var payload schemas.RetentionSettingsRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}
	settings, err := deps.Data.UpdateRetentionSettings(
		request.Context(),
		payload.MessageRetentionDays,
		payload.RawEventRetentionDays,
	)
	if err != nil {
		writeDataManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, retentionSettingsResponse(settings))
}

func previewDataCleanup(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Data == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	var payload schemas.DataCleanupRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}
	preview, err := deps.Data.PreviewCleanup(request.Context(), dataCleanupRequest(payload))
	if err != nil {
		writeDataManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, dataCleanupPreviewResponse(preview))
}

func confirmDataCleanup(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Data == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	var payload schemas.DataCleanupConfirmRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}
	result, err := deps.Data.ConfirmCleanup(
		request.Context(),
		domain.DataCleanupRequest{
			Target:      domain.DataCleanupTarget(payload.Target),
			ChannelSlug: stringFromPointer(payload.ChannelSlug),
			Sender:      stringFromPointer(payload.Sender),
		},
		payload.ConfirmationText,
	)
	if err != nil {
		writeDataManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, dataCleanupResultResponse(result))
}

func dataCleanupRequest(payload schemas.DataCleanupRequest) domain.DataCleanupRequest {
	return domain.DataCleanupRequest{
		Target:      domain.DataCleanupTarget(payload.Target),
		ChannelSlug: stringFromPointer(payload.ChannelSlug),
		Sender:      stringFromPointer(payload.Sender),
	}
}

func writeDataManagementError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, datamanagementusecase.ErrValidation):
		writeError(response, http.StatusBadRequest, "Invalid request body.")
	case errors.Is(err, datamanagementusecase.ErrCleanupCannotExecute):
		writeError(response, http.StatusBadRequest, "Cleanup cannot be executed.")
	case errors.Is(err, datamanagementusecase.ErrCleanupConfirmation):
		writeError(response, http.StatusBadRequest, "Confirmation text does not match.")
	default:
		writeError(response, http.StatusInternalServerError, "Internal server error.")
	}
}

func stringFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
