package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// Get and Len already hide expired entries, so the janitor's only distinct
// promise — reclaiming the map slots those dead entries hold — is invisible
// through the public API. This peeks at the entry map to confirm the janitor
// actually deletes, rather than reads merely skipping expired keys.
func TestJanitorReclaimsExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Second)
		defer c.Close()

		c.Set("tok", "v", 500*time.Millisecond)

		// The entry dies at 500ms; the first sweep at 1s must reclaim it.
		time.Sleep(1500 * time.Millisecond)
		synctest.Wait()

		c.mu.Lock()
		n := len(c.entries)
		c.mu.Unlock()
		if n != 0 {
			t.Errorf("janitor left %d expired entries in the map, want 0 reclaimed", n)
		}
	})
}
