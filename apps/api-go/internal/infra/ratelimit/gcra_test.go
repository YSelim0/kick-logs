package ratelimit

import (
	"testing"
)

func newTestLimiter(t *testing.T) *GCRARateLimiter {
	t.Helper()
	l, err := NewGCRA(1000)
	if err != nil {
		t.Fatalf("NewGCRA: %v", err)
	}
	return l
}

func TestGCRA_AllowsBurst(t *testing.T) {
	l := newTestLimiter(t)
	for i := 0; i <= 3; i++ {
		result, err := l.RateLimit("key1", 10, 60, 3)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if result.Limited {
			t.Fatalf("request %d should be allowed (burst=3, so first 4 pass)", i)
		}
	}
}

func TestGCRA_BlocksAfterBurst(t *testing.T) {
	l := newTestLimiter(t)
	for i := 0; i <= 2; i++ {
		_, err := l.RateLimit("key2", 10, 60, 2)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
	result, err := l.RateLimit("key2", 10, 60, 2)
	if err != nil {
		t.Fatalf("excess request: %v", err)
	}
	if !result.Limited {
		t.Fatal("excess request should be limited")
	}
	if result.RetryAfter < 1 {
		t.Fatalf("RetryAfter should be >= 1, got %d", result.RetryAfter)
	}
}

func TestGCRA_IndependentKeys(t *testing.T) {
	l := newTestLimiter(t)
	burst := 1
	for i := 0; i <= burst; i++ {
		_, _ = l.RateLimit("keyA", 10, 60, burst)
	}
	result, err := l.RateLimit("keyB", 10, 60, burst)
	if err != nil {
		t.Fatalf("keyB: %v", err)
	}
	if result.Limited {
		t.Fatal("keyB should be independent from keyA")
	}
}

func TestGCRA_IndependentConfigs(t *testing.T) {
	l := newTestLimiter(t)
	for i := 0; i <= 1; i++ {
		_, _ = l.RateLimit("shared-key", 2, 60, 1)
	}
	result, err := l.RateLimit("shared-key", 100, 60, 50)
	if err != nil {
		t.Fatalf("different config: %v", err)
	}
	if result.Limited {
		t.Fatal("different config tuple should have independent state")
	}
}
