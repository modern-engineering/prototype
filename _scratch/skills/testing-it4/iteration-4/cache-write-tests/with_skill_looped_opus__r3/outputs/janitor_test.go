package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// Because expiry is enforced on read, Get and Len already hide an expired
// entry, so the janitor's real contribution, reclaiming that entry's memory,
// is invisible through the exported API. This peeks at the backing map to
// confirm a sweep physically deletes, rather than letting expired tokens
// accumulate until the process exhausts memory.
func TestJanitorReclaimsExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Minute) // sweeps once a minute
		defer c.Close()

		c.Set("token", "user-42", 10*time.Second) // expires long before the first sweep

		c.mu.Lock()
		stored := len(c.entries)
		c.mu.Unlock()
		if stored != 1 {
			t.Fatalf("len(entries) right after Set = %d, want 1", stored)
		}

		// The first sweep lands at t=1m and clears the entry, expired since
		// t=10s, out of the map entirely rather than merely hiding it.
		time.Sleep(time.Minute + time.Second)

		c.mu.Lock()
		remaining := len(c.entries)
		c.mu.Unlock()
		if remaining != 0 {
			t.Errorf("len(entries) after a sweep = %d, want 0", remaining)
		}
	})
}
