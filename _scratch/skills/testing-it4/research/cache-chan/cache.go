// Package cache provides an in-memory key-value store with per-entry
// expiry, intended for short-lived values such as session tokens.
package cache

import (
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
	// Ownership of the entry map travels through entries: receiving
	// the map grants exclusive access, and sending it back releases
	// it. At every method boundary exactly one of the two channels is
	// ready: either entries holds the map, or stop is closed. Close
	// receives the map for good and closes stop, so after Close every
	// select below takes the stop branch.
	entries chan map[string]entry
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
	c := &Cache{
		entries: make(chan map[string]entry, 1),
		stop:    make(chan struct{}),
	}
	c.entries <- make(map[string]entry)
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
			select {
			case m := <-c.entries:
				now := time.Now()
				for key, e := range m {
					if expired(e, now) {
						delete(m, key)
					}
				}
				c.entries <- m
			case <-c.stop:
				// Close won the map between the tick and the sweep;
				// it is never coming back.
				return
			}
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
	select {
	case m := <-c.entries:
		e := entry{value: value}
		if ttl > 0 {
			e.deadline = time.Now().Add(ttl)
		}
		m[key] = e
		c.entries <- m
	case <-c.stop:
	}
}

// Get returns the value under key and whether key held a live entry.
// Expired entries are misses even before the janitor removes them.
// After Close, Get reports a miss for every key.
func (c *Cache) Get(key string) (string, bool) {
	select {
	case m := <-c.entries:
		e, ok := m[key]
		c.entries <- m
		if !ok || expired(e, time.Now()) {
			return "", false
		}
		return e.value, true
	case <-c.stop:
		return "", false
	}
}

// Len returns the number of live entries. Expired entries are not
// counted even before the janitor removes them.
func (c *Cache) Len() int {
	select {
	case m := <-c.entries:
		n := 0
		now := time.Now()
		for _, e := range m {
			if !expired(e, now) {
				n++
			}
		}
		c.entries <- m
		return n
	case <-c.stop:
		return 0
	}
}

// Close stops the janitor goroutine and discards all entries. Close
// is idempotent: calling it again is safe and does nothing. After
// Close, Set does nothing and Get reports a miss for every key.
func (c *Cache) Close() {
	select {
	case <-c.entries:
		// The map is dropped here, never to be returned; closing stop
		// flips every select in the package to its stop branch.
		close(c.stop)
	case <-c.stop:
	}
}
