package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

// The janitor's promise to delete expired entries is invisible through Get and
// Len, which mask an expired entry whether or not it still occupies the map. So
// this test peeks at the map directly to confirm the janitor reclaims the
// expired entry on its sweep while leaving a non-expiring one untouched.
func TestJanitorReclaimsExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Second)
		defer c.Close()

		c.Set("expiring", "v", 500*time.Millisecond)
		c.Set("permanent", "p", 0)

		time.Sleep(2 * time.Second)
		synctest.Wait()

		c.mu.Lock()
		defer c.mu.Unlock()
		if len(c.entries) != 1 {
			t.Fatalf("after a sweep, len(entries) = %d; want 1", len(c.entries))
		}
		if _, ok := c.entries["expiring"]; ok {
			t.Error("janitor left the expired entry in the map")
		}
		if _, ok := c.entries["permanent"]; !ok {
			t.Error("janitor removed the non-expiring entry")
		}
	})
}
