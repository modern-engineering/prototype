# Critique: limiter-write-tests / with_skill_looped_fable__r3

Reviewed as the package maintainer, bound to skill-v4 (SKILL.md, exemplar,
references/concurrency.md, references/helpers.md, references/tables-and-subtests.md)
and to the package's prose via `go doc -all`. Source bodies were not opened.
Verified: `go vet` clean, `go test -race -count=2` passes.

Verdict: close to mergeable, and the strongest limiter run I have seen, but I
refuse the panic table as committed. Findings ranked by severity.

## limiter_test.go

### 1. BLOCKER — the parked case must be live at hand-off (lines 150-155)

The invalid-rate case sits commented out in the committed file. The doctrine
is explicit: comment a bug-panic out to see the work through, then restore it
at hand-off, never dropping it silently. Hand-off happened; the case is still
dead code. The comment's own justification — the panic "crashes the whole
test process instead of failing one test" — argues against canon: a crash is
a valid test failure, exactly like a hang. That crash is the honest, loud
failure that forces the fix; parking it converts a red suite into a green one
with a note the author may never read. The standing ruling on this exact
input class (a rate large enough to floor the refill interval to zero) is:
definitely test it, even though the constructor technically accepts it.
Surfacing the finding in report.md was right; the committed artifact's shape
is wrong.

### 2. BLOCKER — the parked row is transplanted, and it pins an unpromised panic (lines 142-159)

This package's prose promises panics only for non-positive rate and negative
burst. No cap. Yet the parked row carries the skill exemplar's exact
`2_000_000_000` value and its "rate beyond one token per nanosecond" phrasing
— from a package whose docs DO promise a 1e9 cap. That row was copied, not
planned from this package's go doc, which undercuts the report's go-doc-first
claim. Restored as-is inside a table titled for promised panics, it would pin
a panic the prose never made, presuming the author's decision (cap with a
doc'd panic) before the author made it. Siding with prose, the live form is
its own failing test: a doc-valid huge rate must yield a working limiter, and
today it crashes. The comment also mirrors implementation internals
("time.NewTicker(0) panics inside the refill goroutine") and echoes the
report's remediation menu into source; both rot the moment the author picks
either fix. Author-context belongs in the report and the commit body, where
it already lives.

### 3. NIT — the zero-burst doc comment echoes instead of reasoning (lines 110-113)

"The package doc promises that burst 0 stores nothing:" then a paraphrase of
the body's three assertions, which the in-body comments at lines 123-124 and
131-132 restate again. Stating the contract clause is wanted for a
non-trivial test; the attribution preamble and the triple narration are not.
Trim to the scenario rationale and let the in-body comments carry the phases.
Not a blocker.

### What holds up

Example first, complexity building down the file, helper after first use.
Example_gateway's in-function comments speak to users and translate the
non-meaningful `New(2, 2)`; the idempotence comment sits on the explicit
`Close` call site, as I asked. TestThrottleLifecycle is a genuine user flow
with hard-coded virtual-time results, and the exact-500ms assertion in the
zero-burst test actually discriminates banked-token bugs from
served-while-waiting: good test. Nothing pins concurrency the docs do not
promise. Yes, most of this file is the exemplar under a rename; the fixture's
contract happens to coincide, the transplanted math re-derives correctly
against this prose, and imitation is the method, so I do not refuse on that —
finding 2 is where the copying actually bit.

## report.md

### 4. MAJOR — the suite demonstrates a package smell three times and the report never flags it (report.md line 5; test lines 19, 54, 116)

Every `New` call site needs a comment translating the bare ints: "Two tokens
per second, and up to two held for a burst" (line 19), "one token per second,
two to start" (line 54), and by line 116 the translation is skipped entirely
and the doc comment has to carry it. Tests repeatedly translating an API
value mean production users need the translation too: that is the secondary
review goal, and it is unfulfilled. The report raises two API findings but
misses the one its own file proves most often. Flag the constructor for a
rethink (a frequency- or interval-flavored signature would erase all three
comments).

### 5. MODERATE — the sharper prose silence goes unasked (report.md line 5)

Wait's doc says "every subsequent Wait call returns ErrClosed" — pointedly
excluding a waiter already parked when Close lands. For a limiter whose
typical caller falls back to Wait, Close-versus-in-flight-Wait is the more
user-visible open question, and it is readable from go doc alone. The suite
rightly pins nothing there, but the report asks only about Allow draining
leftover tokens after Close. Ask the author both: released with ErrClosed,
left to its context, or refused as a promise.

### Minor

The crash finding says New "promises a panic only for non-positive rate";
the doc also promises one for negative burst. In a drift report, quote the
promise precisely.

## go.mod

Fine: go 1.25 covers synctest.Test and t.Context.
