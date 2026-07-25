package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A login flow is the typical caller: store the token under the session
// ID at login, look it up on every request, and shut the store down when
// the service drains.
func Example_sessionTokens() {
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	// A fresh login stores the session's token for the next hour.
	c.Set("sess-1", "token-alice", time.Hour)
	if token, ok := c.Get("sess-1"); ok {
		fmt.Println(token)
	}

	// A re-login on the same session replaces both the token and its
	// expiry.
	c.Set("sess-1", "token-alice-2", time.Hour)
	token, _ := c.Get("sess-1")
	fmt.Println(token)

	// A session that never logged in is a miss.
	_, ok := c.Get("sess-404")
	fmt.Println(ok)

	// Output:
	// token-alice
	// token-alice-2
	// false
}

// The typical service story end to end: tokens go in with a TTL, reads
// hit until the deadline and miss from the deadline on, a replacement
// restarts the clock, a zero TTL pins an entry until shutdown, and Close
// empties the cache and turns every later Set into a no-op.
func TestSessionLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Sweeps land an hour apart, so every expiry below is observed on
		// read long before the janitor could have removed the entry.
		c := cache.New(time.Hour)
		defer c.Close()

		if v, ok := c.Get("alice"); ok {
			t.Errorf(`Get("alice") on an empty cache = %q, true, want a miss`, v)
		}

		c.Set("alice", "token-a", time.Second)
		c.Set("bob", "token-b", 2*time.Second)
		c.Set("carol", "token-c", 0) // no expiry
		if v, ok := c.Get("alice"); !ok || v != "token-a" {
			t.Errorf(`Get("alice") = %q, %v, want "token-a", true`, v, ok)
		}
		if n := c.Len(); n != 3 {
			t.Errorf("Len() = %d, want 3 live entries", n)
		}

		// "Expires ttl from now" includes the deadline itself: at exactly
		// the 1s mark the entry is already a miss.
		time.Sleep(time.Second)
		if v, ok := c.Get("alice"); ok {
			t.Errorf(`Get("alice") exactly at its deadline = %q, true, want a miss`, v)
		}
		if v, ok := c.Get("bob"); !ok || v != "token-b" {
			t.Errorf(`Get("bob") halfway through its TTL = %q, %v, want "token-b", true`, v, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d after one expiry, want 2", n)
		}

		// A replacement restarts the clock: bob was due at the 2s mark,
		// but the fresh 2s TTL keeps it alive until 3s.
		c.Set("bob", "token-b2", 2*time.Second)
		time.Sleep(time.Second)
		if v, ok := c.Get("bob"); !ok || v != "token-b2" {
			t.Errorf(`Get("bob") after replacement = %q, %v, want "token-b2", true`, v, ok)
		}

		// A day of janitor sweeps later, the no-expiry entry alone
		// survives.
		time.Sleep(24 * time.Hour)
		if v, ok := c.Get("carol"); !ok || v != "token-c" {
			t.Errorf(`Get("carol") after a day = %q, %v, want "token-c", true`, v, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d after every TTL elapsed, want 1", n)
		}

		// Shutdown discards everything, even the entry with no expiry,
		// and later Sets are dropped.
		c.Close()
		if v, ok := c.Get("carol"); ok {
			t.Errorf(`Get("carol") after Close = %q, true, want a miss`, v)
		}
		c.Set("dave", "token-d", time.Hour)
		if v, ok := c.Get("dave"); ok {
			t.Errorf(`Get("dave") after a post-Close Set = %q, true, want a miss`, v)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
		c.Close() // idempotent: the deferred Close above becomes a no-op too
	})
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
	cases := []struct {
		interval   time.Duration
		invalidity string
	}{
		{interval: 0, invalidity: "zero interval"},
		{interval: -time.Second, invalidity: "negative interval"},
	}
	for _, c := range cases {
		mustPanic(t, c.interval, c.invalidity)
	}
}

// mustPanic fails the test unless New(interval) panics; invalidity names
// what makes the argument invalid in the failure message.
func mustPanic(t *testing.T, interval time.Duration, invalidity string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a %s", interval, invalidity)
		}
	}()
	c := cache.New(interval)
	c.Close() // reached only if the guard is broken; stop the janitor anyway
}
