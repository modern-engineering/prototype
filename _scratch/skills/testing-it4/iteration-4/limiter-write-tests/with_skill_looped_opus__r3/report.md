# limiter tests

Wrote `limiter_test.go` (external `limiter_test`) from your published go-doc.
It carries a runnable `Example_throttle` showing the anticipated call pattern
(Allow on the hot path, fall back to Wait when the bucket runs dry), a synctest
`TestThrottleLifecycle` walking one client story end to end, a
`TestZeroBurstAdmitsOnlyWaitingCallers` for the burst-0 clause in your package
doc, and a `New` panic table.

## Finding: `New` accepts a rate that crashes the process

`New` rejects `rate <= 0`, but not a rate so large that `time.Second/rate`
floors to `0`. `New(2e9, 1)` is accepted; the refill goroutine then calls
`time.NewTicker(0)`, which panics from a background goroutine. No `recover` in
the caller can catch it, so the whole process goes down. Verified in a
throwaway module.

The author's call, not a mechanical fix. Your doc is silent on an upper bound;
whether to promise rejecting `rate > 1e9` is a contract-shaping decision. Two
readings:

- No real caller asks for a billion tokens per second, so an explicit bound
  may be hardening a contract irrelevant to any scenario you anticipate.
- But the current failure is the worst kind: silent acceptance followed by an
  async, unrecoverable process crash from a goroutine the caller never sees. A
  typo or a computed rate reaches it, and the blast radius is the whole binary.

I lean toward guarding it, because a clean up-front panic makes `New`'s
contract uniform ("panics on invalid arguments") and trades a catastrophic
crash for a legible one. That is your decision to make; this test and that
reasoning would land as their own fine-grained commit rather than folded into
the rest.

The huge-rate case lives as a commented-out row in the `New` panic table,
where it belongs beside the other invalid-rate rows. It cannot run live: the
async ticker panic crashes the suite, so no table row or `recover` survives it.
The row is written to pass the moment `New` rejects `rate > 1e9`; uncomment it
at that hand-off.

## Changes from the earlier review

Applied:

- Replaced the standalone bug "test" (an unconditional `t.Errorf` that never
  called `New` and failed 100% regardless of the code) with the commented-out
  panic-table row described above. It is now a real assertion that passes once
  the guard lands, sitting with the concern it extends instead of orphaned
  behind the helper.
- Cut the bug comment's meta-narration ("the test fails on purpose so the gap
  stays visible..."). The remaining comment states the floor-to-zero mechanism
  and the uncomment trigger, nothing about the test's own construction.
- Moved the huge-rate deliberation (promise the rejection, or not?) into this
  report as the author's judgment, out of a flat "add the guard" prescription.
- Relaxed `TestThrottleLifecycle`'s patient-wait assertion from `== 900ms` to
  "blocked, and served within one refill interval." The exact 900ms was
  borrowed implementation phase (ticker started at `New`, 100ms burned), not a
  bound your prose promises. Working go-doc-first, the rate contract promises
  one token per second, so I assert that bound and drop the phase.

Rejected: nothing outright. The 900ms note was one you said you would not die
on; I still relaxed it, because pinning implementation phase contradicts the
go-doc-first stance and the assertion keeps its meaning as a bound.

## Status

`gofmt -l` clean, `go vet ./...` clean, `go build ./...` OK. `go test ./...`
passes, including `-race`. No deliberate failing test remains.
