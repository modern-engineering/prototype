# cache tests

Tests written from the go-doc, exercising the package as a session store rather than symbol by symbol.

External suite (`cache_test.go`, `package cache_test`):

- `Example` — the anticipated call pattern. Mint a token, store it under a session ID with a TTL, read it back while live, find it gone once the TTL lapses. In-function comments repeat the non-Go contract at each call site (defer Close to release the janitor; read-time expiry).
- `TestSessionLifecycle` — one whole-package scenario on synctest's virtual clock. A TTL'd token that reads back live, a no-expiry entry that outlives every sweep, a replacement that restarts the TTL past the original deadline, the read-time miss once the restarted TTL lapses, and a Close that empties the store and turns every later call into a miss. The janitor goroutine runs concurrently throughout; that is the coverage the "safe for concurrent use" clause asks for, so the suite does not hunt for races with unrealistic call patterns.
- `TestNewPanicsOnNonPositiveInterval` — the documented panic on a zero or negative interval.

Internal peephole (`janitor_internal_test.go`, `package cache`):

- `TestJanitorReclaimsUnreadExpiredEntries` guards the one janitor behavior no user can observe. It locks the backing map and confirms an unread, expired entry is actually deleted.

The external suite holds the package to its doc. The internal test is the deliberate exception, and it points at a design question worth your call before this lands.

Design signal — the reclamation promise has no external surface. The package doc promises "A background janitor goroutine removes expired entries every janitorInterval." But Get and Len both enforce expiry on read, so from outside the package a swept entry and an unread-expired one are indistinguishable. The only way to witness reclamation is to lock the private map, which is what the peephole test does. The peephole is justified by the promise-as-written, but needing a peephole to observe a documented behavior is itself the signal: that behavior sits at the wrong altitude in user-facing prose.

Two clean resolutions, and I did not silently pick one:

- Demote the janitor to an internal implementation comment and drop the sentence from the package doc. Reclamation is a memory-management detail, not part of the contract with callers. The peephole test then documents an internal invariant, which is where a lock-and-peek test belongs.
- Or, if reclamation timing is a promise you want to keep to callers, expose something observable (a swept count, a metric) and move the test to the external suite.

The test stays for now so the promise-as-written keeps a regression guard, but it should not survive a decision to demote the prose. This rationale belongs in that test's own fine-grained commit body, separate from the external suite's commit.

No drift otherwise: prose, signatures, and behavior agree.

Verification (Go 1.26.5): `go build ./...`, `go vet ./...`, and `gofmt -l` clean; `go test -race -count=5 ./...` green; the janitor test held at `-count=500`. Example output verified.
