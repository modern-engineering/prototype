package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// The everyday use for a session store: open a cache, stash a token under
// its user's key with a lifetime, and read it back when the next request
// for that user arrives. A key that was never issued is an ordinary miss.
// Close releases the background janitor at shutdown.
func Example() {
	c := cache.New(time.Minute) // the janitor sweeps expired tokens each minute
	defer c.Close()

	c.Set("user-42", "tok-abc123", time.Hour)

	if tok, ok := c.Get("user-42"); ok {
		fmt.Println("user-42 token:", tok)
	}
	if _, ok := c.Get("user-99"); !ok {
		fmt.Println("user-99 has no session")
	}

	// Output:
	// user-42 token: tok-abc123
	// user-99 has no session
}

// One session token's whole life in a single pass on virtual time. It is
// stored, read back, and re-issued in place rather than duplicated. The
// load-bearing promise is that expiry is enforced on read: the moment a
// TTL lapses the entry reads as gone, and it does so well before the
// janitor's first sweep could have removed it. A token given no TTL must
// outlive every sweep. Close finally empties the store and turns Set into
// a no-op and every Get into a miss; the deferred Close then proves the
// call is idempotent.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second) // sweep once a second
		defer c.Close()

		// A token good for three seconds, and one with no expiry.
		c.Set("sess-a", "alpha", 3*time.Second)
		c.Set("sess-b", "beta", 0)
		if v, ok := c.Get("sess-a"); !ok || v != "alpha" {
			t.Errorf(`Get("sess-a") = %q, %v; want "alpha", true`, v, ok)
		}
		if v, ok := c.Get("sess-b"); !ok || v != "beta" {
			t.Errorf(`Get("sess-b") = %q, %v; want "beta", true`, v, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2", n)
		}

		// Re-issuing a key overwrites its value in place; it is one entry.
		c.Set("sess-a", "alpha2", 3*time.Second)
		if v, ok := c.Get("sess-a"); !ok || v != "alpha2" {
			t.Errorf(`Get("sess-a") after re-issue = %q, %v; want "alpha2", true`, v, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() after re-issue = %d, want 2", n)
		}

		// A short-lived token read back just past its deadline but far short
		// of the first sweep at the one-second mark: it must already miss,
		// and Len must not count it, though the janitor has not touched it.
		c.Set("sess-c", "gamma", 50*time.Millisecond)
		time.Sleep(100 * time.Millisecond)
		if _, ok := c.Get("sess-c"); ok {
			t.Error(`Get("sess-c") hit past its TTL; want a miss enforced on read, before any sweep`)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() with an expired-but-unswept entry = %d, want 2", n)
		}

		// Advance past sess-a's deadline; three sweeps run along the way.
		// The expired token is gone and the no-TTL token has survived them all.
		time.Sleep(3 * time.Second)
		if _, ok := c.Get("sess-a"); ok {
			t.Error(`Get("sess-a") hit after its TTL; want a miss`)
		}
		if v, ok := c.Get("sess-b"); !ok || v != "beta" {
			t.Errorf(`Get("sess-b") after sweeps = %q, %v; want "beta", true (no TTL never expires)`, v, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d, want 1 (only the no-TTL token remains)", n)
		}

		// Shutdown discards everything and turns callers away.
		c.Close()
		if _, ok := c.Get("sess-b"); ok {
			t.Error(`Get("sess-b") after Close hit; want a miss`)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
		c.Set("sess-d", "delta", time.Hour)
		if _, ok := c.Get("sess-d"); ok {
			t.Error("Set after Close was honored; want it ignored")
		}
		// The deferred Close is a second call and must be a safe no-op.
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

// mustPanic fails the test unless New(interval) panics; invalidity names
// what makes the interval invalid in the failure message.
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
