package kicksync_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/kicksync"
)

func TestSyncAllResolvesBroadcasterUserID(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, ctx)
	defer db.Close()

	channelRepo := sqliteinfra.NewFollowedChannelRepository(db)
	subRepo := sqliteinfra.NewKickEventSubscriptionRepository(db)

	ch, err := channelRepo.Upsert(ctx, domain.FollowedChannel{
		Slug:           "test-channel",
		DisplayName:    "Test Channel",
		IsEnabled:      true,
		RawPayloadJSON: "{}",
	})
	if err != nil {
		t.Fatalf("Upsert channel: %v", err)
	}
	if ch.BroadcasterUserID != 0 {
		t.Fatal("expected BroadcasterUserID == 0 before sync")
	}

	client := &fakeKickClient{broadcasterUserID: 5555, subIDPrefix: "sub"}
	svc := kicksync.NewService(discardLogger(), channelRepo, subRepo, client, []string{"channel.subscription.new"})

	svc.SyncAll(ctx)

	updated, err := channelRepo.GetBySlug(ctx, "test-channel")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if updated.BroadcasterUserID != 5555 {
		t.Fatalf("BroadcasterUserID = %d, want 5555", updated.BroadcasterUserID)
	}

	subs, err := subRepo.ListByChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("ListByChannel: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("subscriptions len = %d, want 1", len(subs))
	}
	if subs[0].KickSubscriptionID == "" || subs[0].Status != domain.KickEventSubStatusActive {
		t.Fatalf("subscription = %+v", subs[0])
	}
}

func TestEnsureChannelSubscriptionsCreatesAll(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, ctx)
	defer db.Close()

	channelRepo := sqliteinfra.NewFollowedChannelRepository(db)
	subRepo := sqliteinfra.NewKickEventSubscriptionRepository(db)

	ch, _ := channelRepo.Upsert(ctx, domain.FollowedChannel{
		Slug:              "chan2",
		DisplayName:       "Chan2",
		BroadcasterUserID: 9000,
		IsEnabled:         true,
		RawPayloadJSON:    "{}",
	})

	client := &fakeKickClient{broadcasterUserID: 9000, subIDPrefix: "newsub"}
	events := []string{"channel.subscription.new", "channel.subscription.renewal", "channel.subscription.gifts"}
	svc := kicksync.NewService(discardLogger(), channelRepo, subRepo, client, events)

	if err := svc.EnsureChannelSubscriptions(ctx, ch.ID); err != nil {
		t.Fatalf("EnsureChannelSubscriptions: %v", err)
	}

	subs, _ := subRepo.ListByChannel(ctx, ch.ID)
	if len(subs) != 3 {
		t.Fatalf("subscriptions len = %d, want 3", len(subs))
	}
	for _, sub := range subs {
		if sub.Status != domain.KickEventSubStatusActive || sub.KickSubscriptionID == "" {
			t.Fatalf("subscription not active: %+v", sub)
		}
	}
}

func TestEnsureChannelSubscriptionsSkipsExistingActive(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, ctx)
	defer db.Close()

	channelRepo := sqliteinfra.NewFollowedChannelRepository(db)
	subRepo := sqliteinfra.NewKickEventSubscriptionRepository(db)

	ch, _ := channelRepo.Upsert(ctx, domain.FollowedChannel{
		Slug:              "chan3",
		DisplayName:       "Chan3",
		BroadcasterUserID: 7777,
		IsEnabled:         true,
		RawPayloadJSON:    "{}",
	})

	_, _ = subRepo.Upsert(ctx, domain.KickEventSubscription{
		FollowedChannelID:  ch.ID,
		BroadcasterUserID:  7777,
		EventType:          "channel.subscription.new",
		EventVersion:       "v1",
		Method:             "webhook",
		KickSubscriptionID: "existing-sub-id",
		Status:             domain.KickEventSubStatusActive,
	})

	client := &fakeKickClient{broadcasterUserID: 7777, subIDPrefix: "shouldnotcreate"}
	svc := kicksync.NewService(discardLogger(), channelRepo, subRepo, client, []string{"channel.subscription.new"})

	if err := svc.EnsureChannelSubscriptions(ctx, ch.ID); err != nil {
		t.Fatalf("EnsureChannelSubscriptions: %v", err)
	}

	if client.createCalls != 0 {
		t.Fatalf("CreateEventSubscription called %d times, want 0", client.createCalls)
	}
}

func TestRemoveChannelSubscriptionsDeletesAll(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, ctx)
	defer db.Close()

	channelRepo := sqliteinfra.NewFollowedChannelRepository(db)
	subRepo := sqliteinfra.NewKickEventSubscriptionRepository(db)

	ch, _ := channelRepo.Upsert(ctx, domain.FollowedChannel{
		Slug:              "chan4",
		DisplayName:       "Chan4",
		BroadcasterUserID: 6666,
		IsEnabled:         true,
		RawPayloadJSON:    "{}",
	})

	for _, et := range []string{"channel.subscription.new", "channel.subscription.gifts"} {
		_, _ = subRepo.Upsert(ctx, domain.KickEventSubscription{
			FollowedChannelID:  ch.ID,
			BroadcasterUserID:  6666,
			EventType:          et,
			EventVersion:       "v1",
			Method:             "webhook",
			KickSubscriptionID: "kick-" + et,
			Status:             domain.KickEventSubStatusActive,
		})
	}

	client := &fakeKickClient{}
	svc := kicksync.NewService(discardLogger(), channelRepo, subRepo, client, nil)

	if err := svc.RemoveChannelSubscriptions(ctx, ch.ID); err != nil {
		t.Fatalf("RemoveChannelSubscriptions: %v", err)
	}

	if client.deleteCalls != 2 {
		t.Fatalf("DeleteEventSubscription called %d times, want 2", client.deleteCalls)
	}

	subs, _ := subRepo.ListByChannel(ctx, ch.ID)
	for _, sub := range subs {
		if sub.Status != domain.KickEventSubStatusDeleted {
			t.Fatalf("subscription status = %s, want deleted", sub.Status)
		}
	}
}

func TestSyncAllStoresErrorPerChannel(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, ctx)
	defer db.Close()

	channelRepo := sqliteinfra.NewFollowedChannelRepository(db)
	subRepo := sqliteinfra.NewKickEventSubscriptionRepository(db)

	ch, _ := channelRepo.Upsert(ctx, domain.FollowedChannel{
		Slug:              "chan5",
		DisplayName:       "Chan5",
		BroadcasterUserID: 4444,
		IsEnabled:         true,
		RawPayloadJSON:    "{}",
	})

	client := &fakeKickClient{createError: "Kick API unavailable"}
	svc := kicksync.NewService(discardLogger(), channelRepo, subRepo, client, []string{"channel.subscription.new"})

	svc.SyncAll(ctx)

	subs, _ := subRepo.ListByChannel(ctx, ch.ID)
	if len(subs) != 1 {
		t.Fatalf("subs len = %d, want 1", len(subs))
	}
	if subs[0].Status != domain.KickEventSubStatusError || subs[0].LatestSyncError == "" {
		t.Fatalf("subscription = %+v", subs[0])
	}
}

type fakeKickClient struct {
	broadcasterUserID int64
	subIDPrefix       string
	createError       string
	createCalls       int
	deleteCalls       int
	counter           int
}

func (f *fakeKickClient) ResolveBroadcasterUserID(_ context.Context, _ string) (int64, error) {
	return f.broadcasterUserID, nil
}

func (f *fakeKickClient) ListEventSubscriptions(_ context.Context) ([]domain.KickAPIEventSub, error) {
	return nil, nil
}

func (f *fakeKickClient) CreateEventSubscription(_ context.Context, _ int64, eventType string) (domain.KickAPIEventSub, error) {
	f.createCalls++
	if f.createError != "" {
		return domain.KickAPIEventSub{}, fmt.Errorf("%s", f.createError)
	}
	f.counter++
	return domain.KickAPIEventSub{
		SubscriptionID: fmt.Sprintf("%s-%d", f.subIDPrefix, f.counter),
		EventType:      eventType,
		Method:         "webhook",
	}, nil
}

func (f *fakeKickClient) DeleteEventSubscription(_ context.Context, _ string) error {
	f.deleteCalls++
	return nil
}

func openDB(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.sqlite3")
	db, err := sqliteinfra.Open(ctx, path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		db.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
