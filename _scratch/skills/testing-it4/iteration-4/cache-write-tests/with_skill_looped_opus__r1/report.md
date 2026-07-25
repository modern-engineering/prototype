# Committed tests for `cache`

Planned from `go doc -all` before opening `cache.go`. The suite is black-box
`cache_test` except for one internal-package peephole, and walks the flows a
session store actually sees rather than scanning method by method.

- `Example` mints a token, reads it back, keeps a no-expiry service key beside
  the expiring sessions, and misses an unknown key. The no-expiry line is there
  on purpose: `ttl <= 0` never expiring is the non-obvious footgun, and pkgsite
  is where a reader meets it.
- `TestCacheLifecycle` runs the whole story on virtual time: set/hit, a
  read-time miss the instant a TTL lapses (ahead of the janitor's sweep),
  no-expiry via both a negative and a zero TTL, replacement under a key, an
  entry with no deadline surviving a sweep, then Close turning every key into a
  miss and a second Close proving idempotence.
- `TestConcurrentCallers` exercises the type doc's headline promise, "all
  methods are safe for concurrent use," as the real flow it describes: many
  handlers on one cache. It runs outside synctest, on purpose. A bubble only
  advances virtual time while a goroutine is durably blocked, so callers there
  never touch the map at the same instant; the race detector needs genuine
  overlap. `go test -race` is what gives this test its teeth.
- `TestNewPanicsOnNonPositiveInterval` tables the documented panic for zero and
  negative intervals.

## The one white-box test is a judgment call, not settled contract

`TestJanitorReclaimsExpiredEntries` locks `c.mu` and reads `len(c.entries)`. It
is the only place that peeks at internals, and it earns the peek only if we mean
the doc line "a background janitor goroutine removes expired entries" as a
promise. Get and Len already hide an expired entry whether or not it is still in
the map, so nothing about *reclamation* is observable through the public API.

My reading: it is a promise worth holding, because a long-lived session cache
that never frees dead entries is a memory leak, and that is the janitor's whole
reason to exist. But the test pays for that with fragility: `len(c.entries)` is
a proxy for memory, coupled to today's `map` storage, and it breaks the day the
store becomes a `sync.Map` or a shard even though the contract holds. So this
belongs to the owner to decide, and it should land as its own commit whose body
records the choice. If reclamation is really best-effort hygiene, the cleaner
move is to expose a size metric (then the black-box suite can assert it and this
file goes away) or to soften New's prose to say so.

## Coverage honesty

The concurrency clause is now exercised, not assumed. `Len` and the `Get`/`Set`
boundaries are covered by the lifecycle and the concurrent flow. What the suite
does *not* pin: exact memory figures (the white-box test proves deletion, not
bytes), and any ordering between an in-flight call and Close beyond "after Close,
everything misses," which the prose does not promise and I did not invent.

One prose-code agreement worth keeping: Close's synctest requirement (the bubble
will not settle until the janitor goroutine exits) is itself evidence the
goroutine is released, so the shutdown clause is covered as a side effect of the
lifecycle running clean.

## Verification

`go build`, `go vet`, `gofmt -l` clean. `go test -count=50` and
`go test -race -count=10` both pass with zero failures.
