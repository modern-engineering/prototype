# Engineer's note: limiter tests

Wrote `limiter_test.go` as an external `limiter_test` package: an `Example_throttle`
for the typical Allow/Wait call pattern, a `synctest`-based `TestThrottleLifecycle`
covering burst, timeout, patient wait, lull restock, and shutdown, a
`TestZeroBurstDeliversOnlyToWaiters` for the package doc's burst-0 rendezvous
promise, and `TestNewPanicsOnInvalidArguments` for the documented constructor
guards.

Finding, framed for the author: `New` validates `rate > 0` but, unlike its
sibling `bucket.New` (which caps `rate` at 1e9), never guards against a rate
high enough to floor `time.Second/rate` to a zero interval. That's not just an
unhelpful panic message — `New` returns successfully, and the panic instead
happens later, asynchronously, inside the background refill goroutine's
`time.NewTicker` call, which is unrecoverable and crashes the whole process.
`TestNewRejectsARateItCannotServe` proves this out via a subprocess (so the
crash doesn't take the rest of the suite down with it) and fails on purpose:
it's asking for the same upper-bound guard `bucket.New` already has, before
this ships.

Status: `go build ./...` clean, `go vet ./...` clean. `go test ./...` reports
one failing test, `TestNewRejectsARateItCannotServe`, for the reason above; the
other four tests pass.
