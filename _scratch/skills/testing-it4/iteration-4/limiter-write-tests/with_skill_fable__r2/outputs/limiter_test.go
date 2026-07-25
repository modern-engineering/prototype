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

// A gateway handler is the typical caller: try Allow on the hot path,
// and when the bucket runs dry fall back to Wait, which blocks until
// the next refill frees a token.
func Example_throttle() {
	// Two tokens per second, and up to two held for a burst.
	l := limiter.New(2, 2)
	// Always defer Close so the refill goroutine is released even on an
	// early return.
	defer l.Close()

	// The bucket starts full, so the opening burst is admitted at once.
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
func TestLimiterLifecycle(t *testing.T) {
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

		// A lull: four refills arrive during the sleep, but the bucket
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

		// Shutdown: two more refills land before Close, and ErrClosed
		// still wins over the leftover tokens.
		time.Sleep(2500 * time.Millisecond)
		l.Close()
		if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait() after Close = %v, want ErrClosed", err)
		}
	})
}

// With burst 0 there is nowhere to keep a token, so the package doc
// promises a pure pacing mode: Allow never admits, refills that find no
// waiter evaporate, and a caller gets through only by already being
// parked in Wait when one lands.
func TestBurstZeroStoresNoTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0)
		defer l.Close()

		if l.Allow() {
			t.Error("Allow() = true on a fresh burst-0 limiter, want false")
		}

		// Two refills land during the sleep with nobody waiting; a
		// burst-0 limiter must drop them rather than store them.
		time.Sleep(2500 * time.Millisecond)
		if l.Allow() {
			t.Error("Allow() = true after unclaimed refills, want false: burst 0 stores no tokens")
		}

		// A parked waiter is served by the next refill, due at the 3s
		// mark, 500ms from now.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill lands", err)
		}
		if elapsed := time.Since(start); elapsed != 500*time.Millisecond {
			t.Errorf("Wait() returned after %v, want exactly 500ms", elapsed)
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
		// FAILING (deliberate): New lets this rate through, the refill
		// interval floors to zero, and NewTicker(0) panics inside the
		// refill goroutine, crashing the whole process beyond any
		// caller's reach. Either New should reject rates above 1e9 or
		// the docs should bound rate; leaving the case in until the
		// author picks a side.
		{rate: 2_000_000_000, burst: 1, invalidity: "rate beyond one token per nanosecond"},
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
