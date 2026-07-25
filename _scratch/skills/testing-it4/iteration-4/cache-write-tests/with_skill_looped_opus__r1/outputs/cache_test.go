package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// The everyday session-token pattern: mint a token under a key, read it
// back while the session is live, and treat an unknown key as a miss.
func Example() {
	// The janitor interval only governs when dead entries are reclaimed
	// from memory; the per-token TTL is what governs visibility.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	c.Set("session-7f3a", "user-42", time.Hour)

	if user, ok := c.Get("session-7f3a"); ok {
		fmt.Println("request from", user)
	}

	// A non-positive TTL stores a value with no expiry: it lives until it is
	// replaced or the cache is closed. Handy for a long-lived service key that
	// sits beside the expiring session tokens.
	c.Set("service-key", "svc-42", 0)
	if key, ok := c.Get("service-key"); ok {
		fmt.Println("service key", key)
	}

	// A token nobody issued is a miss, not an empty string mistaken for a
	// value.
	if _, ok := c.Get("session-0000"); !ok {
		fmt.Println("no such session")
	}

	// Output:
	// request from user-42
	// service key svc-42
	// no such session
}

// The session store's life story on virtual time: a token read back while
// live, the same token gone the instant its TTL lapses (a read-time miss
// that beats the next janitor sweep), a replacement overwriting a key,
// a no-expiry entry that outlives a janitor sweep, and a shutdown that
// discards everything and turns every key into a miss.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// A one-hour janitor so nothing is swept during the early reads;
		// this isolates expiry-on-read from the janitor's later sweep.
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("sess", "user-42", time.Minute)
		if v, ok := c.Get("sess"); !ok || v != "user-42" {
			t.Fatalf(`Get("sess") = %q, %t; want "user-42", true`, v, ok)
		}
		if n := c.Len(); n != 1 {
			t.Fatalf("Len() = %d after one Set, want 1", n)
		}

		// A minute and a half in, the token's one-minute TTL has lapsed but
		// the hourly janitor has not run: the entry is still in the map,
		// yet Get and Len must already treat it as gone.
		time.Sleep(90 * time.Second)
		if v, ok := c.Get("sess"); ok {
			t.Errorf(`Get("sess") = %q, true after its TTL; want a miss`, v)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() = %d with only an expired entry, want 0", n)
		}

		// A non-positive TTL stores a value with no expiry; negative and zero
		// both take that branch, so exercise the negative one here and the
		// zero one on the replacement below.
		c.Set("perm", "forever", -time.Second)
		if v, ok := c.Get("perm"); !ok || v != "forever" {
			t.Fatalf(`Get("perm") = %q, %t; want "forever", true`, v, ok)
		}

		// Set replaces the entry under an existing key rather than adding a
		// second one.
		c.Set("perm", "updated", 0)
		if v, ok := c.Get("perm"); !ok || v != "updated" {
			t.Fatalf(`Get("perm") = %q, %t after replacement; want "updated", true`, v, ok)
		}
		if n := c.Len(); n != 1 {
			t.Fatalf("Len() = %d, want 1: replacement must not add a key", n)
		}

		// Past the first hourly sweep. An entry with no deadline is never
		// reclaimed, so it is still readable after the janitor has run.
		time.Sleep(time.Hour)
		synctest.Wait()
		if v, ok := c.Get("perm"); !ok || v != "updated" {
			t.Errorf(`Get("perm") = %q, %t after a janitor sweep; want "updated", true`, v, ok)
		}

		// Shutdown discards every entry and closes the store to writes.
		c.Close()
		if v, ok := c.Get("perm"); ok {
			t.Errorf(`Get("perm") = %q, true after Close; want a miss`, v)
		}
		c.Set("late", "value", time.Hour)
		if _, ok := c.Get("late"); ok {
			t.Error(`Get("late") = true after a post-Close Set; want a miss`)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() = %d after Close, want 0", n)
		}

		// Closing again is safe: Close is idempotent, so the deferred Close
		// above becomes a no-op.
		c.Close()
	})
}

// The type doc promises every method is safe for concurrent use, which is the
// whole reason a session store exists: many request handlers reach into one
// cache at once. This runs that flow directly, so the race detector guards the
// promise: a batch of callers each read a shared token while writing and
// reading back a session of their own.
func TestConcurrentCallers(t *testing.T) {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("shared", "everyone", time.Hour)

	const handlers = 8
	var wg sync.WaitGroup
	for i := range handlers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, ok := c.Get("shared"); !ok || v != "everyone" {
				t.Errorf(`Get("shared") = %q, %t; want "everyone", true`, v, ok)
			}
			key := fmt.Sprintf("sess-%d", i)
			c.Set(key, "live", time.Hour)
			if v, ok := c.Get(key); !ok || v != "live" {
				t.Errorf(`Get(%q) = %q, %t; want "live", true`, key, v, ok)
			}
		}()
	}
	wg.Wait()
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		mustPanic(t, interval)
	}
}

// mustPanic fails the test unless New(interval) panics; interval is echoed
// in the failure message to name the argument that should have been rejected.
func mustPanic(t *testing.T, interval time.Duration) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic on a non-positive interval", interval)
		}
	}()
	c := cache.New(interval)
	c.Close() // reached only if the guard is broken; release the janitor anyway
}
