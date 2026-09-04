package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	watchedsendersusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/watchedsenders"
)

func RegisterAdminWatchedSenderRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/watched-senders", func(response http.ResponseWriter, request *http.Request) {
		listWatchedSenders(response, request, deps)
	})
	mux.HandleFunc("POST /admin/watched-senders", func(response http.ResponseWriter, request *http.Request) {
		addWatchedSender(response, request, deps)
	})
	mux.HandleFunc("DELETE /admin/watched-senders/{sender_id}", func(response http.ResponseWriter, request *http.Request) {
		removeWatchedSender(response, request, deps)
	})
}

func listWatchedSenders(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.WatchedSenders == nil {
		writeJSON(response, http.StatusOK, []schemas.WatchedSenderResponse{})
		return
	}

	senders, err := deps.WatchedSenders.List(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	payload := make([]schemas.WatchedSenderResponse, 0, len(senders))
	for _, sender := range senders {
		payload = append(payload, watchedSenderResponse(sender))
	}
	writeJSON(response, http.StatusOK, payload)
}

func addWatchedSender(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.WatchedSenders == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	var payload schemas.AddWatchedSenderRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}

	sender, err := deps.WatchedSenders.Add(request.Context(), payload.Username)
	if err != nil {
		switch {
		case errors.Is(err, watchedsendersusecase.ErrValidation):
			writeError(response, http.StatusBadRequest, "Invalid username.")
		case errors.Is(err, watchedsendersusecase.ErrAlreadyWatched):
			writeError(response, http.StatusConflict, "Username is already watched.")
		default:
			writeError(response, http.StatusInternalServerError, "Internal server error.")
		}
		return
	}

	writeJSON(response, http.StatusCreated, watchedSenderResponse(sender))
}

func removeWatchedSender(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.WatchedSenders == nil {
		writeError(response, http.StatusNotFound, "Watched sender not found.")
		return
	}

	senderID, err := strconv.ParseInt(request.PathValue("sender_id"), 10, 64)
	if err != nil || senderID <= 0 {
		writeError(response, http.StatusNotFound, "Watched sender not found.")
		return
	}

	if err := deps.WatchedSenders.Remove(request.Context(), senderID); err != nil {
		if errors.Is(err, watchedsendersusecase.ErrSenderNotFound) {
			writeError(response, http.StatusNotFound, "Watched sender not found.")
			return
		}
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	writeJSON(response, http.StatusOK, statusResponse{Status: "ok"})
}
