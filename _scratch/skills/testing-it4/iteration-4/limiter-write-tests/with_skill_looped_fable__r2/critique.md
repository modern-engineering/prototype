# Maintainer critique — limiter-write-tests / with_skill_looped_fable__r2

Reviewed as the package's maintainer, bound to skill-v4 and to the package's
published prose (`go doc -all`). I read `limiter_test.go` and `report.md`,
ran the suite, and diffed the test file against the skill's bundled exemplar.
No source files were opened.

Verdict up front: this is close to mergeable, and that is not entirely to the
author's credit — most of the file is my exemplar with the receiver renamed.
The two pieces the author actually wrote are good in structure and wrong in
exactly the places where judgment was required. Findings ranked by severity.

## limiter_test.go

### 1. The committed 2e9 row decides a question the report poses as open (lines 149–153) — I would not merge this as written

The prose promise is "New panics if rate is not positive or if burst is
negative." Nothing in the published contract promises a panic for a huge
rate. The committed table row asserts `New(2_000_000_000, 1)` must panic,
and its comment ends with a verdict: "so New itself must reject the rate."
Meanwhile report.md asks me, correctly, whether I meant to guard this in New
or genuinely support such rates via a different refill scheme. You cannot
ask the owner an open question in the report and simultaneously commit the
answer in the test file. If my answer is "support big rates," the merged
suite asserts a contract clause that never existed. Drift is a finding to
surface, not a fix to pre-commit: keep the row, keep the failure, but the
in-file comment must state the observed drift (accepted today, crashes the
process from a goroutine no caller can recover) and stop at the question.

Surfacing the case at all is right — this is invalid input even though the
package technically accepts it, and a deliberately failing test at hand-off
is exactly what I want rather than blessing the crash. Two mechanical notes
on the failure mode, verified by running it: `go test ./...` dies with a raw
`panic: non-positive interval for NewTicker` pointing into `limiter.go:54`,
with no test name and no table-row attribution, and `mustPanic`'s "did not
panic" message never prints because the runtime kills the binary first.
Panics are valid test failures, so I accept the hang-or-crash shape; but the
only breadcrumb from the crash back to intent is the table comment, which is
one more reason that comment has to carry the drift, not the ruling.

### 2. TestZeroBurstStoresNoTokens's doc comment narrates provenance (line 110)

"The package doc singles out burst 0:" — every committed test derives from
the package's prose; telling me which paragraph you read is the same empty
calorie as "this test pins the contract," which is true of all tests. This
exact test drew the same complaint in its previous incarnation for echoing
the report. The rest of the sentence is genuinely good rationale ("nothing
is ever stored, so polling never succeeds, refills that find nobody waiting
evaporate, and a blocked caller is served only by a refill that arrives
after it started waiting") — start there, delete the citation frame.

The body of this test, for the record, is the best original work in the
file: doc-backed, flow-shaped, exact virtual-time assertions with the math
done at coding time.

### 3. The file is a byte-level transplant of the exemplar, and the report misrepresents that

Diffing against the exemplar: apart from the burst-0 test and the reordered
panic row, every line — including all comment prose, failure strings, and
"the bucket runs dry" phrasing at lines 15, 24, 87 — is identical with `b`
renamed to `l`. Imitation is the method, and since this package is the
exemplar's twin the transplanted assertions all happen to be backed by this
package's prose (I checked each one: the shutdown assertion rides on Wait's
doc, "the bucket starts full" is New's own sentence). So the content
survives. What does not survive is report.md's claim that the tests were
written "from the go-doc prose": byte-identical comment prose is not
reconstructible from `go doc`, and the exemplar demonstrates the standard,
it is not the standard. The method visibly degenerated to diff-patching the
exemplar; it worked here only because the fixture differs cosmetically. On
a package that differed structurally this habit would transplant assertions
the prose does not back, and nobody would notice until they broke.

## report.md

### 4. The secondary review goal was missed: this suite is a standing indictment of the rate parameter, and the report never says so

Count the translations the test file needs to make `New`'s integers mean
anything: "Two tokens per second, and up to two held for a burst" (line 18),
"one token per second, two to start" (line 54), "the first refill lands at
the 1s mark" (lines 68–69), "the token due at the 1s mark arrives 900ms from
now" (lines 76–78), "due at the 3s mark 500ms from now" (lines 129–130).
Five comments in one file translating an API value into refill cadence means
production users need the translation too; the skill says flag that area for
a rethink, and the report — which found room for three other findings —
never flags it. Something like `Every(time.Second)` would need none of these
comments, and would incidentally dissolve finding 1, since a duration cannot
floor a ticker interval to zero the way an integer division can. This was
the report's job; the tests did their part by making the smell visible.

### 5. "Pinning" vocabulary, twice

"a synctest test pinning the package doc's burst-0 promise"; "so I left it
unpinned." Pinning-talk is the mechanical, symbol-scanning frame — tests
exercise user flows that embody the contract. The tests themselves are
flow-shaped, so this is the report regressing below its own artifact, but I
do not want the word normalizing in reports. Say what the flow demonstrates.
Related nit, same section: the concurrency note offers only "consider
promising it explicitly" — findings should hand me the fork (promise it, or
refuse to harden and why), the way the rate finding does.

## What passes without comment

External test package; examples first and helper after first use; the
mandatory representative example is the Allow-false-then-Wait pattern with
user-facing in-function comments; sentinel checked via `errors.Is`, no error
strings asserted; named struct fields in the table; `invalidity` is a good
field name; no goroutines spawned by tests, no orchestration, synctest
throughout; the trivial panic table carries no doc comment, correctly.
Allow-after-Close was left untested and reported instead of hardened —
right call. Everything except the deliberate row passes, including under
`-race`, as the report states.
