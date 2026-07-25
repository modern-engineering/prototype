package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: stash each token under its
// session ID with the session's lifetime, look it up on every request,
// and let expiry log the session out.
func Example_sessionTokens() {
	// Sweep expired sessions once a minute.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on
	// an early return.
	defer c.Close()

	c.Set("session-42", "alice", 200*time.Millisecond)

	// Within its TTL the token is live.
	if user, ok := c.Get("session-42"); ok {
		fmt.Println("request served for", user)
	}

	// Expiry is enforced on read: past the TTL the token is a miss
	// even though the janitor has not swept yet.
	time.Sleep(500 * time.Millisecond)
	if _, ok := c.Get("session-42"); !ok {
		fmt.Println("session expired, log in again")
	}

	// Output:
	// request served for alice
	// session expired, log in again
}

// The typical service story end to end: tokens go in with a lifetime,
// lookups hit until the lifetime runs out, a re-issued token starts a
// fresh lifetime, a rotated key pinned without a TTL never expires,
// and shutdown empties the cache for good.
func TestSessionTokenLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// An hour between sweeps keeps the janitor out of the picture:
		// every expiry observed below is enforced on read.
		c := cache.New(time.Hour)
		defer c.Close()

		// A fresh cache is empty.
		if _, ok := c.Get("session-1"); ok {
			t.Error(`Get("session-1") hit on a fresh cache, want a miss`)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() = %d on a fresh cache, want 0", n)
		}

		// Two credentials arrive: a session token with a lifetime and
		// an API key on a longer one.
		c.Set("session-1", "alice", time.Minute)
		c.Set("api-key", "hunter2", 2*time.Hour)
		if got, ok := c.Get("session-1"); !ok || got != "alice" {
			t.Errorf(`Get("session-1") = %q, %v, want "alice", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d after two Sets, want 2", n)
		}

		// Half the lifetime in, the token is re-issued: the replacement
		// expires a full minute from now, not thirty seconds.
		time.Sleep(30 * time.Second)
		c.Set("session-1", "alice-2", time.Minute)
		time.Sleep(45 * time.Second)
		if got, ok := c.Get("session-1"); !ok || got != "alice-2" {
			t.Errorf(`Get("session-1") = %q, %v past the original deadline, want the re-issued "alice-2", true`, got, ok)
		}

		// The remaining 15s run out. The token expires exactly at its
		// deadline, and with the janitor's first sweep still 58 minutes
		// away, the miss and the count below are the readers' own doing.
		time.Sleep(15 * time.Second)
		if _, ok := c.Get("session-1"); ok {
			t.Error(`Get("session-1") hit at its deadline, want a miss`)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d with one entry expired, want 1", n)
		}

		// The API key is rotated with a negative TTL: the replacement
		// sheds the old two-hour deadline and is pinned for good.
		c.Set("api-key", "hunter2-rotated", -time.Second)
		time.Sleep(4 * time.Hour)
		if got, ok := c.Get("api-key"); !ok || got != "hunter2-rotated" {
			t.Errorf(`Get("api-key") = %q, %v four hours after rotation, want "hunter2-rotated", true`, got, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d with only the pinned key live, want 1", n)
		}

		// Shutdown: everything is discarded, writes are ignored, and a
		// second Close is a safe no-op.
		c.Close()
		if _, ok := c.Get("api-key"); ok {
			t.Error(`Get("api-key") hit after Close, want a miss`)
		}
		c.Set("session-9", "mallory", time.Minute)
		if _, ok := c.Get("session-9"); ok {
			t.Error("Get() hit for a key Set after Close, want Set to do nothing")
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() = %d after Close, want 0", n)
		}
		c.Close()
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

// mustPanic fails the test unless New(interval) panics; invalidity
// names what makes the argument invalid in the failure message.
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
