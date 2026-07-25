package limiter

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"
)

func TestInvalidNew(t *testing.T) {
	for _, tt := range []struct {
		rate, burst int
	}{
		{0, 1},  // rate must be positive
		{-1, 1}, // rate must be positive
		{1, -1}, // burst must not be negative
	} {
		t.Run(fmt.Sprintf("New(%d, %d)", tt.rate, tt.burst), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic, want panic", tt.rate, tt.burst)
				}
			}()
			New(tt.rate, tt.burst)
		})
	}
}

// TestAllow exercises Allow's documented contract: the bucket starts full,
// Allow never blocks, and refilled tokens never accumulate past burst.
func TestAllow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New(1, 2) // 1 token/sec, burst 2
		defer l.Close()

		if !l.Allow() {
			t.Fatal("Allow() = false right after New, want true (bucket starts full)")
		}
		if !l.Allow() {
			t.Fatal("Allow() = false, want true (burst is 2)")
		}
		if l.Allow() {
			t.Fatal("Allow() = true, want false (bucket drained)")
		}

		// Three refill intervals pass while the bucket is empty; only
		// burst (2) tokens survive, the rest are dropped.
		time.Sleep(3 * time.Second)
		synctest.Wait()

		if !l.Allow() {
			t.Fatal("Allow() = false, want true (one refilled token)")
		}
		if !l.Allow() {
			t.Fatal("Allow() = false, want true (a second refilled token)")
		}
		if l.Allow() {
			t.Fatal("Allow() = true, want false (refills never exceed burst)")
		}
	})
}

// TestWaitBlocksUntilRefill pins Wait's documented contract: it blocks
// until a token is available, then consumes exactly one.
func TestWaitBlocksUntilRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New(1, 1) // 1 token/sec, burst 1
		defer l.Close()

		if !l.Allow() {
			t.Fatal("Allow() = false right after New, want true (bucket starts full)")
		}

		done := make(chan error, 1)
		go func() { done <- l.Wait(context.Background()) }()
		synctest.Wait()

		select {
		case err := <-done:
			t.Fatalf("Wait() returned early with %v, want blocked until refill", err)
		default:
		}

		time.Sleep(time.Second)
		synctest.Wait()

		if err := <-done; err != nil {
			t.Fatalf("Wait() = %v, want nil", err)
		}
		if l.Allow() {
			t.Fatal("Allow() = true right after Wait, want false (Wait consumed the token)")
		}
	})
}

func TestWaitReturnsContextError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New(1, 0) // no token ever available without a refill
		defer l.Close()

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- l.Wait(ctx) }()
		synctest.Wait()

		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait() = %v, want context.Canceled", err)
		}
	})
}

func TestWaitReturnsErrClosedAfterClose(t *testing.T) {
	l := New(1, 0)
	l.Close()

	if err := l.Wait(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Wait() after Close = %v, want ErrClosed", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	l := New(1, 1)
	l.Close()
	l.Close() // must not panic
}

// TestZeroBurst pins the package doc's promise for a limiter created with
// burst 0: Allow always reports false, and Wait proceeds only once a
// refill arrives while the caller is already waiting.
func TestZeroBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New(1, 0)
		defer l.Close()

		if l.Allow() {
			t.Fatal("Allow() = true for a zero-burst limiter, want false")
		}

		done := make(chan error, 1)
		go func() { done <- l.Wait(context.Background()) }()
		synctest.Wait()

		select {
		case err := <-done:
			t.Fatalf("Wait() returned early with %v, want blocked until refill", err)
		default:
		}

		time.Sleep(time.Second)
		synctest.Wait()

		if err := <-done; err != nil {
			t.Fatalf("Wait() = %v, want nil", err)
		}
	})
}
