package routes

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/directory"
)

func RegisterDirectoryRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /directory/users", func(w http.ResponseWriter, r *http.Request) {
		getDirectory(w, r, deps, domain.DirectoryUsers)
	})
	mux.HandleFunc("GET /directory/channels", func(w http.ResponseWriter, r *http.Request) {
		getDirectory(w, r, deps, domain.DirectoryChannels)
	})
}

func getDirectory(w http.ResponseWriter, r *http.Request, deps Dependencies, kind domain.DirectoryKind) {
	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
		return
	}
	for key, values := range params {
		if len(values) != 1 || (key != "prefix" && key != "limit" && key != "after" && key != "cursor") {
			writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
			return
		}
	}
	if params.Has("after") && params.Has("cursor") {
		writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
		return
	}
	limit := directory.DefaultLimit
	if params.Has("limit") {
		limit, err = strconv.Atoi(params.Get("limit"))
		if err != nil || limit < 1 || limit > directory.MaxLimit {
			writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
			return
		}
	}
	after := params.Get("after")
	if params.Has("cursor") {
		after = params.Get("cursor")
	}
	if deps.Directory == nil {
		writeError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	page, err := deps.Directory.Search(r.Context(), kind, params.Get("prefix"), limit, after)
	if errors.Is(err, directory.ErrInvalidQuery) {
		writeError(w, http.StatusUnprocessableEntity, "Invalid query parameters.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	response := schemas.DirectoryResponse{Items: make([]schemas.DirectoryIdentityResponse, 0, len(page.Items))}
	for _, item := range page.Items {
		identity := schemas.DirectoryIdentityResponse{ID: item.ID, Name: item.Name, Slug: item.Slug}
		if item.ProfileImageURL != "" {
			identity.ProfileImageURL = &item.ProfileImageURL
		}
		response.Items = append(response.Items, identity)
	}
	if page.NextCursor != "" {
		response.NextCursor = &page.NextCursor
	}
	writeJSON(w, http.StatusOK, response)
}
