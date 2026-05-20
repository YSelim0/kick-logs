package listener

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreakerOpensAfterThresholdFailures(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, NewBackoff(200*time.Millisecond, time.Second, 2), testLogger())

	if err := cb.Wait(context.Background()); err != nil {
		t.Fatalf("closed Wait() error = %v", err)
	}
	cb.RecordFailure()
	cb.RecordFailure()
	if state := cb.State(); state.State != "closed" {
		t.Fatalf("state before threshold = %#v", state)
	}
	start := time.Now()
	cb.RecordFailure()
	if state := cb.State(); state.State != "open" {
		t.Fatalf("state after threshold = %#v", state)
	}

	if err := cb.Wait(context.Background()); err != nil {
		t.Fatalf("open Wait() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed < 90*time.Millisecond {
		t.Fatalf("open Wait() returned too fast (%v), expected to honor open window", elapsed)
	}
}

func TestCircuitBreakerClosesOnProbeSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, NewBackoff(20*time.Millisecond, 100*time.Millisecond, 2), testLogger())
	cb.RecordFailure()
	if state := cb.State(); state.State != "open" {
		t.Fatalf("state = %#v", state)
	}
	if err := cb.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	cb.RecordSuccess()
	if state := cb.State(); state.State != "closed" || state.Failures != 0 {
		t.Fatalf("state after success = %#v", state)
	}
}

func TestCircuitBreakerReOpensWithLongerDelayOnProbeFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, NewBackoff(20*time.Millisecond, time.Second, 4), testLogger())
	cb.RecordFailure()
	first := cb.State().CurrentDelay
	if err := cb.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	cb.RecordFailure()
	second := cb.State().CurrentDelay
	if second <= first {
		t.Fatalf("expected second open delay (%v) longer than first (%v)", second, first)
	}
}

func TestCircuitBreakerWaitRespectsContextCancellation(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, NewBackoff(time.Second, time.Second, 1), testLogger())
	cb.RecordFailure()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cb.Wait(ctx); err == nil {
		t.Fatal("Wait() returned nil for cancelled context")
	}
}

func TestCircuitBreakerSharedAcrossGoroutines(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, NewBackoff(200*time.Millisecond, time.Second, 2), testLogger())
	start := time.Now()
	cb.RecordFailure()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cb.Wait(context.Background()); err != nil {
				t.Errorf("Wait() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if elapsed := time.Since(start); elapsed < 90*time.Millisecond {
		t.Fatalf("Wait() returned too fast on open breaker: %v", elapsed)
	}
}
