package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: mint a token with a TTL on
// login, look it up on every request, and overwrite it on refresh.
func Example_sessionStore() {
	// Sweep expired sessions out of memory once a minute.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on
	// an early return.
	defer c.Close()

	// Login: the token lives for an hour; an hour from now this
	// lookup starts missing.
	c.Set("user-42", "tok-A", time.Hour)
	if tok, ok := c.Get("user-42"); ok {
		fmt.Println("request sees", tok)
	}

	// A token that was never issued is a miss.
	_, ok := c.Get("user-7")
	fmt.Println("unknown user, hit:", ok)

	// Refresh: Set replaces the old token, and its TTL starts over.
	c.Set("user-42", "tok-B", time.Hour)
	if tok, ok := c.Get("user-42"); ok {
		fmt.Println("after refresh:", tok)
	}

	// Logout everywhere: Close discards every entry, so lookups miss.
	// Closing again is safe, so the deferred Close becomes a no-op.
	c.Close()
	_, ok = c.Get("user-42")
	fmt.Println("after close, hit:", ok)

	// Output:
	// request sees tok-A
	// unknown user, hit: false
	// after refresh: tok-B
	// after close, hit: false
}

// A session store's day end to end: a rush of concurrent logins mints
// tokens that miss the moment their TTL runs out, well before the
// janitor's first sweep; a refresh restarts the clock; replacing an
// entry crosses expiry classes in both directions; entries without a
// TTL outlive every sweep; and Close empties the store for good while
// staying safe to call twice.
func TestSessionLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// The janitor sweeps only once an hour, so every expiry
		// observed in the first hour is enforced by the read itself.
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("pinned", "cfg", 0)
		c.Set("archive", "old", -time.Second)

		// Both are live: zero and negative TTLs mean no expiry, not
		// instant expiry.
		if got, ok := c.Get("pinned"); !ok || got != "cfg" {
			t.Errorf(`Get("pinned") = %q, %t, want "cfg", true`, got, ok)
		}
		if got, ok := c.Get("archive"); !ok || got != "old" {
			t.Errorf(`Get("archive") = %q, %t, want "old", true`, got, ok)
		}

		// A login rush: the anticipated caller is a fleet of request
		// handlers, and the docs promise every method is safe for
		// concurrent use. Each handler mints its own token and reads
		// a shared entry.
		var wg sync.WaitGroup
		for i := range 3 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				key := fmt.Sprintf("user-%d", i)
				tok := fmt.Sprintf("tok-%d", i)
				c.Set(key, tok, 10*time.Minute)
				if got, ok := c.Get(key); !ok || got != tok {
					t.Errorf("Get(%q) = %q, %t, want %q, true", key, got, ok, tok)
				}
				if got, ok := c.Get("pinned"); !ok || got != "cfg" {
					t.Errorf(`Get("pinned") = %q, %t, want "cfg", true`, got, ok)
				}
			}()
		}
		wg.Wait()
		if n := c.Len(); n != 5 {
			t.Errorf("Len() = %d, want 5: two config entries and a token per handler", n)
		}

		// Five minutes in, user-1 refreshes: the replacement's
		// ten-minute TTL starts over from now.
		time.Sleep(5 * time.Minute)
		c.Set("user-1", "tok-1b", 10*time.Minute)
		// User-2 elects "remember me": the same token, stored again
		// without a TTL. Set replaces the whole entry, so the
		// original ten-minute deadline goes with it.
		c.Set("user-2", "tok-2", 0)

		// Ten minutes after login the unrefreshed token misses, at
		// its deadline exactly; the janitor's first sweep is still 50
		// minutes away, so the miss is expiry-on-read. The
		// remember-me token sails past the deadline it was minted with.
		time.Sleep(5 * time.Minute)
		if got, ok := c.Get("user-0"); ok {
			t.Errorf(`Get("user-0") at its deadline = %q, true, want a miss`, got)
		}
		if got, ok := c.Get("user-2"); !ok || got != "tok-2" {
			t.Errorf(`Get("user-2") past its cleared deadline = %q, %t, want "tok-2", true`, got, ok)
		}
		if n := c.Len(); n != 4 {
			t.Errorf("Len() = %d, want 4: an expired entry must not be counted", n)
		}

		// Four minutes later the original deadline is long gone, but
		// the refreshed token has a minute left.
		time.Sleep(4 * time.Minute)
		if got, ok := c.Get("user-1"); !ok || got != "tok-1b" {
			t.Errorf(`Get("user-1") after a refresh = %q, %t, want "tok-1b", true`, got, ok)
		}
		// That minute passes and the refreshed deadline lands: ten
		// minutes from the refresh, not from login.
		time.Sleep(time.Minute)
		if got, ok := c.Get("user-1"); ok {
			t.Errorf(`Get("user-1") at its refreshed deadline = %q, true, want a miss`, got)
		}

		// A day passes and the janitor sweeps two dozen times; the
		// entries stored without a TTL survive every sweep.
		time.Sleep(24 * time.Hour)
		if got, ok := c.Get("pinned"); !ok || got != "cfg" {
			t.Errorf(`Get("pinned") a day later = %q, %t, want "cfg", true`, got, ok)
		}
		if got, ok := c.Get("archive"); !ok || got != "old" {
			t.Errorf(`Get("archive") a day later = %q, %t, want "old", true`, got, ok)
		}
		if got, ok := c.Get("user-2"); !ok || got != "tok-2" {
			t.Errorf(`Get("user-2") a day later = %q, %t, want "tok-2", true`, got, ok)
		}

		// Replacement crosses expiry classes the other way too: a
		// ten-minute Set over the never-expiring archive puts it on
		// the clock.
		c.Set("archive", "old", 10*time.Minute)
		time.Sleep(10 * time.Minute)
		if got, ok := c.Get("archive"); ok {
			t.Errorf(`Get("archive") at its re-set deadline = %q, true, want a miss`, got)
		}

		// Shutdown discards even the entries that would never expire,
		// and a late Set is dropped rather than stored.
		c.Close()
		if got, ok := c.Get("pinned"); ok {
			t.Errorf(`Get("pinned") after Close = %q, true, want a miss`, got)
		}
		c.Set("late", "tok-3", time.Hour)
		if got, ok := c.Get("late"); ok {
			t.Errorf(`Get("late") = %q, true, want a miss: Set after Close must store nothing`, got)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
	})
}

func TestNewPanicsOnInvalidInterval(t *testing.T) {
	cases := []struct {
		interval   time.Duration
		invalidity string
	}{
		{interval: 0, invalidity: "zero janitor interval"},
		{interval: -time.Minute, invalidity: "negative janitor interval"},
	}
	for _, c := range cases {
		mustPanic(t, c.interval, c.invalidity)
	}
}

// mustPanic fails the test unless New(interval) panics; invalidity
// names what makes the interval invalid in the failure message.
func mustPanic(t *testing.T, interval time.Duration, invalidity string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a %s", interval, invalidity)
		}
	}()
	c := cache.New(interval)
	c.Close() // reached only if the guard is broken; release the janitor anyway
}
