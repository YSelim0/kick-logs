package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type AdminUserRepository struct {
	db *sql.DB
}

func NewAdminUserRepository(db *sql.DB) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

func (repo *AdminUserRepository) Upsert(ctx context.Context, user domain.AdminUser) (domain.AdminUser, error) {
	user.Email = normalizeEmail(user.Email)
	if user.Email == "" {
		return domain.AdminUser{}, fmt.Errorf("admin email is required")
	}
	if user.PasswordHash == "" {
		return domain.AdminUser{}, fmt.Errorf("admin password hash is required")
	}
	if user.Role == "" {
		user.Role = domain.AdminRoleAdmin
	}

	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	existing, err := repo.GetByEmail(ctx, user.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.AdminUser{}, err
	}

	if errors.Is(err, sql.ErrNoRows) {
		result, err := repo.db.ExecContext(
			ctx,
			`INSERT INTO admin_users (email, password_hash, role, is_active, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			user.Email,
			user.PasswordHash,
			string(user.Role),
			boolToInt(user.IsActive),
			formatTime(user.CreatedAt),
			formatTime(user.UpdatedAt),
		)
		if err != nil {
			return domain.AdminUser{}, fmt.Errorf("insert admin user: %w", err)
		}
		user.ID, _ = result.LastInsertId()
		return user, nil
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`UPDATE admin_users
		 SET password_hash = ?, role = ?, is_active = ?, updated_at = ?
		 WHERE id = ?`,
		user.PasswordHash,
		string(user.Role),
		boolToInt(user.IsActive),
		formatTime(user.UpdatedAt),
		existing.ID,
	); err != nil {
		return domain.AdminUser{}, fmt.Errorf("update admin user: %w", err)
	}
	return repo.GetByEmail(ctx, user.Email)
}

func (repo *AdminUserRepository) GetByEmail(ctx context.Context, email string) (domain.AdminUser, error) {
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, role, is_active, created_at, updated_at
		 FROM admin_users
		 WHERE email = ?`,
		normalizeEmail(email),
	)
	return scanAdminUser(row)
}

func (repo *AdminUserRepository) ListActive(ctx context.Context) ([]domain.AdminUser, error) {
	rows, err := repo.db.QueryContext(
		ctx,
		`SELECT id, email, password_hash, role, is_active, created_at, updated_at
		 FROM admin_users
		 WHERE is_active = 1
		 ORDER BY email ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active admin users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.AdminUser, 0)
	for rows.Next() {
		user, err := scanAdminUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active admin users: %w", err)
	}
	return users, nil
}

func SeedSuperAdmin(ctx context.Context, repo *AdminUserRepository, email string, password string) (domain.AdminUser, error) {
	email = normalizeEmail(email)
	if email == "" {
		return domain.AdminUser{}, fmt.Errorf("default super admin email is required")
	}
	if password == "" {
		return domain.AdminUser{}, fmt.Errorf("default super admin password is required")
	}

	existing, err := repo.GetByEmail(ctx, email)
	if err == nil {
		if existing.Role == domain.AdminRoleSuperAdmin && existing.IsActive {
			return existing, nil
		}
		existing.Role = domain.AdminRoleSuperAdmin
		existing.IsActive = true
		return repo.Upsert(ctx, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.AdminUser{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AdminUser{}, fmt.Errorf("hash default super admin password: %w", err)
	}

	return repo.Upsert(ctx, domain.AdminUser{
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.AdminRoleSuperAdmin,
		IsActive:     true,
	})
}

type adminUserScanner interface {
	Scan(dest ...any) error
}

func scanAdminUser(scanner adminUserScanner) (domain.AdminUser, error) {
	var user domain.AdminUser
	var role string
	var isActive int
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(&user.ID, &user.Email, &user.PasswordHash, &role, &isActive, &createdAt, &updatedAt); err != nil {
		return domain.AdminUser{}, err
	}
	user.Role = domain.AdminRole(role)
	user.IsActive = intToBool(isActive)
	user.CreatedAt = parseTime(createdAt)
	user.UpdatedAt = parseTime(updatedAt)
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
