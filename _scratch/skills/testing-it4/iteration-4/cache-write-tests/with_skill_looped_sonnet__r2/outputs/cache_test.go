package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// The package's own motivating case: a session token lands with a TTL,
// answers Get while it is live, and disappears the instant it lapses,
// well before the janitor would ever get around to sweeping it.
func Example_sessionToken() {
	c := cache.New(time.Hour) // a janitor slower than the TTL below on purpose
	defer c.Close()

	c.Set("session-a1", "dana", 200*time.Millisecond)

	if user, ok := c.Get("session-a1"); ok {
		fmt.Println("authenticated as", user)
	}

	time.Sleep(time.Second) // five times the TTL: past any doubt on any machine

	if _, ok := c.Get("session-a1"); !ok {
		fmt.Println("session-a1 expired")
	}

	// Closing again is safe: Close is idempotent, so the deferred Close
	// above becomes a no-op.
	c.Close()

	// Output:
	// authenticated as dana
	// session-a1 expired
}

// A cache exercised the way a session store really uses one: entries with
// no expiry, entries that lapse on their own, a refresh that replaces
// rather than extends a deadline, and a Close that revokes everything at
// once regardless of what the janitor has gotten to.
func TestCacheLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour) // far slower than every TTL exercised below
		defer c.Close()

		if _, ok := c.Get("never-set"); ok {
			t.Error(`Get("never-set") = _, true for a key that was never Set, want false`)
		}

		c.Set("refreshed", "v1", 800*time.Millisecond)
		c.Set("permanent", "v2", 0)
		c.Set("pinned", "v3", -500*time.Millisecond) // negative TTL: same as zero
		if got := c.Len(); got != 3 {
			t.Fatalf("Len() = %d, want 3 right after three Sets", got)
		}

		// Replace "refreshed" partway through its original TTL. The new
		// deadline is 1200ms out from now (the 300ms mark), landing at the
		// 1500ms mark, not extending the original 800ms deadline.
		time.Sleep(300 * time.Millisecond)
		c.Set("refreshed", "v1b", 1200*time.Millisecond)

		// 900ms further brings the running total to 1200ms, short of the
		// 1500ms mark: the refreshed entry must still be live.
		time.Sleep(900 * time.Millisecond)
		if v, ok := c.Get("refreshed"); !ok || v != "v1b" {
			t.Fatalf(`Get("refreshed") = %q, %v, want "v1b", true: a Set before the old deadline should replace it outright`, v, ok)
		}

		// 400ms more brings the running total to 1600ms, past the 1500ms
		// mark: Get and Len both drop it on read, an hour before the
		// janitor is next due.
		time.Sleep(400 * time.Millisecond)
		if _, ok := c.Get("refreshed"); ok {
			t.Error(`Get("refreshed") = _, true past its refreshed deadline, want false`)
		}
		if got := c.Len(); got != 2 {
			t.Errorf("Len() = %d, want 2 once one of three entries has expired", got)
		}
		if v, ok := c.Get("permanent"); !ok || v != "v2" {
			t.Errorf(`Get("permanent") = %q, %v, want "v2", true: a zero TTL never expires on its own`, v, ok)
		}
		if v, ok := c.Get("pinned"); !ok || v != "v3" {
			t.Errorf(`Get("pinned") = %q, %v, want "v3", true: a negative TTL never expires on its own`, v, ok)
		}

		// Close discards everything at once, including entries that never
		// carried a real deadline, and turns every later Set into a no-op.
		c.Close()
		if _, ok := c.Get("permanent"); ok {
			t.Error("Get() after Close = _, true, want false")
		}
		if got := c.Len(); got != 0 {
			t.Errorf("Len() after Close = %d, want 0", got)
		}
		c.Set("too-late", "v4", 0)
		if _, ok := c.Get("too-late"); ok {
			t.Error("Get() for a key Set after Close = _, true, want false")
		}

		c.Close() // idempotent: a second call must not panic or block
	})
}

// The docs promise every method is safe for concurrent callers, the shape
// a real service takes: many request handlers touching their own session
// key at once, and two handlers refreshing the same session at once, all
// with no lock of their own.
func TestConcurrentAccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		var wg sync.WaitGroup
		for i := range 8 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("session-%d", i)
				c.Set(key, "token", time.Minute)
				if v, ok := c.Get(key); !ok || v != "token" {
					t.Errorf("Get(%q) = %q, %v, want %q, true", key, v, ok, "token")
				}
			}(i)
		}

		// Two handlers racing to refresh the same session: whichever Set
		// wins, Get must see one writer's value, never a miss or a mix.
		for _, v := range []string{"first-refresh", "second-refresh"} {
			wg.Add(1)
			go func(v string) {
				defer wg.Done()
				c.Set("shared-session", v, time.Minute)
				if got, ok := c.Get("shared-session"); !ok || (got != "first-refresh" && got != "second-refresh") {
					t.Errorf(`Get("shared-session") = %q, %v, want "first-refresh" or "second-refresh", true`, got, ok)
				}
			}(v)
		}
		wg.Wait()

		if got := c.Len(); got != 9 {
			t.Errorf("Len() = %d, want 9 after 8 distinct sessions and one shared one", got)
		}
	})
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

// mustPanic fails the test unless New(interval) panics; invalidity names
// what makes interval invalid in the failure message.
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
