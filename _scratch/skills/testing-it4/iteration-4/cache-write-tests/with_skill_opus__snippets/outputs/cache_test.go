package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// This example caches a session token under its ID, reads it back, and shows
// that an unknown ID is a miss. Size the TTL to the session's lifetime and the
// janitor interval to how promptly expired tokens should be reclaimed.
func ExampleCache() {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("session-abc", "user-42", 30*time.Minute)

	token, ok := c.Get("session-abc")
	fmt.Println(token, ok)

	_, ok = c.Get("session-xyz")
	fmt.Println(ok)
	// Output:
	// user-42 true
	// false
}

// Read-side expiry must hide an entry the moment its TTL elapses, before the
// janitor could have swept. The janitor interval is an hour so no sweep runs
// during the test, leaving read-side expiry as the only thing that can turn the
// entry into a miss for Get and drop it from Len.
func TestEntryExpiresAfterTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("tok", "secret", 30*time.Minute)

		time.Sleep(20 * time.Minute)
		if v, ok := c.Get("tok"); !ok || v != "secret" {
			t.Errorf("Get before TTL = %q, %v; want \"secret\", true", v, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len before TTL = %d; want 1", n)
		}

		time.Sleep(20 * time.Minute)
		if v, ok := c.Get("tok"); ok {
			t.Errorf("Get after TTL = %q, %v; want \"\", false", v, ok)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len after TTL = %d; want 0", n)
		}
	})
}

func TestSetReplacesEntry(t *testing.T) {
	c := cache.New(time.Minute)
	defer c.Close()

	c.Set("tok", "first", 0)
	c.Set("tok", "second", 0)

	if v, ok := c.Get("tok"); !ok || v != "second" {
		t.Errorf("Get after replace = %q, %v; want \"second\", true", v, ok)
	}
	if n := c.Len(); n != 1 {
		t.Errorf("Len after replacing one key = %d; want 1", n)
	}
}

// A non-positive TTL stores an entry without any expiry, so it stays a hit
// however long the cache lives. Time is advanced far past any ordinary session
// to show the entry endures.
func TestZeroTTLNeverExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Hour)
		defer c.Close()

		c.Set("tok", "forever", 0)

		time.Sleep(30 * time.Minute)
		if v, ok := c.Get("tok"); !ok || v != "forever" {
			t.Errorf("Get = %q, %v; want \"forever\", true", v, ok)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len = %d; want 1", n)
		}
	})
}

// A closed cache discards its entries and refuses new ones: Get misses every
// key, Set is a no-op, and a second Close is harmless.
func TestClosedCacheIsInert(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := cache.New(time.Minute)
		c.Set("tok", "secret", 0)

		c.Close()

		if _, ok := c.Get("tok"); ok {
			t.Error("Get after Close reported a hit; want miss")
		}
		c.Set("tok2", "again", 0)
		if _, ok := c.Get("tok2"); ok {
			t.Error("Set after Close stored a value; want no-op")
		}

		c.Close()
	})
}

// New rejects a janitor interval that is not positive: a zero or negative
// ticker interval has no sensible sweep schedule.
func TestNewRejectsNonPositiveInterval(t *testing.T) {
	for _, tt := range []struct {
		name     string
		interval time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mustPanicNew(t, tt.interval)
		})
	}
}

func mustPanicNew(t *testing.T, interval time.Duration) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("New(%v) did not panic; the janitor interval must be positive", interval)
		}
	}()
	cache.New(interval).Close() // if New wrongly returns, release the janitor it started
}
