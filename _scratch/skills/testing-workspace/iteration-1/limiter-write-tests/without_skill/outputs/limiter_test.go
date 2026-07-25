package limiter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewPanicsOnNonPositiveRate(t *testing.T) {
	for _, rate := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, 1) did not panic", rate)
				}
			}()
			New(rate, 1)
		}()
	}
}

func TestNewPanicsOnNegativeBurst(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New(1, -1) did not panic")
		}
	}()
	New(1, -1)
}

// The bucket starts full: exactly burst Allow calls succeed before any
// refill can plausibly arrive.
func TestBucketStartsFull(t *testing.T) {
	const burst = 3
	l := New(1, burst) // 1s interval: no refill lands during this test.
	defer l.Close()

	for i := range burst {
		if !l.Allow() {
			t.Fatalf("Allow call %d = false, want true", i+1)
		}
	}
	if l.Allow() {
		t.Fatalf("Allow call %d = true, want false (bucket drained)", burst+1)
	}
}

// With burst 0 the bucket stores nothing, so Allow never succeeds even
// while refills are being generated.
func TestZeroBurstAllowAlwaysFalse(t *testing.T) {
	l := New(1000, 0)
	defer l.Close()

	deadline := time.Now().Add(50 * time.Millisecond)
	for time.Now().Before(deadline) {
		if l.Allow() {
			t.Fatal("Allow = true, want false for a burst-0 limiter")
		}
	}
}

// A drained bucket becomes usable again once the refill goroutine adds
// a token.
func TestRefillReplenishesToken(t *testing.T) {
	l := New(100, 1) // refill every 10ms
	defer l.Close()

	if !l.Allow() {
		t.Fatal("Allow = false on a fresh limiter, want true")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if l.Allow() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("no token refilled within 2s at rate 100/s")
}

// Tokens beyond burst are dropped: after ample refill time only burst
// tokens are stored.
func TestRefillCapsAtBurst(t *testing.T) {
	const burst = 2
	l := New(1000, burst) // refill every 1ms

	time.Sleep(100 * time.Millisecond) // far more refills than capacity
	l.Close()
	time.Sleep(10 * time.Millisecond) // let any in-flight refill land

	got := 0
	for l.Allow() {
		got++
	}
	if got != burst {
		t.Fatalf("drained %d tokens after saturation, want exactly %d", got, burst)
	}
}

func TestWaitReturnsImmediatelyWithToken(t *testing.T) {
	l := New(1, 1)
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait = %v, want nil", err)
	}
}

// Wait on an empty burst-0 bucket parks until the refill goroutine
// hands over a token.
func TestWaitBlocksUntilRefill(t *testing.T) {
	l := New(100, 0) // refill every 10ms
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait = %v, want nil once a refill arrives", err)
	}
}

func TestWaitHonorsContextCancellation(t *testing.T) {
	l := New(1, 0) // no token for ~1s
	defer l.Close()

	t.Run("deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait = %v, want %v", err, context.DeadlineExceeded)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		errc := make(chan error, 1)
		go func() { errc <- l.Wait(ctx) }()
		cancel()
		select {
		case err := <-errc:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Wait = %v, want %v", err, context.Canceled)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Wait did not return after context cancellation")
		}
	})
}

// After Close, Wait reports ErrClosed even if unused tokens remain in
// the bucket.
func TestWaitAfterCloseReturnsErrClosed(t *testing.T) {
	l := New(1, 1)
	l.Close()

	if err := l.Wait(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Wait after Close = %v, want %v", err, ErrClosed)
	}
}

// Close unblocks a Wait that is already parked.
func TestCloseUnblocksPendingWait(t *testing.T) {
	l := New(1, 0) // no token for ~1s

	errc := make(chan error, 1)
	go func() { errc <- l.Wait(context.Background()) }()

	time.Sleep(20 * time.Millisecond) // let Wait park
	l.Close()

	select {
	case err := <-errc:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("Wait = %v, want %v", err, ErrClosed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after Close")
	}
}

// Allow keeps handing out tokens that were already banked before Close;
// it only stops once the bucket is drained.
func TestAllowAfterCloseDrainsRemainingTokens(t *testing.T) {
	l := New(1, 2)
	l.Close()

	if !l.Allow() || !l.Allow() {
		t.Fatal("Allow = false for tokens banked before Close, want true")
	}
	if l.Allow() {
		t.Fatal("Allow = true on a drained, closed limiter, want false")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	l := New(1, 1)
	l.Close()
	l.Close() // must not panic
}

// Smoke test for the race detector: Allow, Wait, and Close from many
// goroutines at once must not race or deadlock.
func TestConcurrentAllowWaitClose(t *testing.T) {
	l := New(1000, 8)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			for l.Wait(ctx) == nil {
			}
		}()
	}
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				l.Allow()
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	l.Close()
	l.Close()
	wg.Wait()
}
