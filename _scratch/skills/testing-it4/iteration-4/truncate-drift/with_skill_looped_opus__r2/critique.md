# Maintainer critique — truncate-drift / with_skill_looped_opus__r2

Reviewed as the package maintainer, bound to skill-v4 and the package prose
(`go doc -all`). I did not open `truncate.go`. Verified state: `go vet` clean,
`Example` passes, `TestHeadAndTailCountRunes` passes all 8 cases,
`TestNegativeLimitYieldsEmptyString` fails on the two Head cases only (Tail's
negative cases pass). The drift is real and it is Head-only.

Credit where due: the runes-not-bytes table is correct and covers the contract
(over-limit-unchanged, exact-length, empty, zero, wide/accented runes); the
drift is surfaced in both a test doc comment and report.md rather than absorbed
silently, and the red test holds Head to its published promise instead of
blessing the panic. That part is exactly right. The problems are structural.

## truncate_test.go

### 1. (HIGH) Prose sub-test names — this table should not use `t.Run` at all

`TestHeadAndTailCountRunes` wraps every row in a sub-test whose name is a prose
description: `"doc example"`, `"wide rune"`, `"limit exceeds length"`,
`"limit equals length"`, `"empty input"`, `"zero limit"`. The runner renders
these as `.../limit_exceeds_length`, `.../wide_rune` — spaces mashed into
underscores. That is the textbook smell: a sub-test earns its `t.Run` only when
its name reads like a top-level Go symbol (no spaces). I ruled on this exact
package already: "I would not use sub-tests, this is a classic case of test
descriptions that don't fit as test names." Drop `t.Run` and run one continuous
loop. The failure messages already print `Head(%q, %d)` / `Tail(%q, %d)`, so the
inputs name themselves; the `name` field is then dead weight duplicating a log
label the runner would print anyway. This is a merge blocker.

### 2. (HIGH) `wantEmpty` masks the code under test and wraps a function that cannot panic

`wantEmpty(t, name string, f func(string, int) string, ...)` abstracts the
function under test behind a func value and re-supplies its identity as a
hand-wired `"Head"` / `"Tail"` string. That is precisely the indirection I said
I do not want to see: the error strings are wired by hand, the actual call is
hidden behind `f`, and `name` can silently drift out of sync with the function
passed. Worse, it is applied to Tail, which has the `n <= 0` guard and never
panics — the run confirms Tail's negative cases pass. So the entire
recover apparatus is dead ceremony for half its callers. The shape I endorsed
in a sibling run was a Head-specific recovered helper (a `headRecovered`), not a
generic function-passer. Only Head is the drift; only Head needs recover.

### 3. (MEDIUM) The negative-limit drift should fold in, not stand alone

I already called a standalone negative test on this package "weird" and asked to
fold it into the table with a comment. Nothing changed here: it is still a
separate top-level `TestNegativeLimitYieldsEmptyString` carrying its own helper.
The clean consolidation: Tail's negative case is plain-compare code — it belongs
as an ordinary row in the merged table (`{"café", -5, "", ""}` reads fine).
Head's negative case is the only genuinely different code (it panics), so give
Head one focused recovered assertion and comment the divergence there. As
written, the same Head/Tail input space is smeared across two structures for no
gain. The two rows in the negative table also do not discriminate: the `n <= 0`
guard is byte-content-blind, so ASCII vs. multi-byte proves nothing extra — one
input suffices.

### 4. (MEDIUM) The lone `Example` cheaps out and drops the per-symbol examples

The example is two working lines: a breadcrumb `Head(path,4)+"…"+Tail(path,5)`.
The scenario framing is good and the comment speaks to users about the scenario
rather than narrating the code — that fixes the audience flaw from the prior
run. But it under-shows the contract: nothing demonstrates the headline
"returns s unchanged" on an over-generous limit, nothing shows the negative→
empty promise, and it does not even reproduce the package doc's own worked
example `Head("héllo", 2) == "hé"`. It also collapses what worked best before —
per-function examples that pkgsite attributes to Head and Tail individually —
into a single unnamed `Example`, so neither symbol gets an attributed example.
Why cheap out? If a reader reached the example, let them see the contract at
once; an `ExampleHead` and `ExampleTail` (or a richer composed example) is the
reach here, not a two-liner.

### 5. (LOW-MED) Example call-site literals go unexplained

`Head(path, 4)` and `Tail(path, 5)` carry the only real work in the example and
have no adjacent comment. The `4` and `5` are non-meaningful literals chosen to
land on "café" / "señor"; in-function example comments are prime pkgsite real
estate and non-meaningful literals at a call site want a word. The leading block
carries the scenario, which is acceptable, but the working line itself is bare.

## report.md

Accurate and appropriately scoped: names the drift, states the prose-over-code
decision and its justification (Head/Tail symmetry implies a forgotten guard),
confirms production code was left untouched for the owner, and reports the exact
failing surface. No complaints. The author-context inference the drift deserves
("do we actually want to promise this for negative n, or is hardening
unwarranted?") would live in the commit body, not here — out of scope for a
report, but worth carrying into the commit when this lands.
