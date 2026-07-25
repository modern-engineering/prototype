# limiter tests

Wrote an external `limiter_test` suite from your go-doc contract: a runnable `Example_throttle`, a whole-package `TestThrottleLifecycle` under `testing/synctest` (burst, deadline, exact 900ms wait, restock capped at burst, ErrClosed winning over leftover tokens), a `TestZeroBurstStoresNothing` pinning the burst-0 mode your package doc calls out, and a `TestNewPanicsOnInvalidArguments` table.

Heads up: `New` guards `rate <= 0` and `burst < 0` but nothing on the top end, so `New(2_000_000_000, 1)` floors `time.Second/rate` to `0`, and `time.NewTicker(0)` panics inside the refill goroutine — that crashes the whole process and the caller's `recover` can't catch it (verified). I left that as a commented-out panic-table row; either doc `rate` as unbounded-caller-beware or add an upper guard (e.g. `rate > 1e9`) and I'll uncomment it.

Status: `go build ./...`, `go vet ./...`, and `go test -race ./...` all pass (4 tests green).
