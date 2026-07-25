package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A service front door is the typical caller: stash the session token
// under its ID at login, look it up on each request, and let the TTL
// sign the user out.
func Example_sessionTokens() {
	// Expiry is enforced on read, so a coarse sweep interval only bounds
	// memory: an expired token is invisible to Get either way.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	// A login stores the token; each request looks it up.
	c.Set("sess-alice", "token-1", time.Minute)
	if token, ok := c.Get("sess-alice"); ok {
		fmt.Println("alice is signed in as", token)
	}

	// A session nobody created is a plain miss.
	if _, ok := c.Get("sess-mallory"); !ok {
		fmt.Println("mallory is not signed in")
	}

	// Once its TTL elapses a token is gone, swept or not.
	c.Set("sess-bob", "token-2", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	if _, ok := c.Get("sess-bob"); !ok {
		fmt.Println("bob's session expired")
	}

	// Closing again is safe: Close is idempotent, so the deferred Close
	// above becomes a no-op.
	c.Close()

	// Output:
	// alice is signed in as token-1
	// mallory is not signed in
	// bob's session expired
}

// The session store's whole story: a token is a miss once its deadline
// passes, the reader sees the expiry before the janitor's first sweep,
// replacing a token renews, drops, or arms its deadline, live and
// TTL-less tokens survive every sweep, and Close discards even the
// entries that would have lived forever.
func TestExpiryLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second) // sweeps at 1s, 2s, 3s, ...
		defer c.Close()

		if _, ok := c.Get("session"); ok {
			t.Error(`Get("session") hit on a fresh cache, want a miss`)
		}

		c.Set("session", "alice-1", 2*time.Second)
		c.Set("refresh", "alice-refresh", 0) // no TTL: lives until Close
		c.Set("blip", "one-shot", 500*time.Millisecond)

		// All three are live on arrival.
		if got, ok := c.Get("session"); !ok || got != "alice-1" {
			t.Errorf(`Get("session") = %q, %t, want "alice-1", true`, got, ok)
		}
		if got, ok := c.Get("refresh"); !ok || got != "alice-refresh" {
			t.Errorf(`Get("refresh") = %q, %t, want "alice-refresh", true`, got, ok)
		}
		if got, ok := c.Get("blip"); !ok || got != "one-shot" {
			t.Errorf(`Get("blip") = %q, %t, want "one-shot", true`, got, ok)
		}
		if n := c.Len(); n != 3 {
			t.Errorf("Len() = %d, want 3", n)
		}

		// Past its 0.5s deadline the token is a miss even though the
		// first sweep is still 400ms away: the miss below is the
		// reader's own doing, not the janitor's.
		time.Sleep(600 * time.Millisecond)
		if _, ok := c.Get("blip"); ok {
			t.Error(`Get("blip") hit past its deadline, want a miss before any sweep`)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d once blip expired, want 2", n)
		}

		// Replacing a live token renews both value and deadline: the
		// original lease would end 2s in, the renewed one 3.5s in.
		// blip comes back for a second life with no TTL at all.
		time.Sleep(900 * time.Millisecond) // 1.5s in
		c.Set("session", "alice-2", 2*time.Second)
		c.Set("blip", "one-shot-2", 0)
		time.Sleep(time.Second) // 2.5s in, past the original session deadline
		if got, ok := c.Get("session"); !ok || got != "alice-2" {
			t.Errorf(`Get("session") = %q, %t after replacement, want "alice-2", true`, got, ok)
		}
		if got, ok := c.Get("blip"); !ok || got != "one-shot-2" {
			t.Errorf(`Get("blip") = %q, %t stored without a TTL, want "one-shot-2", true`, got, ok)
		}

		// Replacement swings the deadline both ways: a negative TTL
		// behaves like zero, discarding session's pending 3.5s deadline,
		// while a positive TTL arms a deadline on blip where none existed.
		c.Set("session", "alice-3", -time.Minute)
		c.Set("blip", "one-shot-3", time.Second) // deadline 3.5s in
		time.Sleep(2 * time.Second)              // 4.5s in
		if got, ok := c.Get("session"); !ok || got != "alice-3" {
			t.Errorf(`Get("session") = %q, %t past its discarded deadline, want "alice-3", true`, got, ok)
		}
		if _, ok := c.Get("blip"); ok {
			t.Error(`Get("blip") hit past the deadline its replacement armed, want a miss`)
		}

		// Four sweeps have run by now; the two remaining tokens must
		// have survived them all.
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d after four sweeps, want 2", n)
		}

		// Close discards even the immortal entries and turns Set into a
		// no-op; closing again is safe. Get misses everything after
		// Close no matter what Set did, so Len is the assertion that the
		// late token was truly not stored.
		c.Close()
		if _, ok := c.Get("refresh"); ok {
			t.Error(`Get("refresh") hit after Close, want a miss`)
		}
		c.Set("late", "ignored", 0)
		if n := c.Len(); n != 0 {
			t.Errorf("Len() = %d after Close and a late Set, want 0", n)
		}
		c.Close()
	})
}

// Every method promises safety for concurrent use, and a session store
// earns its keep under exactly that load: logins writing tokens while
// request handlers look them up. Each goroutine plays one user's login
// and lookups and asserts its own token round-trips; the final count
// audits that no write was lost.
func TestConcurrentUse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second)
		defer c.Close()

		var wg sync.WaitGroup
		for i := range 8 {
			wg.Go(func() {
				key := fmt.Sprintf("sess-%d", i)
				token := fmt.Sprintf("token-%d", i)
				c.Set(key, token, time.Minute)
				if got, ok := c.Get(key); !ok || got != token {
					t.Errorf("Get(%q) = %q, %t, want %q, true", key, got, ok, token)
				}
				if got, ok := c.Get(key); !ok || got != token {
					t.Errorf("Get(%q) = %q, %t on a second lookup, want %q, true", key, got, ok, token)
				}
			})
		}
		wg.Wait()
		if n := c.Len(); n != 8 {
			t.Errorf("Len() = %d after 8 concurrent logins, want 8", n)
		}
	})
}

func TestNewPanicsOnNonpositiveInterval(t *testing.T) {
	cases := []struct {
		interval   time.Duration
		invalidity string
	}{
		{interval: 0, invalidity: "zero interval"},
		{interval: -time.Minute, invalidity: "negative interval"},
	}
	for _, c := range cases {
		mustPanic(t, c.interval, c.invalidity)
	}
}

// Fails the test unless New(interval) panics; invalidity names what
// makes the interval invalid in the failure message.
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
