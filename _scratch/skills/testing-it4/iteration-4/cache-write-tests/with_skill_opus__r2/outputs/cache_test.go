package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// The everyday use: stash a session token that outlives the request,
// read it back on the next request, and let a missing key stand for a
// user with no session. Always Close the cache so its janitor goroutine
// is released.
func Example() {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("alice", "token-abc", 15*time.Minute)

	if token, ok := c.Get("alice"); ok {
		fmt.Println("alice:", token)
	}
	if _, ok := c.Get("bob"); !ok {
		fmt.Println("bob: no session")
	}
	fmt.Println("live sessions:", c.Len())

	// Output:
	// alice: token-abc
	// bob: no session
	// live sessions: 1
}

// One scenario carries the whole contract because a session cache is
// lived through, not called piecemeal: a token is stored with a TTL,
// stays a hit right up to its deadline, and turns into a miss the instant
// it passes. The interesting promise is that read-time expiry beats the
// janitor, so the interval here (a minute) is deliberately far longer
// than the token's TTL (30s): at the 30s mark the janitor has not swept
// even once, yet the entry is already gone from Get and Len. Alongside
// that: an entry set with a non-positive TTL never expires, Set replaces
// rather than duplicates, and Close leaves the cache inert. Virtual time
// keeps all of this exact and instant; running in a bubble also proves
// the janitor is released, since a leaked goroutine would fail the test.
func TestSessionCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Minute)
		defer c.Close()

		c.Set("session", "token-abc", 30*time.Second)
		if v, ok := c.Get("session"); !ok || v != "token-abc" {
			t.Fatalf(`Get("session") = %q, %v; want "token-abc", true`, v, ok)
		}

		// A non-positive TTL means no expiry at all.
		c.Set("persistent", "keep-me", 0)
		if n := c.Len(); n != 2 {
			t.Fatalf("Len() = %d, want 2 live entries", n)
		}

		time.Sleep(29 * time.Second)
		if _, ok := c.Get("session"); !ok {
			t.Error(`Get("session") a second before its deadline = miss, want hit`)
		}

		// At the deadline the token expires on read, well before the
		// once-a-minute janitor's first sweep at the 60s mark.
		time.Sleep(time.Second)
		if v, ok := c.Get("session"); ok {
			t.Errorf(`Get("session") at its deadline = %q, true; want a miss`, v)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d after the token expired, want 1 (the no-TTL entry)", n)
		}

		// The no-TTL entry outlives any sweep.
		time.Sleep(time.Hour)
		if v, ok := c.Get("persistent"); !ok || v != "keep-me" {
			t.Errorf(`Get("persistent") an hour on = %q, %v; want "keep-me", true`, v, ok)
		}

		// A fresh Set on the expired key replaces it in place.
		c.Set("session", "token-xyz", time.Minute)
		if v, ok := c.Get("session"); !ok || v != "token-xyz" {
			t.Errorf(`Get("session") after re-Set = %q, %v; want "token-xyz", true`, v, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d after re-Set, want 2", n)
		}

		// Shutdown empties the cache and turns every method inert.
		c.Close()
		if v, ok := c.Get("session"); ok {
			t.Errorf(`Get("session") after Close = %q, true; want a miss`, v)
		}
		c.Set("session", "revenant", time.Minute)
		if _, ok := c.Get("session"); ok {
			t.Error(`Get("session") after a post-Close Set = hit; want a miss`)
		}
		// Idempotent: the deferred Close will call it once more.
		c.Close()
	})
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
	cases := []struct {
		interval time.Duration
		why      string
	}{
		{interval: 0, why: "zero interval"},
		{interval: -time.Second, why: "negative interval"},
	}
	for _, c := range cases {
		mustPanic(t, c.interval, c.why)
	}
}

// mustPanic fails the test unless New(interval) panics; why names what
// makes the interval invalid in the failure message.
func mustPanic(t *testing.T, interval time.Duration, why string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a %s", interval, why)
		}
	}()
	c := cache.New(interval) // returns only if the guard is broken
	c.Close()                // release the janitor if New wrongly succeeded
}
