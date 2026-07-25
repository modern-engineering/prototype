# limiter tests

Wrote `limiter_test.go` (external `limiter_test`) from the package doc: a runnable `Example_throttle`, a `synctest` `TestThrottleLifecycle` (burst, ctx-timeout, refill-served Wait, lull capped at burst, ErrClosed-after-Close), a `TestBurstZeroHoldsNoTokens` for the doc's burst-0 clause, and `TestNewPanicsOnInvalidArguments`. `go build ./...` and `go test ./...` both pass (also clean under `-race -count=3`).

Finding for the author: `New` accepts any rate above 1e9, which floors `time.Second/rate` to a 0 interval; `time.NewTicker(0)` then panics inside the refill goroutine and takes the whole process down (uncatchable), yet the doc only promises a panic for a non-positive rate. That over-large-rate case is written but commented out in the panic table (a live case crashes the test binary) -- please either guard it in `New` like the interval-zero case demands, or document that callers must keep rate <= 1e9, then I'll uncomment it.
