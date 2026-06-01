package listener

import (
	"sync"
	"time"
)

type senderProfileWriteGate struct {
	mu      sync.Mutex
	ttl     time.Duration
	last    map[int64]time.Time
	nowFunc func() time.Time
}

func newSenderProfileWriteGate(ttl time.Duration) *senderProfileWriteGate {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &senderProfileWriteGate{
		ttl:     ttl,
		last:    make(map[int64]time.Time),
		nowFunc: time.Now,
	}
}

func (gate *senderProfileWriteGate) ShouldWrite(kickUserID int64) bool {
	if gate == nil || kickUserID < 1 {
		return true
	}
	now := gate.nowFunc().UTC()

	gate.mu.Lock()
	defer gate.mu.Unlock()

	if previous, ok := gate.last[kickUserID]; ok && now.Sub(previous) < gate.ttl {
		return false
	}
	gate.last[kickUserID] = now
	return true
}
