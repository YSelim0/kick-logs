package listener

import (
	"math/rand"
	"time"
)

// Backoff produces non-decreasing delays bounded by Max with multiplicative growth.
// Next adds full jitter so concurrent callers do not synchronize their retries.
type Backoff struct {
	Initial    time.Duration
	Max        time.Duration
	Multiplier float64
	attempts   int
	current    time.Duration
}

func NewBackoff(initial, max time.Duration, multiplier float64) *Backoff {
	if initial <= 0 {
		initial = time.Second
	}
	if max < initial {
		max = initial
	}
	if multiplier < 1 {
		multiplier = 2
	}
	return &Backoff{Initial: initial, Max: max, Multiplier: multiplier}
}

func (b *Backoff) Next() time.Duration {
	if b.current == 0 {
		b.current = b.Initial
	}
	delay := b.current
	if delay > b.Max {
		delay = b.Max
	}
	b.attempts++
	next := time.Duration(float64(b.current) * b.Multiplier)
	if next > b.Max {
		next = b.Max
	}
	b.current = next
	return jitter(delay)
}

func (b *Backoff) Reset() {
	b.attempts = 0
	b.current = 0
}

func (b *Backoff) Attempts() int {
	return b.attempts
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	return time.Duration(rand.Int63n(int64(d)) + int64(d)/2)
}
