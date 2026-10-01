package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
)

func directoryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "directory.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestDirectoryMigrationIndexesExistingMetadata(t *testing.T) {
	db := directoryTestDB(t)
	ctx := context.Background()
	// Simulate an upgrade with historical identities already present.
	for _, migration := range migrations.SQLiteMigrations() {
		if migration.Version >= 9 {
			break
		}
		for _, statement := range migration.Statements {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				t.Fatal(err)
			}
		}
	}
	_, err := NewSenderProfileRepository(db).Upsert(ctx, domain.SenderProfile{KickUserID: 1, Username: "Old_User", Slug: "old_user"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewFollowedChannelRepository(db).Upsert(ctx, domain.FollowedChannel{Slug: "old_channel", DisplayName: "Old_Channel", IsEnabled: false})
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations.SQLiteMigrations() {
		if migration.Version != 9 {
			continue
		}
		for _, statement := range migration.Statements {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, kind := range []domain.DirectoryKind{domain.DirectoryUsers, domain.DirectoryChannels} {
		items, err := NewDirectoryRepository(db).Search(ctx, kind, domain.DirectoryQuery{Prefix: "old-", Limit: 51})
		if err != nil || len(items) != 1 {
			t.Fatalf("upgraded %s: %v, %v", kind, items, err)
		}
	}
}

func TestDirectoryQueriesUseBothPrefixIndexes(t *testing.T) {
	db := directoryTestDB(t)
	if err := migrations.ApplySQLite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind  domain.DirectoryKind
		table string
	}{
		{domain.DirectoryUsers, "sender_profiles"}, {domain.DirectoryChannels, "followed_channels"},
	} {
		for _, after := range []domain.DirectoryPosition{{}, {Slug: "yavu-a", ID: 20}} {
			statement, args, err := directoryStatement(tc.kind, domain.DirectoryQuery{Prefix: "yavu", Limit: 51, After: after})
			if err != nil {
				t.Fatal(err)
			}
			rows, err := db.Query("EXPLAIN QUERY PLAN "+statement, args...)
			if err != nil {
				t.Fatal(err)
			}
			var plan strings.Builder
			for rows.Next() {
				var id, parent, unused int
				var detail string
				if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
					t.Fatal(err)
				}
				plan.WriteString(detail + "\n")
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
			for _, column := range []string{"name", "slug"} {
				want := "SEARCH " + tc.table + " USING INDEX idx_" + tc.table + "_directory_" + column
				if !strings.Contains(strings.ReplaceAll(plan.String(), "COVERING INDEX", "INDEX"), want) {
					t.Fatalf("missing indexed prefix lookup %q:\n%s", want, plan.String())
				}
			}
			if strings.Contains(plan.String(), "SCAN "+tc.table) || strings.Contains(plan.String(), "SCAN identity") {
				t.Fatalf("full identity-table scan:\n%s", plan.String())
			}
		}
	}
}

func TestDirectoryRejectsUnsafeRepositoryQueries(t *testing.T) {
	db := directoryTestDB(t)
	repo := NewDirectoryRepository(db)
	for _, tc := range []struct {
		kind  domain.DirectoryKind
		query domain.DirectoryQuery
	}{
		{"users; DROP TABLE sender_profiles", domain.DirectoryQuery{Prefix: "ab", Limit: 50}},
		{domain.DirectoryUsers, domain.DirectoryQuery{Prefix: "ab", Limit: 0}},
		{domain.DirectoryUsers, domain.DirectoryQuery{Prefix: "ab", Limit: 102}},
		{domain.DirectoryUsers, domain.DirectoryQuery{Limit: 50}},
	} {
		if _, err := repo.Search(context.Background(), tc.kind, tc.query); err == nil {
			t.Fatal("invalid query accepted")
		}
	}
}
