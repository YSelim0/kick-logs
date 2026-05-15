package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type FollowedChannelRepository struct {
	db *sql.DB
}

func NewFollowedChannelRepository(db *sql.DB) *FollowedChannelRepository {
	return &FollowedChannelRepository{db: db}
}

func (repo *FollowedChannelRepository) Upsert(ctx context.Context, channel domain.FollowedChannel) (domain.FollowedChannel, error) {
	channel.Slug = normalizeSlug(channel.Slug)
	if channel.Slug == "" {
		return domain.FollowedChannel{}, fmt.Errorf("channel slug is required")
	}
	if channel.DisplayName == "" {
		channel.DisplayName = channel.Slug
	}
	if channel.RawPayloadJSON == "" {
		channel.RawPayloadJSON = "{}"
	}

	now := time.Now().UTC()
	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = now
	}
	channel.UpdatedAt = now

	existing, err := repo.GetBySlug(ctx, channel.Slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.FollowedChannel{}, err
	}

	args := []any{
		nullInt64(channel.KickChannelID),
		nullInt64(channel.KickChatroomID),
		channel.Slug,
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
	}

	if errors.Is(err, sql.ErrNoRows) {
		result, err := repo.db.ExecContext(
			ctx,
			`INSERT INTO followed_channels (
				kick_channel_id, kick_chatroom_id, slug, display_name, profile_image_url,
				banner_image_url, is_enabled, raw_payload_json, created_at, updated_at,
				last_resolved_at, last_message_at, last_listener_error
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			args...,
		)
		if err != nil {
			return domain.FollowedChannel{}, fmt.Errorf("insert followed channel: %w", err)
		}
		channel.ID, _ = result.LastInsertId()
		return channel, nil
	}

	args = append(args, existing.ID)
	if _, err := repo.db.ExecContext(
		ctx,
		`UPDATE followed_channels
		 SET kick_channel_id = ?, kick_chatroom_id = ?, slug = ?, display_name = ?,
		     profile_image_url = ?, banner_image_url = ?, is_enabled = ?, raw_payload_json = ?,
		     created_at = ?, updated_at = ?, last_resolved_at = ?, last_message_at = ?,
		     last_listener_error = ?
		 WHERE id = ?`,
		args...,
	); err != nil {
		return domain.FollowedChannel{}, fmt.Errorf("update followed channel: %w", err)
	}
	return repo.GetBySlug(ctx, channel.Slug)
}

func (repo *FollowedChannelRepository) GetBySlug(ctx context.Context, slug string) (domain.FollowedChannel, error) {
	row := repo.db.QueryRowContext(
		ctx,
		`SELECT id, kick_channel_id, kick_chatroom_id, slug, display_name, profile_image_url,
		        banner_image_url, is_enabled, raw_payload_json, created_at, updated_at,
		        last_resolved_at, last_message_at, last_listener_error
		 FROM followed_channels
		 WHERE slug = ?`,
		normalizeSlug(slug),
	)
	return scanFollowedChannel(row)
}

func (repo *FollowedChannelRepository) ListEnabled(ctx context.Context) ([]domain.FollowedChannel, error) {
	rows, err := repo.db.QueryContext(
		ctx,
		`SELECT id, kick_channel_id, kick_chatroom_id, slug, display_name, profile_image_url,
		        banner_image_url, is_enabled, raw_payload_json, created_at, updated_at,
		        last_resolved_at, last_message_at, last_listener_error
		 FROM followed_channels
		 WHERE is_enabled = 1
		 ORDER BY slug ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled followed channels: %w", err)
	}
	defer rows.Close()

	channels := make([]domain.FollowedChannel, 0)
	for rows.Next() {
		channel, err := scanFollowedChannel(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled followed channels: %w", err)
	}
	return channels, nil
}

type followedChannelScanner interface {
	Scan(dest ...any) error
}

func scanFollowedChannel(scanner followedChannelScanner) (domain.FollowedChannel, error) {
	var channel domain.FollowedChannel
	var kickChannelID sql.NullInt64
	var kickChatroomID sql.NullInt64
	var isEnabled int
	var createdAt string
	var updatedAt string
	var lastResolvedAt string
	var lastMessageAt string
	if err := scanner.Scan(
		&channel.ID,
		&kickChannelID,
		&kickChatroomID,
		&channel.Slug,
		&channel.DisplayName,
		&channel.ProfileImageURL,
		&channel.BannerImageURL,
		&isEnabled,
		&channel.RawPayloadJSON,
		&createdAt,
		&updatedAt,
		&lastResolvedAt,
		&lastMessageAt,
		&channel.LastListenerError,
	); err != nil {
		return domain.FollowedChannel{}, err
	}
	channel.KickChannelID = scanInt64(kickChannelID)
	channel.KickChatroomID = scanInt64(kickChatroomID)
	channel.IsEnabled = intToBool(isEnabled)
	channel.CreatedAt = parseTime(createdAt)
	channel.UpdatedAt = parseTime(updatedAt)
	channel.LastResolvedAt = parseTime(lastResolvedAt)
	channel.LastMessageAt = parseTime(lastMessageAt)
	return channel, nil
}

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}
