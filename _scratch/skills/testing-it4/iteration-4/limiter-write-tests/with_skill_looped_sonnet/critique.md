# Maintainer critique: limiter_test.go (with_skill, looped sonnet)

Reviewed strictly from `go doc -all` for `example.invalid/limiter`, the two
committed files (`limiter_test.go`, `report.md`), and the skill's own
exemplar (`bucket_test.go`). Non-test source was not opened.

The file is, line for line, a rename of the exemplar (`bucket`→`limiter`,
`b`→`l`) plus one new test (`TestZeroBurstHandsTokensOnlyToAWaitingCaller`)
and one deletion (the exemplar's upper-bound panic case). That's the right
instinct — imitation is the method — but it means the only two places this
run actually exercised judgment are exactly where it falls down below.
`gofmt`, `go vet`, `golangci-lint`, and `go test -race -count=5` are all
clean, so this is a judgment review, not a mechanics one.

## Findings, most severe first

### 1. A previously-litigated crash bug ships with zero test trace anywhere in the committed file — `limiter_test.go:133-145`

`go doc` promises `New panics if rate is not positive or if burst is
negative` — no upper bound. The exemplar's `New`, by contrast, explicitly
guards `rate > 1e9` with the comment "a rate above 1e9 would make the
refill interval zero," and its test table carries a matching case
(`bucket_test.go:117`, `rate: 2_000_000_000, invalidity: "rate beyond one
token per nanosecond"`). This run's table dropped that case outright:

```
{rate: 0, burst: 1, invalidity: "zero rate"},
{rate: -1, burst: 1, invalidity: "negative rate"},
{rate: 1, burst: -1, invalidity: "negative burst"},
```

`report.md` explains why: without the exemplar's guard, a large enough
rate floors `time.Second/time.Duration(rate)` to zero, `time.NewTicker`
panics inside the background goroutine, and that's an unrecoverable,
process-crashing panic. All true. But this is verbatim the scenario the
maintainer already ruled on for this package's sibling: "a large enough
rate gives a zero ticker... **Definitely test this!** It's part of the
invalid input, even if the package *technically* accepts it." The correct
response to "this is hard to test" is the skill's own closing line: "a
test that proves hard to write is a signal to redesign, rethink, or try
harder" — not silence. Options that were available and untried: a
subprocess-isolated test (re-exec the test binary with a flag, assert on
the exit code/stderr — the standard Go pattern for asserting a crash), a
`t.Skip("...")` stub naming the gap in the file itself, or proposing the
one-line fix the exemplar already carries. Instead the only record of this
gap is a paragraph in `report.md`, a same-session, throwaway artifact. A
maintainer who reads `limiter_test.go` in six months, or a fresh go-doc-first
reviewer following the skill exactly as written, has zero signal that this
case was ever considered and would have to rediscover it from scratch.
This is the one thing in this diff I would block on.

### 2. The one new test doesn't prove the contract its own doc comment claims — `limiter_test.go:110-131`

`TestZeroBurstHandsTokensOnlyToAWaitingCaller`'s doc comment asserts two
things: a token "goes straight to whoever is already parked in Wait," and
"a mere probe through Allow always misses it." The body only demonstrates
the first half. The final assertion (`l.Allow()` false right after `Wait`
already consumed the token) is true for *any* burst size — it adds no
evidence that burst-0 specifically drops unclaimed refills rather than
buffering them. What's missing is the actual distinguishing case: let a
refill interval pass with nobody in `Wait` (e.g. `time.Sleep(1200 *
time.Millisecond)` before ever calling `Wait`), then assert `Allow()` is
still false. Without that, this test can't tell "zero burst" apart from
"buffer of size zero that still queues one pending refill for the next
caller," which is exactly the ambiguity the doc comment claims to resolve.

### 3. The suite never tests the ordering `Wait`'s own doc promises between a done context and an available token

`go doc` reads: "Wait blocks until a token is available, consumes it, and
returns nil. If ctx is done first, Wait returns ctx.Err()." Every `Wait`
call in this suite (and in the exemplar it copies) either has a context
that times out with no token ever available, or has a token ready with a
live context — never both a token available and an already-done context
at the moment of the call. The exemplar's own implementation makes this
observable: a channel receive and a `ctx.Done()` receive in the same
`select` are equally ready, and Go picks among ready cases arbitrarily, so
"if ctx is done first" is not actually guaranteed by a `select` with no
priority — it's a race the doc states as a certainty. This is derivable
from the go-doc text alone (a temporal/ordering claim deserves a test that
exercises the ordering), and it's untested in both this file and the
exemplar it was copied from. I'm not asking this run to fix the exemplar,
but "I copied a gap along with everything else" isn't a pass either —
flag it, don't silently inherit it.

### 4. The one substantive edit in this diff has no home in the committed history

Dropping the exemplar's boundary case is a real, judgment-laden decision —
exactly the kind of thing the skill says belongs in a commit body: "non-
trivial tests land as fine-grained commits carrying [the author's
inferences]." Nothing here indicates this ever happened as its own commit,
with its own message pointing at the finding; the reasoning lives only in
freeform `report.md` prose. If I `git log` this file next year, I get
nothing that explains why the exemplar's fourth case vanished.

### 5. `report.md`'s own arithmetic, used to justify skipping the test, is off — `report.md:5`

"a rate large enough to floor `time.Second/time.Duration(rate)` to zero
(e.g. `rate >= 1e9`)" — `time.Second` is `1e9` nanoseconds, so at `rate ==
1e9` the interval is `1e9/1e9 == 1ns`, not zero; the floor only hits at
`rate > 1e9` (the exemplar's own guard comment says exactly this: "a rate
**above** 1e9"). The overall conclusion survives (such rates do exist and
do crash the process), but a write-up used as the entire justification for
shipping a known gap untested should have its own numbers checked before
it's trusted.

## What's fine

`Example_throttle` and `TestThrottleLifecycle` are faithful, correctly
adapted copies of the (maintainer-approved) exemplar — comments speak to
users, not to maintainers or to Go syntax; no goroutines are spawned
outside `synctest`; no error strings are asserted; `gofmt`/`vet`/lint/race
are all clean. The new test's placement (after the lifecycle test, before
the panic table) and its doc comment's tone are both right. None of that
offsets finding 1.
