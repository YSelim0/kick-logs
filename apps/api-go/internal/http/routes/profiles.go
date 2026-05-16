package routes

import (
	"errors"
	"net/http"
	"strings"

	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
)

func RegisterProfileRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /users/{slug}/analytics", func(response http.ResponseWriter, request *http.Request) {
		getUserProfile(response, request, deps)
	})
	mux.HandleFunc("GET /channels/{slug}/analytics", func(response http.ResponseWriter, request *http.Request) {
		getChannelProfile(response, request, deps)
	})
}

func getUserProfile(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Profiles == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	slug, ok := profileSlug(response, request)
	if !ok {
		return
	}
	profile, err := deps.Profiles.UserProfile(request.Context(), slug)
	if err != nil {
		writeProfileUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, userProfileResponse(profile))
}

func getChannelProfile(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Profiles == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	slug, ok := profileSlug(response, request)
	if !ok {
		return
	}
	profile, err := deps.Profiles.ChannelProfile(request.Context(), slug)
	if err != nil {
		writeProfileUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, channelProfileResponse(profile))
}

func profileSlug(response http.ResponseWriter, request *http.Request) (string, bool) {
	slug := strings.TrimSpace(request.PathValue("slug"))
	if slug == "" || len(slug) > 160 {
		writeError(response, http.StatusUnprocessableEntity, "Invalid profile slug.")
		return "", false
	}
	return slug, true
}

func writeProfileUseCaseError(response http.ResponseWriter, err error) {
	if errors.Is(err, profilesusecase.ErrSenderNotFound) {
		writeError(response, http.StatusNotFound, "Sender profile not found.")
		return
	}
	if errors.Is(err, profilesusecase.ErrChannelNotFound) {
		writeError(response, http.StatusNotFound, "Channel profile not found.")
		return
	}
	writeError(response, http.StatusInternalServerError, "Internal server error.")
}
