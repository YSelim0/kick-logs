package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAuthentication     = errors.New("authentication required")
	ErrInvalidSession     = errors.New("invalid session")
	ErrSuperAdminRequired = errors.New("super admin required")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrValidation         = errors.New("validation failed")
)

type Service struct {
	users  ports.AdminUserRepository
	hasher ports.PasswordHasher
	tokens ports.TokenService
}

func NewService(
	users ports.AdminUserRepository,
	hasher ports.PasswordHasher,
	tokens ports.TokenService,
) *Service {
	return &Service{users: users, hasher: hasher, tokens: tokens}
}

func (service *Service) Login(ctx context.Context, email string, password string) (domain.AdminUser, string, error) {
	if !validLength(email, 3, 320) || !validLength(password, 1, 256) {
		return domain.AdminUser{}, "", ErrInvalidCredentials
	}

	user, err := service.users.GetByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AdminUser{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return domain.AdminUser{}, "", err
	}
	if !user.IsActive || !service.hasher.Verify(password, user.PasswordHash) {
		return domain.AdminUser{}, "", ErrInvalidCredentials
	}

	token, err := service.tokens.CreateAccessToken(user.ID)
	if err != nil {
		return domain.AdminUser{}, "", err
	}
	return user, token, nil
}

func (service *Service) CurrentUser(ctx context.Context, token string) (domain.AdminUser, error) {
	if strings.TrimSpace(token) == "" {
		return domain.AdminUser{}, ErrAuthentication
	}

	userID, ok := service.tokens.GetUserID(token)
	if !ok {
		return domain.AdminUser{}, ErrInvalidSession
	}

	user, err := service.users.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AdminUser{}, ErrInvalidSession
	}
	if err != nil {
		return domain.AdminUser{}, err
	}
	if !user.IsActive {
		return domain.AdminUser{}, ErrInvalidSession
	}
	return user, nil
}

func (service *Service) ListUsers(ctx context.Context) ([]domain.AdminUser, error) {
	return service.users.ListActive(ctx)
}

func (service *Service) CreateAdminUser(
	ctx context.Context,
	currentUser domain.AdminUser,
	email string,
	password string,
) (domain.AdminUser, error) {
	if currentUser.Role != domain.AdminRoleSuperAdmin {
		return domain.AdminUser{}, ErrSuperAdminRequired
	}
	if !validLength(email, 3, 320) || !validLength(password, 8, 256) {
		return domain.AdminUser{}, fmt.Errorf("%w: invalid admin user payload", ErrValidation)
	}

	if _, err := service.users.GetByEmail(ctx, email); err == nil {
		return domain.AdminUser{}, ErrUserAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.AdminUser{}, err
	}

	hash, err := service.hasher.Hash(password)
	if err != nil {
		return domain.AdminUser{}, err
	}

	return service.users.Upsert(ctx, domain.AdminUser{
		Email:        email,
		PasswordHash: hash,
		Role:         domain.AdminRoleAdmin,
		IsActive:     true,
	})
}

func validLength(value string, minLength int, maxLength int) bool {
	trimmed := strings.TrimSpace(value)
	length := len(trimmed)
	return length >= minLength && length <= maxLength
}
