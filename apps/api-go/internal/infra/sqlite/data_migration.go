package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type DataMigrationRepository struct {
	db *sql.DB
}

func NewDataMigrationRepository(db *sql.DB) *DataMigrationRepository {
	return &DataMigrationRepository{db: db}
}

func (repo *DataMigrationRepository) UpsertAdminUser(ctx context.Context, user domain.AdminUser) error {
	if user.ID < 1 {
		return fmt.Errorf("migrated admin user id is required")
	}
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO admin_users (id, email, password_hash, role, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			password_hash = excluded.password_hash,
			role = excluded.role,
			is_active = excluded.is_active,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at`,
		user.ID,
		normalizeEmail(user.Email),
		user.PasswordHash,
		string(user.Role),
		boolToInt(user.IsActive),
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
	)
	if err != nil {
		if updateErr := repo.updateAdminUserByEmail(ctx, user); updateErr == nil {
			return nil
		}
		return fmt.Errorf("migrate admin user %d: %w", user.ID, err)
	}
	return nil
}

func (repo *DataMigrationRepository) updateAdminUserByEmail(ctx context.Context, user domain.AdminUser) error {
	result, err := repo.db.ExecContext(
		ctx,
		`UPDATE admin_users
		 SET id = ?, password_hash = ?, role = ?, is_active = ?, created_at = ?, updated_at = ?
		 WHERE email = ?`,
		user.ID,
		user.PasswordHash,
		string(user.Role),
		boolToInt(user.IsActive),
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
		normalizeEmail(user.Email),
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repo *DataMigrationRepository) UpsertFollowedChannel(ctx context.Context, channel domain.FollowedChannel) error {
	if channel.ID < 1 {
		return fmt.Errorf("migrated followed channel id is required")
	}
	if channel.RawPayloadJSON == "" {
		channel.RawPayloadJSON = "{}"
	}
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO followed_channels (
			id, kick_channel_id, kick_chatroom_id, slug, display_name, profile_image_url,
			banner_image_url, is_enabled, raw_payload_json, created_at, updated_at,
			last_resolved_at, last_message_at, last_listener_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kick_channel_id = excluded.kick_channel_id,
			kick_chatroom_id = excluded.kick_chatroom_id,
			slug = excluded.slug,
			display_name = excluded.display_name,
			profile_image_url = excluded.profile_image_url,
			banner_image_url = excluded.banner_image_url,
			is_enabled = excluded.is_enabled,
			raw_payload_json = excluded.raw_payload_json,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			last_resolved_at = excluded.last_resolved_at,
			last_message_at = excluded.last_message_at,
			last_listener_error = excluded.last_listener_error`,
		channel.ID,
		nullInt64(channel.KickChannelID),
		nullInt64(channel.KickChatroomID),
		normalizeSlug(channel.Slug),
		channel.DisplayName,
		channel.ProfileImageURL,
		channel.BannerImageURL,
		boolToInt(channel.IsEnabled),
		channel.RawPayloadJSON,
		formatTime(channel.CreatedAt),
		formatTime(channel.UpdatedAt),
		formatTime(channel.LastResolvedAt),
		formatTime(channel.LastMessageAt),
		channel.LastListenerError,
	)
	if err != nil {
		if updateErr := repo.updateFollowedChannelBySlug(ctx, channel); updateErr == nil {
			return nil
		}
		return fmt.Errorf("migrate followed channel %d: %w", channel.ID, err)
	}
	return nil
}

func (repo *DataMigrationRepository) updateFollowedChannelBySlug(ctx context.Context, channel domain.FollowedChannel) error {
	result, err := repo.db.ExecContext(
		ctx,
		`UPDATE followed_channels
		 SET id = ?, kick_channel_id = ?, kick_chatroom_id = ?, display_name = ?,
		     profile_image_url = ?, banner_image_url = ?, is_enabled = ?, raw_payload_json = ?,
		     created_at = ?, updated_at = ?, last_resolved_at = ?, last_message_at = ?,
		     last_listener_error = ?
		 WHERE slug = ?`,
		channel.ID,
		nullInt64(channel.KickChannelID),
		nullInt64(channel.KickChatroomID),
		channel.DisplayName,
		channel.ProfileImageURL,
		channel.BannerImageURL,
		boolToInt(channel.IsEnabled),
		channel.RawPayloadJSON,
		formatTime(channel.CreatedAt),
		formatTime(channel.UpdatedAt),
		formatTime(channel.LastResolvedAt),
		formatTime(channel.LastMessageAt),
		channel.LastListenerError,
		normalizeSlug(channel.Slug),
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repo *DataMigrationRepository) UpsertSenderProfile(ctx context.Context, sender domain.SenderProfile) error {
	if sender.ID < 1 {
		return fmt.Errorf("migrated sender profile id is required")
	}
	if sender.RawProfilePayloadJSON == "" {
		sender.RawProfilePayloadJSON = "{}"
	}
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO sender_profiles (
			id, kick_user_id, username, slug, profile_image_url, last_seen_color,
			raw_profile_payload_json, created_at, updated_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kick_user_id = excluded.kick_user_id,
			username = excluded.username,
			slug = excluded.slug,
			profile_image_url = excluded.profile_image_url,
			last_seen_color = excluded.last_seen_color,
			raw_profile_payload_json = excluded.raw_profile_payload_json,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			last_seen_at = excluded.last_seen_at`,
		sender.ID,
		sender.KickUserID,
		sender.Username,
		normalizeSlug(sender.Slug),
		sender.ProfileImageURL,
		sender.LastSeenColor,
		sender.RawProfilePayloadJSON,
		formatTime(sender.CreatedAt),
		formatTime(sender.UpdatedAt),
		formatTime(sender.LastSeenAt),
	)
	if err != nil {
		if updateErr := repo.updateSenderProfileByNaturalKey(ctx, sender); updateErr == nil {
			return nil
		}
		return fmt.Errorf("migrate sender profile %d: %w", sender.ID, err)
	}
	return nil
}

func (repo *DataMigrationRepository) updateSenderProfileByNaturalKey(ctx context.Context, sender domain.SenderProfile) error {
	result, err := repo.db.ExecContext(
		ctx,
		`UPDATE sender_profiles
		 SET id = ?, kick_user_id = ?, username = ?, slug = ?, profile_image_url = ?, last_seen_color = ?,
		     raw_profile_payload_json = ?, created_at = ?, updated_at = ?, last_seen_at = ?
		 WHERE kick_user_id = ? OR slug = ?`,
		sender.ID,
		sender.KickUserID,
		sender.Username,
		normalizeSlug(sender.Slug),
		sender.ProfileImageURL,
		sender.LastSeenColor,
		sender.RawProfilePayloadJSON,
		formatTime(sender.CreatedAt),
		formatTime(sender.UpdatedAt),
		formatTime(sender.LastSeenAt),
		sender.KickUserID,
		normalizeSlug(sender.Slug),
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repo *DataMigrationRepository) UpsertRetentionSettings(ctx context.Context, settings domain.RetentionSettings) error {
	if settings.ID != 1 {
		return fmt.Errorf("unsupported retention settings id %d; expected singleton id 1", settings.ID)
	}
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO retention_settings (
			id, message_retention_days, raw_event_retention_days, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			message_retention_days = excluded.message_retention_days,
			raw_event_retention_days = excluded.raw_event_retention_days,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at`,
		settings.ID,
		nullableInt(settings.MessageRetentionDays),
		nullableInt(settings.RawEventRetentionDays),
		formatTime(settings.CreatedAt),
		formatTime(settings.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("migrate retention settings: %w", err)
	}
	return nil
}

func (repo *DataMigrationRepository) UpsertWorkerHeartbeat(ctx context.Context, heartbeat domain.ListenerHeartbeat) error {
	if heartbeat.MetadataJSON == "" {
		heartbeat.MetadataJSON = "{}"
	}
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO worker_heartbeats (
			service_name, last_seen_at, metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(service_name) DO UPDATE SET
			last_seen_at = excluded.last_seen_at,
			metadata_json = excluded.metadata_json,
			updated_at = excluded.updated_at`,
		heartbeat.ServiceName,
		formatTime(heartbeat.LastSeenAt),
		heartbeat.MetadataJSON,
		formatTime(heartbeat.LastSeenAt),
		formatTime(heartbeat.LastSeenAt),
	)
	if err != nil {
		return fmt.Errorf("migrate worker heartbeat %q: %w", heartbeat.ServiceName, err)
	}
	return nil
}

func (repo *DataMigrationRepository) ControlCounts(ctx context.Context) (domain.MigrationCounts, error) {
	counts := domain.MigrationCounts{}
	tables := map[string]*int64{
		"admin_users":        &counts.AdminUsers,
		"followed_channels":  &counts.FollowedChannels,
		"sender_profiles":    &counts.SenderProfiles,
		"retention_settings": &counts.RetentionSettings,
		"worker_heartbeats":  &counts.WorkerHeartbeats,
	}
	for table, target := range tables {
		if err := repo.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(target); err != nil {
			return domain.MigrationCounts{}, fmt.Errorf("count sqlite table %s: %w", table, err)
		}
	}
	return counts, nil
}

func (repo *DataMigrationRepository) FindAdminUser(ctx context.Context, id int64) (domain.AdminUser, error) {
	return scanAdminUser(repo.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, role, is_active, created_at, updated_at
		 FROM admin_users WHERE id = ?`,
		id,
	))
}

func (repo *DataMigrationRepository) FindFollowedChannel(ctx context.Context, id int64) (domain.FollowedChannel, error) {
	return scanFollowedChannel(repo.db.QueryRowContext(
		ctx,
		`SELECT id, kick_channel_id, kick_chatroom_id, slug, display_name, profile_image_url,
		        banner_image_url, is_enabled, raw_payload_json, created_at, updated_at,
		        last_resolved_at, last_message_at, last_listener_error
		 FROM followed_channels WHERE id = ?`,
		id,
	))
}

func (repo *DataMigrationRepository) FindSenderProfile(ctx context.Context, id int64) (domain.SenderProfile, error) {
	return scanSenderProfile(repo.db.QueryRowContext(
		ctx,
		`SELECT id, kick_user_id, username, slug, profile_image_url, last_seen_color,
		        raw_profile_payload_json, created_at, updated_at, last_seen_at
		 FROM sender_profiles WHERE id = ?`,
		id,
	))
}

func (repo *DataMigrationRepository) RecordRun(ctx context.Context, run domain.DataMigrationRun) error {
	if run.RunID == "" {
		return fmt.Errorf("migration run id is required")
	}
	_, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO data_migration_runs (
			run_id, name, mode, status, source_counts_json, destination_counts_json,
			validation_json, error_message, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			status = excluded.status,
			source_counts_json = excluded.source_counts_json,
			destination_counts_json = excluded.destination_counts_json,
			validation_json = excluded.validation_json,
			error_message = excluded.error_message,
			finished_at = excluded.finished_at`,
		run.RunID,
		run.Name,
		run.Mode,
		run.Status,
		run.SourceCountsJSON,
		run.DestinationCountsJSON,
		run.ValidationJSON,
		run.ErrorMessage,
		formatTime(run.StartedAt),
		formatTime(run.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("record data migration run: %w", err)
	}
	return nil
}

func nullableInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}
