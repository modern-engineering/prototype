package limiter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/limiter"
)

// A new limiter starts with a full bucket: Allow hands out one token
// per call without blocking until the bucket is empty, and one token
// refills per interval at the configured rate.
func TestAllowDrainsBurstThenRefills(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 2)
		defer l.Close()

		for i := range 2 {
			if !l.Allow() {
				t.Fatalf("Allow() = false for token %d of a full burst-2 bucket, want true", i+1)
			}
		}
		if l.Allow() {
			t.Fatal("Allow() = true on an empty bucket, want false")
		}

		time.Sleep(time.Second) // one refill interval at rate 1
		synctest.Wait()
		if !l.Allow() {
			t.Fatal("Allow() = false one refill interval after draining, want true")
		}
		if l.Allow() {
			t.Fatal("Allow() = true before the next refill, want false")
		}
	})
}

// The bucket holds at most burst unused tokens; refills beyond that
// capacity are dropped, so an idle limiter never exceeds its burst.
func TestBucketCapsAtBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(100, 2)
		defer l.Close()

		time.Sleep(time.Second) // 100 refill intervals pass unused
		synctest.Wait()
		for i := range 2 {
			if !l.Allow() {
				t.Fatalf("Allow() = false for token %d of a burst-2 bucket, want true", i+1)
			}
		}
		if l.Allow() {
			t.Fatal("Allow() = true for a third token from a burst-2 bucket, want false")
		}
	})
}

// With burst 0 the limiter stores no tokens: Allow always reports
// false, and Wait proceeds only when a refill arrives while the caller
// is already blocked, one interval after New.
func TestWaitBlocksUntilRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0)
		defer l.Close()

		if l.Allow() {
			t.Fatal("Allow() = true with burst 0, want false")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		start := time.Now()
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait() = %v, want nil", err)
		}
		if d := time.Since(start); d != time.Second {
			t.Fatalf("Wait returned after %v, want one refill interval (1s)", d)
		}
	})
}

// When the context is done before a token becomes available, Wait
// returns the context's error.
func TestWaitHonorsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0)
		defer l.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait() = %v, want %v", err, context.DeadlineExceeded)
		}
	})
}

// Close is idempotent, and once it has returned every Wait call
// reports ErrClosed even while tokens remain in the bucket.
func TestCloseStopsWait(t *testing.T) {
	l := limiter.New(1, 1) // starts full, so one token remains after Close
	l.Close()
	l.Close() // second call must be a no-op

	if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
		t.Fatalf("Wait() after Close = %v, want %v", err, limiter.ErrClosed)
	}
}

// New rejects a non-positive rate and a negative burst by panicking.
func TestNewPanicsOnBadArguments(t *testing.T) {
	for _, tt := range []struct {
		name        string
		rate, burst int
	}{
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"negative burst", 1, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic", tt.rate, tt.burst)
				}
			}()
			limiter.New(tt.rate, tt.burst)
		})
	}
}

func ExampleLimiter_Allow() {
	l := limiter.New(1, 2) // 1 token per second, burst of 2; the bucket starts full
	defer l.Close()

	fmt.Println(l.Allow())
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())
	// Output:
	// true
	// true
	// false
}
