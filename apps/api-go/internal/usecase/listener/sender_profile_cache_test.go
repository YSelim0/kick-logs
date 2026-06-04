package listener

import (
	"testing"
	"time"
)

func TestSenderProfileWriteGateAllowsFirstAndAfterTTL(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	gate := newSenderProfileWriteGate(10 * time.Minute)
	gate.nowFunc = func() time.Time { return now }

	if !gate.ShouldWrite(42) {
		t.Fatal("first write blocked")
	}
	if gate.ShouldWrite(42) {
		t.Fatal("second write within ttl allowed")
	}

	now = now.Add(10 * time.Minute)
	if !gate.ShouldWrite(42) {
		t.Fatal("write after ttl blocked")
	}
}

func TestSenderProfileWriteGateSeparatesSenders(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	gate := newSenderProfileWriteGate(time.Hour)
	gate.nowFunc = func() time.Time { return now }

	if !gate.ShouldWrite(1) {
		t.Fatal("sender 1 first write blocked")
	}
	if !gate.ShouldWrite(2) {
		t.Fatal("sender 2 first write blocked")
	}
}
