package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: a login deposits the token
// with its lifetime, each request looks it up, and an expired or
// unknown token is simply a miss.
func Example_sessionTokens() {
	// Sweep expired sessions once a minute.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on
	// an early return.
	defer c.Close()

	// A login stores the session token for its 30-minute lifetime.
	c.Set("session-abc", "user-1", 30*time.Minute)

	// A zero (or negative) ttl stores an entry with no expiry: the
	// signing key lives until it is replaced or the cache is closed.
	c.Set("signing-key", "k1", 0)

	// A request presents the token; a hit authenticates it.
	if user, ok := c.Get("session-abc"); ok {
		fmt.Println("authenticated as", user)
	}

	// A token the cache never saw, or one past its TTL, is a miss.
	if _, ok := c.Get("session-xyz"); !ok {
		fmt.Println("session-xyz rejected")
	}

	// Output:
	// authenticated as user-1
	// session-xyz rejected
}

// The store's whole day in one sitting: a miss on the empty cache, a
// refresh that carries a live token past its first deadline, a death
// at the very instant of the refreshed one, a Set that revives the
// dead key, a two-sweep lull that only the no-expiry entries outlive,
// a replacement that ends the signing key's immortality, and a
// shutdown that blanks every read and write. The janitor sweeps
// hourly, so the expiries observed before the lull, and the replaced
// key's death after it, come from read-time enforcement, never from
// the sweep.
func TestSessionTokenLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		if v, ok := c.Get("token"); ok {
			t.Errorf(`Get("token") on an empty cache = %q, true, want a miss`, v)
		}

		c.Set("token", "alice", 2*time.Second)
		c.Set("api-key", "s3cret", 0)            // zero ttl: no expiry
		c.Set("signing-key", "k1", -time.Minute) // negative ttl: no expiry as well
		if n := c.Len(); n != 3 {
			t.Errorf("Len() = %d, want 3 after three Sets", n)
		}

		// Halfway through the TTL the token is still live.
		time.Sleep(time.Second)
		if v, ok := c.Get("token"); !ok || v != "alice" {
			t.Errorf(`Get("token") 1s into a 2s TTL = %q, %t, want "alice", true`, v, ok)
		}

		// A refresh: Set replaces the live entry and restarts its 2s TTL.
		c.Set("token", "alice", 2*time.Second)

		// The original deadline passes with the token still live: the
		// refresh replaced the old entry, deadline and all.
		time.Sleep(time.Second)
		if v, ok := c.Get("token"); !ok || v != "alice" {
			t.Errorf(`Get("token") at its pre-refresh deadline = %q, %t, want "alice", true`, v, ok)
		}

		// The prose reads inclusive: "from then on Get misses it"
		// starts at the deadline instant itself. Asserted as read,
		// pending the owner's word.
		time.Sleep(time.Second)
		if v, ok := c.Get("token"); ok {
			t.Errorf(`Get("token") at its deadline = %q, true, want a miss`, v)
		}

		// A nanosecond later the boundary question is moot, and the
		// janitor's first sweep is still an hour off: the expired
		// token vanishes from Len on read alone.
		time.Sleep(time.Nanosecond)
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2 once the token has expired", n)
		}

		// Setting the expired key again revives it with a fresh TTL.
		c.Set("token", "bob", time.Second)
		if v, ok := c.Get("token"); !ok || v != "bob" {
			t.Errorf(`Get("token") after a fresh Set = %q, %t, want "bob", true`, v, ok)
		}

		// A lull long enough for two janitor sweeps: the no-expiry
		// entries come through it intact, and the expired token stays
		// a miss.
		time.Sleep(2 * time.Hour)
		if v, ok := c.Get("api-key"); !ok || v != "s3cret" {
			t.Errorf(`Get("api-key") after the lull = %q, %t, want "s3cret", true`, v, ok)
		}
		if v, ok := c.Get("signing-key"); !ok || v != "k1" {
			t.Errorf(`Get("signing-key") after the lull = %q, %t, want "k1", true`, v, ok)
		}
		if v, ok := c.Get("token"); ok {
			t.Errorf(`Get("token") after the lull = %q, true, want a miss`, v)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d after the lull, want 2: only the no-expiry entries survive", n)
		}

		// Replacing the no-expiry signing key ends its immortality:
		// the new entry carries the short deadline it was given and
		// dies by it.
		c.Set("signing-key", "k2", time.Second)
		if v, ok := c.Get("signing-key"); !ok || v != "k2" {
			t.Errorf(`Get("signing-key") after its replacement = %q, %t, want "k2", true`, v, ok)
		}
		time.Sleep(2 * time.Second)
		if v, ok := c.Get("signing-key"); ok {
			t.Errorf(`Get("signing-key") past its replacement TTL = %q, true, want a miss`, v)
		}

		// Shutdown discards everything: reads miss, writes are ignored,
		// and closing again is a safe no-op.
		c.Close()
		if v, ok := c.Get("api-key"); ok {
			t.Errorf(`Get("api-key") after Close = %q, true, want a miss`, v)
		}
		c.Set("late", "ignored", time.Minute)
		if v, ok := c.Get("late"); ok {
			t.Errorf(`Get("late") = %q, true, want a miss: Set after Close must do nothing`, v)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() = %d after Close, want 0: Close discards all entries", n)
		}
		c.Close()
	})
}

// A session store lives under concurrent request handlers, and the
// docs promise every method is safe for that. Eight handlers each
// deposit and read back their own token while all of them share reads
// of one signing key; the final count proves no write was lost, and
// the race detector watches the rest.
func TestConcurrentUse(t *testing.T) {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("signing-key", "k1", 0) // no expiry: every handler reads it

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token := fmt.Sprintf("session-%d", i)
			c.Set(token, "user", 30*time.Minute)
			if v, ok := c.Get(token); !ok || v != "user" {
				t.Errorf(`Get(%q) = %q, %t, want "user", true`, token, v, ok)
			}
			if v, ok := c.Get("signing-key"); !ok || v != "k1" {
				t.Errorf(`Get("signing-key") = %q, %t, want "k1", true`, v, ok)
			}
		}()
	}
	wg.Wait()

	if n := c.Len(); n != 9 {
		t.Errorf("Len() = %d after 8 concurrent Sets, want 9 with the signing key", n)
	}
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
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
	c.Close() // reached only if the guard is broken; stop the janitor anyway
}
