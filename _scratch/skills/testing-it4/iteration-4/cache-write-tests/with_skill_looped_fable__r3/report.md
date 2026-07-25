# Committed tests for the expiring cache

Wrote `cache_test.go` (external `package cache_test`, go-doc-first): a runnable `Example_sessionTokens` for the anticipated login/lookup/expiry pattern, closing with the idempotent second Close; a `TestExpiryLifecycle` synctest scenario covering read-side expiry before any sweep, replacement moving the deadline in every direction (renewed, dropped via non-positive TTL, and armed on a previously TTL-less entry), TTL-less survival across four sweeps, Close semantics, and the janitor's release; a `TestConcurrentUse` exercising the doc's concurrent-use promise under its natural load, concurrent logins and lookups with each goroutine asserting its own token; and a panic table for `New`.

Findings for the author: "from then on Get misses it" leaves the exact deadline instant ambiguous, and no real-time caller can observe that instant, so the suite asserts only strictly past the deadline; promise the boundary explicitly (inclusive or exclusive) if you want it held. `Len` after `Close` returning 0 is only implied by Close's "discards all entries"; consider promising it in Len's doc. Set-after-Close is observable only through `Len`, since Get misses everything post-Close regardless.

Status: gofmt clean; `go vet` clean; `go build ./...` OK; `go test ./...` and `go test -race -count=3` pass.
