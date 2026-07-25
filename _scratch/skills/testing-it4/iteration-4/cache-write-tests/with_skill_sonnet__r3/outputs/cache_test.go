package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store needs no reaper of its own: entries expire on their own
// schedule, and once the store is closed it quietly stops serving anyone.
func Example_sessionStore() {
	sessions := cache.New(time.Minute)
	defer sessions.Close()

	sessions.Set("session-1", "alice", 30*time.Minute)
	if v, ok := sessions.Get("session-1"); ok {
		fmt.Println(v)
	}
	fmt.Println(sessions.Len())

	sessions.Close()
	if _, ok := sessions.Get("session-1"); !ok {
		fmt.Println("session lookup after Close finds nothing")
	}

	// Output:
	// alice
	// 1
	// session lookup after Close finds nothing
}

// The typical client story end to end: a short-lived entry read before and
// after it expires, a permanent entry that outlives it, a replacement that
// resets rather than extends a ttl, and a shutdown that empties the table
// and turns every later call into a miss.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second) // janitor sweeps once a second
		defer c.Close()

		c.Set("a", "1", 500*time.Millisecond)
		c.Set("b", "2", 0) // no ttl: lives until replaced or closed

		if v, ok := c.Get("a"); !ok || v != "1" {
			t.Fatalf(`Get("a") = %q, %v, want "1", true`, v, ok)
		}
		if got, want := c.Len(), 2; got != want {
			t.Fatalf("Len() = %d, want %d", got, want)
		}

		// "a" expires at the 500ms mark, well ahead of the janitor's first
		// sweep at 1s, so this miss must come from the read-time check.
		time.Sleep(600 * time.Millisecond)
		if _, ok := c.Get("a"); ok {
			t.Error(`Get("a") after its ttl elapsed = hit, want miss`)
		}
		if got, want := c.Len(), 1; got != want {
			t.Errorf("Len() after \"a\" expired = %d, want %d", got, want)
		}
		if v, ok := c.Get("b"); !ok || v != "2" {
			t.Errorf(`Get("b") with no ttl = %q, %v, want "2", true`, v, ok)
		}

		// Replacing a live entry resets its ttl; it must not extend the
		// original 500ms deadline that "b" never had in the first place.
		c.Set("b", "3", 200*time.Millisecond)
		time.Sleep(300 * time.Millisecond)
		if _, ok := c.Get("b"); ok {
			t.Error(`Get("b") after its replacement ttl elapsed = hit, want miss`)
		}

		c.Close()
		c.Set("c", "4", time.Minute)
		if _, ok := c.Get("c"); ok {
			t.Error("Get after Close = hit, want miss: Set must no-op once closed")
		}
		if got := c.Len(); got != 0 {
			t.Errorf("Len() after Close = %d, want 0", got)
		}

		c.Close() // idempotent: must not panic or block
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
// what makes the interval invalid in the failure message.
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
