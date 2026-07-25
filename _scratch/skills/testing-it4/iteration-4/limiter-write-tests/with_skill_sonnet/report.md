# Tests for `limiter`

## What I did

Wrote `limiter_test.go` as an external `limiter_test` package: a runnable
`Example_throttle` for the typical Allow-then-Wait call pattern, a
`synctest`-driven `TestLimiterLifecycle` covering the burst/dry/timeout/
refill/lull/shutdown story end to end, a `TestZeroBurstServesOnlyAWaitingCaller`
exercising the package doc's explicit burst-0 promise, and
`TestNewPanicsOnInvalidArguments` for the constructor's documented panics.

While drafting the zero-burst test I hit a real flake: sleeping exactly to
a tick boundary before calling `Wait` ties the sleep's wakeup against the
ticker's tick at the same virtual instant, and `testing/synctest` leaves
same-instant wakeup order unspecified, so the token occasionally transfers
before the intended elapsed time. Fixed by parking in `Wait` with headroom
before the next tick (matching the "patient wait" pattern already used for
the burst>0 lifecycle test); confirmed stable over 50 repeats and 10 runs
under `-race`.

## Findings (for the author)

`New`'s doc says it panics if rate isn't positive, but there's no upper
bound: a rate above ~1e9 makes `time.Second/time.Duration(rate)` floor to
0, and the refill goroutine's `time.NewTicker(0)` then panics on its own,
unrecovered, crashing the whole process instead of failing at `New`. I
left this case commented out in `TestNewPanicsOnInvalidArguments` (with a
pointer back here) rather than let it take down the suite — worth adding
the same upper-bound guard before this ships.

## Status

`go build ./...` and `go test ./...` both pass (also green under `-race`,
`go vet`, and `gofmt -l`).
