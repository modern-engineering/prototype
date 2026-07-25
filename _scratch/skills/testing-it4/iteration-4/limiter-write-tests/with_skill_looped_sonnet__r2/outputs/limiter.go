// Package limiter provides a token-bucket rate limiter driven by a
// background refill goroutine.
//
// A limiter created with burst 0 stores no tokens: Allow always reports
// false, and Wait proceeds only when a refill arrives while the caller
// is already waiting.
package limiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrClosed is returned by Wait once the limiter has been closed.
var ErrClosed = errors.New("limiter: closed")

// Limiter is a token-bucket rate limiter. Tokens accumulate at a fixed
// rate up to a maximum of burst; each permitted event consumes one
// token.
type Limiter struct {
	tokens    chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// New returns a Limiter that refills at rate tokens per second and
// holds at most burst unused tokens. The limiter starts full. New starts
// a background refill goroutine; callers must release it with Close.
//
// New panics if rate is not between 1 and 1e9 tokens per second, or if
// burst is negative.
func New(rate, burst int) *Limiter {
	// A rate above 1e9 would make the refill interval zero.
	if rate <= 0 || rate > 1e9 {
		panic("limiter: rate must be between 1 and 1e9 tokens per second")
	}
	if burst < 0 {
		panic("limiter: burst must not be negative")
	}
	l := &Limiter{
		tokens: make(chan struct{}, burst),
		done:   make(chan struct{}),
	}
	for range burst {
		l.tokens <- struct{}{}
	}
	go l.refill(time.Second / time.Duration(rate))
	return l
}

// refill adds one token per interval until the limiter is closed.
// Tokens beyond the limiter's capacity are dropped.
func (l *Limiter) refill(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-l.done:
			return
		case <-ticker.C:
			select {
			case l.tokens <- struct{}{}:
			default:
			}
		}
	}
}

// Allow reports whether a token is available right now, consuming one
// if so. It never blocks.
func (l *Limiter) Allow() bool {
	select {
	case <-l.tokens:
		return true
	default:
		return false
	}
}

// Wait blocks until a token is available, consumes it, and returns nil.
// If ctx is done first, Wait returns ctx.Err(). Once Close has
// returned, every subsequent Wait call returns ErrClosed.
func (l *Limiter) Wait(ctx context.Context) error {
	select {
	case <-l.done:
		return ErrClosed
	default:
	}
	select {
	case <-l.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-l.done:
		return ErrClosed
	}
}

// Close stops the refill goroutine. Close is idempotent: calling it
// more than once is safe, and every call after the first does nothing.
func (l *Limiter) Close() {
	l.closeOnce.Do(func() { close(l.done) })
}
