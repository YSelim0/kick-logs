package routes

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	datamanagementusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/data_management"
	messageimportusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messageimport"
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
	mux.HandleFunc("POST /admin/data-management/import/preview", func(response http.ResponseWriter, request *http.Request) {
		previewMessageImport(response, request, deps)
	})
	mux.HandleFunc("POST /admin/data-management/import/confirm", func(response http.ResponseWriter, request *http.Request) {
		confirmMessageImport(response, request, deps)
	})
}

func previewMessageImport(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.MessageImport == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	payload, limit, ok := readImportUpload(response, request, deps)
	if !ok {
		return
	}
	preview, err := deps.MessageImport.Preview(request.Context(), payload, limit)
	if err != nil {
		writeMessageImportError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, messageImportPreviewResponse(preview))
}

func confirmMessageImport(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.MessageImport == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	payload, limit, ok := readImportUpload(response, request, deps)
	if !ok {
		return
	}
	result, err := deps.MessageImport.Confirm(
		request.Context(),
		payload,
		limit,
		request.FormValue("confirmation_text"),
	)
	if err != nil {
		writeMessageImportError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, messageImportResultResponse(result))
}

// readImportUpload reads the uploaded export file, bounded by the configured
// upload size so a large upload cannot exhaust the API container's memory.
func readImportUpload(
	response http.ResponseWriter,
	request *http.Request,
	deps Dependencies,
) ([]byte, int, bool) {
	maxUploadBytes := deps.Config.MessageImportMaxUploadBytes
	if maxUploadBytes <= 0 {
		maxUploadBytes = 16 * 1024 * 1024
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxUploadBytes)

	if err := request.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid or too large upload.")
		return nil, 0, false
	}

	file, _, err := request.FormFile("file")
	if err != nil {
		writeError(response, http.StatusBadRequest, "Missing export file.")
		return nil, 0, false
	}
	defer file.Close()

	payload, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		writeError(response, http.StatusBadRequest, "Could not read export file.")
		return nil, 0, false
	}

	limit := 0
	if raw := strings.TrimSpace(request.FormValue("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeError(response, http.StatusBadRequest, "Invalid limit.")
			return nil, 0, false
		}
		limit = parsed
	}
	return payload, limit, true
}

func writeMessageImportError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, messageimportusecase.ErrValidation):
		writeError(response, http.StatusBadRequest, err.Error())
	case errors.Is(err, messageimportusecase.ErrCannotExecute):
		writeError(response, http.StatusBadRequest, "Import cannot be executed.")
	case errors.Is(err, messageimportusecase.ErrConfirmation):
		writeError(response, http.StatusBadRequest, "Confirmation text does not match.")
	default:
		writeError(response, http.StatusInternalServerError, "Internal server error.")
	}
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
