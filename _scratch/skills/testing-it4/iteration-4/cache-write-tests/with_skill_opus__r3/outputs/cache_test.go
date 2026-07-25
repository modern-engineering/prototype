package cache_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"example.invalid/cache"
)

// A cache of session tokens: store each token under its session id with a
// time-to-live, read it back while it is live, and let it fall out on its
// own once the TTL elapses, with no explicit deletion.
func Example() {
	// The janitor interval only bounds how long dead entries linger in
	// memory; a minute keeps it out of the way for this short example.
	c := cache.New(time.Minute)
	// Always defer Close so the janitor goroutine is released even on an
	// early return.
	defer c.Close()

	// A logged-in session token, good for the length of the session.
	c.Set("session-42", "token-abc", time.Hour)
	token, ok := c.Get("session-42")
	fmt.Println(token, ok)

	// A one-time passcode expires on its own. Expiry is enforced on read,
	// so the token misses as soon as its TTL passes, before the janitor
	// would sweep it.
	c.Set("otp-42", "123456", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	_, ok = c.Get("otp-42")
	fmt.Println(ok)

	// Output:
	// token-abc true
	// false
}

// The session-token story end to end: two tokens stored, one with a TTL and
// one permanent, a replacement that resets the value, a read after the TTL
// that misses on its own before any janitor sweep, and a shutdown that
// discards everything and refuses later writes.
func TestSessionTokenLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// A janitor interval far longer than any TTL below, so every expiry
		// observed here is enforced on read, not by a background sweep.
		c := cache.New(10 * time.Second)
		defer c.Close()

		// An expiring session token alongside a permanent entry (ttl <= 0).
		c.Set("session", "alice", 2*time.Second)
		c.Set("region", "eu-west", 0)

		if v, ok := c.Get("session"); !ok || v != "alice" {
			t.Errorf(`Get("session") = %q, %v; want "alice", true`, v, ok)
		}
		if v, ok := c.Get("region"); !ok || v != "eu-west" {
			t.Errorf(`Get("region") = %q, %v; want "eu-west", true`, v, ok)
		}
		if n := c.Len(); n != 2 {
			t.Errorf("Len() = %d, want 2 with both entries live", n)
		}

		// Re-authenticating replaces the token, resetting both its value and
		// its TTL to a fresh 2s from now.
		c.Set("session", "bob", 2*time.Second)
		if v, ok := c.Get("session"); !ok || v != "bob" {
			t.Errorf(`Get("session") after replace = %q, %v; want "bob", true`, v, ok)
		}

		// Still short of the deadline: the token is live.
		time.Sleep(1 * time.Second)
		if _, ok := c.Get("session"); !ok {
			t.Error(`Get("session") = miss at 1s, want hit before the 2s TTL`)
		}

		// Past the deadline, and still well before the first janitor sweep:
		// the token misses on read and Len stops counting it, while the
		// permanent entry is untouched.
		time.Sleep(1500 * time.Millisecond)
		if _, ok := c.Get("session"); ok {
			t.Error(`Get("session") = hit after its TTL, want miss on read before any janitor sweep`)
		}
		if n := c.Len(); n != 1 {
			t.Errorf("Len() = %d after the token expired, want 1 (the permanent entry)", n)
		}
		if v, ok := c.Get("region"); !ok || v != "eu-west" {
			t.Errorf(`Get("region") = %q, %v; want "eu-west", true — a zero TTL never expires`, v, ok)
		}

		// Shutdown discards every entry and turns future reads into misses.
		c.Close()
		if _, ok := c.Get("region"); ok {
			t.Error(`Get("region") after Close = hit, want miss`)
		}
		// A write after Close is dropped, so the value must not reappear.
		c.Set("region", "eu-west", 0)
		if _, ok := c.Get("region"); ok {
			t.Error(`Get after Set-after-Close = hit, want miss: Set must do nothing once closed`)
		}
		if n := c.Len(); n != 0 {
			t.Errorf("Len() after Close = %d, want 0", n)
		}
	})
}

func TestNewPanicsOnNonPositiveInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		mustPanic(t, interval)
	}
}

// mustPanic fails the test unless New(interval) panics; interval is echoed
// in the failure message to name the offending argument.
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
