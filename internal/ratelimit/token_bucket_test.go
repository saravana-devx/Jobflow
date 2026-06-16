package ratelimit

import (
	"testing"
	"time"
)

// TestTokenBucket_InitialCapacity verifies a fresh bucket starts full: exactly
// `capacity` immediate acquires succeed, and the next one fails.
func TestTokenBucket_InitialCapacity(t *testing.T) {
	tests := []struct {
		name     string
		rate     int
		capacity int
	}{
		{name: "single token bucket", rate: 1, capacity: 1},
		{name: "typical bucket", rate: 2, capacity: 5},
		{name: "large bucket", rate: 10, capacity: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(tt.rate, tt.capacity)
			defer tb.Stop()

			for i := 0; i < tt.capacity; i++ {
				if !tb.TryAcquire() {
					t.Fatalf("acquire %d of %d failed on a full bucket", i+1, tt.capacity)
				}
			}

			if tb.TryAcquire() {
				t.Fatalf("acquire succeeded after draining all %d tokens; want empty", tt.capacity)
			}
		})
	}
}

// TestTokenBucket_Refills verifies the background ticker tops the bucket back
// up: after draining, waiting longer than one refill interval yields a token.
func TestTokenBucket_Refills(t *testing.T) {
	// rate=10 → roughly one token every 100ms.
	tb := NewTokenBucket(10, 1)
	defer tb.Stop()

	if !tb.TryAcquire() {
		t.Fatal("expected the single starting token to be available")
	}
	if tb.TryAcquire() {
		t.Fatal("bucket should be empty immediately after draining")
	}

	// Wait comfortably past one refill interval (~100ms) to avoid flakiness.
	time.Sleep(300 * time.Millisecond)

	if !tb.TryAcquire() {
		t.Fatal("expected a refilled token after waiting past the refill interval")
	}
}

// TestTokenBucket_RefillCappedAtCapacity verifies refill never overfills: after
// many ticks elapse, draining yields at most `capacity` tokens. Stop() freezes
// the refill goroutine first so the measurement is deterministic.
func TestTokenBucket_RefillCappedAtCapacity(t *testing.T) {
	const capacity = 2
	tb := NewTokenBucket(100, capacity) // fast refill (~10ms), tiny capacity

	time.Sleep(100 * time.Millisecond) // let many refill ticks fire
	tb.Stop()                          // freeze refills before counting

	got := 0
	for tb.TryAcquire() {
		got++
	}

	if got != capacity {
		t.Fatalf("drained %d tokens; want capacity=%d (refill must not exceed capacity)", got, capacity)
	}
}
