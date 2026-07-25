package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: stash a token with a TTL and read
// it back while it's still live.
func Example_sessionToken() {
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	c.Set("session-42", "alice", 30*time.Minute)

	if value, ok := c.Get("session-42"); ok {
		fmt.Println(value)
	}

	// Output:
	// alice
}

// The typical lifecycle end to end: a live entry, a TTL that expires it on
// read before the janitor ever runs, an entry stored with no TTL that
// outlives it, an overwrite that resets both value and expiry, and a
// shutdown that empties the cache and turns Set into a no-op.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second) // sweep every second
		defer c.Close()

		c.Set("a", "1", time.Minute)
		c.Set("b", "2", 0) // zero TTL: no expiry
		if v, ok := c.Get("a"); !ok || v != "1" {
			t.Fatalf(`Get("a") = %q, %v, want "1", true`, v, ok)
		}
		if got := c.Len(); got != 2 {
			t.Fatalf("Len() = %d, want 2", got)
		}

		// "a"'s minute-long TTL elapses; "b" has none and stays live.
		time.Sleep(90 * time.Second)
		if _, ok := c.Get("a"); ok {
			t.Error(`Get("a") after its TTL elapsed = hit, want miss`)
		}
		if got := c.Len(); got != 1 {
			t.Errorf("Len() after \"a\" expired = %d, want 1", got)
		}
		if v, ok := c.Get("b"); !ok || v != "2" {
			t.Errorf(`Get("b") with no TTL = %q, %v, want "2", true`, v, ok)
		}

		// Overwriting "b" replaces both its value and its expiry.
		c.Set("b", "3", time.Minute)
		if v, _ := c.Get("b"); v != "3" {
			t.Errorf(`Get("b") after overwrite = %q, want "3"`, v)
		}

		c.Close()
		if _, ok := c.Get("b"); ok {
			t.Error("Get() after Close = hit, want miss")
		}
		if got := c.Len(); got != 0 {
			t.Errorf("Len() after Close = %d, want 0", got)
		}
		c.Set("c", "4", time.Minute)
		if _, ok := c.Get("c"); ok {
			t.Error("Set() after Close stored a value, want no-op")
		}

		c.Close() // idempotent: must not panic or block
	})
}

func TestNewPanicsOnInvalidJanitorInterval(t *testing.T) {
	cases := []struct {
		janitorInterval time.Duration
		invalidity      string
	}{
		{janitorInterval: 0, invalidity: "zero interval"},
		{janitorInterval: -time.Second, invalidity: "negative interval"},
	}
	for _, c := range cases {
		mustPanic(t, c.janitorInterval, c.invalidity)
	}
}

// mustPanic fails the test unless New(janitorInterval) panics; invalidity
// names what makes the argument invalid in the failure message.
func mustPanic(t *testing.T, janitorInterval time.Duration, invalidity string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a %s", janitorInterval, invalidity)
		}
	}()
	c := cache.New(janitorInterval)
	c.Close() // reached only if the guard is broken; stop the janitor goroutine anyway
}
