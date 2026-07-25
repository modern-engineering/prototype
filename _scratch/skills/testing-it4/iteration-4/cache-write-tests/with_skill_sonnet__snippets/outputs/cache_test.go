package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// This example stores a session token behind a TTL and reads it back
// while it is still live.
func ExampleCache() {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("session-42", "alice", 30*time.Minute)

	value, ok := c.Get("session-42")
	fmt.Println(value, ok)
	// Output:
	// alice true
}

// The janitor interval (1h) outlives the sleep below (1m) by design, so
// the miss it proves comes from Get and Len enforcing expiry on read,
// not from the janitor ever having swept.
func TestExpiryIsEnforcedOnReadBeforeJanitorSweeps(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("short", "expires", time.Minute)
		c.Set("forever", "stays", 0)

		if got, ok := c.Get("short"); !ok || got != "expires" {
			t.Fatalf(`Get("short") = %q, %t before its ttl; want "expires", true`, got, ok)
		}
		if n := c.Len(); n != 2 {
			t.Fatalf("Len() = %d before the ttl; want 2", n)
		}

		time.Sleep(time.Minute)
		synctest.Wait()

		if _, ok := c.Get("short"); ok {
			t.Fatalf(`Get("short") = true after its ttl; want a miss`)
		}
		if n := c.Len(); n != 1 {
			t.Fatalf("Len() = %d after the ttl; want 1 (the forever entry only)", n)
		}
		if got, ok := c.Get("forever"); !ok || got != "stays" {
			t.Fatalf(`Get("forever") = %q, %t; want "stays", true (a zero ttl never expires)`, got, ok)
		}
	})
}

// Close's promise to stop the janitor goroutine has no direct
// observation point from outside the package; a goroutine left running
// would hang this test rather than fail it visibly.
func TestCloseIsIdempotentAndDisablesTheCache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Second)
		c.Set("k", "v", 0)

		c.Close()
		c.Close()

		if _, ok := c.Get("k"); ok {
			t.Fatalf(`Get("k") = true after Close; want a miss`)
		}
		c.Set("k", "v2", 0)
		if _, ok := c.Get("k"); ok {
			t.Fatalf("Set after Close stored a value; want it to do nothing")
		}
	})
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
	for _, tt := range nonPositiveIntervals {
		t.Run(fmt.Sprintf("interval=%s", tt.interval), func(t *testing.T) {
			defer mustPanic(t, tt.interval, tt.reason)
			cache.New(tt.interval)
		})
	}
}

var nonPositiveIntervals = []struct {
	interval time.Duration
	reason   string
}{
	{0, "zero"},
	{-time.Second, "negative"},
}

// mustPanic reports a failure naming the interval and why it is
// invalid when the deferred call site did not panic.
func mustPanic(t *testing.T, interval time.Duration, reason string) {
	t.Helper()
	if recover() == nil {
		t.Errorf("New(%s) did not panic; a %s janitor interval is invalid", interval, reason)
	}
}
