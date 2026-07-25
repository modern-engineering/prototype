package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A session store is the typical caller: set a token with a TTL at login,
// read it back on later requests, and see it disappear once it expires.
func Example_sessionTokens() {
	c := cache.New(time.Hour) // janitor interval; won't fire during this example
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	c.Set("session-42", "tok-abc", 20*time.Millisecond)
	if v, ok := c.Get("session-42"); ok {
		fmt.Println(v)
	}

	// Long past the 20ms TTL, so the token is already gone from Get on its
	// own, well before the janitor's hour-long sweep could remove it.
	time.Sleep(40 * time.Millisecond)
	_, ok := c.Get("session-42")
	fmt.Println(ok)

	// Output:
	// tok-abc
	// false
}

// The typical session-cache story end to end: a token that outlives its
// TTL becomes invisible to Get and Len before the janitor ever runs, a
// replacement resets a still-live token's deadline to run from the
// replacement rather than from whatever it displaced, a permanent entry
// survives every wait, and Close discards everything and turns every
// later call into a no-op or a miss.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour) // interval far beyond this test's clock advances
		defer c.Close()

		// A short-lived token, live for the moment.
		c.Set("session", "tok-v1", time.Second)
		if v, ok := c.Get("session"); !ok || v != "tok-v1" {
			t.Fatalf(`Get("session") = %q, %v, want "tok-v1", true`, v, ok)
		}

		// A one-time password good for five seconds.
		c.Set("otp", "445566", 5*time.Second)
		if got := c.Len(); got != 2 {
			t.Fatalf("Len() = %d, want 2", got)
		}

		// Replace the token while it is still live, well ahead of its
		// original one-second deadline.
		time.Sleep(500 * time.Millisecond)
		c.Set("session", "tok-v2", 4*time.Second)
		if v, _ := c.Get("session"); v != "tok-v2" {
			t.Errorf(`Get("session") after replace = %q, want "tok-v2"`, v)
		}

		// Past the original token's one-second deadline, but the
		// replacement's own four-second deadline, counted from the
		// replace rather than the token it displaced, still has time left.
		time.Sleep(900 * time.Millisecond)
		if v, ok := c.Get("session"); !ok || v != "tok-v2" {
			t.Errorf(`Get("session") at 1.4s = %q, %v, want "tok-v2", true`, v, ok)
		}

		// Now past the replacement's own deadline too, while the OTP is
		// still shy of its five-second one.
		time.Sleep(3599 * time.Millisecond)
		if _, ok := c.Get("session"); ok {
			t.Error(`Get("session") at 4.999s = true, want false: its own replaced deadline has passed`)
		}
		if v, ok := c.Get("otp"); !ok || v != "445566" {
			t.Fatalf(`Get("otp") at 4.999s = %q, %v, want "445566", true`, v, ok)
		}

		// Past the deadline, Get and Len drop it on their own, long before
		// the one-hour janitor interval could.
		time.Sleep(2 * time.Millisecond)
		if _, ok := c.Get("otp"); ok {
			t.Error(`Get("otp") past its deadline = true, want false`)
		}
		if got := c.Len(); got != 0 {
			t.Errorf("Len() = %d, want 0 once both entries are gone", got)
		}

		// A permanent entry: zero TTL means no expiry at all.
		c.Set("token", "final", 0)
		if got := c.Len(); got != 1 {
			t.Fatalf("Len() = %d, want 1", got)
		}

		// Close discards every entry, including one that was never expired.
		c.Close()
		if _, ok := c.Get("token"); ok {
			t.Error("Get() after Close = true, want false")
		}
		if got := c.Len(); got != 0 {
			t.Errorf("Len() after Close = %d, want 0", got)
		}
		c.Set("new", "value", 0)
		if got := c.Len(); got != 0 {
			t.Errorf("Len() after Set following Close = %d, want 0", got)
		}
		c.Close() // idempotent
	})
}

// Sessions are set once at login and read on every later request from
// whatever goroutine handles it; that pattern must not race, and each
// goroutine's own reads must see what it just wrote.
func TestConcurrentAccessIsSafe(t *testing.T) {
	c := cache.New(time.Hour)
	defer c.Close()

	const callers = 8
	var wg sync.WaitGroup
	for n := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("session-%d", n)
			c.Set(key, "tok", time.Minute)
			for range 100 {
				if v, ok := c.Get(key); !ok || v != "tok" {
					t.Errorf("Get(%q) = %q, %v, want %q, true", key, v, ok, "tok")
				}
				c.Len()
			}
		}()
	}
	wg.Wait()

	if got := c.Len(); got != callers {
		t.Errorf("Len() = %d, want %d once every caller has set its key", got, callers)
	}
}

func TestNewPanicsOnInvalidJanitorInterval(t *testing.T) {
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

// Fails the test unless New(interval) panics; invalidity names what makes
// interval invalid in the failure message.
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
