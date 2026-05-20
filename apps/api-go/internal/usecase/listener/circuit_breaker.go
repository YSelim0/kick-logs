package listener

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// CircuitBreaker coordinates ClickHouse access across listener goroutines.
// Closed: calls pass straight through.
// Open: calls block on Wait until the open window expires.
// After the open window the next caller probes ClickHouse. RecordSuccess closes
// the breaker; RecordFailure re-opens it with a longer delay.
type CircuitBreaker struct {
	mu                sync.Mutex
	threshold         int
	state             int32 // 0 = closed, 1 = open
	openUntil         time.Time
	failures          int
	backoff           *Backoff
	logger            *slog.Logger
	name              string
	lastStateChangeAt time.Time
	currentDelay      time.Duration
}

const (
	circuitStateClosed int32 = 0
	circuitStateOpen   int32 = 1
)

func NewCircuitBreaker(name string, threshold int, backoff *Backoff, logger *slog.Logger) *CircuitBreaker {
	if threshold < 1 {
		threshold = 5
	}
	if backoff == nil {
		backoff = NewBackoff(time.Second, 30*time.Second, 2)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &CircuitBreaker{
		threshold: threshold,
		backoff:   backoff,
		logger:    logger,
		name:      name,
	}
}

// Wait blocks until the breaker would accept a call or ctx is cancelled.
// Returns ctx.Err() on cancellation. Returns nil when the caller may proceed.
func (cb *CircuitBreaker) Wait(ctx context.Context) error {
	for {
		cb.mu.Lock()
		if cb.state == circuitStateClosed {
			cb.mu.Unlock()
			return nil
		}
		now := time.Now()
		if !now.Before(cb.openUntil) {
			cb.state = circuitStateClosed
			cb.failures = 0
			cb.lastStateChangeAt = now
			cb.logger.Info(
				"circuit breaker probe window opened",
				"breaker", cb.name,
			)
			cb.mu.Unlock()
			return nil
		}
		remaining := cb.openUntil.Sub(now)
		cb.mu.Unlock()

		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// RecordSuccess resets the failure counter and closes the breaker.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == circuitStateOpen {
		cb.logger.Info(
			"circuit breaker closed after probe success",
			"breaker", cb.name,
		)
	}
	cb.state = circuitStateClosed
	cb.failures = 0
	cb.currentDelay = 0
	cb.backoff.Reset()
	cb.lastStateChangeAt = time.Now()
}

// RecordFailure increments the failure counter and opens the breaker once the
// threshold is reached. Each subsequent failure while open increases the open
// window by the backoff strategy.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures < cb.threshold && cb.state == circuitStateClosed {
		return
	}
	delay := cb.backoff.Next()
	cb.state = circuitStateOpen
	cb.openUntil = time.Now().Add(delay)
	cb.currentDelay = delay
	cb.lastStateChangeAt = time.Now()
	cb.logger.Info(
		"circuit breaker opened",
		"breaker", cb.name,
		"delay", delay.String(),
		"failures", cb.failures,
	)
}

// State returns a snapshot of the breaker state for metrics or logs.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	stateName := "closed"
	if cb.state == circuitStateOpen {
		stateName = "open"
	}
	return CircuitBreakerState{
		Name:         cb.name,
		State:        stateName,
		Failures:     cb.failures,
		CurrentDelay: cb.currentDelay,
		OpenUntil:    cb.openUntil,
		ChangedAt:    cb.lastStateChangeAt,
	}
}

type CircuitBreakerState struct {
	Name         string
	State        string
	Failures     int
	CurrentDelay time.Duration
	OpenUntil    time.Time
	ChangedAt    time.Time
}
