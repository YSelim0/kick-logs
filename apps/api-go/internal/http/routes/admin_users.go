package routes

import (
	"errors"
	"net/http"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
)

func RegisterAdminUserRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/users", func(response http.ResponseWriter, request *http.Request) {
		listAdminUsers(response, request, deps)
	})
	mux.HandleFunc("POST /admin/users", func(response http.ResponseWriter, request *http.Request) {
		createAdminUser(response, request, deps)
	})
}

func listAdminUsers(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}

	users, err := deps.Auth.ListUsers(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	payload := make([]schemas.AdminUserResponse, 0, len(users))
	for _, user := range users {
		payload = append(payload, adminUserResponse(user))
	}
	writeJSON(response, http.StatusOK, payload)
}

func createAdminUser(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	currentUser, ok := requireSuperAdmin(response, request, deps.Auth, deps.Config)
	if !ok {
		return
	}

	var payload schemas.CreateAdminUserRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeError(response, http.StatusBadRequest, "Invalid request body.")
		return
	}

	createdUser, err := deps.Auth.CreateAdminUser(
		request.Context(),
		currentUser,
		payload.Email,
		payload.Password,
	)
	if err != nil {
		switch {
		case errors.Is(err, authusecase.ErrUserAlreadyExists):
			writeError(response, http.StatusConflict, "User email already exists.")
		case errors.Is(err, authusecase.ErrSuperAdminRequired):
			writeError(response, http.StatusForbidden, "Super admin role required.")
		case errors.Is(err, authusecase.ErrValidation):
			writeError(response, http.StatusBadRequest, "Invalid request body.")
		default:
			writeError(response, http.StatusInternalServerError, "Internal server error.")
		}
		return
	}

	writeJSON(response, http.StatusCreated, adminUserResponse(createdUser))
}
