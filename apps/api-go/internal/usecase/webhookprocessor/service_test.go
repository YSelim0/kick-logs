package webhookprocessor_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/webhookprocessor"
)

func TestProcessorWritesPeriodsAndMarksProcessed(t *testing.T) {
	ctx := context.Background()

	channel := domain.FollowedChannel{ID: 1, BroadcasterUserID: 9000, Slug: "chan", DisplayName: "Chan"}
	inbox := newFakeInbox()
	channels := newFakeChannelRepo(channel)
	periods := &fakePeriodRepo{}

	payload, _ := json.Marshal(map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"subscriber":  map[string]any{"user_id": 1001, "username": "sub1"},
		"created_at":  "2026-06-01T10:00:00Z",
		"expires_at":  "2026-07-01T10:00:00Z",
	})
	inbox.add(domain.KickWebhookEvent{
		MessageID:      "msg-proc-001",
		EventType:      webhookprocessor.EventTypeNew,
		EventVersion:   "v1",
		RawPayloadJSON: string(payload),
		Status:         domain.WebhookEventStatusPending,
		ReceivedAt:     time.Now().UTC(),
	})

	svc := webhookprocessor.NewService(discardLogger(), inbox, channels, periods, 10, 5)
	svc.ProcessBatchOnce(ctx)

	if len(periods.inserted) != 1 {
		t.Fatalf("periods inserted = %d, want 1", len(periods.inserted))
	}
	if inbox.events[0].Status != domain.WebhookEventStatusProcessed {
		t.Fatalf("status = %s, want processed", inbox.events[0].Status)
	}
}

func TestProcessorHandlesGiftWithMultipleGiftees(t *testing.T) {
	ctx := context.Background()

	channel := domain.FollowedChannel{ID: 1, BroadcasterUserID: 9000, Slug: "chan", DisplayName: "Chan"}
	inbox := newFakeInbox()
	channels := newFakeChannelRepo(channel)
	periods := &fakePeriodRepo{}

	payload, _ := json.Marshal(map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"gifter":      map[string]any{"user_id": 5000, "username": "gifter"},
		"recipients": []map[string]any{
			{"user_id": 6001, "username": "g1"},
			{"user_id": 6002, "username": "g2"},
		},
		"created_at": "2026-06-01T10:00:00Z",
		"expires_at": "2026-07-01T10:00:00Z",
	})
	inbox.add(domain.KickWebhookEvent{
		MessageID:      "msg-gift-proc",
		EventType:      webhookprocessor.EventTypeGifts,
		EventVersion:   "v1",
		RawPayloadJSON: string(payload),
		Status:         domain.WebhookEventStatusPending,
		ReceivedAt:     time.Now().UTC(),
	})

	svc := webhookprocessor.NewService(discardLogger(), inbox, channels, periods, 10, 5)
	svc.ProcessBatchOnce(ctx)

	if len(periods.inserted) != 2 {
		t.Fatalf("periods inserted = %d, want 2", len(periods.inserted))
	}
	if inbox.events[0].Status != domain.WebhookEventStatusProcessed {
		t.Fatalf("status = %s", inbox.events[0].Status)
	}
}

func TestProcessorIgnoresUnknownBroadcaster(t *testing.T) {
	ctx := context.Background()

	inbox := newFakeInbox()
	channels := newFakeChannelRepo() // no channels
	periods := &fakePeriodRepo{}

	payload, _ := json.Marshal(map[string]any{
		"broadcaster": map[string]any{"user_id": 99999},
		"subscriber":  map[string]any{"user_id": 1},
	})
	inbox.add(domain.KickWebhookEvent{
		MessageID:      "msg-unknown-broadcaster",
		EventType:      webhookprocessor.EventTypeNew,
		RawPayloadJSON: string(payload),
		Status:         domain.WebhookEventStatusPending,
		ReceivedAt:     time.Now().UTC(),
	})

	svc := webhookprocessor.NewService(discardLogger(), inbox, channels, periods, 10, 5)
	svc.ProcessBatchOnce(ctx)

	if len(periods.inserted) != 0 {
		t.Fatalf("periods inserted = %d, want 0", len(periods.inserted))
	}
	if inbox.events[0].Status != domain.WebhookEventStatusIgnored {
		t.Fatalf("status = %s, want ignored", inbox.events[0].Status)
	}
}

func TestProcessorIgnoresUnsupportedEventType(t *testing.T) {
	ctx := context.Background()

	channel := domain.FollowedChannel{ID: 1, BroadcasterUserID: 9000, Slug: "chan", DisplayName: "Chan"}
	inbox := newFakeInbox()
	channels := newFakeChannelRepo(channel)
	periods := &fakePeriodRepo{}

	payload, _ := json.Marshal(map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
	})
	inbox.add(domain.KickWebhookEvent{
		MessageID:      "msg-unsupported",
		EventType:      "channel.ban",
		RawPayloadJSON: string(payload),
		Status:         domain.WebhookEventStatusPending,
		ReceivedAt:     time.Now().UTC(),
	})

	svc := webhookprocessor.NewService(discardLogger(), inbox, channels, periods, 10, 5)
	svc.ProcessBatchOnce(ctx)

	if inbox.events[0].Status != domain.WebhookEventStatusIgnored {
		t.Fatalf("status = %s, want ignored", inbox.events[0].Status)
	}
}

func TestProcessorRetriesAndExhausts(t *testing.T) {
	ctx := context.Background()

	channel := domain.FollowedChannel{ID: 1, BroadcasterUserID: 9000, Slug: "chan", DisplayName: "Chan"}
	inbox := newFakeInbox()
	channels := newFakeChannelRepo(channel)
	periods := &fakePeriodRepo{failInsert: true}

	payload, _ := json.Marshal(map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"subscriber":  map[string]any{"user_id": 1001},
		"created_at":  "2026-06-01T10:00:00Z",
	})
	inbox.add(domain.KickWebhookEvent{
		MessageID:      "msg-retry",
		EventType:      webhookprocessor.EventTypeNew,
		RawPayloadJSON: string(payload),
		Status:         domain.WebhookEventStatusPending,
		ReceivedAt:     time.Now().UTC(),
	})

	svc := webhookprocessor.NewService(discardLogger(), inbox, channels, periods, 10, 3)

	for i := 0; i < 3; i++ {
		svc.ProcessBatchOnce(ctx)
	}

	if inbox.events[0].Status != domain.WebhookEventStatusFailed {
		t.Fatalf("status after exhaustion = %s, want failed", inbox.events[0].Status)
	}
}

// --- fakes ---

type fakeInbox struct {
	events []domain.KickWebhookEvent
}

func newFakeInbox() *fakeInbox { return &fakeInbox{} }

func (f *fakeInbox) add(e domain.KickWebhookEvent) { f.events = append(f.events, e) }

func (f *fakeInbox) InsertIdempotent(_ context.Context, e domain.KickWebhookEvent) error {
	for _, existing := range f.events {
		if existing.MessageID == e.MessageID {
			return nil
		}
	}
	f.events = append(f.events, e)
	return nil
}

func (f *fakeInbox) GetByMessageID(_ context.Context, id string) (domain.KickWebhookEvent, error) {
	for _, e := range f.events {
		if e.MessageID == id {
			return e, nil
		}
	}
	return domain.KickWebhookEvent{}, sql.ErrNoRows
}

func (f *fakeInbox) ListPending(_ context.Context, limit int, maxAttempts int) ([]domain.KickWebhookEvent, error) {
	var out []domain.KickWebhookEvent
	for _, e := range f.events {
		if e.Status == domain.WebhookEventStatusPending && e.Attempts < maxAttempts {
			out = append(out, e)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (f *fakeInbox) MarkProcessed(_ context.Context, id string) error {
	for i := range f.events {
		if f.events[i].MessageID == id {
			f.events[i].Status = domain.WebhookEventStatusProcessed
		}
	}
	return nil
}

func (f *fakeInbox) MarkFailed(_ context.Context, id string, errMsg string, maxAttempts int) error {
	for i := range f.events {
		if f.events[i].MessageID == id {
			f.events[i].Attempts++
			f.events[i].ErrorMessage = errMsg
			if f.events[i].Attempts >= maxAttempts {
				f.events[i].Status = domain.WebhookEventStatusFailed
			}
		}
	}
	return nil
}

func (f *fakeInbox) MarkIgnored(_ context.Context, id string, reason string) error {
	for i := range f.events {
		if f.events[i].MessageID == id {
			f.events[i].Status = domain.WebhookEventStatusIgnored
			f.events[i].ErrorMessage = reason
		}
	}
	return nil
}

func (f *fakeInbox) CountByStatus(_ context.Context) (map[string]int64, error) { return nil, nil }
func (f *fakeInbox) LatestReceivedAt(_ context.Context) (time.Time, error)     { return time.Time{}, nil }

type fakeChannelRepo struct {
	channels []domain.FollowedChannel
}

func newFakeChannelRepo(chs ...domain.FollowedChannel) *fakeChannelRepo {
	return &fakeChannelRepo{channels: chs}
}

func (r *fakeChannelRepo) GetByBroadcasterUserID(_ context.Context, id int64) (domain.FollowedChannel, error) {
	for _, ch := range r.channels {
		if ch.BroadcasterUserID == id {
			return ch, nil
		}
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (r *fakeChannelRepo) Upsert(_ context.Context, ch domain.FollowedChannel) (domain.FollowedChannel, error) {
	return ch, nil
}
func (r *fakeChannelRepo) GetByID(_ context.Context, _ int64) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{}, nil
}
func (r *fakeChannelRepo) GetBySlug(_ context.Context, _ string) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{}, sql.ErrNoRows
}
func (r *fakeChannelRepo) GetByChatroomID(_ context.Context, _ int64) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{}, sql.ErrNoRows
}
func (r *fakeChannelRepo) List(_ context.Context) ([]domain.FollowedChannel, error) {
	return r.channels, nil
}
func (r *fakeChannelRepo) ListEnabled(_ context.Context) ([]domain.FollowedChannel, error) {
	var out []domain.FollowedChannel
	for _, ch := range r.channels {
		if ch.IsEnabled {
			out = append(out, ch)
		}
	}
	return out, nil
}
func (r *fakeChannelRepo) Disable(_ context.Context, _ int64) (domain.FollowedChannel, error) {
	return domain.FollowedChannel{}, nil
}

type fakePeriodRepo struct {
	inserted   []domain.ChannelSubscriptionPeriod
	failInsert bool
}

func (r *fakePeriodRepo) InsertBatch(_ context.Context, periods []domain.ChannelSubscriptionPeriod) error {
	if r.failInsert {
		return errFakeInsert
	}
	r.inserted = append(r.inserted, periods...)
	return nil
}

func (r *fakePeriodRepo) ActiveSummary(_ context.Context, _ int64) (domain.ChannelSubscriptionSummary, error) {
	return domain.ChannelSubscriptionSummary{}, nil
}

var errFakeInsert = fmt.Errorf("fake insert error")

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
