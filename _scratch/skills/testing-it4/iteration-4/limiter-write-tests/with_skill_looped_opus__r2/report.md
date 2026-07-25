# limiter tests

Wrote `outputs/limiter_test.go` (external `limiter_test`) from your godoc: a
runnable `Example_throttle` (Allow on the hot path, Wait when dry), a synctest
`TestThrottleLifecycle` (burst, ctx timeout, refill timing, burst cap,
ErrClosed-after-Close), `TestBurstZeroBanksNothing` for the documented burst-0
mode, and a `TestNewPanicsOnInvalidArguments` table covering exactly what New's
doc promises: rate not positive and burst negative.

## Finding: New accepts a rate it cannot honor, then crashes off-goroutine

New's doc promises a panic only when "rate is not positive or burst is
negative." A rate above 1e9 is positive, so the prose is silent about it. It
does not imply an upper guard and none is missing by the letter of the doc.

The code, though, cannot honor such a rate. `time.Second / time.Duration(rate)`
floors to a zero interval once `rate > 1e9`, and `time.NewTicker(0)` panics.
That panic fires in the detached refill goroutine that New spawns, not at the
call site. I confirmed it in a throwaway: `New(2_000_000_000, 1)` returns a
non-nil limiter, a `recover()` in the calling goroutine catches nothing, and the
process aborts a moment later with `panic: non-positive interval for NewTicker`
from `refill`. It takes the whole test binary down with it.

So this is not a dropped guard; it is a gap between a silent promise and a value
the implementation quietly accepts and then dies on. Two ways to close it:

- Bless the value in prose: document that rates above 1e9 are rejected or
  clamped, and make the code do whatever the prose then says.
- Reject it synchronously in New, mirroring the existing `rate <= 0` guard.

My inclination is the synchronous guard. Every other precondition already fails
loudly at the call site where the caller can act on it; a rate that instead
corrupts into an off-goroutine crash is strictly worse, because it evades
recover, aborts unrelated tests, and carries no stack pointing back at the
caller's New. A synchronous check also makes the invalid input testable in
place: it would slot straight into the existing `mustPanic` table as one more
row. That is the shape I would want before committing a test for it, which is
why the suite carries the finding rather than a disabled test. Your call on
whether the contract should harden this way.

## Applying the critique

- Cut "This pins that promise." from `TestBurstZeroBanksNothing` (applied):
  every committed test pins a promise, so the clause said nothing. The two
  sentences before it, which explain why burst-0 is its own mode, stay.
- Removed the disabled `2e9` panic row and its comment (applied): the comment
  invented a "sibling" package that `go doc` does not show and pointed at
  report.md, baking the hand-off process into permanent source. The row could
  never come back as a live `mustPanic` assertion either, since the panic is
  uncatchable in place (see the finding); a commented corpse is the wrong
  artifact. The redesign ask now lives in this report instead.
- Re-checked the file against limiter's own prose (applied in spirit): the
  lifecycle and example legitimately resemble the skill's exemplar because
  limiter shares the exemplar's New/Allow/Wait/Close/ErrClosed contract
  verbatim. The two places limiter's doc actually diverges are handled on
  limiter's terms, not the exemplar's: its extra burst-0 package-doc paragraph
  drives `TestBurstZeroBanksNothing`, and its narrower "not positive" panic
  clause is why the panic table stops at three rows and the large-rate case is
  a finding rather than a test. I did not churn the correct, shared-contract
  tests just to disguise the resemblance; that would trade a real defect for
  cosmetic difference.
- Rewrote this finding (applied): the earlier draft said New "dropped the
  upper-rate guard your doc implies," which misread "not positive" as implying
  a guard, and it referenced the exemplar's sibling. Both are gone.

Status: `go build ./...` clean; `go vet ./...` clean; `go test ./...` PASS, also
under `-race`.
