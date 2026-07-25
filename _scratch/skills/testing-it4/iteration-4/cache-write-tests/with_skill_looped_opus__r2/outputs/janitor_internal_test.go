package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// The janitor's distinct promise is memory reclamation: it deletes entries
// whose TTL has lapsed even when no caller ever reads them. Get and Len
// enforce expiry on read, so from outside the package a swept entry and an
// unread expired one are indistinguishable; only the backing map, inspected
// here under the lock, tells them apart.
func TestJanitorReclaimsUnreadExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Second) // sweep every second
		defer c.Close()

		c.Set("stale", "token", 10*time.Second)
		c.mu.Lock()
		n := len(c.entries)
		c.mu.Unlock()
		if n != 1 {
			t.Fatalf("len(entries) = %d right after Set, want 1", n)
		}

		// Advance past the 10s TTL to an instant between sweeps without ever
		// reading the entry, so only the janitor can account for its removal
		// and the wake lands strictly after the sweep that reclaims it.
		time.Sleep(11500 * time.Millisecond)
		c.mu.Lock()
		n = len(c.entries)
		c.mu.Unlock()
		if n != 0 {
			t.Errorf("len(entries) = %d after the janitor swept, want 0", n)
		}
	})
}
