package listener

import (
	"testing"
	"time"
)

func TestBackoffProducesNonDecreasingDelays(t *testing.T) {
	b := NewBackoff(100*time.Millisecond, 800*time.Millisecond, 2)
	want := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		800 * time.Millisecond,
	}
	for i, base := range want {
		got := b.Next()
		lower := base / 2
		upper := base + base/2
		if got < lower || got > upper {
			t.Fatalf("Next() #%d = %v, want jitter window [%v, %v]", i, got, lower, upper)
		}
	}
}

func TestBackoffResetReturnsToInitial(t *testing.T) {
	b := NewBackoff(50*time.Millisecond, time.Second, 3)
	b.Next()
	b.Next()
	b.Reset()
	got := b.Next()
	if got < 25*time.Millisecond || got > 75*time.Millisecond {
		t.Fatalf("after reset Next() = %v", got)
	}
}
