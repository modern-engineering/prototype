package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// Get and Len already hide expired entries on every read, so the public
// API alone can never tell the janitor's sweep apart from a no-op; only a
// peek at the unexported map confirms New's doc-promised janitor actually
// deletes the entry rather than leaving the read-time filter to carry the
// promise forever.
func TestJanitorRemovesExpiredEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Second) // sweep every second
		defer c.Close()

		c.Set("stale", "value", 500*time.Millisecond)
		time.Sleep(2 * time.Second) // past the TTL and a janitor sweep

		c.mu.Lock()
		_, present := c.entries["stale"]
		c.mu.Unlock()
		if present {
			t.Error("janitor left an expired entry in the map after a sweep")
		}
	})
}
