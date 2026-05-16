package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type SenderProfileRepository struct {
	db *sql.DB
}

func NewSenderProfileRepository(db *sql.DB) *SenderProfileRepository {
	return &SenderProfileRepository{db: db}
}

func (repo *SenderProfileRepository) Upsert(ctx context.Context, sender domain.SenderProfile) (domain.SenderProfile, error) {
	sender.Slug = normalizeSlug(sender.Slug)
	if sender.KickUserID < 1 {
		return domain.SenderProfile{}, fmt.Errorf("sender kick user id is required")
	}
	if sender.Username == "" {
		return domain.SenderProfile{}, fmt.Errorf("sender username is required")
	}
	if sender.Slug == "" {
		sender.Slug = normalizeSlug(sender.Username)
	}
	if sender.RawProfilePayloadJSON == "" {
		sender.RawProfilePayloadJSON = "{}"
	}

	now := time.Now().UTC()
	if sender.CreatedAt.IsZero() {
		sender.CreatedAt = now
	}
	sender.UpdatedAt = now
	if sender.LastSeenAt.IsZero() {
		sender.LastSeenAt = now
	}

	args := []any{
		sender.KickUserID,
		sender.Username,
		sender.Slug,
		sender.ProfileImageURL,
		sender.LastSeenColor,
		sender.RawProfilePayloadJSON,
		formatTime(sender.CreatedAt),
		formatTime(sender.UpdatedAt),
		formatTime(sender.LastSeenAt),
	}
	if _, err := repo.db.ExecContext(
		ctx,
		`INSERT INTO sender_profiles (
			kick_user_id, username, slug, profile_image_url, last_seen_color,
			raw_profile_payload_json, created_at, updated_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(kick_user_id) DO UPDATE SET
			username = excluded.username,
			slug = excluded.slug,
			profile_image_url = excluded.profile_image_url,
			last_seen_color = excluded.last_seen_color,
			raw_profile_payload_json = excluded.raw_profile_payload_json,
			updated_at = excluded.updated_at,
			last_seen_at = excluded.last_seen_at
		ON CONFLICT(slug) DO UPDATE SET
			kick_user_id = excluded.kick_user_id,
			username = excluded.username,
			profile_image_url = excluded.profile_image_url,
			last_seen_color = excluded.last_seen_color,
			raw_profile_payload_json = excluded.raw_profile_payload_json,
			updated_at = excluded.updated_at,
			last_seen_at = excluded.last_seen_at`,
		args...,
	); err != nil {
		return domain.SenderProfile{}, fmt.Errorf("upsert sender profile: %w", err)
	}
	return repo.GetByKickUserID(ctx, sender.KickUserID)
}

func (repo *SenderProfileRepository) GetByKickUserID(ctx context.Context, kickUserID int64) (domain.SenderProfile, error) {
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT id, kick_user_id, username, slug, profile_image_url, last_seen_color,
		        raw_profile_payload_json, created_at, updated_at, last_seen_at
		 FROM sender_profiles
		 WHERE kick_user_id = ?`,
		kickUserID,
	)
	return scanSenderProfile(row)
}

func (repo *SenderProfileRepository) GetBySlug(ctx context.Context, slug string) (domain.SenderProfile, error) {
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT id, kick_user_id, username, slug, profile_image_url, last_seen_color,
		        raw_profile_payload_json, created_at, updated_at, last_seen_at
		 FROM sender_profiles
		 WHERE slug = ?`,
		normalizeSlug(slug),
	)
	return scanSenderProfile(row)
}

type senderProfileScanner interface {
	Scan(dest ...any) error
}

func scanSenderProfile(scanner senderProfileScanner) (domain.SenderProfile, error) {
	var sender domain.SenderProfile
	var createdAt string
	var updatedAt string
	var lastSeenAt string
	if err := scanner.Scan(
		&sender.ID,
		&sender.KickUserID,
		&sender.Username,
		&sender.Slug,
		&sender.ProfileImageURL,
		&sender.LastSeenColor,
		&sender.RawProfilePayloadJSON,
		&createdAt,
		&updatedAt,
		&lastSeenAt,
	); err != nil {
		return domain.SenderProfile{}, err
	}
	sender.CreatedAt = parseTime(createdAt)
	sender.UpdatedAt = parseTime(updatedAt)
	sender.LastSeenAt = parseTime(lastSeenAt)
	return sender, nil
}
