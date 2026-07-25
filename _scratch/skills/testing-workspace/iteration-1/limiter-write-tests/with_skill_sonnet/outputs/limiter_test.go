package limiter_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/limiter"
)

// The token bucket starts full, Allow drains it without blocking, and
// refills arrive at the configured rate up to the burst capacity.
func TestLimiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := limiter.New(1, 2) // 1 token/sec, burst of 2
		defer lim.Close()

		if !lim.Allow() {
			t.Fatal("Allow() = false, want true: the bucket starts full")
		}
		if !lim.Allow() {
			t.Fatal("Allow() = false, want true: the bucket starts full")
		}
		if lim.Allow() {
			t.Fatal("Allow() = true, want false once the burst is exhausted")
		}

		// Just under one interval: no refill has arrived yet.
		time.Sleep(999 * time.Millisecond)
		synctest.Wait()
		if lim.Allow() {
			t.Fatal("Allow() = true before the refill interval elapsed")
		}

		// One interval elapses: exactly one token refills.
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if !lim.Allow() {
			t.Fatal("Allow() = false, want true after one refill interval")
		}
		if lim.Allow() {
			t.Fatal("Allow() = true, want false: only one token should have refilled")
		}
	})
}

// Wait blocks a caller until a token becomes available from a
// refill, then returns nil. With a zero burst the bucket stores no
// tokens of its own, so the only way Wait can succeed is by catching
// a refill while already waiting.
func TestLimiterWaitBlocksUntilRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := limiter.New(1, 0)
		defer lim.Close()

		if lim.Allow() {
			t.Fatal("Allow() = true, want false: burst 0 stores no tokens")
		}

		waited := make(chan error, 1)
		go func() {
			waited <- lim.Wait(context.Background())
		}()
		synctest.Wait()

		select {
		case err := <-waited:
			t.Fatalf("Wait() returned %v before any token refilled", err)
		default:
		}

		time.Sleep(time.Second)
		synctest.Wait()

		select {
		case err := <-waited:
			if err != nil {
				t.Fatalf("Wait() = %v, want nil once a token refilled", err)
			}
		default:
			t.Fatal("Wait did not return after a token refilled")
		}
	})
}

// Wait reports the context's error when the context is done before a
// token arrives.
func TestLimiterWaitContextCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := limiter.New(1, 0)
		defer lim.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := lim.Wait(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait(canceled ctx) = %v, want context.Canceled", err)
		}
	})
}

// Once Close has returned, every subsequent Wait call reports
// ErrClosed instead of blocking, and Close itself is safe to call
// more than once.
func TestLimiterCloseStopsWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := limiter.New(1, 0)

		lim.Close()
		lim.Close()

		if err := lim.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
			t.Fatalf("Wait() after Close = %v, want ErrClosed", err)
		}
	})
}

// New validates its arguments: rate must be positive and burst must
// not be negative.
func TestLimiterNewPanics(t *testing.T) {
	tests := []struct {
		name        string
		rate, burst int
	}{
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"negative burst", 1, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("New did not panic")
				}
			}()
			lim := limiter.New(tt.rate, tt.burst)
			lim.Close()
		})
	}
}
