package limiter_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/limiter"
)

// A bursty client is the typical caller: try Allow on the hot path, and
// when the bucket runs dry fall back to Wait, which blocks until the next
// refill frees a token.
func Example_throttle() {
	// Two tokens per second, and up to two held for a burst.
	l := limiter.New(2, 2)
	// Always defer Close so the refill goroutine is released even on an
	// early return.
	defer l.Close()

	// The bucket starts full, so the opening burst is admitted at once.
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())

	// The burst is spent. Rather than drop the next request, block until a
	// token arrives.
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

		// A patient caller: the timed-out wait above already burned 100ms of
		// the first refill interval, so the token due at the 1s mark arrives
		// 900ms from now.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill lands", err)
		}
		if elapsed := time.Since(start); elapsed != 900*time.Millisecond {
			t.Errorf("Wait() returned after %v, want exactly 900ms", elapsed)
		}

		// A lull: four refills arrive during the sleep, but the bucket keeps
		// only burst of them.
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

// Burst 0 is a mode the package doc singles out: the bucket banks nothing,
// so Allow is always false and Wait is served only by a refill that lands
// while the caller is already blocked on it. A refill that finds no waiter
// is dropped, never saved for later.
func TestZeroBurstServesOnlyWaiters(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(2, 0) // two refills per second, nothing retained
		defer l.Close()

		// Nothing is ever banked: not at construction, and not after refills
		// have come and gone with no waiter. The 1.2s pause lands between the
		// 1s and 1.5s ticks, so the 0.5s and 1s refills both drop while no one
		// is receiving, and Allow still finds the bucket empty.
		if l.Allow() {
			t.Error("Allow() = true on a fresh burst-0 limiter, want false: no token is stored")
		}
		time.Sleep(1200 * time.Millisecond)
		if l.Allow() {
			t.Error("Allow() = true after unwatched refills, want false: a refill with no waiter is dropped")
		}

		// A blocked caller is a receiver, so the next refill hands its token
		// straight over. The wait starts at 1.2s, so the token due at the
		// 1.5s tick arrives 300ms later.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill reaches the waiter", err)
		}
		if elapsed := time.Since(start); elapsed != 300*time.Millisecond {
			t.Errorf("Wait() returned after %v, want 300ms (the next refill tick)", elapsed)
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
	}
	for _, c := range cases {
		mustPanic(t, c.rate, c.burst, c.invalidity)
	}
}

// mustPanic fails the test unless New(rate, burst) panics; invalidity names
// what makes the arguments invalid in the failure message.
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

// TestNewRejectsIntervalFlooringRate documents a gap the limiter's own doc
// is silent about. New guards rate <= 0, but not a rate so large that
// time.Second/rate floors to a zero refill interval. time.NewTicker(0) then
// panics on the background refill goroutine, which no caller can recover, so
// the whole process dies instead of the caller getting a defensible panic
// from New (the way rate <= 0 already behaves). The zero-flooring rate is
// driven in a child process so the crash cannot take the rest of the suite
// down with it; the test fails until New rejects such a rate. See report.md.
func TestNewRejectsIntervalFlooringRate(t *testing.T) {
	if os.Getenv("LIMITER_CRASH_CHILD") == "1" {
		limiter.New(2_000_000_000, 1) // 1e9ns / 2e9 floors to a 0ns interval
		time.Sleep(time.Second)       // let the refill goroutine reach NewTicker
		return                        // reached only once New guards the rate
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNewRejectsIntervalFlooringRate$", "-test.v")
	cmd.Env = append(os.Environ(), "LIMITER_CRASH_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("New(2e9, 1) crashed the process instead of rejecting the rate: %v\n"+
			"a zero-flooring rate must panic in New (like rate <= 0), not on the refill goroutine\n%s", err, out)
	}
}
