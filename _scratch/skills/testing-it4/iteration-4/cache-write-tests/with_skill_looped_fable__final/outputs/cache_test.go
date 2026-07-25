package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// Session tokens are stored with a TTL; lookups report whether a live
// token was found, so a miss covers both "never stored" and "expired".
func ExampleCache() {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("session-alice", "token-123", 30*time.Minute)

	if token, ok := c.Get("session-alice"); ok {
		fmt.Println("alice:", token)
	}
	if _, ok := c.Get("session-bob"); !ok {
		fmt.Println("bob: no session")
	}

	// After its TTL passes, a stored token reads exactly like one
	// that was never stored.
	c.Set("session-carol", "token-456", time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	if _, ok := c.Get("session-carol"); !ok {
		fmt.Println("carol: no session")
	}
	// Output:
	// alice: token-123
	// bob: no session
	// carol: no session
}

// A session-token cache in ordinary use: tokens are stored with a TTL,
// looked up while fresh, refreshed to extend their life, and gone once
// the TTL passes. The janitor interval is an hour so it never fires
// here: misses after expiry prove the read-side enforcement the docs
// promise, not the sweep. Close empties the cache and turns every
// later call into a no-op.
func TestCache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		t.Cleanup(c.Close)

		c.Set("alice", "token-a", 10*time.Minute)
		c.Set("bob", "token-b", 10*time.Minute)
		if got, ok := c.Get("alice"); !ok || got != "token-a" {
			t.Errorf(`Get("alice") = %q, %t, want "token-a", true`, got, ok)
		}
		if got, ok := c.Get("carol"); ok {
			t.Errorf(`Get("carol") = %q, true, want a miss for a key never set`, got)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2", n)
		}

		// Refreshing a session is a plain Set: the replacement expires
		// ten minutes from now, not from the original Set.
		time.Sleep(5 * time.Minute)
		c.Set("alice", "token-a2", 10*time.Minute)

		time.Sleep(6 * time.Minute) // alice has 4m left; bob expired 1m ago
		if got, ok := c.Get("alice"); !ok || got != "token-a2" {
			t.Errorf(`Get("alice") after refresh = %q, %t, want "token-a2", true`, got, ok)
		}
		if got, ok := c.Get("bob"); ok {
			t.Errorf(`Get("bob") = %q, true, want a miss 1m past its TTL`, got)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d, want 1", n)
		}

		c.Close()
		if got, ok := c.Get("alice"); ok {
			t.Errorf(`Get("alice") after Close = %q, true, want a miss`, got)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
		c.Set("dave", "token-d", 10*time.Minute)
		if got, ok := c.Get("dave"); ok {
			t.Errorf(`Get("dave") = %q, true, want a miss for a Set made after Close`, got)
		}
		c.Close() // a second Close must be a safe no-op
	})
}

// Handler goroutines share one cache, each storing and looking up its
// own session: the anticipated multi-goroutine flow behind the docs'
// promise that all methods are safe for concurrent use.
func TestConcurrentUse(t *testing.T) {
	c := cache.New(time.Hour)
	t.Cleanup(c.Close)

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Go(func() {
			key := fmt.Sprintf("session-%d", i)
			token := fmt.Sprintf("token-%d", i)
			c.Set(key, token, time.Hour)
			if got, ok := c.Get(key); !ok || got != token {
				t.Errorf("Get(%q) = %q, %t, want %q, true", key, got, ok, token)
			}
		})
	}
	wg.Wait()

	if n := c.Len(); n != 16 {
		t.Errorf("Len() = %d, want 16", n)
	}
}

// An entry stored with a zero or negative TTL has no expiry: it must
// outlive any amount of time and any number of janitor sweeps, until
// it is replaced or the cache is closed.
func TestEntryWithoutTTLNeverExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Minute)
		t.Cleanup(c.Close)

		c.Set("forever", "v", 0)
		c.Set("negative", "v", -time.Second)

		time.Sleep(24 * time.Hour)
		if got, ok := c.Get("forever"); !ok || got != "v" {
			t.Errorf(`Get("forever") = %q, %t, want "v", true for an entry stored with ttl 0`, got, ok)
		}
		if got, ok := c.Get("negative"); !ok || got != "v" {
			t.Errorf(`Get("negative") = %q, %t, want "v", true for an entry stored with a negative ttl`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2", n)
		}

		// "Until it is replaced": a later Set with a TTL takes over,
		// and the entry expires like any other.
		c.Set("forever", "v2", time.Minute)
		time.Sleep(2 * time.Minute)
		if got, ok := c.Get("forever"); ok {
			t.Errorf(`Get("forever") = %q, true, want a miss after the entry was replaced with a 1m TTL`, got)
		}
	})
}

func TestNewRejectsNonPositiveJanitorInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		t.Run(fmt.Sprint(interval), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%v) did not panic", interval)
				}
			}()
			c := cache.New(interval)
			c.Close()
		})
	}
}
