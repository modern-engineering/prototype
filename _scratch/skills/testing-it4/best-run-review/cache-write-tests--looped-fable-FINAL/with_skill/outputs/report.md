# Cache test report

`cache_test.go` is an external `package cache_test` suite, planned go-doc-first from the exported docs. The file builds complexity as it reads:

- `ExampleCache` — the anticipated call pattern for pkgsite: store, hit, and both flavors of miss, never stored and expired. The expired miss uses a millisecond TTL and a real sleep; the deadline is fixed at Set time and `time.Sleep` only ever overshoots, so the miss cannot flake even on a weak runner.
- `TestCache` — the whole-package session-token scenario under `testing/synctest`: store, hit, refresh resets the TTL, expiry visible to Get and Len before the janitor ever sweeps, Close discards and disables.
- `TestConcurrentUse` — the docs promise all methods are safe for concurrent use, so sixteen handler goroutines each store and look up their own session, asserting inside the goroutines under a WaitGroup. This test is the suite's concurrency evidence: in the synctest tests the janitor and the caller never overlap, since virtual time advances only while every goroutine in the bubble is blocked.
- `TestEntryWithoutTTLNeverExpires` — zero and negative TTLs ride 1440 janitor sweeps of virtual time with values intact, until a replacing Set gives the entry a real TTL and it expires like any other.
- `TestNewRejectsNonPositiveJanitorInterval` — the documented panic, a two-case table.

The janitor's sweep is unobservable through the exported API (expiry is enforced on read), so the sweeps are exercised rather than asserted.

No doc/code drift found. One author call remains: the code expires an entry at exactly t+ttl (`!now.Before(deadline)`), and the prose ("from then on") only arguably promises the boundary instant, so I did not pin it — say the word and it becomes a one-line case.

Status: `gofmt`, `go vet`, and `go build ./...` clean; `go test ./...` and `go test -race -count=1 ./...` pass, every test function confirmed run under `-v`.
