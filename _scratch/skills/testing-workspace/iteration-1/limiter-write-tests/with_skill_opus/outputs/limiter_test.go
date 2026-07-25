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

// The bucket starts full, so the first burst events are permitted and the
// next is denied. Time is frozen by the bubble, so no refill interferes.
func TestNewAllowConsumesFullBucket(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const burst = 3
		l := limiter.New(10, burst)
		defer l.Close()
		synctest.Wait()

		for i := 0; i < burst; i++ {
			if !l.Allow() {
				t.Fatalf("Allow #%d on a full bucket = false, want true", i+1)
			}
		}
		if l.Allow() {
			t.Error("Allow on a drained bucket = true, want false")
		}
	})
}

// Tokens accumulate at one per interval and stop at the burst cap; overflow
// refills are dropped rather than stored.
func TestRefillReplenishesUpToBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			rate     = 10
			burst    = 5
			interval = time.Second / rate
		)
		l := limiter.New(rate, burst)
		defer l.Close()
		synctest.Wait() // the refill goroutine has armed its ticker at time zero

		for i := 0; i < burst; i++ {
			l.Allow() // drain the initial fill
		}
		if l.Allow() {
			t.Fatal("Allow on a drained bucket = true, want false")
		}

		// One interval yields exactly one fresh token.
		time.Sleep(interval)
		synctest.Wait()
		if !l.Allow() {
			t.Error("Allow one interval after draining = false, want true")
		}
		if l.Allow() {
			t.Error("Allow drew a second token from a single refill")
		}

		// Idling far past burst intervals still leaves at most burst tokens.
		time.Sleep(2 * burst * interval)
		synctest.Wait()
		for i := 0; i < burst; i++ {
			if !l.Allow() {
				t.Errorf("Allow #%d after a long idle = false, want a full bucket", i+1)
			}
		}
		if l.Allow() {
			t.Errorf("Allow beyond burst = true; bucket exceeded its cap of %d", burst)
		}
	})
}

// Wait returns nil at once when a token is ready and consumes exactly one.
func TestWaitConsumesAvailableToken(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const burst = 3
		l := limiter.New(10, burst)
		defer l.Close()
		synctest.Wait()

		if err := l.Wait(context.Background()); err != nil {
			t.Fatalf("Wait on a full bucket = %v, want nil", err)
		}
		for i := 0; i < burst-1; i++ {
			if !l.Allow() {
				t.Fatalf("Allow #%d after one Wait = false, want true", i+1)
			}
		}
		if l.Allow() {
			t.Error("bucket held more than burst tokens after one Wait")
		}
	})
}

// A burst-0 bucket stores nothing: Allow is always false and Wait blocks until
// a refill arrives while the caller is parked, then proceeds.
func TestWaitProceedsOnRefillWhileWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const interval = time.Second / 10
		l := limiter.New(10, 0)
		defer l.Close()
		synctest.Wait()

		if l.Allow() {
			t.Error("Allow with burst 0 = true, want false")
		}

		errc := make(chan error, 1)
		go func() { errc <- l.Wait(context.Background()) }()
		synctest.Wait() // the caller is now parked, waiting for a refill

		select {
		case err := <-errc:
			t.Fatalf("Wait returned %v before any refill arrived", err)
		default:
		}

		time.Sleep(interval) // a refill arrives while the caller waits
		synctest.Wait()
		if err := <-errc; err != nil {
			t.Errorf("Wait after a refill = %v, want nil", err)
		}
	})
}

// When the context ends before a token is free, Wait reports the context's own
// error, whether the context was canceled or its deadline passed.
func TestWaitReturnsContextError(t *testing.T) {
	t.Run("canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			l := limiter.New(1, 1) // interval 1s, far longer than this test runs
			defer l.Close()
			synctest.Wait()
			if !l.Allow() {
				t.Fatal("Allow on a full bucket = false, want true")
			}

			ctx, cancel := context.WithCancel(context.Background())
			errc := make(chan error, 1)
			go func() { errc <- l.Wait(ctx) }()
			synctest.Wait() // the caller is parked on the empty bucket

			cancel()
			synctest.Wait()
			if err := <-errc; !errors.Is(err, context.Canceled) {
				t.Errorf("Wait after cancel = %v, want context.Canceled", err)
			}
		})
	})

	t.Run("deadline", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			l := limiter.New(1, 1) // interval 1s, so no refill beats the deadline
			defer l.Close()
			synctest.Wait()
			if !l.Allow() {
				t.Fatal("Allow on a full bucket = false, want true")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			errc := make(chan error, 1)
			go func() { errc <- l.Wait(ctx) }()
			synctest.Wait() // the caller is parked on the empty bucket

			time.Sleep(10 * time.Millisecond)
			synctest.Wait()
			if err := <-errc; !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("Wait after deadline = %v, want context.DeadlineExceeded", err)
			}
		})
	})
}

// Close releases a parked caller with ErrClosed and makes every later Wait
// return ErrClosed too. The bubble also proves Close lets the refill goroutine
// exit: an unclosed limiter would leave a lingering goroutine and fail.
func TestCloseReleasesWaitersAndRejectsWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := limiter.New(1, 1)
		defer l.Close()
		synctest.Wait()
		if !l.Allow() {
			t.Fatal("Allow on a full bucket = false, want true")
		}

		errc := make(chan error, 1)
		go func() { errc <- l.Wait(context.Background()) }()
		synctest.Wait() // the caller is parked on the empty bucket

		l.Close()
		synctest.Wait()
		if err := <-errc; !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait released by Close = %v, want ErrClosed", err)
		}
		if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
			t.Errorf("Wait after Close = %v, want ErrClosed", err)
		}
	})
}

// Repeated Close calls are safe; the limiter stays closed.
func TestCloseIsIdempotent(t *testing.T) {
	l := limiter.New(1, 1)
	l.Close()
	l.Close()
	if err := l.Wait(context.Background()); !errors.Is(err, limiter.ErrClosed) {
		t.Errorf("Wait after repeated Close = %v, want ErrClosed", err)
	}
}

// New rejects arguments that cannot describe a bucket: a non-positive rate has
// no interval, and a negative burst has no capacity.
func TestNewPanicsOnInvalidArguments(t *testing.T) {
	for _, tt := range []struct {
		name        string
		rate, burst int
	}{
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"negative burst", 1, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic", tt.rate, tt.burst)
				}
			}()
			limiter.New(tt.rate, tt.burst)
		})
	}
}

func ExampleLimiter_Allow() {
	l := limiter.New(1, 1) // the bucket starts full with one token
	defer l.Close()
	fmt.Println(l.Allow())
	// Output: true
}
