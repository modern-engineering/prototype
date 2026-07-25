package limiter_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/limiter"
)

// TestLimiter walks the typical gateway flow: a full bucket admits its burst
// at once, then further events pace out to the refill rate.
func TestLimiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// rate 2 => one refill every 500ms; burst 2 => the bucket starts
		// with two tokens.
		l := limiter.New(2, 2)
		defer l.Close()

		// The bucket starts full: two events are admitted immediately, and
		// the third is refused because Allow never blocks.
		if !l.Allow() {
			t.Fatal("Allow denied the first token from a full bucket")
		}
		if !l.Allow() {
			t.Fatal("Allow denied the second token from a full bucket")
		}
		if l.Allow() {
			t.Fatal("Allow admitted a third token from a bucket of burst 2")
		}

		// With the bucket drained, Wait blocks until the refill goroutine
		// delivers a token one interval later. Virtual time makes the
		// interval exact.
		start := time.Now()
		if err := l.Wait(context.Background()); err != nil {
			t.Fatalf("Wait on a drained bucket returned %v, want nil", err)
		}
		if elapsed := time.Since(start); elapsed != 500*time.Millisecond {
			t.Errorf("Wait unblocked after %v, want one 500ms refill interval", elapsed)
		}
	})
}

// TestBurstZeroAdmitsOnlyOnRefill pins the documented burst-0 behavior: no
// token is ever stored, so Allow can never succeed and Wait proceeds only when
// a refill meets a waiter.
func TestBurstZeroAdmitsOnlyOnRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// rate 1 => one refill per second; burst 0 => the bucket holds
		// nothing.
		l := limiter.New(1, 0)
		defer l.Close()

		if l.Allow() {
			t.Fatal("Allow admitted a token from a bucket of burst 0")
		}

		start := time.Now()
		if err := l.Wait(context.Background()); err != nil {
			t.Fatalf("Wait returned %v, want nil once a refill arrived", err)
		}
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Errorf("Wait unblocked after %v, want one 1s refill interval", elapsed)
		}
	})
}

// TestWaitHonorsContextDeadline checks that a deadline reached before the next
// refill surfaces the context error rather than a token.
func TestWaitHonorsContextDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// rate 1 => the first refill is a second away; the 500ms deadline
		// fires first.
		l := limiter.New(1, 0)
		defer l.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err := l.Wait(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait past its deadline returned %v, want context.DeadlineExceeded", err)
		}
	})
}

// TestWaitHonorsContextCancellation checks that cancelling the context unblocks
// a waiting caller with the context error.
func TestWaitHonorsContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0)
		defer l.Close()

		ctx, cancel := context.WithCancel(context.Background())
		errc := make(chan error, 1)
		go func() { errc <- l.Wait(ctx) }()

		// Let the waiter reach its blocking select before cancelling, so the
		// cancellation is observed mid-wait.
		synctest.Wait()
		cancel()

		if err := <-errc; !errors.Is(err, context.Canceled) {
			t.Errorf("Wait on a cancelled context returned %v, want context.Canceled", err)
		}
	})
}

// TestWaitAfterCloseReturnsErrClosed checks that a closed limiter refuses every
// Wait, even while a token is still buffered.
func TestWaitAfterCloseReturnsErrClosed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 1) // full bucket: a token is available
		l.Close()

		if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait after Close returned %v, want ErrClosed", err)
		}
	})
}

// TestCloseIsIdempotent checks that closing more than once is safe.
func TestCloseIsIdempotent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 1)
		l.Close()
		l.Close() // a second Close must not panic on the closed done channel
	})
}

// TestNewRejectsInvalidArguments checks that New validates its arguments before
// starting the refill goroutine, panicking on each invalid combination.
func TestNewRejectsInvalidArguments(t *testing.T) {
	cases := []struct {
		name        string
		rate, burst int
	}{
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"negative burst", 1, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic", c.rate, c.burst)
				}
			}()
			limiter.New(c.rate, c.burst)
		})
	}
}
