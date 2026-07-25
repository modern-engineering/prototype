# Committed tests for the expiring-cache package

## What I did

I planned the tests from the exported prose alone before touching the source, then wrote one committed file, `outputs/cache_test.go`, in an external `package cache_test` so the compiler enforces the user's seat. It holds three tests:

- `Example_sessionTokens` — the anticipated call pattern for the package's stated purpose: store a session token with a TTL, serve a request off a hit, and watch expiry log the session out. This is the only real-time test (0.5s).
- `TestSessionTokenLifecycle` — a whole-package scenario inside a `testing/synctest` bubble on virtual time. It walks a service's story: fresh-cache misses, two credentials stored, a re-issued token whose replacement restarts the clock, expiry landing exactly at the deadline with the janitor still 58 minutes away (so the miss is read-side enforcement), a key rotated with a negative TTL that sheds its old deadline and survives four sweeps, then Close: reads miss, writes are ignored, `Len` is 0, and a second Close is a no-op.
- `TestNewPanicsOnInvalidInterval` — a small table through a recover-helper for the documented panic on zero and negative janitor intervals.

## Findings (questions for you as the author)

1. **Exact-deadline semantics are pinned.** `expired` uses `!now.Before(deadline)`, so a Get at exactly `ttl` from Set is a miss. Your doc, "expires ttl from now; from then on Get misses it", reads as inclusive of that instant, so I asserted the boundary. If you consider the boundary incidental rather than promised, say so and I'll loosen that step; otherwise it is now contract.
2. **The janitor is unobservable through the exported API.** Because expiry is enforced on read, a swept and an unswept expired entry look identical to `Get` and `Len`; the janitor is purely memory reclamation. The tests can only prove it never removes live entries (the pinned key outlives four sweep intervals) and that Close releases the goroutine (a leaked janitor deadlocks the synctest bubble). If sweep behavior ever needs to be observable, that is an API question, not a test one.
3. **Concurrency promise taken as read.** The docs promise all methods are safe for concurrent use. I did not commit a statistical race hunt; the scenario already runs the readers and writers against the live janitor goroutine, and the suite passes under `-race -count=2`.

No prose/code drift surfaced: every documented clause (TTL expiry on read, replacement resets the clock, zero-or-negative TTL pins, Close discards/ignores/idempotent, New's panic) checked out against behavior.

## Final status

`go build ./...` clean. `go test ./...` passes; also verified with `go vet`, `gofmt -l` (clean), and `go test -race -count=2 ./...` (passes, ~1.9s). Toolchain: go1.26.5 on darwin/arm64.
