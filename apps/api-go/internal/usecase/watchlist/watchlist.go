// Package watchlist gates outbound sender-message alerts: it matches chat
// messages against a dynamically refreshed watchlist and rate-limits repeat
// sends per sender before handing off to a ports.SenderMessageNotifier.
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
// cache's best-effort rule. The watchlist itself is admin-managed (SQLite)
// and refreshed at runtime via SetUsernames, so adding/removing a watched
// username never requires a process restart.
type WatchlistService struct {
	notifier ports.SenderMessageNotifier
	logger   *slog.Logger

	mu        sync.Mutex
	cooldown  time.Duration
	usernames map[string]bool
	lastSent  map[string]time.Time
}

// NewWatchlistService returns nil when no notifier is configured, so
// callers can invoke methods on a nil receiver unconditionally without an
// extra nil check at every call site. The watchlist starts empty; callers
// populate it with SetUsernames.
func NewWatchlistService(cooldown time.Duration, notifier ports.SenderMessageNotifier, logger *slog.Logger) *WatchlistService {
	if notifier == nil {
		return nil
	}
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &WatchlistService{
		cooldown:  cooldown,
		notifier:  notifier,
		logger:    logger,
		usernames: make(map[string]bool),
		lastSent:  make(map[string]time.Time),
	}
}

// SetUsernames replaces the current watchlist. Safe to call concurrently
// with Notify and on a nil receiver (no-op); intended to be called
// periodically by the caller (a poll of the admin-managed SQLite list) so
// watchlist changes take effect without a process restart.
func (s *WatchlistService) SetUsernames(usernames []string) {
	if s == nil {
		return
	}
	set := make(map[string]bool, len(usernames))
	for _, username := range usernames {
		key := normalizeUsername(username)
		if key != "" {
			set[key] = true
		}
	}
	s.mu.Lock()
	s.usernames = set
	s.mu.Unlock()
}

// SetCooldown replaces the per-sender cooldown window. Safe to call
// concurrently with Notify and on a nil receiver (no-op); intended to be
// called periodically by the caller (a poll of the admin-managed cooldown
// setting) so a cooldown change takes effect without a process restart. A
// non-positive duration is ignored so a bad read cannot disable the
// cooldown entirely.
func (s *WatchlistService) SetCooldown(cooldown time.Duration) {
	if s == nil || cooldown <= 0 {
		return
	}
	s.mu.Lock()
	s.cooldown = cooldown
	s.mu.Unlock()
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
		if key == "" || !s.matchAndReserve(key) {
			continue
		}
		if err := s.notifier.NotifySenderMessage(ctx, message); err != nil {
			s.logger.Error("failed to send watched-sender notification", "sender", message.SenderUsername, "error", err)
		}
	}
}

// matchAndReserve reports whether key is on the watchlist and not within
// its cooldown window, marking it as just-sent if so. The membership check
// and cooldown reservation share one lock so a concurrent SetUsernames
// cannot interleave with a Notify decision.
func (s *WatchlistService) matchAndReserve(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.usernames[key] {
		return false
	}
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
