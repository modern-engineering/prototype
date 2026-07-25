package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// Get and Len already treat an expired entry as gone on their own, so
// neither one can tell a janitor that swept the map from one that never
// ran; only a look at the map itself catches a janitor that silently
// stopped sweeping.
func TestJanitorSweepsExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Second)
		defer c.Close()

		c.Set("session-1", "alice", 500*time.Millisecond)

		time.Sleep(1500 * time.Millisecond) // past the TTL and a full sweep

		c.mu.Lock()
		n := len(c.entries)
		c.mu.Unlock()
		if n != 0 {
			t.Errorf("entries map holds %d entries after a sweep past expiry, want 0", n)
		}
	})
}
