package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type DirectoryRepository struct{ db *sql.DB }

var _ ports.DirectoryRepository = (*DirectoryRepository)(nil)

func NewDirectoryRepository(db *sql.DB) *DirectoryRepository {
	return &DirectoryRepository{db: db}
}

func (repo *DirectoryRepository) Search(ctx context.Context, kind domain.DirectoryKind, query domain.DirectoryQuery) ([]domain.DirectoryIdentity, error) {
	statement, args, err := directoryStatement(kind, query)
	if err != nil {
		return nil, err
	}
	rows, err := repo.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("search directory: %w", err)
	}
	defer rows.Close()
	items := make([]domain.DirectoryIdentity, 0, query.Limit)
	for rows.Next() {
		var item domain.DirectoryIdentity
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.ProfileImageURL, &item.SortSlug); err != nil {
			return nil, fmt.Errorf("scan directory identity: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate directory: %w", err)
	}
	return items, nil
}

func directoryStatement(kind domain.DirectoryKind, query domain.DirectoryQuery) (string, []any, error) {
	var table, name string
	switch kind {
	case domain.DirectoryUsers:
		table, name = "sender_profiles", "username"
	case domain.DirectoryChannels:
		table, name = "followed_channels", "display_name"
	default:
		return "", nil, fmt.Errorf("invalid directory kind")
	}
	if query.Limit < 1 || query.Limit > 101 || query.Prefix == "" {
		return "", nil, fmt.Errorf("invalid directory page")
	}
	normalizedName := "lower(replace(trim(" + name + "), '_', '-'))"
	normalizedSlug := "lower(replace(trim(slug), '_', '-'))"
	pattern := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(query.Prefix) + "%"
	upper := directoryPrefixUpperBound(query.Prefix)
	// Separate indexed ranges prevent ORDER BY from choosing a full metadata-table
	// scan. UNION deduplicates identities matching both their name and their slug.
	statement := fmt.Sprintf(`WITH matches AS (
		SELECT id FROM %s WHERE %s >= ? AND %s < ? AND %s LIKE ? ESCAPE '\'
		UNION
		SELECT id FROM %s WHERE %s >= ? AND %s < ? AND %s LIKE ? ESCAPE '\'
	)
	SELECT identity.id, %s, slug, profile_image_url, %s AS sort_slug
	FROM matches JOIN %s AS identity ON identity.id = matches.id
	WHERE (%s, identity.id) > (?, ?)
	ORDER BY sort_slug ASC, identity.id ASC LIMIT ?`,
		table, normalizedName, normalizedName, normalizedName,
		table, normalizedSlug, normalizedSlug, normalizedSlug,
		name, normalizedSlug, table, normalizedSlug)
	return statement, []any{query.Prefix, upper, pattern, query.Prefix, upper, pattern, query.After.Slug, query.After.ID, query.Limit}, nil
}

// SQLite BINARY order compares UTF-8 bytes; the exclusive successor bounds all
// strings beginning with prefix, including literal percent and underscore input.
func directoryPrefixUpperBound(prefix string) string {
	upper := []byte(prefix)
	upper[len(upper)-1]++
	return string(upper)
}
