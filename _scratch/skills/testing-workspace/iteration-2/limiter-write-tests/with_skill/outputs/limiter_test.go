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

// A gateway creates one limiter, gates each request with Allow or
// Wait, and releases the refill goroutine with Close on shutdown.
func Example() {
	l := limiter.New(100, 2)
	defer l.Close()

	fmt.Println("allowed:", l.Allow())
	fmt.Println("wait:", l.Wait(context.Background()))
	// Output:
	// allowed: true
	// wait: <nil>
}

func TestInvalidNew(t *testing.T) {
	// New promises to panic if rate is not positive or burst is negative.
	for _, args := range []struct{ rate, burst int }{
		{0, 1},
		{-1, 1},
		{1, -1},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic", args.rate, args.burst)
				}
			}()
			limiter.New(args.rate, args.burst)
		}()
	}
}

func TestBucketStartsFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 2)
		defer l.Close()
		if !l.Allow() {
			t.Error("first Allow() = false, want true from a full bucket")
		}
		if !l.Allow() {
			t.Error("second Allow() = false, want true from a full bucket")
		}
		if l.Allow() {
			t.Error("third Allow() = true, want false once the burst is spent")
		}
	})
}

func TestWaitBlocksUntilRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(4, 0) // one token every 250ms, none stored
		defer l.Close()
		start := time.Now()
		if err := l.Wait(context.Background()); err != nil {
			t.Fatalf("Wait() = %v, want nil once a refill arrives", err)
		}
		if d := time.Since(start); d != 250*time.Millisecond {
			t.Errorf("Wait() returned after %v, want 250ms at 4 tokens per second", d)
		}
	})
}

func TestWaitHonorsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0) // next refill a full second away
		defer l.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait() = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestWaitAfterClose(t *testing.T) {
	l := limiter.New(1, 1)
	l.Close()
	l.Close() // Close is documented as idempotent
	// ErrClosed wins even though an unused token is still in the bucket.
	if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
		t.Errorf("Wait() after Close = %v, want ErrClosed", err)
	}
}

func TestBurstCapsStoredTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(10, 2)
		defer l.Close()
		time.Sleep(time.Second) // ten refills against a bucket that holds two
		if !l.Allow() {
			t.Error("first Allow() = false, want true")
		}
		if !l.Allow() {
			t.Error("second Allow() = false, want true")
		}
		if l.Allow() {
			t.Error("third Allow() = true, want false: tokens beyond burst are dropped")
		}
	})
}

func TestZeroBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(10, 0)
		defer l.Close()
		time.Sleep(time.Second) // refills arrive but none can be stored
		if l.Allow() {
			t.Error("Allow() = true, want false: a burst-0 limiter stores no tokens")
		}
		if err := l.Wait(context.Background()); err != nil {
			t.Errorf("Wait() = %v, want nil when a refill arrives mid-wait", err)
		}
	})
}
