package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
)

func TestRawEventQueueEnqueueAndListPending(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventQueueRepository(db)
	base := time.Now().UTC().Add(-time.Minute)
	items := []domain.RawEventQueueItem{
		{RawEventID: "a", ChannelID: 1, ChatroomID: 11, ChannelSlug: "hype", KickMessageID: "m-a", EnqueuedAt: base.Add(0)},
		{RawEventID: "b", ChannelID: 2, ChatroomID: 22, ChannelSlug: "alpha", KickMessageID: "m-b", EnqueuedAt: base.Add(time.Second)},
		{RawEventID: "c", ChannelID: 3, ChatroomID: 33, ChannelSlug: "beta", KickMessageID: "m-c", EnqueuedAt: base.Add(2 * time.Second)},
	}
	if err := repo.EnqueueBatch(ctx, items); err != nil {
		t.Fatalf("EnqueueBatch() error = %v", err)
	}

	if err := repo.EnqueueBatch(ctx, items[:1]); err != nil {
		t.Fatalf("EnqueueBatch() duplicate error = %v", err)
	}

	pending, err := repo.ListPending(ctx, 10, 5)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(pending) != 3 {
		t.Fatalf("ListPending() len = %d, want 3", len(pending))
	}
	if pending[0].RawEventID != "a" || pending[1].RawEventID != "b" || pending[2].RawEventID != "c" {
		t.Fatalf("ListPending() order = %v %v %v", pending[0].RawEventID, pending[1].RawEventID, pending[2].RawEventID)
	}
	if pending[0].Status != domain.RawEventQueueStatusPending {
		t.Fatalf("ListPending() status = %q", pending[0].Status)
	}
	if pending[0].ChannelSlug != "hype" || pending[0].ChannelID != 1 {
		t.Fatalf("ListPending() metadata = %#v", pending[0])
	}
}

func TestRawEventQueueClaimAndRelease(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventQueueRepository(db)
	if err := repo.Enqueue(ctx, domain.RawEventQueueItem{RawEventID: "x"}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	claimed, err := repo.Claim(ctx, "x", "worker-1")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if !claimed {
		t.Fatal("Claim() = false, want true")
	}
	again, err := repo.Claim(ctx, "x", "worker-2")
	if err != nil {
		t.Fatalf("Claim() second error = %v", err)
	}
	if again {
		t.Fatal("Claim() second = true, want false (already claimed)")
	}

	item, err := repo.GetByID(ctx, "x")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.Status != domain.RawEventQueueStatusClaimed || item.ClaimedBy != "worker-1" {
		t.Fatalf("GetByID() = %#v", item)
	}

	if err := repo.Release(ctx, "x", "worker-1"); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	item, err = repo.GetByID(ctx, "x")
	if err != nil {
		t.Fatalf("GetByID() after release error = %v", err)
	}
	if item.Status != domain.RawEventQueueStatusPending || item.ClaimedBy != "" {
		t.Fatalf("after release = %#v", item)
	}
}

func TestRawEventQueueMarkProcessedDeletesRow(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventQueueRepository(db)
	if err := repo.Enqueue(ctx, domain.RawEventQueueItem{RawEventID: "x"}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if _, err := repo.Claim(ctx, "x", "worker-1"); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if err := repo.MarkProcessed(ctx, "x"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	if err := repo.MarkProcessed(ctx, "x"); err != nil {
		t.Fatalf("MarkProcessed() idempotent error = %v", err)
	}
	if _, err := repo.GetByID(ctx, "x"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetByID() error = %v, want sql.ErrNoRows", err)
	}
	pending, err := repo.ListPending(ctx, 10, 5)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("ListPending() len = %d, want 0", len(pending))
	}
}

func TestRawEventQueueMarkFailedAndExhaustion(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventQueueRepository(db)
	if err := repo.Enqueue(ctx, domain.RawEventQueueItem{RawEventID: "x"}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if _, err := repo.Claim(ctx, "x", "worker-1"); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if err := repo.MarkFailed(ctx, "x", "boom", 3); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	item, err := repo.GetByID(ctx, "x")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.Status != domain.RawEventQueueStatusPending {
		t.Fatalf("status after 1 fail = %q, want pending", item.Status)
	}
	if item.Attempts != 1 || item.LastError != "boom" {
		t.Fatalf("item after fail = %#v", item)
	}

	if _, err := repo.Claim(ctx, "x", "worker-1"); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if err := repo.MarkFailed(ctx, "x", "again", 3); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if _, err := repo.Claim(ctx, "x", "worker-1"); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if err := repo.MarkFailed(ctx, "x", "last", 3); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	item, err = repo.GetByID(ctx, "x")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.Status != domain.RawEventQueueStatusFailed {
		t.Fatalf("status after maxAttempts = %q, want failed", item.Status)
	}
	if item.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3", item.Attempts)
	}
}

func TestRawEventQueueCountAndOldestAge(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventQueueRepository(db)
	old := time.Now().UTC().Add(-30 * time.Second)
	items := []domain.RawEventQueueItem{
		{RawEventID: "a", EnqueuedAt: old},
		{RawEventID: "b", EnqueuedAt: time.Now().UTC()},
	}
	if err := repo.EnqueueBatch(ctx, items); err != nil {
		t.Fatalf("EnqueueBatch() error = %v", err)
	}

	count, err := repo.CountPending(ctx, 5)
	if err != nil {
		t.Fatalf("CountPending() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountPending() = %d, want 2", count)
	}

	age, err := repo.OldestPendingAge(ctx, 5)
	if err != nil {
		t.Fatalf("OldestPendingAge() error = %v", err)
	}
	if age < 25*time.Second {
		t.Fatalf("OldestPendingAge() = %v, want >= 25s", age)
	}

	if err := repo.MarkProcessed(ctx, "a"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	count, err = repo.CountPending(ctx, 5)
	if err != nil {
		t.Fatalf("CountPending() after processed error = %v", err)
	}
	if count != 1 {
		t.Fatalf("CountPending() after processed = %d, want 1", count)
	}
}

func TestRawEventQueueRecoverStaleClaims(t *testing.T) {
	ctx := context.Background()
	db, _ := openMigratedSQLite(t, ctx)
	defer db.Close()

	repo := sqlite.NewRawEventQueueRepository(db)
	if err := repo.Enqueue(ctx, domain.RawEventQueueItem{RawEventID: "stuck"}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if _, err := repo.Claim(ctx, "stuck", "worker-1"); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	_, err := db.ExecContext(
		ctx,
		`UPDATE raw_event_queue SET claimed_at = ? WHERE raw_event_id = ?;`,
		time.Now().UTC().Add(-10*time.Minute).Format(time.RFC3339Nano),
		"stuck",
	)
	if err != nil {
		t.Fatalf("backdate claim error = %v", err)
	}

	recovered, err := repo.RecoverStaleClaims(ctx, time.Minute)
	if err != nil {
		t.Fatalf("RecoverStaleClaims() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("RecoverStaleClaims() = %d, want 1", recovered)
	}
	item, err := repo.GetByID(ctx, "stuck")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.Status != domain.RawEventQueueStatusPending {
		t.Fatalf("recovered status = %q", item.Status)
	}
}
