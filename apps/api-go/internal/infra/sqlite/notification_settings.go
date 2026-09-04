package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

// NotificationSettingsRepository stores the single admin-editable
// watched-sender notification cooldown row. defaultCooldownSeconds seeds the
// row the first time it is read (before any admin edit), matching the
// process's NOTIFY_EMAIL_COOLDOWN_SECONDS startup default so an operator who
// already configured that env var keeps the same behavior after upgrading.
type NotificationSettingsRepository struct {
	db                     *sql.DB
	defaultCooldownSeconds int
}

func NewNotificationSettingsRepository(db *sql.DB, defaultCooldownSeconds int) *NotificationSettingsRepository {
	if defaultCooldownSeconds <= 0 {
		defaultCooldownSeconds = 600
	}
	return &NotificationSettingsRepository{db: db, defaultCooldownSeconds: defaultCooldownSeconds}
}

func (repo *NotificationSettingsRepository) GetNotificationSettings(ctx context.Context) (domain.NotificationSettings, error) {
	settings, err := repo.readNotificationSettings(ctx)
	if err == nil {
		return settings, nil
	}
	if err != sql.ErrNoRows {
		return domain.NotificationSettings{}, err
	}

	if err := repo.ensureNotificationSettings(ctx); err != nil {
		return domain.NotificationSettings{}, err
	}
	return repo.readNotificationSettings(ctx)
}

func (repo *NotificationSettingsRepository) readNotificationSettings(ctx context.Context) (domain.NotificationSettings, error) {
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT cooldown_seconds, updated_at FROM notification_settings WHERE id = 1`,
	)
	var (
		settings  domain.NotificationSettings
		updatedAt string
	)
	if err := row.Scan(&settings.CooldownSeconds, &updatedAt); err != nil {
		return domain.NotificationSettings{}, err
	}
	settings.UpdatedAt = parseTime(updatedAt)
	return settings, nil
}

func (repo *NotificationSettingsRepository) ensureNotificationSettings(ctx context.Context) error {
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO notification_settings (id, cooldown_seconds, updated_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		repo.defaultCooldownSeconds,
		formatTime(time.Now().UTC()),
	)
	if err != nil {
		return fmt.Errorf("seed notification settings: %w", err)
	}
	return nil
}

func (repo *NotificationSettingsRepository) UpdateNotificationSettings(
	ctx context.Context,
	settings domain.NotificationSettings,
) (domain.NotificationSettings, error) {
	if _, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO notification_settings (id, cooldown_seconds, updated_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			cooldown_seconds = excluded.cooldown_seconds,
			updated_at = excluded.updated_at`,
		settings.CooldownSeconds,
		formatTime(time.Now().UTC()),
	); err != nil {
		return domain.NotificationSettings{}, fmt.Errorf("update notification settings: %w", err)
	}
	return repo.GetNotificationSettings(ctx)
}
