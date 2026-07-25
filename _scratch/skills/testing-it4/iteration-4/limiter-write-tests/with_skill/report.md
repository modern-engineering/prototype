# limiter tests — engineer's note

Wrote an external `limiter_test` suite from the go-doc contract: a runnable `Example_throttle` (the gateway's Allow-then-Wait pattern), a synctest lifecycle scenario (burst, deadline expiry, exact refill timing, at-most-burst restock, ErrClosed winning over leftover tokens), a synctest test for the package doc's burst-0 promise, and a panic table for New's documented guards.
Finding for you before the PR: New's docs accept any positive rate, but anything above 1e9 truncates the refill interval to zero and the refill goroutine panics in `time.NewTicker`, crashing the process asynchronously — `TestRateBeyondOneTokenPerNanosecond` is deliberately failing (last in file, reason in its comment) so you can decide: reject such rates in New, or cap the interval at 1ns.
Question, no test committed: after Close, Allow still hands out leftover tokens; if callers must be turned away entirely, promise it in Allow's docs and I'll pin it.
Status: `go build ./...` clean; `go test ./...` fails only on the deliberate test above (panic trace points at refill); all other tests, example included, pass and are race-clean under `-race`.
