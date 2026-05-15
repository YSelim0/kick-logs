package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
)

func RegisterAdminChannelRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/channels", func(response http.ResponseWriter, request *http.Request) {
		listChannels(response, request, deps)
	})
	mux.HandleFunc("POST /admin/channels", func(response http.ResponseWriter, request *http.Request) {
		addChannel(response, request, deps)
	})
	mux.HandleFunc("DELETE /admin/channels/{channel_id}", func(response http.ResponseWriter, request *http.Request) {
		disableChannel(response, request, deps)
	})
}

func listChannels(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}

	channels, err := deps.Channels.List(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	payload := make([]schemas.ChannelResponse, 0, len(channels))
	for _, channel := range channels {
		payload = append(payload, channelResponse(channel))
	}
	writeJSON(response, http.StatusOK, payload)
}

func addChannel(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}

	var payload schemas.AddChannelRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}

	channel, err := deps.Channels.Add(request.Context(), payload.Slug)
	if err != nil {
		switch {
		case errors.Is(err, channelsusecase.ErrChannelResolution):
			writeError(response, http.StatusUnprocessableEntity, "Kick channel could not be resolved.")
		case errors.Is(err, channelsusecase.ErrValidation):
			writeError(response, http.StatusBadRequest, "Invalid request body.")
		default:
			writeError(response, http.StatusInternalServerError, "Internal server error.")
		}
		return
	}

	writeJSON(response, http.StatusCreated, channelResponse(channel))
}

func disableChannel(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}

	channelID, err := strconv.ParseInt(request.PathValue("channel_id"), 10, 64)
	if err != nil || channelID <= 0 {
		writeError(response, http.StatusNotFound, "Channel not found.")
		return
	}

	channel, err := deps.Channels.Disable(request.Context(), channelID)
	if err != nil {
		if errors.Is(err, channelsusecase.ErrChannelNotFound) {
			writeError(response, http.StatusNotFound, "Channel not found.")
			return
		}
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	writeJSON(response, http.StatusOK, channelResponse(channel))
}
