package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: stash a token behind its ID with a
// short TTL, serve it back on lookup, and let it quietly fall out of the
// cache once the session has expired.
func Example_sessionTokens() {
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	c.Set("session-42", "tok_abc123", 20*time.Millisecond)

	if token, ok := c.Get("session-42"); ok {
		fmt.Println(token)
	}

	// The session outlives its TTL well before the janitor's next sweep,
	// so this miss comes from read-time expiry, not a background delete.
	time.Sleep(40 * time.Millisecond)
	_, ok := c.Get("session-42")
	fmt.Println(ok)

	// Output:
	// tok_abc123
	// false
}

// The typical client story end to end: two tokens with different lifetimes
// share one cache, replacing a key resets its deadline, an expiry shows up
// long before the janitor's next sweep could have caused it, and a shutdown
// makes every later call inert rather than a write into a discarded map.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// A ten-second sweep is long enough that every check below lands
		// well before the janitor's first tick, so a miss can only come
		// from read-time expiry.
		c := cache.New(10 * time.Second)
		defer c.Close()

		c.Set("session-1", "tok-a", time.Second)
		c.Set("permanent", "tok-b", 0) // ttl <= 0: no expiry

		if v, ok := c.Get("session-1"); !ok || v != "tok-a" {
			t.Fatalf("Get(%q) = %q, %v, want %q, true", "session-1", v, ok, "tok-a")
		}
		if v, ok := c.Get("permanent"); !ok || v != "tok-b" {
			t.Fatalf("Get(%q) = %q, %v, want %q, true", "permanent", v, ok, "tok-b")
		}
		if n := c.Len(); n != 2 {
			t.Fatalf("Len() = %d, want 2", n)
		}
		if _, ok := c.Get("missing"); ok {
			t.Error("Get() on a never-set key reported a hit")
		}

		// Setting session-1 again replaces both its value and its
		// deadline: the shorter ttl below governs from here on, not the
		// one-second ttl set above.
		c.Set("session-1", "tok-a2", 500*time.Millisecond)
		time.Sleep(600 * time.Millisecond)

		if _, ok := c.Get("session-1"); ok {
			t.Error("Get() = hit for session-1 past its replaced ttl, want a miss")
		}
		if v, ok := c.Get("permanent"); !ok || v != "tok-b" {
			t.Fatalf("Get(%q) = %q, %v, want %q, true (no ttl set)", "permanent", v, ok, "tok-b")
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d, want 1 once session-1 has expired", n)
		}

		// Close discards the entry map; Set and Get afterward must stay
		// inert rather than write into or read past a nil map.
		c.Close()
		c.Close() // idempotent
		c.Set("permanent", "tok-c", 0)
		if v, ok := c.Get("permanent"); ok {
			t.Errorf("Get() after Close = %q, true, want a miss", v)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
	})
}

func TestNewPanicsOnInvalidJanitorInterval(t *testing.T) {
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
// what makes interval invalid in the failure message.
func mustPanic(t *testing.T, interval time.Duration, invalidity string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a %s", interval, invalidity)
		}
	}()
	c := cache.New(interval)
	c.Close() // reached only if the guard is broken; stop the janitor goroutine anyway
}
