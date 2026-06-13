package routes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/middleware"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	requestsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/requests"
)

func RegisterUserRequestRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("POST /requests", func(response http.ResponseWriter, request *http.Request) {
		createUserRequest(response, request, deps)
	})
	mux.HandleFunc("GET /admin/requests", func(response http.ResponseWriter, request *http.Request) {
		listUserRequests(response, request, deps)
	})
	mux.HandleFunc("GET /admin/requests/{request_id}", func(response http.ResponseWriter, request *http.Request) {
		getUserRequest(response, request, deps)
	})
	mux.HandleFunc("POST /admin/requests/{request_id}/status", func(response http.ResponseWriter, request *http.Request) {
		updateUserRequestStatus(response, request, deps)
	})
	mux.HandleFunc("POST /admin/requests/{request_id}/notes", func(response http.ResponseWriter, request *http.Request) {
		addUserRequestNote(response, request, deps)
	})
	mux.HandleFunc("POST /admin/requests/{request_id}/archive", func(response http.ResponseWriter, request *http.Request) {
		archiveUserRequest(response, request, deps)
	})
}

func createUserRequest(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Requests == nil {
		writeError(response, http.StatusServiceUnavailable, "Request storage unavailable.")
		return
	}

	var payload schemas.CreateUserRequestRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if strings.TrimSpace(payload.Website) != "" {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}

	clientIP := middleware.ClientIP(request, deps.Config.RateLimitTrustProxy, deps.Config.RateLimitClientIPHeader)
	created, err := deps.Requests.Create(request.Context(), requestsusecase.CreateInput{
		Type:               payload.Type,
		Title:              payload.Title,
		Message:            payload.Message,
		ChannelSlug:        payload.ChannelSlug,
		ChannelDisplayName: payload.ChannelDisplayName,
		Contact:            payload.Contact,
		IPHash:             hashRequestMetadata(clientIP, deps.Config.JWTSecretKey),
		UserAgentHash:      hashRequestMetadata(request.UserAgent(), deps.Config.JWTSecretKey),
	})
	if err != nil {
		writeUserRequestError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, schemas.CreateUserRequestResponse{RequestID: created.ID})
}

func listUserRequests(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Requests == nil {
		writeError(response, http.StatusServiceUnavailable, "Request storage unavailable.")
		return
	}

	filter, err := parseUserRequestFilter(request)
	if err != nil {
		writeUserRequestError(response, err)
		return
	}
	items, err := deps.Requests.List(request.Context(), filter)
	if err != nil {
		writeUserRequestError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, userRequestsResponse(items))
}

func getUserRequest(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	detail, ok := userRequestDetail(response, request, deps)
	if !ok {
		return
	}
	writeJSON(response, http.StatusOK, userRequestDetailResponse(detail))
}

func updateUserRequestStatus(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	admin, ok := requireAdmin(response, request, deps.Auth, deps.Config)
	if !ok {
		return
	}
	if deps.Requests == nil {
		writeError(response, http.StatusServiceUnavailable, "Request storage unavailable.")
		return
	}

	var payload schemas.UpdateUserRequestStatusRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}
	detail, err := deps.Requests.ChangeStatus(request.Context(), request.PathValue("request_id"), payload.Status, admin.ID)
	if err != nil {
		writeUserRequestError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, userRequestDetailResponse(detail))
}

func addUserRequestNote(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	admin, ok := requireAdmin(response, request, deps.Auth, deps.Config)
	if !ok {
		return
	}
	if deps.Requests == nil {
		writeError(response, http.StatusServiceUnavailable, "Request storage unavailable.")
		return
	}

	var payload schemas.AddUserRequestNoteRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}
	detail, err := deps.Requests.AddNote(request.Context(), request.PathValue("request_id"), payload.Note, admin.ID)
	if err != nil {
		writeUserRequestError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, userRequestDetailResponse(detail))
}

func archiveUserRequest(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	admin, ok := requireAdmin(response, request, deps.Auth, deps.Config)
	if !ok {
		return
	}
	if deps.Requests == nil {
		writeError(response, http.StatusServiceUnavailable, "Request storage unavailable.")
		return
	}

	detail, err := deps.Requests.Archive(request.Context(), request.PathValue("request_id"), admin.ID)
	if err != nil {
		writeUserRequestError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, userRequestDetailResponse(detail))
}

func userRequestDetail(
	response http.ResponseWriter,
	request *http.Request,
	deps Dependencies,
) (domain.UserRequestDetail, bool) {
	if deps.Requests == nil {
		writeError(response, http.StatusServiceUnavailable, "Request storage unavailable.")
		return domain.UserRequestDetail{}, false
	}
	detail, err := deps.Requests.Detail(request.Context(), request.PathValue("request_id"))
	if err != nil {
		writeUserRequestError(response, err)
		return domain.UserRequestDetail{}, false
	}
	return detail, true
}

func parseUserRequestFilter(request *http.Request) (domain.UserRequestListFilter, error) {
	query := request.URL.Query()
	filter := domain.UserRequestListFilter{
		Type:   domain.UserRequestType(strings.TrimSpace(query.Get("type"))),
		Status: domain.UserRequestStatus(strings.TrimSpace(query.Get("status"))),
		Query:  strings.TrimSpace(query.Get("q")),
	}

	if rawArchived := strings.TrimSpace(query.Get("archived")); rawArchived != "" {
		archived, err := strconv.ParseBool(rawArchived)
		if err != nil {
			return domain.UserRequestListFilter{}, requestsusecase.ErrValidation
		}
		filter.Archived = &archived
	}

	var err error
	if filter.Start, err = optionalTime(query.Get("start")); err != nil {
		return domain.UserRequestListFilter{}, requestsusecase.ErrValidation
	}
	if filter.End, err = optionalTime(query.Get("end")); err != nil {
		return domain.UserRequestListFilter{}, requestsusecase.ErrValidation
	}

	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		limit, err := strconv.ParseUint(rawLimit, 10, 64)
		if err != nil || limit < 1 {
			return domain.UserRequestListFilter{}, requestsusecase.ErrValidation
		}
		filter.Limit = limit
	}
	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		offset, err := strconv.ParseUint(rawOffset, 10, 64)
		if err != nil {
			return domain.UserRequestListFilter{}, requestsusecase.ErrValidation
		}
		filter.Offset = offset
	}

	return filter, nil
}

func userRequestsResponse(states []domain.UserRequestState) schemas.UserRequestsResponse {
	items := make([]schemas.UserRequestResponse, 0, len(states))
	for _, state := range states {
		items = append(items, userRequestResponse(state))
	}
	return schemas.UserRequestsResponse{Items: items, Count: len(items)}
}

func userRequestDetailResponse(detail domain.UserRequestDetail) schemas.UserRequestDetailResponse {
	events := make([]schemas.UserRequestEventResponse, 0, len(detail.Events))
	for _, event := range detail.Events {
		events = append(events, schemas.UserRequestEventResponse{
			EventID:   event.ID,
			RequestID: event.RequestID,
			EventType: string(event.EventType),
			Status:    string(event.Status),
			Note:      event.Note,
			AdminID:   event.AdminID,
			CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return schemas.UserRequestDetailResponse{
		Request: userRequestResponse(detail.State),
		Events:  events,
	}
}

func userRequestResponse(state domain.UserRequestState) schemas.UserRequestResponse {
	return schemas.UserRequestResponse{
		RequestID:          state.Request.ID,
		Type:               string(state.Request.Type),
		Title:              state.Request.Title,
		Message:            state.Request.Message,
		ChannelSlug:        nullableString(state.Request.ChannelSlug),
		ChannelDisplayName: nullableString(state.Request.ChannelDisplayName),
		Contact:            nullableString(state.Request.Contact),
		CurrentStatus:      string(state.CurrentStatus),
		IsArchived:         state.IsArchived,
		CreatedAt:          state.Request.CreatedAt.UTC().Format(time.RFC3339),
		LatestEventAt:      state.LatestEventAt.UTC().Format(time.RFC3339),
	}
}

func hashRequestMetadata(value string, secret string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	key := strings.TrimSpace(secret)
	if key == "" {
		key = "kick-logs-request-metadata"
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func writeUserRequestError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, requestsusecase.ErrValidation):
		writeError(response, http.StatusBadRequest, "Invalid request body.")
	case errors.Is(err, requestsusecase.ErrNotFound):
		writeError(response, http.StatusNotFound, "Request not found.")
	default:
		writeError(response, http.StatusInternalServerError, "Internal server error.")
	}
}
