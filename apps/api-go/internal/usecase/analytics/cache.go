package analytics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

const (
	analyticsCacheFreshTTL = time.Hour
	analyticsCacheStaleTTL = 24 * time.Hour
	analyticsCacheMaxItems = 128
)

type analyticsCache struct {
	mu       sync.Mutex
	entries  map[string]analyticsCacheEntry
	inflight map[string]*analyticsCacheCall
	now      func() time.Time
}

type analyticsCacheEntry struct {
	value      any
	expiresAt  time.Time
	staleUntil time.Time
	accessedAt time.Time
}

type analyticsCacheCall struct {
	done  chan struct{}
	value any
	err   error
}

func newAnalyticsCache() *analyticsCache {
	return &analyticsCache{
		entries:  map[string]analyticsCacheEntry{},
		inflight: map[string]*analyticsCacheCall{},
		now:      time.Now,
	}
}

func cachedAnalyticsValue[T any](
	ctx context.Context,
	cache *analyticsCache,
	key string,
	fetch func(context.Context) (T, error),
) (T, error) {
	if cache == nil || key == "" {
		return fetch(ctx)
	}

	if value, ok := cache.fresh(key); ok {
		if typed, ok := value.(T); ok {
			return typed, nil
		}
		cache.delete(key)
	}

	call, owner := cache.begin(key)
	if !owner {
		select {
		case <-call.done:
			if call.err != nil {
				var zero T
				return zero, call.err
			}
			if typed, ok := call.value.(T); ok {
				return typed, nil
			}
			var zero T
			return zero, fmt.Errorf("analytics cache type mismatch for %s", key)
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		}
	}

	value, err := fetch(ctx)
	cache.finish(key, call, value, err)
	if call.err != nil {
		var zero T
		return zero, call.err
	}
	if typed, ok := call.value.(T); ok {
		return typed, nil
	}
	var zero T
	return zero, fmt.Errorf("analytics cache type mismatch for %s", key)
}

func (cache *analyticsCache) fresh(key string) (any, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	entry, ok := cache.entries[key]
	if !ok {
		return nil, false
	}
	now := cache.now()
	if now.After(entry.expiresAt) {
		return nil, false
	}
	entry.accessedAt = now
	cache.entries[key] = entry
	return entry.value, true
}

func (cache *analyticsCache) begin(key string) (*analyticsCacheCall, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if call, ok := cache.inflight[key]; ok {
		return call, false
	}
	call := &analyticsCacheCall{done: make(chan struct{})}
	cache.inflight[key] = call
	return call, true
}

func (cache *analyticsCache) finish(key string, call *analyticsCacheCall, value any, err error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	now := cache.now()
	if err == nil {
		cache.entries[key] = analyticsCacheEntry{
			value:      value,
			expiresAt:  now.Add(analyticsCacheFreshTTL),
			staleUntil: now.Add(analyticsCacheStaleTTL),
			accessedAt: now,
		}
		cache.evict(now)
		call.value = value
	} else if stale, ok := cache.staleLocked(key, now); ok {
		call.value = stale
	} else {
		call.err = err
	}
	delete(cache.inflight, key)
	close(call.done)
}

func (cache *analyticsCache) staleLocked(key string, now time.Time) (any, bool) {
	entry, ok := cache.entries[key]
	if !ok || now.After(entry.staleUntil) {
		return nil, false
	}
	entry.accessedAt = now
	cache.entries[key] = entry
	return entry.value, true
}

func (cache *analyticsCache) evict(now time.Time) {
	if len(cache.entries) <= analyticsCacheMaxItems {
		return
	}
	for key, entry := range cache.entries {
		if now.After(entry.staleUntil) {
			delete(cache.entries, key)
		}
	}
	for len(cache.entries) > analyticsCacheMaxItems {
		var oldestKey string
		var oldestAt time.Time
		for key, entry := range cache.entries {
			if oldestKey == "" || entry.accessedAt.Before(oldestAt) {
				oldestKey = key
				oldestAt = entry.accessedAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(cache.entries, oldestKey)
	}
}

func (cache *analyticsCache) delete(key string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.entries, key)
}

func globalAnalyticsCacheable(filter domain.AnalyticsFilter) bool {
	return strings.TrimSpace(filter.Query) == "" &&
		strings.TrimSpace(filter.Sender) == "" &&
		strings.TrimSpace(filter.Channel) == ""
}

func analyticsFilterCacheKey(filter domain.AnalyticsFilter) string {
	return fmt.Sprintf(
		"start=%s:end=%s",
		analyticsTimeCacheKey(filter.Start),
		analyticsTimeCacheKey(filter.End),
	)
}

func analyticsTimeCacheKey(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
