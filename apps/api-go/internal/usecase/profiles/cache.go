package profiles

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	profileCacheFreshTTL = time.Hour
	profileCacheStaleTTL = 24 * time.Hour
	profileCacheMaxItems = 128
)

type profileCache struct {
	mu       sync.Mutex
	entries  map[string]profileCacheEntry
	inflight map[string]*profileCacheCall
	now      func() time.Time
}

type profileCacheEntry struct {
	value      any
	expiresAt  time.Time
	staleUntil time.Time
	accessedAt time.Time
}

type profileCacheCall struct {
	done  chan struct{}
	value any
	err   error
}

func newProfileCache() *profileCache {
	return &profileCache{
		entries:  map[string]profileCacheEntry{},
		inflight: map[string]*profileCacheCall{},
		now:      time.Now,
	}
}

func cachedProfileValue[T any](
	ctx context.Context,
	cache *profileCache,
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
			return zero, fmt.Errorf("profile cache type mismatch for %s", key)
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
	return zero, fmt.Errorf("profile cache type mismatch for %s", key)
}

func (cache *profileCache) fresh(key string) (any, bool) {
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

func (cache *profileCache) begin(key string) (*profileCacheCall, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if call, ok := cache.inflight[key]; ok {
		return call, false
	}
	call := &profileCacheCall{done: make(chan struct{})}
	cache.inflight[key] = call
	return call, true
}

func (cache *profileCache) finish(key string, call *profileCacheCall, value any, err error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	now := cache.now()
	if err == nil {
		cache.entries[key] = profileCacheEntry{
			value:      value,
			expiresAt:  now.Add(profileCacheFreshTTL),
			staleUntil: now.Add(profileCacheStaleTTL),
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

func (cache *profileCache) staleLocked(key string, now time.Time) (any, bool) {
	entry, ok := cache.entries[key]
	if !ok || now.After(entry.staleUntil) {
		return nil, false
	}
	entry.accessedAt = now
	cache.entries[key] = entry
	return entry.value, true
}

func (cache *profileCache) evict(now time.Time) {
	if len(cache.entries) <= profileCacheMaxItems {
		return
	}
	for key, entry := range cache.entries {
		if now.After(entry.staleUntil) {
			delete(cache.entries, key)
		}
	}
	for len(cache.entries) > profileCacheMaxItems {
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

func (cache *profileCache) delete(key string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.entries, key)
}

func profileCacheKey(kind string, slug string) string {
	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return ""
	}
	return kind + ":" + cleaned
}
