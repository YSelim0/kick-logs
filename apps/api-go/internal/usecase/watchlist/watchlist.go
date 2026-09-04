// Package watchlist gates outbound sender-message alerts: it matches chat
// messages against a configured watchlist and rate-limits repeat sends per
// sender before handing off to a ports.SenderMessageNotifier.
package watchlist

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

// WatchlistService checks normalized chat messages against a set of watched
// sender usernames and forwards a match to the notifier, at most once per
// sender per cooldown window. It must never block or fail chat ingestion:
// callers should treat it as fire-and-forget, matching the sender-profile
// cache's best-effort rule.
type WatchlistService struct {
	usernames map[string]bool
	cooldown  time.Duration
	notifier  ports.SenderMessageNotifier
	logger    *slog.Logger

	mu       sync.Mutex
	lastSent map[string]time.Time
}

// NewWatchlistService returns nil when the feature is not configured
// (no usernames or no notifier), so callers can invoke methods on a nil
// receiver unconditionally without an extra nil check at every call site.
func NewWatchlistService(usernames []string, cooldown time.Duration, notifier ports.SenderMessageNotifier, logger *slog.Logger) *WatchlistService {
	if notifier == nil || len(usernames) == 0 {
		return nil
	}
	set := make(map[string]bool, len(usernames))
	for _, username := range usernames {
		key := normalizeUsername(username)
		if key != "" {
			set[key] = true
		}
	}
	if len(set) == 0 {
		return nil
	}
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &WatchlistService{
		usernames: set,
		cooldown:  cooldown,
		notifier:  notifier,
		logger:    logger,
		lastSent:  make(map[string]time.Time),
	}
}

// Notify checks each message against the watchlist and sends at most one
// notification per sender per cooldown window. Safe to call on a nil
// receiver (no-op) and intended to be run in its own goroutine by the
// caller so a slow/blocked mail server cannot delay chat ingestion acks.
func (s *WatchlistService) Notify(ctx context.Context, messages []domain.ChatMessage) {
	if s == nil {
		return
	}
	for _, message := range messages {
		key := normalizeUsername(message.SenderUsername)
		if key == "" || !s.usernames[key] {
			continue
		}
		if !s.shouldSend(key) {
			continue
		}
		if err := s.notifier.NotifySenderMessage(ctx, message); err != nil {
			s.logger.Error("failed to send watched-sender notification", "sender", message.SenderUsername, "error", err)
		}
	}
}

func (s *WatchlistService) shouldSend(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if last, ok := s.lastSent[key]; ok && now.Sub(last) < s.cooldown {
		return false
	}
	s.lastSent[key] = now
	return true
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
