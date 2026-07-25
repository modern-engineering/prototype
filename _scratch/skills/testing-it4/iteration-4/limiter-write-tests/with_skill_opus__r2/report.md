# limiter tests

Wrote an external `limiter_test.go` from your doc comments first: a runnable `Example_throttle`, a whole-lifecycle `TestThrottleLifecycle` under `testing/synctest` (burst, ctx-timeout, patient wait, lull capped at burst, ErrClosed winning over leftover tokens), a `TestZeroBurstServesOnlyWaiters` pinning your documented burst-0 mode, and `TestNewPanicsOnInvalidArguments` for the promised zero/negative-rate and negative-burst panics.

Finding: `New` guards `rate <= 0` but not a rate that floors `time.Second/rate` to 0. `New(2e9, 1)` returns fine, then `time.NewTicker(0)` panics on the refill goroutine and kills the whole process, unrecoverable by the caller. Was dropping an upper-rate bound intentional? A `rate > 1e9` guard (panic in `New`, like `rate <= 0`) fixes it.

Status: `go build ./...` clean; `go test ./...` has one deliberate FAIL, `TestNewRejectsIntervalFlooringRate`, which drives that crash in a child process (so it cannot take the suite down) and stays red until `New` rejects the rate. All other tests pass, also under `-race`.
