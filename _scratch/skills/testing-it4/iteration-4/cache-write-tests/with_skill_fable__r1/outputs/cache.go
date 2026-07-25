// Package cache provides an in-memory key-value store with per-entry
// expiry, intended for short-lived values such as session tokens.
package cache

import (
	"sync"
	"time"
)

// entry pairs a value with its expiry deadline; zero means no expiry.
type entry struct {
	value    string
	deadline time.Time
}

// Cache is an in-memory string store whose entries expire after a
// per-entry TTL. A background janitor goroutine removes expired
// entries, but expiry is enforced on read: an expired entry is
// invisible to Get and Len even before the janitor removes it.
//
// All methods are safe for concurrent use by multiple goroutines.
type Cache struct {
	mu      sync.Mutex
	entries map[string]entry
	closed  bool
	stop    chan struct{}
}

// New returns an empty Cache and starts its background janitor
// goroutine, which removes expired entries every janitorInterval.
// Callers must release the goroutine with Close. New panics if
// janitorInterval is zero or negative.
func New(janitorInterval time.Duration) *Cache {
	if janitorInterval <= 0 {
		panic("cache: janitor interval must be positive")
	}
	c := &Cache{entries: make(map[string]entry), stop: make(chan struct{})}
	go c.janitor(janitorInterval)
	return c
}

// janitor deletes expired entries every sweepEvery until Close.
func (c *Cache) janitor(sweepEvery time.Duration) {
	tick := time.NewTicker(sweepEvery)
	defer tick.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-tick.C:
			now := time.Now()
			c.mu.Lock()
			for key, e := range c.entries {
				if expired(e, now) {
					delete(c.entries, key)
				}
			}
			c.mu.Unlock()
		}
	}
}

// expired reports whether e's deadline has been reached as of now.
func expired(e entry, now time.Time) bool {
	return !e.deadline.IsZero() && !now.Before(e.deadline)
}

// Set stores value under key, replacing any existing entry. The entry
// expires ttl from now; from then on Get misses it and Len does not
// count it. If ttl is zero or negative, the entry is stored without
// an expiry and lives until it is replaced or the cache is closed.
// After Close, Set does nothing.
func (c *Cache) Set(key, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	e := entry{value: value}
	if ttl > 0 {
		e.deadline = time.Now().Add(ttl)
	}
	c.entries[key] = e
}

// Get returns the value under key and whether key held a live entry.
// Expired entries are misses even before the janitor removes them.
// After Close, Get reports a miss for every key.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || expired(e, time.Now()) {
		return "", false
	}
	return e.value, true
}

// Len returns the number of live entries. Expired entries are not
// counted even before the janitor removes them.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	now := time.Now()
	for _, e := range c.entries {
		if !expired(e, now) {
			n++
		}
	}
	return n
}

// Close stops the janitor goroutine and discards all entries. Close
// is idempotent: calling it again is safe and does nothing. After
// Close, Set does nothing and Get reports a miss for every key.
func (c *Cache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.entries = nil
	close(c.stop)
}
