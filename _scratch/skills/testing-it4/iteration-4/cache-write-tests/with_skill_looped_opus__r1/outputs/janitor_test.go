package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// The janitor's only promise beyond expiry-on-read is that it reclaims the
// memory of dead entries, which the public API cannot show: Get and Len
// already hide an expired entry whether or not it is still in the map. So
// this peeks at the map directly to prove the sweep deletes, not merely
// hides.
func TestJanitorReclaimsExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Minute)
		defer c.Close()

		c.Set("sess", "user-42", time.Second)

		// Past both the one-second TTL and the first minute-long sweep.
		time.Sleep(90 * time.Second)
		synctest.Wait()

		c.mu.Lock()
		n := len(c.entries)
		c.mu.Unlock()
		if n != 0 {
			t.Errorf("map holds %d entries after a janitor sweep, want 0: expired keys must be deleted, not just hidden", n)
		}
	})
}
