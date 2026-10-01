package directory

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var ErrInvalidQuery = errors.New("invalid directory query")

const (
	DefaultLimit    = 50
	MaxLimit        = 100
	MaxPrefixLength = 160
)

type Service struct{ repository ports.DirectoryRepository }

func NewService(repository ports.DirectoryRepository) *Service {
	return &Service{repository: repository}
}

type cursor struct {
	Version int                  `json:"v"`
	Kind    domain.DirectoryKind `json:"kind"`
	Prefix  string               `json:"prefix"`
	Slug    string               `json:"slug"`
	ID      int64                `json:"id"`
}

func (service *Service) Search(ctx context.Context, kind domain.DirectoryKind, prefix string, limit int, after string) (domain.DirectoryPage, error) {
	if kind != domain.DirectoryUsers && kind != domain.DirectoryChannels {
		return domain.DirectoryPage{}, ErrInvalidQuery
	}
	if !validText(prefix) {
		return domain.DirectoryPage{}, ErrInvalidQuery
	}
	prefix = normalize(prefix)
	if length := utf8.RuneCountInString(prefix); length < 2 || length > MaxPrefixLength {
		return domain.DirectoryPage{}, ErrInvalidQuery
	}
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < 1 || limit > MaxLimit {
		return domain.DirectoryPage{}, ErrInvalidQuery
	}
	query := domain.DirectoryQuery{Prefix: prefix, Limit: limit + 1}
	if after != "" {
		position, err := decodeCursor(after, kind, prefix)
		if err != nil {
			return domain.DirectoryPage{}, err
		}
		query.After = position
	}
	items, err := service.repository.Search(ctx, kind, query)
	if err != nil {
		return domain.DirectoryPage{}, err
	}
	if items == nil {
		items = []domain.DirectoryIdentity{}
	}
	page := domain.DirectoryPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		encoded, err := json.Marshal(cursor{Version: 1, Kind: kind, Prefix: prefix, Slug: last.SortSlug, ID: last.ID})
		if err != nil {
			return domain.DirectoryPage{}, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}

func decodeCursor(raw string, kind domain.DirectoryKind, prefix string) (domain.DirectoryPosition, error) {
	if len(raw) > 4096 {
		return domain.DirectoryPosition{}, ErrInvalidQuery
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil {
		return domain.DirectoryPosition{}, ErrInvalidQuery
	}
	var value cursor
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return domain.DirectoryPosition{}, ErrInvalidQuery
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return domain.DirectoryPosition{}, ErrInvalidQuery
	}
	if value.Version != 1 || value.Kind != kind || value.Prefix != prefix || value.ID < 1 ||
		value.Slug == "" || utf8.RuneCountInString(value.Slug) > MaxPrefixLength || !validText(value.Slug) || normalize(value.Slug) != value.Slug {
		return domain.DirectoryPosition{}, ErrInvalidQuery
	}
	return domain.DirectoryPosition{Slug: value.Slug, ID: value.ID}, nil
}

func normalize(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "-")
}

func validText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
