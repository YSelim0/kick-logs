package httpapi

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	ratelimitinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/ratelimit"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/directory"
)

type directoryTestPage struct {
	Items []struct {
		ID              int64   `json:"id"`
		Name            string  `json:"name"`
		Slug            string  `json:"slug"`
		ProfileImageURL *string `json:"profile_image_url"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func newDirectoryTestRouter(t *testing.T, rateLimited bool) (http.Handler, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "directory.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{RateLimitEnabled: rateLimited}
	deps := routes.Dependencies{Directory: directory.NewService(sqlite.NewDirectoryRepository(db))}
	if rateLimited {
		deps.RateLimiter, err = ratelimitinfra.NewGCRA(1000)
		if err != nil {
			t.Fatal(err)
		}
	}
	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), deps), db
}

func seedDirectoryIdentity(t *testing.T, db *sql.DB, kind, slug, name string, n int) {
	t.Helper()
	var err error
	if kind == "users" {
		_, err = sqlite.NewSenderProfileRepository(db).Upsert(context.Background(), domain.SenderProfile{
			KickUserID: int64(n), Slug: slug, Username: name, ProfileImageURL: "https://example.com/avatar.png",
		})
	} else {
		_, err = sqlite.NewFollowedChannelRepository(db).Upsert(context.Background(), domain.FollowedChannel{
			Slug: slug, DisplayName: name, IsEnabled: false, ProfileImageURL: "https://example.com/avatar.png",
		})
	}
	if err != nil {
		t.Fatal(err)
	}
}

func directoryRequest(t *testing.T, router http.Handler, path string, status int) directoryTestPage {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != status {
		t.Fatalf("GET %s: status %d, want %d: %s", path, w.Code, status, w.Body.String())
	}
	var page directoryTestPage
	if status == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.Items == nil {
			t.Fatal("items must be an array, not null")
		}
		for _, forbidden := range []string{"message_count", "latest_message_at", "first_message_at", "total", "count"} {
			if strings.Contains(w.Body.String(), `"`+forbidden+`"`) {
				t.Fatalf("analytics field in directory: %s", w.Body.String())
			}
		}
	}
	return page
}

func TestDirectoryPrefixCaseWildcardsAndHistoricalIdentities(t *testing.T) {
	for _, kind := range []string{"users", "channels"} {
		t.Run(kind, func(t *testing.T) {
			router, db := newDirectoryTestRouter(t, false)
			for i, entry := range [][2]string{
				{"yavuz", "YaVuZ"}, {"notyavuz", "notYavuz"}, {"yavu-two", "Yavu_Two"},
				{"other", "YavuAlias"}, {"ya%literal", "Ya%Literal"}, {"yaXliteral", "YaXLiteral"},
				{"ya_literal", "Ya_Literal"}, {"ya-literal-two", "Ya-Literal-Two"},
				{`ya\literal`, `Ya\Literal`}, {"inject", "ya' OR 1=1 --"},
			} {
				seedDirectoryIdentity(t, db, kind, entry[0], entry[1], i+1)
			}
			for _, tc := range []struct {
				prefix string
				want   []string
			}{
				{"  YaVu  ", []string{"other", "yavu-two", "yavuz"}},
				{"ya%", []string{"ya%literal"}},
				{"ya_", []string{"ya_literal", "ya-literal-two"}},
				{"ya-", []string{"ya_literal", "ya-literal-two"}},
				{`ya\`, []string{`ya\literal`}},
				{"ya' OR 1=1 --", []string{"inject"}},
				{"zz", []string{}},
			} {
				page := directoryRequest(t, router, "/directory/"+kind+"?prefix="+url.QueryEscape(tc.prefix), http.StatusOK)
				got := []string{}
				for _, item := range page.Items {
					got = append(got, item.Slug)
					if item.Name == "" || item.ProfileImageURL == nil {
						t.Fatalf("lost metadata: %+v", item)
					}
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("prefix %q: got %v, want %v", tc.prefix, got, tc.want)
				}
				if page.NextCursor != nil {
					t.Fatal("unexpected next page")
				}
			}
		})
	}
}

func TestDirectoryKeysetPagesAndCursorScope(t *testing.T) {
	for _, kind := range []string{"users", "channels"} {
		t.Run(kind, func(t *testing.T) {
			router, db := newDirectoryTestRouter(t, false)
			for i, slug := range []string{"test_b", "test-b", "test-a", "test-c"} {
				seedDirectoryIdentity(t, db, kind, slug, slug, i+1)
			}
			base := "/directory/" + kind + "?prefix=test&limit=2"
			first := directoryRequest(t, router, base, 200)
			if len(first.Items) != 2 || first.NextCursor == nil || first.Items[0].Slug != "test-a" || first.Items[1].Slug != "test_b" {
				t.Fatalf("first page: %+v", first)
			}
			seedDirectoryIdentity(t, db, kind, "test-0", "test-0", 5)
			second := directoryRequest(t, router, base+"&after="+url.QueryEscape(*first.NextCursor), 200)
			if len(second.Items) != 2 || second.NextCursor != nil || second.Items[0].Slug != "test-b" || second.Items[1].Slug != "test-c" {
				t.Fatalf("second page: %+v", second)
			}
			withCursor := directoryRequest(t, router, base+"&cursor="+url.QueryEscape(*first.NextCursor), 200)
			if !reflect.DeepEqual(second, withCursor) {
				t.Fatal("cursor alias differs from after")
			}
			directoryRequest(t, router, "/directory/"+kind+"?prefix=other&after="+url.QueryEscape(*first.NextCursor), 422)
			otherKind := "users"
			if kind == "users" {
				otherKind = "channels"
			}
			directoryRequest(t, router, "/directory/"+otherKind+"?prefix=test&after="+url.QueryEscape(*first.NextCursor), 422)
		})
	}
}

func TestDirectoryBoundedDefaultAndMaximumPages(t *testing.T) {
	router, db := newDirectoryTestRouter(t, false)
	for i := 1; i <= 101; i++ {
		seedDirectoryIdentity(t, db, "users", fmt.Sprintf("page-%03d", i), "Page", i)
	}
	for _, tc := range []struct {
		query string
		size  int
	}{{"", 50}, {"&limit=100", 100}, {"&limit=1", 1}} {
		page := directoryRequest(t, router, "/directory/users?prefix=page"+tc.query, 200)
		if len(page.Items) != tc.size || page.NextCursor == nil {
			t.Fatalf("page: got %d items, cursor %v", len(page.Items), page.NextCursor)
		}
	}
}

func TestDirectoryRejectsMalformedParameters(t *testing.T) {
	router, _ := newDirectoryTestRouter(t, false)
	for _, query := range []string{
		"", "prefix=", "prefix=a", "prefix=%20%20", "prefix=ab%00", "prefix=ab%0A", "prefix=%FF%FF",
		"prefix=" + strings.Repeat("a", 161), "prefix=ab&limit=0", "prefix=ab&limit=-1", "prefix=ab&limit=101",
		"prefix=ab&limit=1.5", "prefix=ab&limit=no", "prefix=ab&limit=", "prefix=ab&limit=999999999999999999999",
		"prefix=ab&after=bad!", "prefix=ab&cursor=bad", "prefix=ab&after=" + strings.Repeat("x", 4097),
		"prefix=ab&after=" + base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`)),
		"prefix=ab&after=x&cursor=y", "prefix=ab&prefix=cd", "prefix=ab&limit=2&limit=3", "prefix=ab&offset=3", "prefix=%ZZ",
	} {
		directoryRequest(t, router, "/directory/users?"+query, 422)
	}
}

func TestDirectoryDatabaseErrorIsGeneric(t *testing.T) {
	router, db := newDirectoryTestRouter(t, false)
	_ = db.Close()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/directory/users?prefix=test", nil))
	if w.Code != 500 || w.Body.String() != "{\"detail\":\"Internal server error.\"}\n" {
		t.Fatalf("response: %d %s", w.Code, w.Body.String())
	}
}

func TestDirectoryRateLimitSharesOnlyDirectoryBucket(t *testing.T) {
	router, _ := newDirectoryTestRouter(t, true)
	for i := 0; i < 16; i++ {
		kind := "users"
		if i%2 == 1 {
			kind = "channels"
		}
		directoryRequest(t, router, "/directory/"+kind+"?prefix=test", 200)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/directory/users?prefix=test", nil))
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limit: %d %s", w.Code, w.Body.String())
	}
	directoryRequest(t, router, "/analytics/top-senders?q=test", 500)
	request := httptest.NewRequest(http.MethodGet, "/directory/users?prefix=test", nil)
	request.RemoteAddr = "10.0.0.2:9000"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, request)
	if w.Code != 200 {
		t.Fatalf("independent IP: %d", w.Code)
	}
}
