package limiter_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/limiter"
)

// A client that hits its ceiling falls back to Wait rather than dropping
// the request: Allow never blocks, and Wait always eventually admits the
// caller or reports why it gave up.
func Example_throttle() {
	// Four requests admitted per second, with two more held in reserve
	// for a burst.
	l := limiter.New(4, 2)
	// Close releases the refill goroutine; deferring it here covers any
	// early return below.
	defer l.Close()

	// A fresh limiter starts full, so both reserved tokens are spent at
	// once.
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())

	// The third request finds the limiter dry. Wait blocks until the next
	// refill instead of turning the caller away.
	if !l.Allow() {
		if err := l.Wait(context.Background()); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("admitted once the limiter refills")
	}

	// The deferred Close above becomes a no-op: calling Close a second
	// time is safe.
	l.Close()

	// Output:
	// true
	// true
	// admitted once the limiter refills
}

// The typical client story end to end: a burst on arrival, an impatient
// wait that gives up, a patient wait served by the following refill, a
// lull that restocks only up to burst, and a shutdown that turns away
// every caller no matter how many tokens are left.
func TestThrottleLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(5, 3) // 200ms between refills, three to start
		defer l.Close()

		// The starting burst: exactly three tokens, then dry.
		if !l.Allow() {
			t.Error("Allow() #1 = false on a fresh limiter, want true")
		}
		if !l.Allow() {
			t.Error("Allow() #2 = false, want true for burst 3")
		}
		if !l.Allow() {
			t.Error("Allow() #3 = false, want true for burst 3")
		}
		if l.Allow() {
			t.Error("Allow() #4 = true, want false once the burst is spent")
		}

		// An impatient caller: the 120ms deadline expires well short of
		// the 200ms refill interval.
		ctx, cancel := context.WithTimeout(t.Context(), 120*time.Millisecond)
		defer cancel()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait() with a 120ms deadline = %v, want context.DeadlineExceeded", err)
		}

		// A patient caller: the wait above already burned 120ms of the
		// 200ms interval, so the next refill lands 80ms from now.
		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill lands", err)
		}
		if elapsed := time.Since(start); elapsed != 80*time.Millisecond {
			t.Errorf("Wait() returned after %v, want exactly 80ms", elapsed)
		}

		// A lull: four more refills arrive during the sleep, but the
		// limiter keeps only burst of them.
		time.Sleep(900 * time.Millisecond)
		if !l.Allow() {
			t.Error("Allow() = false after the lull, want true")
		}
		if !l.Allow() {
			t.Error("Allow() = false on the second restocked token, want true")
		}
		if !l.Allow() {
			t.Error("Allow() = false on the third restocked token, want true")
		}
		if l.Allow() {
			t.Error("Allow() = true beyond burst, want false: extra refills must not pile up")
		}

		// Shutdown: several more refills land unclaimed before Close,
		// and ErrClosed still wins over that leftover stock.
		time.Sleep(700 * time.Millisecond)
		l.Close()
		if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait() after Close = %v, want ErrClosed", err)
		}
	})
}

// The package doc calls out burst 0 as a distinct contract: nothing is
// ever stored, so a refill only reaches a caller already parked in Wait.
func TestZeroBurstOnlyAdmitsAnAlreadyWaitingCaller(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0) // one token per second, no storage
		defer l.Close()

		if l.Allow() {
			t.Error("Allow() = true on a fresh zero-burst limiter, want false")
		}

		// A refill with nobody parked in Wait has nowhere to go: the send
		// inside refill is non-blocking, so the token is simply dropped.
		time.Sleep(time.Second)
		if l.Allow() {
			t.Error("Allow() = true after an unclaimed refill, want false")
		}

		// A caller already blocked in Wait is a ready receiver, so the
		// very next refill reaches it directly.
		if err := l.Wait(t.Context()); err != nil {
			t.Errorf("Wait() = %v, want nil once the next refill lands", err)
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

// A rate above one token per nanosecond is invalid input New still
// accepts: it floors the refill interval to zero, and the background
// refill goroutine crashes on that zero interval instead of New itself
// panicking the way the cases above do. The crash lands on a goroutine
// this test does not own, so recover here cannot catch it; a subprocess
// is the only way to observe it without taking the rest of the suite
// down too.
func TestNewCrashesOnRateAboveOnePerNanosecond(t *testing.T) {
	const childEnvVar = "LIMITER_TEST_TRIGGER_CRASH"
	if os.Getenv(childEnvVar) == "1" {
		limiter.New(2_000_000_000, 1)
		// Give the refill goroutine a chance to run and crash before
		// this process exits on its own.
		time.Sleep(200 * time.Millisecond)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), childEnvVar+"=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess constructing New(2_000_000_000, 1) exited cleanly, want it to crash; output:\n%s", output)
	}
	if !strings.Contains(string(output), "panic:") {
		t.Fatalf("subprocess exited with %v but did not panic; output:\n%s", err, output)
	}
}
