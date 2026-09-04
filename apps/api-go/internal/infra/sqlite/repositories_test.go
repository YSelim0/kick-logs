package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
)

func TestSQLiteMigrationsAreIdempotent(t *testing.T) {
	ctx := context.Background()
	db, path := openMigratedSQLite(t, ctx)
	defer db.Close()

	if err := migrations.ApplySQLite(ctx, db); err != nil {
		t.Fatalf("ApplySQLite() second run error = %v", err)
	}

	stats := sqlite.NewStatsRepository(db, path)
	sizes, err := stats.TableSizes(ctx)
	if err != nil {
		t.Fatalf("TableSizes() error = %v", err)
	}
	if len(sizes) == 0 {
		t.Fatal("TableSizes() returned no tables")
	}
}

func TestAdminUserRepositoryAndSeed(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewAdminUserRepository(db)
	seeded, err := sqlite.SeedSuperAdmin(ctx, repo, "ADMIN@kicklogs.local", "admin123")
	if err != nil {
		t.Fatalf("SeedSuperAdmin() error = %v", err)
	}
	if seeded.Email != "admin@kicklogs.local" {
		t.Fatalf("seeded email = %q", seeded.Email)
	}
	if seeded.Role != domain.AdminRoleSuperAdmin {
		t.Fatalf("seeded role = %q", seeded.Role)
	}
	if seeded.PasswordHash == "admin123" {
		t.Fatal("seeded password hash stored plain password")
	}

	fetched, err := repo.GetByEmail(ctx, "admin@kicklogs.local")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if fetched.ID == 0 || !fetched.IsActive {
		t.Fatalf("fetched admin = %#v", fetched)
	}

	active, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("active users len = %d", len(active))
	}
}

func TestFollowedChannelRepository(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewFollowedChannelRepository(db)
	inserted, err := repo.Upsert(ctx, domain.FollowedChannel{
		KickChannelID:  101,
		KickChatroomID: 202,
		Slug:           "Hype",
		DisplayName:    "Hype",
		IsEnabled:      true,
		RawPayloadJSON: `{"slug":"hype"}`,
		LastResolvedAt: time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if inserted.ID == 0 {
		t.Fatal("inserted channel ID = 0")
	}

	fetched, err := repo.GetBySlug(ctx, "hype")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if fetched.KickChatroomID != 202 || !fetched.IsEnabled {
		t.Fatalf("fetched channel = %#v", fetched)
	}

	enabled, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("enabled channels len = %d", len(enabled))
	}

	byChatroom, err := repo.GetByChatroomID(ctx, 202)
	if err != nil {
		t.Fatalf("GetByChatroomID() error = %v", err)
	}
	if byChatroom.Slug != "hype" {
		t.Fatalf("byChatroom = %#v", byChatroom)
	}
}

func TestSenderProfileAndHeartbeatRepositories(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	senders := sqlite.NewSenderProfileRepository(db)
	sender, err := senders.Upsert(ctx, domain.SenderProfile{
		KickUserID:            456,
		Username:              "Yavuz_User",
		Slug:                  "yavuz-user",
		ProfileImageURL:       "https://example.com/avatar.png",
		LastSeenColor:         "#fff600",
		RawProfilePayloadJSON: `{"id":456}`,
		LastSeenAt:            time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("sender Upsert() error = %v", err)
	}
	if sender.ID == 0 {
		t.Fatal("sender ID = 0")
	}
	fetched, err := senders.GetByKickUserID(ctx, 456)
	if err != nil {
		t.Fatalf("GetByKickUserID() error = %v", err)
	}
	if fetched.ProfileImageURL == "" || fetched.LastSeenColor != "#fff600" {
		t.Fatalf("fetched sender = %#v", fetched)
	}

	heartbeats := sqlite.NewWorkerHeartbeatRepository(db)
	if err := heartbeats.Upsert(ctx, domain.ListenerHeartbeat{
		ServiceName:  "listener",
		LastSeenAt:   time.Date(2026, 5, 16, 13, 0, 0, 0, time.UTC),
		MetadataJSON: `{"worker_count":4}`,
	}); err != nil {
		t.Fatalf("heartbeat Upsert() error = %v", err)
	}
	var metadata string
	if err := db.QueryRowContext(ctx, "SELECT metadata_json FROM worker_heartbeats WHERE service_name = 'listener'").Scan(&metadata); err != nil {
		t.Fatalf("read heartbeat: %v", err)
	}
	if metadata != `{"worker_count":4}` {
		t.Fatalf("metadata = %q", metadata)
	}
}

func TestRawEventClaimRepository(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventClaimRepository(db)
	claimed, err := repo.TryClaim(ctx, "raw-1", "worker-1", time.Minute)
	if err != nil {
		t.Fatalf("TryClaim() error = %v", err)
	}
	if !claimed {
		t.Fatal("first TryClaim() returned false")
	}

	claimed, err = repo.TryClaim(ctx, "raw-1", "worker-2", time.Minute)
	if err != nil {
		t.Fatalf("TryClaim(active) error = %v", err)
	}
	if claimed {
		t.Fatal("active claim was acquired by another worker")
	}

	if err := repo.Release(ctx, "raw-1", "worker-1"); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	claimed, err = repo.TryClaim(ctx, "raw-1", "worker-2", time.Minute)
	if err != nil {
		t.Fatalf("TryClaim(released) error = %v", err)
	}
	if !claimed {
		t.Fatal("released claim was not acquired")
	}

	if err := repo.MarkCompleted(ctx, "raw-1", "worker-2"); err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}
	claimed, err = repo.TryClaim(ctx, "raw-1", "worker-3", time.Minute)
	if err != nil {
		t.Fatalf("TryClaim(completed) error = %v", err)
	}
	if claimed {
		t.Fatal("completed claim was acquired")
	}

	claimed, err = repo.TryClaim(ctx, "raw-expired", "worker-1", time.Minute)
	if err != nil {
		t.Fatalf("TryClaim(expired setup) error = %v", err)
	}
	if !claimed {
		t.Fatal("expired setup claim returned false")
	}
	if _, err := db.ExecContext(
		ctx,
		"UPDATE raw_event_claims SET lease_expires_at = ? WHERE raw_event_id = ?",
		time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano),
		"raw-expired",
	); err != nil {
		t.Fatalf("expire claim: %v", err)
	}
	claimed, err = repo.TryClaim(ctx, "raw-expired", "worker-2", time.Minute)
	if err != nil {
		t.Fatalf("TryClaim(expired takeover) error = %v", err)
	}
	if !claimed {
		t.Fatal("expired claim was not acquired")
	}

	var status string
	var workerID string
	if err := db.QueryRowContext(
		ctx,
		"SELECT status, worker_id FROM raw_event_claims WHERE raw_event_id = ?",
		"raw-expired",
	).Scan(&status, &workerID); err != nil {
		t.Fatalf("read raw event claim: %v", err)
	}
	if status != "claimed" || workerID != "worker-2" {
		t.Fatalf("claim row status=%q worker=%q", status, workerID)
	}
}

func TestSenderProfileRepositoryUpsertHandlesNaturalKeyConflicts(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	senders := sqlite.NewSenderProfileRepository(db)
	first, err := senders.Upsert(ctx, domain.SenderProfile{
		KickUserID:            9001,
		Username:              "FirstName",
		Slug:                  "shared-user",
		RawProfilePayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	updatedByKickID, err := senders.Upsert(ctx, domain.SenderProfile{
		KickUserID:            9001,
		Username:              "UpdatedName",
		Slug:                  "shared-user",
		RawProfilePayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	if updatedByKickID.ID != first.ID || updatedByKickID.Username != "UpdatedName" {
		t.Fatalf("updatedByKickID = %#v, first = %#v", updatedByKickID, first)
	}

	updatedBySlug, err := senders.Upsert(ctx, domain.SenderProfile{
		KickUserID:            9002,
		Username:              "SlugOwner",
		Slug:                  "shared-user",
		RawProfilePayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("slug-conflict Upsert() error = %v", err)
	}
	if updatedBySlug.ID != first.ID || updatedBySlug.KickUserID != 9002 {
		t.Fatalf("updatedBySlug = %#v, first = %#v", updatedBySlug, first)
	}
}

func TestKickWebhookEventRepository(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewKickWebhookEventRepository(db)

	event := domain.KickWebhookEvent{
		MessageID:      "msg-001",
		SubscriptionID: "sub-001",
		EventType:      "channel.subscription.new",
		EventVersion:   "v1",
		RawPayloadJSON: `{"test":true}`,
		ReceivedAt:     time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	if err := repo.InsertIdempotent(ctx, event); err != nil {
		t.Fatalf("InsertIdempotent() error = %v", err)
	}

	if err := repo.InsertIdempotent(ctx, event); err != nil {
		t.Fatalf("InsertIdempotent() duplicate error = %v", err)
	}

	fetched, err := repo.GetByMessageID(ctx, "msg-001")
	if err != nil {
		t.Fatalf("GetByMessageID() error = %v", err)
	}
	if fetched.Status != domain.WebhookEventStatusPending || fetched.Attempts != 0 {
		t.Fatalf("fetched = %#v", fetched)
	}

	pending, err := repo.ListPending(ctx, 10, 5)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending len = %d", len(pending))
	}

	if err := repo.MarkFailed(ctx, "msg-001", "parse error", 5); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	after1, _ := repo.GetByMessageID(ctx, "msg-001")
	if after1.Attempts != 1 || after1.Status != domain.WebhookEventStatusPending {
		t.Fatalf("after 1 failure: attempts=%d status=%s", after1.Attempts, after1.Status)
	}

	for i := 0; i < 4; i++ {
		if err := repo.MarkFailed(ctx, "msg-001", "parse error", 5); err != nil {
			t.Fatalf("MarkFailed() iteration %d error = %v", i, err)
		}
	}
	exhausted, _ := repo.GetByMessageID(ctx, "msg-001")
	if exhausted.Status != domain.WebhookEventStatusFailed {
		t.Fatalf("exhausted status = %s, want failed", exhausted.Status)
	}

	event2 := domain.KickWebhookEvent{
		MessageID:  "msg-002",
		EventType:  "channel.subscription.renewal",
		ReceivedAt: time.Now().UTC(),
	}
	_ = repo.InsertIdempotent(ctx, event2)

	if err := repo.MarkProcessed(ctx, "msg-002"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	processed, _ := repo.GetByMessageID(ctx, "msg-002")
	if processed.Status != domain.WebhookEventStatusProcessed {
		t.Fatalf("processed status = %s", processed.Status)
	}

	event3 := domain.KickWebhookEvent{
		MessageID:  "msg-003",
		EventType:  "channel.subscription.gifts",
		ReceivedAt: time.Now().UTC(),
	}
	_ = repo.InsertIdempotent(ctx, event3)

	if err := repo.MarkIgnored(ctx, "msg-003", "unsupported event type"); err != nil {
		t.Fatalf("MarkIgnored() error = %v", err)
	}
	ignored, _ := repo.GetByMessageID(ctx, "msg-003")
	if ignored.Status != domain.WebhookEventStatusIgnored {
		t.Fatalf("ignored status = %s", ignored.Status)
	}

	counts, err := repo.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}
	if counts[domain.WebhookEventStatusFailed] != 1 || counts[domain.WebhookEventStatusProcessed] != 1 || counts[domain.WebhookEventStatusIgnored] != 1 {
		t.Fatalf("counts = %v", counts)
	}

	latest, err := repo.LatestReceivedAt(ctx)
	if err != nil {
		t.Fatalf("LatestReceivedAt() error = %v", err)
	}
	if latest.IsZero() {
		t.Fatal("LatestReceivedAt() returned zero time")
	}

	old := time.Now().UTC().Add(-8 * 24 * time.Hour)
	if _, err := db.ExecContext(
		ctx,
		`UPDATE kick_webhook_events SET processed_at = ? WHERE message_id IN ('msg-002', 'msg-003')`,
		old.Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("backdate terminal webhook events: %v", err)
	}
	pruned, err := repo.PruneTerminalBefore(ctx, time.Now().UTC().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("PruneTerminalBefore() error = %v", err)
	}
	if pruned != 2 {
		t.Fatalf("pruned = %d, want 2", pruned)
	}
	if _, err := repo.GetByMessageID(ctx, "msg-002"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("processed event after prune err = %v, want sql.ErrNoRows", err)
	}
	if _, err := repo.GetByMessageID(ctx, "msg-003"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ignored event after prune err = %v, want sql.ErrNoRows", err)
	}
	if _, err := repo.GetByMessageID(ctx, "msg-001"); err != nil {
		t.Fatalf("failed event should remain after prune: %v", err)
	}
}

func TestKickEventSubscriptionRepository(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	channelRepo := sqlite.NewFollowedChannelRepository(db)
	ch, err := channelRepo.Upsert(ctx, domain.FollowedChannel{
		Slug:              "test-channel",
		DisplayName:       "Test Channel",
		BroadcasterUserID: 9999,
		IsEnabled:         true,
		RawPayloadJSON:    "{}",
	})
	if err != nil {
		t.Fatalf("channel Upsert() error = %v", err)
	}

	subRepo := sqlite.NewKickEventSubscriptionRepository(db)

	sub := domain.KickEventSubscription{
		FollowedChannelID:  ch.ID,
		BroadcasterUserID:  9999,
		EventType:          "channel.subscription.new",
		EventVersion:       "v1",
		Method:             "webhook",
		KickSubscriptionID: "kick-sub-001",
		Status:             domain.KickEventSubStatusActive,
	}

	inserted, err := subRepo.Upsert(ctx, sub)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if inserted.ID == 0 {
		t.Fatal("inserted ID = 0")
	}

	sub.KickSubscriptionID = "kick-sub-001-updated"
	updated, err := subRepo.Upsert(ctx, sub)
	if err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}
	if updated.ID != inserted.ID {
		t.Fatalf("upsert changed ID: got %d want %d", updated.ID, inserted.ID)
	}
	if updated.KickSubscriptionID != "kick-sub-001-updated" {
		t.Fatalf("KickSubscriptionID not updated: %s", updated.KickSubscriptionID)
	}

	sub2 := domain.KickEventSubscription{
		FollowedChannelID: ch.ID,
		BroadcasterUserID: 9999,
		EventType:         "channel.subscription.renewal",
		EventVersion:      "v1",
		Method:            "webhook",
		Status:            domain.KickEventSubStatusActive,
	}
	_, _ = subRepo.Upsert(ctx, sub2)

	byChannel, err := subRepo.ListByChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("ListByChannel() error = %v", err)
	}
	if len(byChannel) != 2 {
		t.Fatalf("ListByChannel() len = %d, want 2", len(byChannel))
	}

	if err := subRepo.UpdateSyncError(ctx, inserted.ID, "Kick API timeout"); err != nil {
		t.Fatalf("UpdateSyncError() error = %v", err)
	}
	afterError, _ := subRepo.ListByChannel(ctx, ch.ID)
	for _, s := range afterError {
		if s.ID == inserted.ID && s.Status != domain.KickEventSubStatusError {
			t.Fatalf("status after sync error = %s, want error", s.Status)
		}
	}

	if err := subRepo.DeleteByChannel(ctx, ch.ID); err != nil {
		t.Fatalf("DeleteByChannel() error = %v", err)
	}
	afterDelete, _ := subRepo.ListByChannel(ctx, ch.ID)
	for _, s := range afterDelete {
		if s.Status != domain.KickEventSubStatusDeleted {
			t.Fatalf("status after delete = %s, want deleted", s.Status)
		}
	}
}

func TestFollowedChannelBroadcasterUserID(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewFollowedChannelRepository(db)

	ch, err := repo.Upsert(ctx, domain.FollowedChannel{
		Slug:              "broadcaster-test",
		DisplayName:       "Broadcaster Test",
		BroadcasterUserID: 42000,
		IsEnabled:         true,
		RawPayloadJSON:    "{}",
	})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if ch.BroadcasterUserID != 42000 {
		t.Fatalf("BroadcasterUserID = %d, want 42000", ch.BroadcasterUserID)
	}

	fetched, err := repo.GetByBroadcasterUserID(ctx, 42000)
	if err != nil {
		t.Fatalf("GetByBroadcasterUserID() error = %v", err)
	}
	if fetched.Slug != "broadcaster-test" {
		t.Fatalf("fetched slug = %s", fetched.Slug)
	}
}

func TestWatchedSenderRepository(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewWatchedSenderRepository(db)

	created, err := repo.Create(ctx, "Nuriben")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == 0 || created.Username != "Nuriben" {
		t.Fatalf("created sender = %+v", created)
	}

	if _, err := repo.Create(ctx, "nuriben"); err == nil {
		t.Fatal("expected case-insensitive duplicate insert to fail")
	}

	if _, err := repo.Create(ctx, "otheruser"); err != nil {
		t.Fatalf("Create() second sender error = %v", err)
	}

	senders, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(senders) != 2 {
		t.Fatalf("List() len = %d, want 2", len(senders))
	}

	usernames, err := repo.ListUsernames(ctx)
	if err != nil {
		t.Fatalf("ListUsernames() error = %v", err)
	}
	if len(usernames) != 2 {
		t.Fatalf("ListUsernames() len = %d, want 2", len(usernames))
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := repo.Delete(ctx, created.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Delete() second call error = %v, want sql.ErrNoRows", err)
	}

	remaining, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(remaining) != 1 || remaining[0].Username != "otheruser" {
		t.Fatalf("remaining senders = %+v", remaining)
	}
}

func TestNotificationSettingsRepository(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewNotificationSettingsRepository(db, 600)

	seeded, err := repo.GetNotificationSettings(ctx)
	if err != nil {
		t.Fatalf("GetNotificationSettings() error = %v", err)
	}
	if seeded.CooldownSeconds != 600 {
		t.Fatalf("seeded CooldownSeconds = %d, want 600", seeded.CooldownSeconds)
	}

	updated, err := repo.UpdateNotificationSettings(ctx, domain.NotificationSettings{CooldownSeconds: 120})
	if err != nil {
		t.Fatalf("UpdateNotificationSettings() error = %v", err)
	}
	if updated.CooldownSeconds != 120 {
		t.Fatalf("updated CooldownSeconds = %d, want 120", updated.CooldownSeconds)
	}

	reread, err := repo.GetNotificationSettings(ctx)
	if err != nil {
		t.Fatalf("GetNotificationSettings() after update error = %v", err)
	}
	if reread.CooldownSeconds != 120 {
		t.Fatalf("reread CooldownSeconds = %d, want 120", reread.CooldownSeconds)
	}
}

func openMigratedSQLite(t *testing.T, ctx context.Context) (*sql.DB, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "kick-logs.sqlite3")
	db, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		db.Close()
		t.Fatalf("ApplySQLite() error = %v", err)
	}
	return db, path
}
