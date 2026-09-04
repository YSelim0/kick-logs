package watchlist

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type fakeNotifier struct {
	mu    sync.Mutex
	sent  []domain.ChatMessage
	err   error
	calls int
}

func (f *fakeNotifier) NotifySenderMessage(_ context.Context, message domain.ChatMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, message)
	return nil
}

func (f *fakeNotifier) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestNewWatchlistService_DisabledWithoutConfig(t *testing.T) {
	if svc := NewWatchlistService(nil, time.Minute, &fakeNotifier{}, slog.Default()); svc != nil {
		t.Fatalf("expected nil service with no usernames configured")
	}
	if svc := NewWatchlistService([]string{"someone"}, time.Minute, nil, slog.Default()); svc != nil {
		t.Fatalf("expected nil service with no notifier configured")
	}
}

func TestWatchlistService_NotifiesMatchingSenderCaseInsensitive(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := NewWatchlistService([]string{"Nuriben"}, time.Minute, notifier, slog.Default())

	svc.Notify(context.Background(), []domain.ChatMessage{
		{SenderUsername: "nuriben", Content: "selam"},
		{SenderUsername: "someoneelse", Content: "ignored"},
	})

	if notifier.callCount() != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", notifier.callCount())
	}
	if notifier.sent[0].Content != "selam" {
		t.Fatalf("unexpected message notified: %+v", notifier.sent[0])
	}
}

func TestWatchlistService_CooldownSuppressesRepeatSends(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := NewWatchlistService([]string{"nuriben"}, time.Hour, notifier, slog.Default())

	svc.Notify(context.Background(), []domain.ChatMessage{{SenderUsername: "nuriben"}})
	svc.Notify(context.Background(), []domain.ChatMessage{{SenderUsername: "nuriben"}})

	if notifier.callCount() != 1 {
		t.Fatalf("expected cooldown to suppress the second send, got %d calls", notifier.callCount())
	}
}

func TestWatchlistService_NotifierErrorDoesNotPanic(t *testing.T) {
	notifier := &fakeNotifier{err: context.DeadlineExceeded}
	svc := NewWatchlistService([]string{"nuriben"}, time.Minute, notifier, slog.Default())

	svc.Notify(context.Background(), []domain.ChatMessage{{SenderUsername: "nuriben"}})

	if notifier.callCount() != 1 {
		t.Fatalf("expected notifier to be called once despite returning an error, got %d", notifier.callCount())
	}
}

func TestWatchlistService_NilReceiverIsNoop(t *testing.T) {
	var svc *WatchlistService
	svc.Notify(context.Background(), []domain.ChatMessage{{SenderUsername: "nuriben"}})
}
