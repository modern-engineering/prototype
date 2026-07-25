package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: mint a token with a TTL at
// login, look it up on every request, and overwrite it on refresh.
func Example_sessionTokens() {
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on
	// an early return.
	defer c.Close()

	c.Set("alice", "token-1", time.Hour)
	if tok, ok := c.Get("alice"); ok {
		fmt.Println(tok)
	}

	// A refresh replaces the entry, value and deadline alike.
	c.Set("alice", "token-2", time.Hour)
	tok, _ := c.Get("alice")
	fmt.Println(tok)

	// A user who never logged in is a miss.
	_, ok := c.Get("bob")
	fmt.Println(ok)

	// Output:
	// token-1
	// token-2
	// false
}

// The session-store story end to end on the janitor's clock: a token is
// live right up to its deadline and a miss from the deadline on, hours
// before the janitor's first sweep; zero- and negative-TTL entries
// outlive every sweep; and Close empties the cache for good.
func TestSessionTokenLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// The janitor sweeps hourly, long after the 10m token expires,
		// so every expiry observed below is enforced by the reads alone.
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("alice", "token-1", 10*time.Minute)
		c.Set("audit", "trail", 0) // no expiry: lives until replaced or Close

		if got, ok := c.Get("alice"); !ok || got != "token-1" {
			t.Errorf(`Get("alice") = %q, %t, want "token-1", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2", n)
		}

		// One nanosecond before the deadline the token is still live.
		time.Sleep(10*time.Minute - time.Nanosecond)
		if _, ok := c.Get("alice"); !ok {
			t.Error(`Get("alice") just before the deadline = miss, want a hit`)
		}

		// At the deadline exactly, the entry expires: Get misses it and
		// Len stops counting it, with the first sweep still 50m away.
		time.Sleep(time.Nanosecond)
		if got, ok := c.Get("alice"); ok {
			t.Errorf(`Get("alice") at the deadline = %q, true, want a miss`, got)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d, want 1: an expired entry must not be counted", n)
		}

		// A refresh replaces the expired entry, and its negative TTL
		// stores the new token with no expiry at all.
		c.Set("alice", "token-2", -time.Minute)
		if got, ok := c.Get("alice"); !ok || got != "token-2" {
			t.Errorf(`Get("alice") after a refresh = %q, %t, want "token-2", true`, got, ok)
		}

		// Five sweeps pass; both no-expiry entries survive them all.
		time.Sleep(5 * time.Hour)
		if got, ok := c.Get("audit"); !ok || got != "trail" {
			t.Errorf(`Get("audit") after five sweeps = %q, %t, want "trail", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() after five sweeps = %d, want 2", n)
		}

		// Shutdown discards everything, unexpired entries included.
		c.Close()
		if _, ok := c.Get("audit"); ok {
			t.Error(`Get("audit") after Close = hit, want a miss`)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
		c.Set("carol", "token-3", time.Minute)
		if _, ok := c.Get("carol"); ok {
			t.Error(`Get("carol") = hit, want a miss: Set after Close must not store`)
		}

		// Closing again is safe: the deferred Close becomes a no-op.
		c.Close()
	})
}

func TestNewPanicsOnNonpositiveInterval(t *testing.T) {
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
	c.Close() // reached only if the guard is broken; stop the janitor anyway
}
