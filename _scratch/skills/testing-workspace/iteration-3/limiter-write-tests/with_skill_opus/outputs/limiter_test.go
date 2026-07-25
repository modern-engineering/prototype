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

// ExampleLimiter shows the everyday shape of using a limiter: a fresh
// bucket starts full, so the opening burst is allowed at once; a drained
// bucket makes Allow report false without blocking; and Close is safe to
// call more than once, shown here both explicitly and through defer. A
// slow refill rate keeps the example deterministic, since no token can
// arrive during the handful of microseconds these calls take.
func ExampleLimiter() {
	l := limiter.New(1, 2) // 2 tokens to start, one new token per second
	defer l.Close()

	fmt.Println(l.Allow())
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())
	l.Close()

	// Output:
	// true
	// true
	// false
}

// TestZeroBurstStoresNoTokens pins the package doc's headline special
// case: a burst-0 limiter has nowhere to hold a token, so Allow can never
// succeed, not even after refills have had every chance to fire. This is
// foundational; the Wait-on-refill contract below leans on the same
// zero-capacity behavior. synctest lets the refill ticker fire on a
// virtual clock, so "after several ticks" is exact and instant.
func TestZeroBurstStoresNoTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(10, 0) // a refill tick every 100ms, but capacity 0
		defer l.Close()

		if l.Allow() {
			t.Fatal("Allow() = true on a fresh burst-0 limiter, want false")
		}

		time.Sleep(500 * time.Millisecond) // five refill ticks go by
		synctest.Wait()

		if l.Allow() {
			t.Error("Allow() = true after refills on a burst-0 limiter, want false")
		}
	})
}

// TestWaitBlocksUntilRefill pins two promises at once. Wait blocks until a
// token is available and then returns nil; and on a burst-0 limiter the
// only way a token appears is a refill arriving while the caller already
// waits. Because the bucket starts empty, the sole reason Wait can return
// is the first refill tick, so its return time also pins the refill rate:
// one token per 1/rate seconds. synctest advances the virtual clock the
// instant every goroutine is durably blocked, so the main goroutine can
// simply call Wait and let the fake clock reach the tick, no helper
// goroutine required.
func TestWaitBlocksUntilRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(10, 0) // empty bucket; refills once every 100ms
		defer l.Close()

		start := time.Now()
		if err := l.Wait(t.Context()); err != nil {
			t.Fatalf("Wait() = %v, want nil", err)
		}

		// A rate of 10/sec means the first token lands at exactly 100ms.
		if got := time.Since(start); got != 100*time.Millisecond {
			t.Errorf("Wait() returned after %v, want one refill interval of 100ms", got)
		}
	})
}

// TestWaitReturnsContextErrorBeforeRefill pins the promise that a context
// deadline reached before a token wins: Wait returns ctx.Err(). The
// deadline is set well inside the refill interval so it is unambiguously
// the first event to fire, and the check uses errors.Is rather than a
// string match so the package stays free to wrap the error.
func TestWaitReturnsContextErrorBeforeRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0) // a refill only once per second
		defer l.Close()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait() = %v, want context.DeadlineExceeded", err)
		}
	})
}

// TestWaitReturnsErrClosedAfterClose pins the promise that every Wait
// after Close returns ErrClosed. The limiter here still holds its starting
// token, which is the point: a closed limiter turns away every caller
// regardless of what tokens remain.
func TestWaitReturnsErrClosedAfterClose(t *testing.T) {
	l := limiter.New(10, 1)
	l.Close()

	if err := l.Wait(t.Context()); !errors.Is(err, limiter.ErrClosed) {
		t.Errorf("Wait() after Close = %v, want ErrClosed", err)
	}
}

// TestNewPanics pins New's argument guard. New reports invalid arguments
// with a panic rather than an error, so the contract is that no unusable
// limiter is ever handed back. Each case records why its arguments are
// invalid; that reason is printed if the expected panic fails to happen.
func TestNewPanics(t *testing.T) {
	cases := []struct {
		rate, burst int
		why         string
	}{
		{rate: 0, burst: 1, why: "rate must be positive"},
		{rate: -1, burst: 1, why: "rate must be positive"},
		{rate: 1, burst: -1, why: "burst must not be negative"},
	}
	for _, c := range cases {
		mustPanic(t, c.rate, c.burst, c.why)
	}
}

// mustPanic fails the test unless New(rate, burst) panics, naming the
// arguments and why they are invalid so a missing panic reads as a
// sentence.
func mustPanic(t *testing.T, rate, burst int, why string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%d, %d) did not panic, but %s", rate, burst, why)
		}
	}()
	limiter.New(rate, burst)
}
