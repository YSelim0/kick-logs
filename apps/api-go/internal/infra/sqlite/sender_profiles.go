package sqlite

import (
	"context"
	"database/sql"
	"errors"
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

	existing, err := repo.GetByKickUserID(ctx, sender.KickUserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.SenderProfile{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		existing, err = repo.GetBySlug(ctx, sender.Slug)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return domain.SenderProfile{}, err
		}
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
	if errors.Is(err, sql.ErrNoRows) {
		result, err := repo.db.ExecContext(
			ctx,
			`INSERT INTO sender_profiles (
				kick_user_id, username, slug, profile_image_url, last_seen_color,
				raw_profile_payload_json, created_at, updated_at, last_seen_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			args...,
		)
		if err != nil {
			return domain.SenderProfile{}, fmt.Errorf("insert sender profile: %w", err)
		}
		sender.ID, _ = result.LastInsertId()
		return sender, nil
	}

	args = append(args, existing.ID)
	if _, err := repo.db.ExecContext(
		ctx,
		`UPDATE sender_profiles
		 SET kick_user_id = ?, username = ?, slug = ?, profile_image_url = ?,
		     last_seen_color = ?, raw_profile_payload_json = ?, created_at = ?,
		     updated_at = ?, last_seen_at = ?
		 WHERE id = ?`,
		args...,
	); err != nil {
		return domain.SenderProfile{}, fmt.Errorf("update sender profile: %w", err)
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
