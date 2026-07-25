package limiter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// slowRate produces a refill interval long enough (1s) that no refill
// lands during a fast test, making token counts deterministic.
const slowRate = 1

func TestNewPanics(t *testing.T) {
	tests := []struct {
		name        string
		rate, burst int
	}{
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"negative burst", 1, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) did not panic", tt.rate, tt.burst)
				}
			}()
			New(tt.rate, tt.burst)
		})
	}
}

func TestBucketStartsFull(t *testing.T) {
	const burst = 3
	l := New(slowRate, burst)
	defer l.Close()

	for i := range burst {
		if !l.Allow() {
			t.Fatalf("Allow() call %d = false, want true (bucket should start full)", i+1)
		}
	}
	if l.Allow() {
		t.Errorf("Allow() call %d = true, want false (bucket of %d should be empty)", burst+1, burst)
	}
}

func TestAllowZeroBurst(t *testing.T) {
	l := New(1000, 0)
	defer l.Close()

	// With burst 0 no token is ever stored, so Allow must always
	// report false, even after many refill intervals have elapsed.
	time.Sleep(20 * time.Millisecond)
	for i := range 5 {
		if l.Allow() {
			t.Fatalf("Allow() call %d = true, want false with burst 0", i+1)
		}
	}
}

func TestWaitReturnsImmediatelyWhenTokenAvailable(t *testing.T) {
	l := New(slowRate, 1)
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait() = %v, want nil (bucket starts full)", err)
	}
}

func TestWaitConsumesToken(t *testing.T) {
	l := New(slowRate, 1)
	defer l.Close()

	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() = %v, want nil", err)
	}
	if l.Allow() {
		t.Error("Allow() = true after Wait consumed the only token, want false")
	}
}

func TestWaitBlocksUntilRefill(t *testing.T) {
	// burst 0: Wait can only succeed by rendezvousing with the refill
	// goroutine, proving both that Wait blocks and that refill works.
	l := New(100, 0) // 10ms interval
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait() = %v, want nil after a refill arrives", err)
	}
}

func TestWaitContextCancelled(t *testing.T) {
	l := New(slowRate, 0)
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { errc <- l.Wait(ctx) }()
	cancel()

	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait() = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not return after context cancellation")
	}
}

func TestWaitContextDeadlineExceeded(t *testing.T) {
	l := New(slowRate, 0)
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait() = %v, want context.DeadlineExceeded", err)
	}
}

func TestWaitAfterClose(t *testing.T) {
	l := New(slowRate, 2)
	l.Close()

	// The bucket still holds tokens, but the documented contract is
	// that every Wait after Close returns ErrClosed.
	if err := l.Wait(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Wait() after Close = %v, want ErrClosed", err)
	}
}

func TestCloseUnblocksWaiter(t *testing.T) {
	l := New(slowRate, 0)

	errc := make(chan error, 1)
	go func() { errc <- l.Wait(context.Background()) }()

	// Give the goroutine a moment to block inside Wait before closing.
	time.Sleep(10 * time.Millisecond)
	l.Close()

	select {
	case err := <-errc:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("Wait() unblocked by Close = %v, want ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not unblock a pending Wait")
	}
}

func TestCloseIdempotent(t *testing.T) {
	l := New(slowRate, 1)
	l.Close()
	l.Close() // must not panic
}

func TestRefillRestoresTokens(t *testing.T) {
	l := New(100, 1) // 10ms interval
	defer l.Close()

	if !l.Allow() {
		t.Fatal("Allow() = false on a full bucket, want true")
	}

	// Poll instead of asserting on a single sleep so the test tolerates
	// scheduler jitter without becoming flaky.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if l.Allow() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("token was never refilled after being consumed")
}

func TestRefillCappedAtBurst(t *testing.T) {
	const burst = 2
	l := New(5, burst) // 200ms interval
	defer l.Close()

	// Let several refill ticks fire against an already-full bucket.
	time.Sleep(900 * time.Millisecond)

	got := 0
	for l.Allow() {
		got++
		if got > burst {
			break
		}
	}
	if got != burst {
		t.Fatalf("drained %d tokens after overfilling, want exactly %d", got, burst)
	}
}

func TestConcurrentUse(t *testing.T) {
	// Exercise Allow, Wait, and Close from many goroutines so the race
	// detector can observe any unsynchronized access.
	l := New(1000, 10)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			for range 50 {
				l.Allow()
				if err := l.Wait(ctx); err != nil {
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		l.Close()
		l.Close()
	}()
	wg.Wait()

	if err := l.Wait(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Wait() after concurrent Close = %v, want ErrClosed", err)
	}
}
