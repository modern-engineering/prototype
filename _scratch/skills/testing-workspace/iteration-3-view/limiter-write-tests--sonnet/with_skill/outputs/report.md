# limiter: test report

Drafted the plan from `go doc -all` alone, then wrote `limiter_test.go` (external `limiter_test` package): a panic-validation table, the zero-burst rendezvous contract (foundational, tested first), starting-burst fill, rate-capped refill, ctx-cancellation race, Close/ErrClosed idempotence, and Close actually halting refills, plus one `Example` proving double-Close safety in ordinary usage. Timing tests use `testing/synctest` for a virtual clock, so they are deterministic rather than sleep-based flakes-in-waiting.

Finding for the maintainer: the doc promises ErrClosed only for calls *after* Close returns; the code also unblocks an *in-flight* `Wait` the instant a concurrent `Close` runs (via the `<-l.done` case), a valuable shutdown guarantee worth stating explicitly in the doc comment. Left untested since it is not yet a documented promise, per the rule against pinning undocumented concurrency.

Judgment call: skipped tests of concurrent multi-goroutine `Wait`/`Allow` fairness, since neither doc promises concurrency semantics beyond what channels give for free.

Final status: `go build ./...` and `go test ./...` both pass (also reran under `-race -count=20`, no flakes).
