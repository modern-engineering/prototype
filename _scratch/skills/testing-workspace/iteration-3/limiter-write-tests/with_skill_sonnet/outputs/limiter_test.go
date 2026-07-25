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

// TestNewPanicsOnInvalidArguments checks the documented panic contract for
// the two arguments New validates: a non-positive rate and a negative
// burst. Both are caller mistakes the package chooses to fail loudly on,
// rather than silently clamp.
func TestNewPanicsOnInvalidArguments(t *testing.T) {
	cases := []struct {
		rate, burst int
		invalid     string
	}{
		{rate: 0, burst: 1, invalid: "zero rate"},
		{rate: -1, burst: 1, invalid: "negative rate"},
		{rate: 1, burst: -1, invalid: "negative burst"},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic for %s", c.rate, c.burst, c.invalid)
				}
			}()
			limiter.New(c.rate, c.burst)
		}()
	}
}

// TestZeroBurstLimiter pins the package doc's special case, the foundation
// the rest of the suite builds on: with no storage in the bucket, Allow can
// never observe a token (both sides of the handoff are non-blocking, so
// they can never rendezvous), while Wait can, because it parks on the
// channel and so is present to receive the refill goroutine's send.
func TestZeroBurstLimiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 0) // one token per second, no storage
		defer l.Close()

		if l.Allow() {
			t.Fatal("Allow() = true on a zero-burst limiter, want false")
		}

		errc := make(chan error, 1)
		go func() { errc <- l.Wait(context.Background()) }()
		synctest.Wait() // let Wait park on the empty bucket

		select {
		case err := <-errc:
			t.Fatalf("Wait() returned %v before any refill arrived", err)
		default:
		}

		time.Sleep(1500 * time.Millisecond) // past the first refill tick
		synctest.Wait()

		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("Wait() = %v, want nil once a refill arrived", err)
			}
		default:
			t.Fatal("Wait() did not return once a refill arrived while it was parked")
		}
	})
}

// TestBucketStartsFull checks that New hands out burst tokens immediately,
// before any refill tick, and no more than that.
func TestBucketStartsFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 3) // slow refill; only the starting tokens matter here
		defer l.Close()

		for i := range 3 {
			if !l.Allow() {
				t.Fatalf("Allow() = false on starting token %d, want true", i)
			}
		}
		if l.Allow() {
			t.Fatal("Allow() = true once the starting burst was drained, want false")
		}
	})
}

// TestTokensRefillUpToBurst checks the general accumulation rule: tokens
// arrive at the configured rate, and the bucket never holds more than
// burst regardless of how many ticks pass.
func TestTokensRefillUpToBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 2) // one token per second, holds at most two
		defer l.Close()

		for l.Allow() { // drain the starting burst
		}

		time.Sleep(2500 * time.Millisecond) // two ticks land; a third is not yet due

		if !l.Allow() {
			t.Fatal("Allow() = false after two refills, want true")
		}
		if !l.Allow() {
			t.Fatal("Allow() = false after two refills, want true")
		}
		if l.Allow() {
			t.Fatal("Allow() = true after only two refills reached an empty bucket, want false (burst caps accumulation)")
		}
	})
}

// TestWaitReturnsCtxErrWhenContextEndsFirst checks that a context canceled
// before a token is available wins the race deterministically: Wait must
// not block waiting on tokens that will never come in time.
func TestWaitReturnsCtxErrWhenContextEndsFirst(t *testing.T) {
	l := limiter.New(1, 0) // no starting tokens, so only ctx can unblock Wait
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already done before Wait is called

	if err := l.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait(canceled ctx) = %v, want context.Canceled", err)
	}
}

// TestWaitAfterCloseReturnsErrClosed checks the post-Close contract: once
// Close has returned, every subsequent Wait reports ErrClosed rather than
// blocking forever on a bucket nothing refills anymore. Calling Close
// twice here also exercises its documented idempotence.
func TestWaitAfterCloseReturnsErrClosed(t *testing.T) {
	l := limiter.New(1, 0)
	l.Close()
	l.Close() // idempotent: must not panic or block

	if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
		t.Fatalf("Wait() after Close = %v, want ErrClosed", err)
	}
}

// TestCloseStopsRefill checks Close's documented effect on the background
// goroutine: refills must not keep arriving after Close returns. The test
// observes this the only way a user can, through Allow, rather than
// reaching into the goroutine's internals.
func TestCloseStopsRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 1) // one token per second, holds one

		if !l.Allow() {
			t.Fatal("Allow() = false on the starting token, want true")
		}
		l.Close()

		time.Sleep(3 * time.Second) // several refill intervals after Close

		if l.Allow() {
			t.Fatal("Allow() = true after Close, want no further refills")
		}
	})
}

// ExampleLimiter shows a gateway request gated by Allow, and that closing
// a Limiter more than once is safe: the explicit Close below and the
// deferred one that follows it are both honored as documented.
func ExampleLimiter() {
	l := limiter.New(1, 2) // up to two requests immediately, then one per second
	defer l.Close()

	fmt.Println(l.Allow())
	fmt.Println(l.Allow())
	fmt.Println(l.Allow())

	l.Close() // the deferred Close above will be a no-op

	// Output:
	// true
	// true
	// false
}
