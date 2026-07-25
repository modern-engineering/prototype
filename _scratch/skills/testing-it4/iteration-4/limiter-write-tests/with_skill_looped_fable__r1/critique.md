# Maintainer critique — with_skill_looped_fable__r1

Reviewed as the package's maintainer, bound to skill-v4 (SKILL.md, the
bundled exemplar, and the scenario references the file obliges:
`concurrency.md` for the panic/invalid-input work, `helpers.md` for
`mustPanic`). Contract authority taken from `go doc -all` and the package
prose only; no non-test source was opened. Findings ranked by severity.

## limiter_test.go

### 1. BLOCKER — the zero-interval row was not restored at hand-off (lines 145–151)

The skill's ruling is explicit and this run breaks it. `concurrency.md`:
a bug-panic case is commented out "to see the work through", and restored
at hand-off — never handed over parked. The same file names this exact
input as mandatory: "a rate large enough to floor the refill interval to
zero is invalid even though the constructor lets it through." The author
found the drift (the doc promises a panic only for non-positive rate,
yet `New(2_000_000_000, 1)` crashes the binary from the refill goroutine),
wrote it up well, then chose green CI over a live failure and told me to
"restore this row once it does". That inverts the hand-off: the failing
row is mine to decide on, and a crashed `go test` is a valid failure — it
forces the decision, where a comment inside a table waits to be skimmed
past. Blessing the crash by omission is exactly what the skill forbids.
I refuse the merge in this shape: restore the row (or land it as its own
red commit carrying the rationale) and let me resolve the drift.

### 2. MAJOR — stale audit math: "two more refills" is three (lines 96–98)

The shutdown sleep was changed from the exemplar's 2500ms to 2600ms
(a defensible move off the exact-boundary tick at t=8000), but the comment
still says "two more refills land before Close". At t=5500 with an empty
bucket, ticks fire at 6000, 7000, and 8000: three refills land, the cap
keeps two. The assertion passes regardless, which is why this is dangerous:
these derivation comments are the audit trail for the hard-coded timeline,
and this one now lies to whoever next edits the constants. Interrelated
timing values earn a comment precisely so they can be re-derived; a wrong
derivation is worse than none.

### 3. MODERATE — Close's idempotency promise is never deliberately exercised or shown (lines 17–42, 99)

The doc spends a sentence promising "Close is idempotent: calling it more
than once is safe". The suite exercises a double Close only by accident:
the lifecycle's explicit `l.Close()` plus its deferred one, with no word
said. The exemplar's example shows the wanted shape — an explicit second
`Close()` with a user-facing call-site comment that the deferred one
becomes a safe no-op — and this run dropped exactly that while copying
everything around it. Users reading pkgsite never learn the
defer-Close-then-explicit-Close pattern is safe, which is the pattern a
gateway shutdown path will actually hit. Put it back in `Example_gateway`.

### 4. MINOR — reviewer jargon leaking into a committed comment (lines 149–150)

"No test can pin that until New guards the rate" — "pin" is process
language from the review loop, not something a maintainer says about a
test, and the skill names it anti-slop. The report does it too ("I left it
unpinned"). Say what is meant: no test can exercise this without crashing
the binary. Same fix wherever the word appears.

## report.md

### 5. MAJOR — the secondary review goal was missed: the rate→interval translation smell

Every `New` call site in the test file needs a translation comment:
"Two tokens per second, and up to two held for a burst" (example),
"one token per second, two to start" (lifecycle), plus the recurring
"the first refill lands at the 1s mark" arithmetic. SKILL.md is direct:
tests repeatedly translating an API value mean production users need the
translation too — flag that area for a rethink. The report flags three
findings, all good, and walks straight past the one on display in its own
file: `New(rate, burst)` taking bare ints is why the suite reads like a
unit-conversion exercise. A constructor shaped like `Every(time.Second)`
would delete half the comments in this file. This is the report's job and
it did not do it.

## What holds

The file otherwise earned its shape: external `package limiter_test`,
examples first with in-function comments aimed at pkgsite users, a
whole-package lifecycle scenario on virtual time instead of a per-symbol
scan, `TestZeroBurstStoresNoTokens` driven by the package doc's burst-0
paragraph, hard-coded results with the math done at coding time,
`mustPanic` after first use with inputs and invalidity named, no
unpromised-concurrency tests, and drift routed to the owner with
promise-or-refuse framing on the smaller notes. That is why finding 1
stings: the judgment was right up to the last step, then the hand-off
kept the artifact green where the skill says it must be red.
