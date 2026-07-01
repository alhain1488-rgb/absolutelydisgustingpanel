package auth

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	l := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("attempt %d should be allowed", i)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("4th attempt should be blocked")
	}
	// Different key is independent.
	if !l.Allow("5.6.7.8") {
		t.Fatal("different key should be allowed")
	}
}

func TestRateLimiter_WindowResets(t *testing.T) {
	l := NewRateLimiter(1, time.Minute)
	base := time.Now()
	l.now = func() time.Time { return base }
	if !l.Allow("k") {
		t.Fatal("first allowed")
	}
	if l.Allow("k") {
		t.Fatal("second blocked in window")
	}
	// Advance past the window.
	l.now = func() time.Time { return base.Add(2 * time.Minute) }
	if !l.Allow("k") {
		t.Fatal("should be allowed after window reset")
	}
}
