package limiter_test

import (
	"bytes"
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

// A gateway handler is the typical caller: try Allow on the hot path, and
// when the limiter is dry fall back to Wait, which blocks until the next
// refill frees a token.
func Example_throttle() {
	// Two tokens per second, and up to two held for a burst.
	l := limiter.New(2, 2)
	// Always defer Close so the refill goroutine is released even on an
	// early return.
	defer l.Close()

	// The limiter starts full, so the opening burst is admitted at once.
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())

	// The burst is spent. Rather than reject the next request, block
	// until a token arrives.
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

		// An impatient caller: the 100ms deadline expires long before
		// the first refill lands at the 1s mark.
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait() with a 100ms deadline = %v, want context.DeadlineExceeded", err)
		}

		// A patient caller: the timed-out wait above already burned
		// 100ms of the first refill interval, so the token due at the
		// 1s mark arrives 900ms from now.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill lands", err)
		}
		if elapsed := time.Since(start); elapsed != 900*time.Millisecond {
			t.Errorf("Wait() returned after %v, want exactly 900ms", elapsed)
		}

		// A lull: four refills arrive during the sleep, but the
		// limiter keeps only burst of them.
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

		// Shutdown: two more refills land before Close, and
		// ErrClosed still wins over the leftover tokens.
		time.Sleep(2500 * time.Millisecond)
		l.Close()
		if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait() after Close = %v, want ErrClosed", err)
		}
	})
}

// A limiter with no storage still lets a caller through: the doc promises
// the token bypasses storage entirely and goes straight to whoever is
// already parked in Wait. A mere probe through Allow always misses it,
// and so does a refill that lands with nobody parked: it must vanish
// rather than sit ready for the next Allow.
func TestZeroBurstHandsTokensOnlyToAWaitingCaller(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0) // one token per second, none stored
		defer l.Close()

		if l.Allow() {
			t.Error("Allow() = true on a zero-burst limiter, want false: it never stores a token")
		}

		// Let a refill land with nobody in Wait to claim it.
		time.Sleep(1200 * time.Millisecond)
		if l.Allow() {
			t.Error("Allow() = true after an unclaimed refill, want false: zero burst must drop it, not buffer it for the next Allow")
		}

		if err := l.Wait(t.Context()); err != nil {
			t.Errorf("Wait() = %v, want nil: a refill hands its token straight to a caller already waiting", err)
		}

		if l.Allow() {
			t.Error("Allow() = true right after Wait consumed the refill, want false")
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

// New accepts a rate this large: nothing in the doc rules it out. But it
// floors the refill interval (time.Second/rate) to zero, and a zero
// interval panics inside the background goroutine New starts, past any
// recover a caller could install, so it crashes the whole process rather
// than a single call. Asserted in a subprocess so that crash lands on the
// subprocess, not on this test binary.
func TestNewCrashesOnARateThatFloorsTheRefillInterval(t *testing.T) {
	if os.Getenv("LIMITER_TEST_CRASH_HELPER") == "1" {
		l := limiter.New(2_000_000_000, 1) // 2e9/s: time.Second/rate floors to 0
		defer l.Close()
		time.Sleep(time.Second) // give the background goroutine time to panic
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestNewCrashesOnARateThatFloorsTheRefillInterval$")
	cmd.Env = append(os.Environ(), "LIMITER_TEST_CRASH_HELPER=1")
	output, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.Success() {
		t.Fatalf("subprocess exited %v, want a non-zero exit from the background goroutine's panic; output:\n%s", err, output)
	}
	if !bytes.Contains(output, []byte("panic:")) {
		t.Errorf("subprocess exit carried no panic trace; output:\n%s", output)
	}
}
