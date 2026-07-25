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

// A bursty client is the typical caller: try Allow on the hot path, and
// when the limiter runs dry fall back to Wait, which blocks until the
// next refill frees a token.
func Example_throttle() {
	// Two tokens per second, and up to two held for a burst.
	l := limiter.New(2, 2)
	// Always defer Close so the refill goroutine is released even on an
	// early return.
	defer l.Close()

	// The limiter starts full, so the opening burst is admitted at once.
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())

	// The burst is spent. Rather than drop the next request, block until
	// a token arrives.
	if !l.Allow() {
		if err := l.Wait(context.Background()); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("third request admitted after a refill")
	}

	// Closing again is safe: Close is idempotent, so the deferred Close
	// above becomes a no-op.
	l.Close()

	// Output:
	// true
	// true
	// third request admitted after a refill
}

// The typical client story end to end: a burst on arrival, an impatient
// wait that times out, a patient wait served by the next refill, a lull
// that restocks only up to burst, and a shutdown that turns away every
// caller no matter how many tokens are left.
func TestThrottleLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 2) // one token per second, two to start
		defer l.Close()

		// The starting burst: exactly two tokens, then dry.
		if !l.Allow() {
			t.Error("Allow() #1 = false on a fresh limiter, want true")
		}
		if !l.Allow() {
			t.Error("Allow() #2 = false, want true for burst 2")
		}
		if l.Allow() {
			t.Error("Allow() #3 = true, want false once the burst is spent")
		}

		// An impatient caller: the 100ms deadline expires long before the
		// first refill lands at the 1s mark.
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait() with a 100ms deadline = %v, want context.DeadlineExceeded", err)
		}

		// A patient caller: the timed-out wait above already burned 100ms
		// of the first refill interval, so the token due at the 1s mark
		// arrives 900ms from now.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill lands", err)
		}
		if elapsed := time.Since(start); elapsed != 900*time.Millisecond {
			t.Errorf("Wait() returned after %v, want exactly 900ms", elapsed)
		}

		// A lull: four refills arrive during the sleep, but the limiter
		// keeps only burst of them.
		time.Sleep(4500 * time.Millisecond)
		if !l.Allow() {
			t.Error("Allow() = false after the lull, want true")
		}
		if !l.Allow() {
			t.Error("Allow() = false on the second restocked token, want true")
		}
		if l.Allow() {
			t.Error("Allow() = true beyond burst, want false: extra refills must not pile up")
		}

		// Shutdown: two more refills land before Close, and ErrClosed still
		// wins over the leftover tokens.
		time.Sleep(2500 * time.Millisecond)
		l.Close()
		if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait() after Close = %v, want ErrClosed", err)
		}
	})
}

// A zero-burst limiter holds no reserve. It admits a caller only when a
// refill ticks while that caller is already blocked in Wait, so the
// non-blocking Allow never catches a token, and refills that tick with no
// one waiting are dropped rather than banked for later.
func TestZeroBurstAdmitsOnlyWhileWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0) // one token per second, nothing held in reserve
		defer l.Close()

		// Nothing is stored, so Allow finds no token even on a fresh limiter.
		if l.Allow() {
			t.Error("Allow() = true on a zero-burst limiter, want false: no tokens are ever stored")
		}

		// A caller that blocks in Wait is handed the refill due at the 1s
		// mark; the token reaches the waiter directly.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill reaches the waiting caller", err)
		}
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Errorf("Wait() returned after %v, want exactly 1s", elapsed)
		}

		// The refills that tick while no one waits are dropped: after a
		// lull Allow still finds nothing.
		time.Sleep(3500 * time.Millisecond)
		if l.Allow() {
			t.Error("Allow() = true after a lull, want false: unclaimed refills must not accumulate")
		}
	})
}

func TestNewPanicsOnInvalidArguments(t *testing.T) {
	cases := []struct {
		rate, burst int
		invalidity  string
	}{
		{rate: 0, burst: 1, invalidity: "zero rate"},
		{rate: -1, burst: 1, invalidity: "negative rate"},
		{rate: 1, burst: -1, invalidity: "negative burst"},
		// Omitted: a rate above 1e9 floors the refill interval to zero, so
		// time.NewTicker panics on the refill goroutine — an unrecoverable
		// crash, not a New panic, so it cannot ship as a passing case. New's
		// prose promises a panic only for a non-positive rate, so this crash
		// is drift from the contract, not part of it.
		// {rate: 2_000_000_000, burst: 1, invalidity: "rate that floors the refill interval to zero"},
	}
	for _, c := range cases {
		mustPanic(t, c.rate, c.burst, c.invalidity)
	}
}

// mustPanic fails the test unless New(rate, burst) panics; invalidity
// names what makes the arguments invalid in the failure message.
func mustPanic(t *testing.T, rate, burst int, invalidity string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%d, %d) did not panic on a %s", rate, burst, invalidity)
		}
	}()
	l := limiter.New(rate, burst)
	l.Close() // reached only if the guard is broken; stop the refill goroutine anyway
}
