package routes

import (
	"errors"
	"net/http"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
	authusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/auth"
	channelsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/channels"
	datamanagementusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/data_management"
	messagesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messages"
	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
)

type Dependencies struct {
	Config      config.Config
	Auth        *authusecase.Service
	Analytics   *analyticsusecase.Service
	Channels    *channelsusecase.Service
	Messages    *messagesusecase.Service
	Profiles    *profilesusecase.Service
	Data        *datamanagementusecase.Service
	Operations  ports.OperationsRepository
}

func currentUser(request *http.Request, authService *authusecase.Service, cfg config.Config) (domain.AdminUser, error) {
	if authService == nil {
		return domain.AdminUser{}, authusecase.ErrAuthentication
	}
	cookie, err := request.Cookie(cfg.JWTCookieName)
	if err != nil {
		return domain.AdminUser{}, authusecase.ErrAuthentication
	}
	user, err := authService.CurrentUser(request.Context(), cookie.Value)
	if err != nil {
		return domain.AdminUser{}, err
	}
	return user, nil
}

func requireAdmin(response http.ResponseWriter, request *http.Request, authService *authusecase.Service, cfg config.Config) (domain.AdminUser, bool) {
	user, err := currentUser(request, authService, cfg)
	if err == nil {
		return user, true
	}
	if errors.Is(err, authusecase.ErrInvalidSession) {
		writeError(response, http.StatusUnauthorized, "Invalid session.")
		return domain.AdminUser{}, false
	}
	if errors.Is(err, authusecase.ErrAuthentication) {
		writeError(response, http.StatusUnauthorized, "Authentication required.")
		return domain.AdminUser{}, false
	}
	writeError(response, http.StatusInternalServerError, "Internal server error.")
	return domain.AdminUser{}, false
}

func requireSuperAdmin(response http.ResponseWriter, request *http.Request, authService *authusecase.Service, cfg config.Config) (domain.AdminUser, bool) {
	user, ok := requireAdmin(response, request, authService, cfg)
	if !ok {
		return domain.AdminUser{}, false
	}
	if user.Role != domain.AdminRoleSuperAdmin {
		writeError(response, http.StatusForbidden, "Super admin role required.")
		return domain.AdminUser{}, false
	}
	return user, true
}
