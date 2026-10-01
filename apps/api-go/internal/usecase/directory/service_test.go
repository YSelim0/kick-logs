package directory

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type recordingRepository struct {
	query domain.DirectoryQuery
	calls int
	items []domain.DirectoryIdentity
	err   error
}

func (r *recordingRepository) Search(_ context.Context, _ domain.DirectoryKind, query domain.DirectoryQuery) ([]domain.DirectoryIdentity, error) {
	r.calls++
	r.query = query
	return r.items, r.err
}

func TestSearchBoundsLookaheadAndNormalizesPrefix(t *testing.T) {
	repo := &recordingRepository{}
	page, err := NewService(repo).Search(context.Background(), domain.DirectoryUsers, "  Te_ST ", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if repo.query.Prefix != "te-st" || repo.query.Limit != 51 || page.Items == nil || page.NextCursor != "" {
		t.Fatalf("query: %+v, page: %+v", repo.query, page)
	}
	_, err = NewService(repo).Search(context.Background(), domain.DirectoryChannels, "test", 100, "")
	if err != nil || repo.query.Limit != 101 {
		t.Fatalf("lookahead: %+v, %v", repo.query, err)
	}
}

func TestSearchRejectsInvalidInputBeforeRepository(t *testing.T) {
	repo := &recordingRepository{}
	service := NewService(repo)
	for _, prefix := range []string{"", " ", "a", "ab\x00", "ab\n", "\xff\xff", strings.Repeat("x", 161)} {
		if _, err := service.Search(context.Background(), domain.DirectoryUsers, prefix, 50, ""); !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("prefix %q: %v", prefix, err)
		}
	}
	for _, limit := range []int{-1, 101} {
		if _, err := service.Search(context.Background(), domain.DirectoryUsers, "ab", limit, ""); !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	for _, raw := range []string{
		`null`, `{}`, `{"v":2,"kind":"users","prefix":"ab","slug":"ab","id":1}`,
		`{"v":1,"kind":"users","prefix":"ab","slug":"ab","id":-1}`,
		`{"v":1,"kind":"users","prefix":"ab","slug":"AB","id":1}`,
		`{"v":1,"kind":"users","prefix":"ab","slug":"ab","id":1,"extra":1}`,
		`{"v":1,"kind":"users","prefix":"ab","slug":"ab","id":1}{}`,
	} {
		after := base64.RawURLEncoding.EncodeToString([]byte(raw))
		if _, err := service.Search(context.Background(), domain.DirectoryUsers, "ab", 50, after); !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("cursor %q: %v", raw, err)
		}
	}
	if _, err := service.Search(context.Background(), "unknown", "ab", 50, ""); !errors.Is(err, ErrInvalidQuery) {
		t.Fatal(err)
	}
	if repo.calls != 0 {
		t.Fatalf("invalid input reached repository %d times", repo.calls)
	}
}

func TestSearchPropagatesRepositoryErrors(t *testing.T) {
	failure := errors.New("unavailable")
	_, err := NewService(&recordingRepository{err: failure}).Search(context.Background(), domain.DirectoryUsers, "ab", 50, "")
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
