package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// The typical caller mints a session token, stores it under the session ID
// with a time-to-live, reads it back while the session is live, and finds it
// gone once the TTL has lapsed.
func Example() {
	// Sweep expired entries once a minute. Expiry is enforced the instant a
	// caller reads, so the interval only bounds how long dead entries linger
	// in memory.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	c.Set("session-42", "token-abc", time.Hour)
	token, ok := c.Get("session-42")
	fmt.Println(token, ok)

	// A short-lived token is a miss once its TTL elapses.
	c.Set("session-99", "token-xyz", time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	_, ok = c.Get("session-99")
	fmt.Println(ok)

	// Output:
	// token-abc true
	// false
}

// The session-store story end to end on synctest's virtual clock: a token
// stored with a TTL that reads back while live, a permanent entry that
// outlives every sweep, a replacement that restarts the TTL so the entry
// survives past its original deadline, a read-time miss once the restarted
// TTL lapses, and a Close that empties the store and turns every later call
// into a miss.
func TestSessionLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second) // sweep expired entries every second
		defer c.Close()

		c.Set("session", "token-A", 30*time.Second)
		c.Set("service", "key", 0) // non-positive TTL: no expiry

		if got, ok := c.Get("session"); !ok || got != "token-A" {
			t.Errorf(`Get("session") = %q, %v; want "token-A", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2 with two live entries", n)
		}

		// Five seconds short of the deadline the token is still live, even
		// though the janitor has already swept 25 times.
		time.Sleep(25 * time.Second)
		if got, ok := c.Get("session"); !ok || got != "token-A" {
			t.Errorf(`Get("session") after 25s = %q, %v; want "token-A", true`, got, ok)
		}

		// Replacing a live entry restarts the TTL from now: the fresh 30s
		// deadline lands at t=55, past the original t=30, so the token no
		// longer expires at its first deadline.
		c.Set("session", "token-B", 30*time.Second)
		if got, ok := c.Get("session"); !ok || got != "token-B" {
			t.Errorf(`Get("session") after replace = %q, %v; want "token-B", true`, got, ok)
		}

		// Ten seconds on, past that original t=30 deadline, the replaced entry
		// is still live and still counted: only the restarted TTL governs now.
		time.Sleep(10 * time.Second)
		if _, ok := c.Get("session"); !ok {
			t.Error(`Get("session") past its original TTL = miss, want hit once the replace reset the clock`)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2 while both entries are live", n)
		}

		// Past the restarted t=55 deadline the token finally misses and stops
		// counting, while the no-expiry entry survives every sweep.
		time.Sleep(25 * time.Second)
		if _, ok := c.Get("session"); ok {
			t.Error(`Get("session") past its restarted TTL = hit, want miss`)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d, want 1 once the token expired", n)
		}

		// Close empties the store; a second Close is a safe no-op.
		c.Close()
		c.Close()
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
		if _, ok := c.Get("service"); ok {
			t.Error(`Get("service") after Close = hit, want miss`)
		}

		// After Close, Set does nothing.
		c.Set("late", "value", time.Hour)
		if _, ok := c.Get("late"); ok {
			t.Error(`Get("late") stored after Close = hit, want miss`)
		}
	})
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		mustPanic(t, interval)
	}
}

// mustPanic fails the test unless New(interval) panics.
func mustPanic(t *testing.T, interval time.Duration) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a non-positive interval", interval)
		}
	}()
	c := cache.New(interval)
	c.Close() // reached only if the guard is broken; release the janitor anyway
}
