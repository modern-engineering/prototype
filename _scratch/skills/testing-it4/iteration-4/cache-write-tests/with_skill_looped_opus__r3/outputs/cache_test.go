package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session server is the archetypal caller: it caches each freshly issued
// token under a short TTL, keeps long-lived credentials with no TTL at all,
// and treats a key it never stored as a prompt to sign in again.
func Example_sessionTokens() {
	// New's argument is how often the janitor sweeps, not any entry's
	// lifetime; each entry carries its own TTL from its Set call.
	c := cache.New(time.Minute)
	// Always Close so the janitor goroutine is released, even on an early
	// return.
	defer c.Close()

	// A minted session token, good for a quarter hour.
	c.Set("session-abc", "user-42", 15*time.Minute)

	// A zero TTL stores the value without an expiry: it lives until
	// something replaces it or the cache closes.
	c.Set("service-key", "svc-secret", 0)

	// A negative TTL means the same as zero: the entry is stored without an
	// expiry, never treated as already-expired.
	c.Set("api-key", "api-secret", -1)

	if user, ok := c.Get("session-abc"); ok {
		fmt.Println("welcome back,", user)
	}
	if secret, ok := c.Get("service-key"); ok {
		fmt.Println("service credential:", secret)
	}
	if key, ok := c.Get("api-key"); ok {
		fmt.Println("api credential:", key)
	}

	// A key the cache never issued is a miss.
	if _, ok := c.Get("session-xyz"); !ok {
		fmt.Println("session-xyz: please sign in")
	}

	// Closing again is safe: Close is idempotent, so the deferred Close
	// above becomes a no-op.
	c.Close()

	// Output:
	// welcome back, user-42
	// service credential: svc-secret
	// api credential: api-secret
	// session-xyz: please sign in
}

// The whole session-token story on one virtual clock. The janitor is set to
// sweep only once an hour, so every expiry observed here is enforced on read
// rather than by the background sweep: a token expires the instant its TTL
// elapses, a re-issue refreshes that deadline, a zero-TTL credential never
// expires, and Close discards everything and turns Set into a no-op.
func TestExpiryLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("session", "user-42", 15*time.Minute) // deadline at t=15m
		c.Set("service", "svc-key", 0)              // no deadline, ever

		if got, ok := c.Get("session"); !ok || got != "user-42" {
			t.Fatalf(`Get("session") = %q, %v; want "user-42", true`, got, ok)
		}
		if got, ok := c.Get("service"); !ok || got != "svc-key" {
			t.Fatalf(`Get("service") = %q, %v; want "svc-key", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Fatalf("Len() = %d, want 2 with both entries live before the timeline starts", n)
		}

		// At t=10m, five minutes short of the token's expiry, the session is
		// re-issued. The replacement's TTL is measured from now, so its
		// deadline moves to t=25m rather than staying at the original t=15m.
		time.Sleep(10 * time.Minute)
		c.Set("session", "user-99", 15*time.Minute)

		// At t=16m, past the original deadline. The token survives only
		// because the re-issue refreshed it; had Set kept the first deadline,
		// this would already be a miss.
		time.Sleep(6 * time.Minute)
		if got, ok := c.Get("session"); !ok || got != "user-99" {
			t.Errorf(`Get("session") past the original TTL = %q, %v; want "user-99", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() past the original TTL = %d, want 2", n)
		}

		// At t=26m, past the refreshed deadline. The token is a miss on read,
		// still well ahead of the hourly sweep, while the no-TTL credential
		// is untouched.
		time.Sleep(10 * time.Minute)
		if _, ok := c.Get("session"); ok {
			t.Error(`Get("session") past its refreshed TTL = hit; want a miss on read before the sweep`)
		}
		if got, ok := c.Get("service"); !ok || got != "svc-key" {
			t.Errorf(`Get("service") = %q, %v; want "svc-key", true: no TTL means no expiry`, got, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() after the token expired = %d, want 1: only the no-TTL credential", n)
		}

		// Close discards every entry and releases the janitor.
		c.Close()
		c.Set("service", "reborn", time.Minute) // a no-op once closed
		if _, ok := c.Get("service"); ok {
			t.Error(`Get("service") after Close = hit; want a miss for every key`)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}

		c.Close() // idempotent
	})
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
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

// mustPanic fails the test unless New(interval) panics; invalidity names what
// makes the interval invalid in the failure message.
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
