// Package limiter_test exercises the limiter package the way its users
// do: through the exported API only.
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

// TestZeroBurstStoresNoTokens pins the package's documented special
// case for burst 0: refills that arrive with nobody waiting vanish
// instead of accumulating, so Allow can never succeed and Wait
// succeeds only once a refill lands while it is blocked. It comes
// first because later tests lean on the refill cadence it establishes.
// The synctest bubble runs the one-second refill clock on virtual
// time, so the sleeps are free and the elapsed times are exact.
func TestZeroBurstStoresNoTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0)
		defer l.Close()

		if l.Allow() {
			t.Error("Allow() = true immediately after New, want false for burst 0")
		}

		// Two refills arrive with no waiter; burst 0 must drop both.
		time.Sleep(2500 * time.Millisecond)
		if l.Allow() {
			t.Error("Allow() = true after 2.5s of refills, want false for burst 0")
		}

		// The next refill is due half a second from now, at the 3s mark.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill arrives", err)
		}
		if elapsed := time.Since(start); elapsed != 500*time.Millisecond {
			t.Errorf("Wait returned after %v, want exactly 500ms (the next refill)", elapsed)
		}
	})
}

// TestNewPanicsOnInvalidArguments pins the constructor's documented
// failure mode: a rate that is not positive or a negative burst is a
// programming error worth a panic, not a limiter that silently
// misbehaves.
func TestNewPanicsOnInvalidArguments(t *testing.T) {
	cases := []struct {
		rate, burst int
		invalidity  string
	}{
		{0, 1, "zero rate"},
		{-1, 1, "negative rate"},
		{1, -1, "negative burst"},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic for a %s", c.rate, c.burst, c.invalidity)
				}
			}()
			l := limiter.New(c.rate, c.burst)
			l.Close() // reached only if the panic contract is broken; stop the goroutine anyway
		}()
	}
}

// TestBucketStartsFullAndRefillsUpToBurst walks the core token-bucket
// promise end to end: New hands out a full bucket, Allow consumes one
// token per call without blocking, and idle refills accumulate again
// only up to burst, so a long quiet stretch buys no extra tokens.
func TestBucketStartsFullAndRefillsUpToBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 2)
		defer l.Close()

		// The bucket starts full: exactly two tokens, then empty.
		if !l.Allow() {
			t.Error("Allow() #1 = false on a fresh limiter, want true")
		}
		if !l.Allow() {
			t.Error("Allow() #2 = false, want true for burst 2")
		}
		if l.Allow() {
			t.Error("Allow() #3 = true, want false once the bucket is drained")
		}

		// Four refills against a bucket that holds two.
		time.Sleep(4500 * time.Millisecond)
		if !l.Allow() {
			t.Error("Allow() = false after refills, want true")
		}
		if !l.Allow() {
			t.Error("Allow() = false on the second refilled token, want true")
		}
		if l.Allow() {
			t.Error("Allow() = true beyond burst, want false: extra refills must not accumulate")
		}
	})
}

// TestWaitBlocksUntilRefill pins Wait's happy path: on an empty bucket
// it parks instead of failing, wakes with the very next refill, and
// consumes the token it was woken for. With one token per second and
// the bucket drained at time zero, that refill lands exactly one
// second in.
func TestWaitBlocksUntilRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 1)
		defer l.Close()

		// Drain the single starting token so Wait has to block.
		if !l.Allow() {
			t.Fatal("Allow() = false on a fresh limiter, want true")
		}

		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil", err)
		}
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Errorf("Wait returned after %v, want exactly 1s (the first refill)", elapsed)
		}

		// Wait consumed the refilled token; nothing is left over.
		if l.Allow() {
			t.Error("Allow() = true right after Wait, want false: Wait consumed the token")
		}
	})
}

// TestWaitReturnsContextErrorWhenDoneFirst pins the caller's escape
// hatch: a Wait that cannot be served before its context is done gives
// back that context's own error, whether the deadline expires mid-wait
// or the context was already canceled on entry.
func TestWaitReturnsContextErrorWhenDoneFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 1)
		defer l.Close()

		// Empty the bucket; the next refill is a full second away.
		if !l.Allow() {
			t.Fatal("Allow() = false on a fresh limiter, want true")
		}

		// A 100ms deadline expires well before that refill.
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait() = %v, want context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(start); elapsed != 100*time.Millisecond {
			t.Errorf("Wait returned after %v, want exactly the 100ms deadline", elapsed)
		}

		// A context that is already done fails Wait without blocking.
		canceled, cancelNow := context.WithCancel(t.Context())
		cancelNow()
		if err := l.Wait(canceled); !errors.Is(err, context.Canceled) {
			t.Errorf("Wait(canceled ctx) = %v, want context.Canceled", err)
		}
	})
}

// TestWaitAfterCloseReturnsErrClosed pins the shutdown contract: once
// Close has returned, every Wait call reports ErrClosed, even while
// unused tokens still sit in the bucket, so a closed limiter admits no
// stragglers.
func TestWaitAfterCloseReturnsErrClosed(t *testing.T) {
	l := limiter.New(1, 2)
	l.Close()

	// The two starting tokens are still in the bucket; ErrClosed wins anyway.
	if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
		t.Errorf("Wait() after Close = %v, want ErrClosed", err)
	}
	// "Every subsequent call": a second Wait fails the same way.
	if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
		t.Errorf("second Wait() after Close = %v, want ErrClosed", err)
	}
}

// Example throttles a burst of three requests with a bucket that holds
// two tokens, then shuts the limiter down. The deferred Close and the
// explicit one together show that Close is idempotent, and the final
// Wait shows a closed limiter reporting ErrClosed.
func Example() {
	l := limiter.New(1, 2)
	defer l.Close()

	fmt.Println(l.Allow())
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())

	l.Close()
	fmt.Println(errors.Is(l.Wait(context.Background()), limiter.ErrClosed))

	// Output:
	// true
	// true
	// false
	// true
}
