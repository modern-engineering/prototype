# limiter tests — engineer's note

Wrote `limiter_test.go` (external `package limiter_test`, go-doc-first): a runnable `Example_throttle` showing the Allow-then-Wait pattern with deferred Close, a synctest whole-lifecycle scenario (starting burst, deadline-expired Wait via `errors.Is(err, context.DeadlineExceeded)`, refill-served Wait at exact virtual time, lull capped at burst, ErrClosed winning over leftover tokens after Close), a synctest burst-0 scenario for the package doc's explicit promise, and a panic table for New's argument guards.

Finding for you before the PR: New accepts any positive rate, but above 1e9/sec `time.Second/rate` floors to zero and `time.NewTicker` panics inside the refill goroutine, crashing the caller's whole process asynchronously; I left the table row asserting New panics up front (it fails today, deliberately) — if you meant to support such rates, clamp the interval to 1ns and amend New's doc instead, otherwise add the guard and the row goes green. I did not pin unpromised behavior (Allow after Close, a Wait already blocked when Close lands): say the word if you want those promised in the docs and tested.

Status: `go build ./...` clean; `gofmt`/`go vet` clean; `go test ./...` FAILS on exactly that one deliberate row (the unguarded goroutine panic also aborts the binary, which is the bug demonstrated); all other tests, including `-race`, pass.
