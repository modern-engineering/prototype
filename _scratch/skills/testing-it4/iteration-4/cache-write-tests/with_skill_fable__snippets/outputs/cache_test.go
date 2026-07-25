package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// This example stores a session token under its session ID and looks it
// up on a later request.
func Example() {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("sess-42", "token-abc123", time.Hour)

	if token, ok := c.Get("sess-42"); ok {
		fmt.Println(token)
	}
	if _, ok := c.Get("sess-7"); !ok {
		fmt.Println("unknown session")
	}
	// Output:
	// token-abc123
	// unknown session
}

// A session token is visible exactly until its TTL elapses and invisible
// from that instant on. The janitor interval exceeds every TTL here, so
// the misses below prove that reads enforce expiry on their own.
func TestCache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		if _, ok := c.Get("sess-1"); ok {
			t.Error("Get on an empty cache reported a hit")
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() of an empty cache = %d; want 0", n)
		}

		c.Set("sess-1", "alpha", 10*time.Minute)
		c.Set("sess-2", "beta", 30*time.Minute)

		if got, ok := c.Get("sess-1"); !ok || got != "alpha" {
			t.Errorf(`Get("sess-1") = %q, %v; want "alpha", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d; want 2", n)
		}

		// Exactly at sess-1's deadline: "from then on" includes the
		// deadline instant itself.
		time.Sleep(10 * time.Minute)

		if got, ok := c.Get("sess-1"); ok {
			t.Errorf(`Get("sess-1") past its TTL = %q, true; want a miss`, got)
		}
		if got, ok := c.Get("sess-2"); !ok || got != "beta" {
			t.Errorf(`Get("sess-2") = %q, %v; want "beta", true`, got, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() after one expiry = %d; want 1", n)
		}

		time.Sleep(20 * time.Minute)

		if n := c.Len(); n != 0 {
			t.Errorf("Len() after every TTL = %d; want 0", n)
		}
	})
}

// Overwriting a key adopts the new value and the new TTL in both
// directions: a longer TTL keeps the entry past the old deadline, and a
// fresh TTL on a previously unexpiring entry starts its clock.
func TestSetReplacesValueAndTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(24 * time.Hour)
		defer c.Close()

		c.Set("sess", "short", 10*time.Minute)
		c.Set("sess", "long", 40*time.Minute)

		time.Sleep(20 * time.Minute)
		if got, ok := c.Get("sess"); !ok || got != "long" {
			t.Errorf(`Get after extending the TTL = %q, %v; want "long", true`, got, ok)
		}

		c.Set("sess", "forever", 0)
		time.Sleep(48 * time.Hour)
		if got, ok := c.Get("sess"); !ok || got != "forever" {
			t.Errorf(`Get after removing the expiry = %q, %v; want "forever", true`, got, ok)
		}

		c.Set("sess", "brief", time.Minute)
		time.Sleep(time.Minute)
		if got, ok := c.Get("sess"); ok {
			t.Errorf(`Get after restoring a TTL = %q, true; want a miss`, got)
		}
	})
}

// Entries stored with zero or negative TTL never expire: a week of
// hourly janitor sweeps leaves them untouched.
func TestNonPositiveTTLNeverExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("zero", "z", 0)
		c.Set("negative", "n", -time.Minute)

		time.Sleep(7 * 24 * time.Hour)

		if got, ok := c.Get("zero"); !ok || got != "z" {
			t.Errorf(`Get("zero") = %q, %v; want "z", true`, got, ok)
		}
		if got, ok := c.Get("negative"); !ok || got != "n" {
			t.Errorf(`Get("negative") = %q, %v; want "n", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d; want 2", n)
		}
	})
}

// Handlers on separate goroutines store and read back their own
// sessions while the janitor runs; every token must come back intact
// and be counted.
func TestConcurrentClients(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Minute)
		defer c.Close()

		const clients = 8
		for i := range clients {
			go func() {
				key := fmt.Sprintf("sess-%d", i)
				c.Set(key, "token-"+key, time.Hour)
				if got, ok := c.Get(key); !ok || got != "token-"+key {
					t.Errorf("Get(%q) = %q, %v; want %q, true", key, got, ok, "token-"+key)
				}
			}()
		}
		synctest.Wait()

		if n := c.Len(); n != clients {
			t.Errorf("Len() = %d; want %d", n, clients)
		}
	})
}

// After Close the cache stays empty and inert: reads miss, writes are
// dropped, and closing again is a no-op rather than a panic.
func TestCloseIsFinal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Minute)
		c.Set("sess", "token", time.Hour)
		c.Close()

		if got, ok := c.Get("sess"); ok {
			t.Errorf("Get after Close = %q, true; want a miss", got)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d; want 0", n)
		}

		c.Set("sess", "token", time.Hour)
		if _, ok := c.Get("sess"); ok {
			t.Error("Set after Close stored an entry")
		}

		c.Close()
	})
}

// A cache whose janitor could never run must not be constructed: zero
// and negative sweep intervals panic instead.
func TestNewRejectsNonPositiveInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Minute} {
		mustPanic(t, fmt.Sprintf("New(%v)", interval), func() {
			cache.New(interval)
		})
	}
}

// Fails the test, naming the offending call, when fn returns without
// panicking.
func mustPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s returned; want panic", what)
		}
	}()
	fn()
}
