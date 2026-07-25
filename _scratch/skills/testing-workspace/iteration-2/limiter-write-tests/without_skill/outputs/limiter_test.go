package limiter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// waitErr runs l.Wait(ctx) in a goroutine and returns a channel that
// yields its result, so tests can bound how long they block on Wait.
func waitErr(l *Limiter, ctx context.Context) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- l.Wait(ctx) }()
	return ch
}

// receive fails the test if no result arrives on ch within the deadline.
func receive(t *testing.T, ch <-chan error, timeout time.Duration) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		t.Fatalf("Wait did not return within %v", timeout)
		return nil
	}
}

func TestNewPanicsOnInvalidArguments(t *testing.T) {
	cases := []struct {
		name        string
		rate, burst int
	}{
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"negative burst", 1, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("New(%d, %d) did not panic", tc.rate, tc.burst)
				}
			}()
			New(tc.rate, tc.burst)
		})
	}
}

func TestNewAcceptsZeroBurst(t *testing.T) {
	l := New(1, 0)
	defer l.Close()
	if l.Allow() {
		t.Error("Allow() = true for a zero-burst limiter; want false")
	}
}

func TestBucketStartsFull(t *testing.T) {
	const burst = 5
	// Rate 1 means the next refill is a full second away, so the six
	// Allow calls below observe only the initial tokens.
	l := New(1, burst)
	defer l.Close()

	for i := range burst {
		if !l.Allow() {
			t.Fatalf("Allow() call %d = false; want true (bucket starts full)", i+1)
		}
	}
	if l.Allow() {
		t.Errorf("Allow() call %d = true after draining the bucket; want false", burst+1)
	}
}

func TestAllowDoesNotBlockWhenEmpty(t *testing.T) {
	l := New(1, 0)
	defer l.Close()

	done := make(chan bool, 1)
	go func() { done <- l.Allow() }()
	select {
	case got := <-done:
		if got {
			t.Error("Allow() = true on an empty bucket; want false")
		}
	case <-time.After(time.Second):
		t.Fatal("Allow blocked on an empty bucket")
	}
}

func TestRefillReplenishesTokens(t *testing.T) {
	l := New(100, 1) // one token every 10ms
	defer l.Close()

	if !l.Allow() {
		t.Fatal("Allow() = false on a fresh limiter; want true")
	}

	deadline := time.Now().Add(2 * time.Second)
	for !l.Allow() {
		if time.Now().After(deadline) {
			t.Fatal("bucket was not refilled within 2s")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestRefillDropsTokensBeyondBurst(t *testing.T) {
	const burst = 2
	// 50ms interval: several refill ticks fire into the already-full
	// bucket during the sleep, and the drain below finishes long before
	// the tick that follows it.
	l := New(20, burst)
	defer l.Close()

	time.Sleep(200 * time.Millisecond)

	for i := range burst {
		if !l.Allow() {
			t.Fatalf("Allow() call %d = false; want true", i+1)
		}
	}
	if l.Allow() {
		t.Errorf("Allow() call %d = true; want false (tokens beyond burst must be dropped)", burst+1)
	}
}

func TestWaitReturnsImmediatelyWhenTokenAvailable(t *testing.T) {
	l := New(1, 1)
	defer l.Close()

	if err := receive(t, waitErr(l, context.Background()), time.Second); err != nil {
		t.Fatalf("Wait() = %v; want nil", err)
	}
	if l.Allow() {
		t.Error("Allow() = true after Wait consumed the only token; want false")
	}
}

func TestWaitBlocksUntilRefill(t *testing.T) {
	// Burst 0: the only way Wait can proceed is a refill arriving while
	// it is already blocked.
	l := New(100, 0)
	defer l.Close()

	if err := receive(t, waitErr(l, context.Background()), 2*time.Second); err != nil {
		t.Fatalf("Wait() = %v; want nil after a refill", err)
	}
}

func TestWaitReturnsContextError(t *testing.T) {
	t.Run("canceled", func(t *testing.T) {
		l := New(1, 0) // next refill a full second away
		defer l.Close()

		ctx, cancel := context.WithCancel(context.Background())
		ch := waitErr(l, ctx)
		cancel()
		if err := receive(t, ch, time.Second); !errors.Is(err, context.Canceled) {
			t.Errorf("Wait() = %v; want %v", err, context.Canceled)
		}
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		l := New(1, 0)
		defer l.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if err := receive(t, waitErr(l, ctx), time.Second); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait() = %v; want %v", err, context.DeadlineExceeded)
		}
	})
}

func TestWaitAfterCloseReturnsErrClosed(t *testing.T) {
	l := New(1, 1) // bucket holds a token, yet closed wins
	l.Close()

	if err := l.Wait(context.Background()); !errors.Is(err, ErrClosed) {
		t.Errorf("Wait() after Close = %v; want %v", err, ErrClosed)
	}
}

func TestCloseUnblocksWait(t *testing.T) {
	l := New(1, 0) // no token and no imminent refill: Wait must block

	ch := waitErr(l, context.Background())
	time.Sleep(20 * time.Millisecond) // let Wait reach its select
	l.Close()

	if err := receive(t, ch, time.Second); !errors.Is(err, ErrClosed) {
		t.Errorf("Wait() = %v; want %v after Close", err, ErrClosed)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	l := New(1, 1)
	l.Close()
	l.Close() // must not panic

	if err := l.Wait(context.Background()); !errors.Is(err, ErrClosed) {
		t.Errorf("Wait() after double Close = %v; want %v", err, ErrClosed)
	}
}

func TestCloseStopsRefill(t *testing.T) {
	l := New(1000, 1)
	l.Close()

	// Give the refill goroutine ample ticks to observe done and exit,
	// then drain whatever it left behind.
	time.Sleep(100 * time.Millisecond)
	for l.Allow() {
	}

	time.Sleep(50 * time.Millisecond)
	if l.Allow() {
		t.Error("Allow() = true after Close and drain; want false (refill must have stopped)")
	}
}

func TestConcurrentUse(t *testing.T) {
	l := New(1000, 8)

	const goroutines = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range goroutines {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			for range 100 {
				l.Allow()
			}
		}()
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()
			err := l.Wait(ctx)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrClosed) {
				t.Errorf("Wait() = %v; want nil, deadline exceeded, or ErrClosed", err)
			}
		}()
	}
	close(start)

	time.Sleep(10 * time.Millisecond)
	l.Close() // racing Close against Allow and Wait must be safe
	wg.Wait()
}
