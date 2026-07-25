package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session token is set with a short TTL, read back while still valid,
// and gone once that TTL elapses.
func Example() {
	c := cache.New(time.Hour) // the janitor never gets a chance to run in this example
	defer c.Close()

	c.Set("session-42", "alice", 20*time.Millisecond)
	if _, ok := c.Get("session-42"); ok {
		fmt.Println("session-42 is live")
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get("session-42"); !ok {
		fmt.Println("session-42 expired")
	}

	// Output:
	// session-42 is live
	// session-42 expired
}

// The typical client story end to end: a token with a TTL and a permanent
// one coexist, the TTL'd token is gone on read well before the janitor's
// own sweep could reach it, Set overwrites a key outright TTL included,
// and once Close runs every key misses and every Set is a no-op no matter
// how live the entry would otherwise be.
func TestSessionLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Minute) // sweep interval far past every TTL below
		defer c.Close()

		c.Set("session-1", "alice", 100*time.Millisecond)
		c.Set("session-2", "bob", 0) // no TTL: permanent until replaced or closed

		if v, ok := c.Get("session-1"); !ok || v != "alice" {
			t.Errorf(`Get("session-1") = %q, %v, want "alice", true`, v, ok)
		}
		if got := c.Len(); got != 2 {
			t.Errorf("Len() = %d, want 2 with both entries live", got)
		}

		time.Sleep(150 * time.Millisecond) // past session-1's TTL, nowhere near a sweep
		if _, ok := c.Get("session-1"); ok {
			t.Error(`Get("session-1") = _, true after its TTL elapsed, want false`)
		}
		if got := c.Len(); got != 1 {
			t.Errorf("Len() = %d after session-1 expired, want 1 for the permanent entry", got)
		}

		c.Set("session-2", "bob-reauthenticated", 100*time.Millisecond)
		if v, ok := c.Get("session-2"); !ok || v != "bob-reauthenticated" {
			t.Errorf(`Get("session-2") = %q, %v, want "bob-reauthenticated", true`, v, ok)
		}

		c.Close()
		if _, ok := c.Get("session-2"); ok {
			t.Error(`Get("session-2") = _, true after Close, want false`)
		}
		if got := c.Len(); got != 0 {
			t.Errorf("Len() = %d after Close, want 0", got)
		}

		c.Set("session-3", "carol", 0)
		if _, ok := c.Get("session-3"); ok {
			t.Error(`Get("session-3") = _, true for a Set issued after Close, want the Set to do nothing`)
		}

		c.Close() // idempotent: must not panic
	})
}

func TestNewPanicsOnInvalidInterval(t *testing.T) {
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
