// Black-box tests for package limiter.
//
// Timing-sensitive tests assert only lower bounds (a ticker never fires
// early) or use generous deadlines, so they stay stable on loaded CI
// machines.
package limiter_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.invalid/limiter"
)

// mustPanic asserts that fn panics.
func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected panic, got none")
		}
	}()
	fn()
}

func TestNewValidation(t *testing.T) {
	t.Run("zero rate", func(t *testing.T) {
		mustPanic(t, func() { limiter.New(0, 1) })
	})
	t.Run("negative rate", func(t *testing.T) {
		mustPanic(t, func() { limiter.New(-1, 1) })
	})
	t.Run("negative burst", func(t *testing.T) {
		mustPanic(t, func() { limiter.New(1, -1) })
	})
	t.Run("zero burst is allowed", func(t *testing.T) {
		l := limiter.New(1, 0)
		l.Close()
	})
}

// The bucket starts full: exactly burst Allow calls succeed before the
// first refill. rate=1 keeps the first refill a full second away, far
// beyond the test's runtime.
func TestBucketStartsFull(t *testing.T) {
	const burst = 3
	l := limiter.New(1, burst)
	defer l.Close()

	for i := range burst {
		if !l.Allow() {
			t.Fatalf("Allow() #%d = false, want true (bucket should start full)", i+1)
		}
	}
	if l.Allow() {
		t.Fatalf("Allow() #%d = true, want false (bucket should be empty)", burst+1)
	}
}

// With burst 0 the limiter stores no tokens, so Allow must always
// report false.
func TestAllowZeroBurst(t *testing.T) {
	l := limiter.New(1, 0)
	defer l.Close()

	for range 10 {
		if l.Allow() {
			t.Fatal("Allow() = true, want false for a zero-burst limiter")
		}
	}
}

// Wait consumes an available token without blocking on the refill.
func TestWaitWithAvailableToken(t *testing.T) {
	l := limiter.New(1, 1)
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait() = %v, want nil", err)
	}
}

// A zero-burst limiter serves Wait only from refill ticks, so n
// sequential waits cannot complete before n ticker intervals have
// elapsed. Only the lower bound is asserted; the upper bound is the
// generous context deadline.
func TestWaitPacedByRefill(t *testing.T) {
	const (
		rate     = 100 // one token per 10ms
		waits    = 3
		interval = time.Second / rate
	)
	start := time.Now()
	l := limiter.New(rate, 0)
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := range waits {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait() #%d = %v, want nil", i+1, err)
		}
	}
	// Allow a small margin below the theoretical minimum for clock
	// granularity.
	if min := waits*interval - 5*time.Millisecond; time.Since(start) < min {
		t.Fatalf("%d waits completed in %v, want at least %v", waits, time.Since(start), min)
	}
}

// Wait returns ctx.Err() when the context expires before a token
// arrives. rate=1 keeps the first refill (1s) far beyond the 50ms
// deadline.
func TestWaitContextDeadline(t *testing.T) {
	l := limiter.New(1, 0)
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait() = %v, want %v", err, context.DeadlineExceeded)
	}
}

// Wait returns ctx.Err() for a context that is already cancelled when
// no token is available.
func TestWaitPreCancelledContext(t *testing.T) {
	l := limiter.New(1, 0)
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait() = %v, want %v", err, context.Canceled)
	}
}

// After Close returns, Wait reports ErrClosed even while unused tokens
// remain in the bucket.
func TestWaitAfterClose(t *testing.T) {
	l := limiter.New(1, 2)
	l.Close()

	for i := range 3 {
		if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
			t.Fatalf("Wait() #%d after Close = %v, want %v", i+1, err, limiter.ErrClosed)
		}
	}
}

// Close wakes a Wait that is already blocked on the refill.
func TestCloseWakesBlockedWait(t *testing.T) {
	l := limiter.New(1, 0)

	errc := make(chan error, 1)
	go func() { errc <- l.Wait(context.Background()) }()

	time.Sleep(10 * time.Millisecond) // let Wait park in its select
	l.Close()

	select {
	case err := <-errc:
		if !errors.Is(err, limiter.ErrClosed) {
			t.Fatalf("Wait() = %v, want %v", err, limiter.ErrClosed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not return within 2s of Close")
	}
}

func TestCloseIdempotent(t *testing.T) {
	l := limiter.New(1, 1)
	l.Close()
	l.Close() // must not panic

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(l.Close) // concurrent Close must be safe too
	}
	wg.Wait()
}

// Pins current behavior: Allow keeps handing out tokens left in the
// bucket after Close, while Wait refuses with ErrClosed. The two
// methods disagree about a closed limiter; see report for the
// maintainer question.
func TestAllowAfterCloseDrainsLeftoverTokens(t *testing.T) {
	l := limiter.New(1, 2)
	l.Close()

	for i := range 2 {
		if !l.Allow() {
			t.Fatalf("Allow() #%d after Close = false, want true (leftover token)", i+1)
		}
	}
	if l.Allow() {
		t.Fatal("Allow() = true after leftover tokens were drained, want false")
	}
}

// The bucket never holds more than burst tokens, no matter how many
// refill intervals pass.
func TestRefillCapsAtBurst(t *testing.T) {
	const burst = 2
	l := limiter.New(1000, burst) // 1ms interval: many refills during the sleep
	for i := range burst {
		if !l.Allow() {
			t.Fatalf("Allow() #%d = false, want true", i+1)
		}
	}

	time.Sleep(25 * time.Millisecond) // dozens of refill ticks
	l.Close()
	// One refill send may still be in flight when Close returns. Let it
	// resolve against the full bucket (where it is dropped) before
	// counting, so the count is deterministic.
	time.Sleep(10 * time.Millisecond)

	count := 0
	for l.Allow() {
		count++
	}
	if count != burst {
		t.Fatalf("bucket held %d tokens after refills, want exactly %d (burst)", count, burst)
	}
}

// Concurrent Allow calls hand out each token exactly once. Close is
// called first so the count cannot drift from refills: with rate=1 the
// first tick is a second away, and after Close no tick is delivered.
func TestConcurrentAllowExactlyBurst(t *testing.T) {
	const (
		burst      = 64
		goroutines = 8
		attempts   = 16 // per goroutine; 128 attempts for 64 tokens
	)
	l := limiter.New(1, burst)
	l.Close()

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range attempts {
				if l.Allow() {
					allowed.Add(1)
				}
			}
		})
	}
	wg.Wait()

	if got := allowed.Load(); got != burst {
		t.Fatalf("concurrent Allow succeeded %d times, want exactly %d", got, burst)
	}
}

// Waiters racing with Close must each either obtain a token or observe
// ErrClosed; nothing may hang or return an unexpected error. Primarily
// a race-detector target.
func TestConcurrentWaitAndClose(t *testing.T) {
	l := limiter.New(200, 4)

	const waiters = 8
	errs := make(chan error, waiters)
	for range waiters {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			errs <- l.Wait(ctx)
		}()
	}

	time.Sleep(20 * time.Millisecond)
	l.Close()

	for i := range waiters {
		select {
		case err := <-errs:
			if err != nil && !errors.Is(err, limiter.ErrClosed) {
				t.Fatalf("Wait() = %v, want nil or %v", err, limiter.ErrClosed)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("waiter %d did not return after Close", i+1)
		}
	}
}
