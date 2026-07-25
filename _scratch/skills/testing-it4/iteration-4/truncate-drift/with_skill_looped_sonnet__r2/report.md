# truncate package: committed tests

## What I did

Followed the `testing` skill's go-doc-first method: ran `go doc -all` on the
copied module before opening `truncate.go`, read the exemplar `bucket`
package and its test end to end, then wrote `outputs/truncate_test.go` as an
external `truncate_test` package with:

- `Example_preview`, the mandatory runnable example, showing a caller
  shortening a display label with `Head` and marking the cut with an
  ellipsis, using a multi-byte name to demonstrate rune-safety. The guard
  around the `fmt.Println` call now carries an in-function comment
  explaining why it's there: `Head` returns the input unchanged once it's
  already short enough, so the example only appends the ellipsis when a
  cut actually happened.
- `TestTruncate`, one table driving both `Head` and `Tail` (their doc
  comments describe the same three behaviors in the same words): an
  in-range cut, an exact-length boundary, a limit past the string's length,
  zero, and cuts landing on either side of a multi-byte rune.
- `TestTailTreatsNegativeNAsZero` and `TestHeadPanicsWhereDocPromisesZero`,
  two separate tests isolating the one behavior where the two functions'
  docs and implementations diverge (see Findings). These replace the
  earlier single `TestNegativeNIsTreatedAsZero`.

## Findings

`Head`'s and `Tail`'s doc comments make an identical promise: "A negative n
is treated as zero: ... returns the empty string." `Tail` has the guard
(`if n <= 0 { return "" }`); `Head` does not. Calling `Head(s, -1)` falls
through to `r[:n]` with a negative index and panics.

Author-context: this reads as an oversight, not a deliberate contract
change. The two doc comments are worded identically, `Tail` already carries
the exact guard the docs describe, and nothing in the source suggests
`Head` dropped the check on purpose. Per the skill's authority order (prose
over implementation, absent evidence the code moved deliberately), the
test asserts the documented contract rather than the buggy behavior, so it
fails against `Head` today. This is left in rather than dropped or softened
with a `recover`, per the skill: an unexpected panic is a valid, honest
test failure, and papering over it would hide a real pre-ship bug (any
caller passing user-controlled negative input to `Head` crashes the
process). The fix is a one-line guard mirroring `Tail`'s; flagging it here
rather than patching `truncate.go` myself, since the ask was for tests, not
a fix.

**Consequence for CI visibility:** because Go's test binary always runs
every `Test` to completion before any `Example`, and
`TestHeadPanicsWhereDocPromisesZero` panics unrecovered, the package's one
mandatory runnable example, `Example_preview`, currently has no standing
verification from the default `go test ./...` invocation — it only ran
here because I invoked `go test -run Example_preview` by hand. This isn't
fixable by reordering the file (execution order is Tests-then-Examples
regardless of source position) and isn't worth fixing by recovering the
panic, which would paper over the real bug. The gap closes on its own the
day `Head` gets `Tail`'s guard: at that point
`TestHeadPanicsWhereDocPromisesZero` passes instead of aborting the binary,
and `Example_preview` runs under the plain invocation again.

## Response to maintainer critique

1. **`Example_preview` had zero in-function comments (refuse to merge).**
   Applied. Added a comment on the guard explaining the non-obvious part
   of the call site: why `short != name` is checked at all (`Head`'s
   passthrough-when-short behavior), which is exactly the kind of
   non-Go contract detail the skill says belongs at the call site rather
   than left implicit.
2. **The suite's design hides `Example_preview` from ordinary `go test`
   runs until the bug is fixed.** Applied, but as report prose rather than
   a source comment. The skill's anti-slop guidance rules out explaining
   Go's Test/Example execution order inside the test file itself; the
   report is where author-context and process consequences belong, so the
   callout above spells out the gap explicitly instead of leaving it as
   the previous version's aside inside the "Final status" section.
3. **`TestNegativeNIsTreatedAsZero`'s `Head` assertion was dead code
   today.** Applied, but by restructuring rather than adding a caveat
   comment. Splitting the test (see next point) means
   `TestHeadPanicsWhereDocPromisesZero` no longer pretends to reach a
   comparison it can't: the whole test is now honestly about the panic,
   named for it, with nothing left implying an unreached `t.Errorf` is
   doing any work.
4. **The test's name asserted a property the test itself disproves for
   `Head`.** Applied. Split the single `TestNegativeNIsTreatedAsZero` into
   `TestTailTreatsNegativeNAsZero` (passes, states Tail's real contract)
   and `TestHeadPanicsWhereDocPromisesZero` (fails, named for the actual
   behavior). This isn't just a rename: once `Head` and `Tail` diverge in
   *kind* of outcome (return a value vs. panic) rather than degree, they
   fail the table-test smell test from `references/tables-and-subtests.md`
   ("if conditions that specialize the test code based on some field
   signal two separate tests") even though there's no literal `if` here —
   the two functions no longer share one test's worth of code once one of
   them panics. A `go test -list` or bare CI failure line now reads
   `FAIL: TestHeadPanicsWhereDocPromisesZero`, which states the actual,
   already-diagnosed condition instead of a package-wide claim the run
   itself refutes.
5. **No commit trail carries the drift finding's rationale.** Not applied
   here. `outputs/` in this workspace is a flat, untracked scratch
   directory with no per-run repository to commit into (confirmed: no
   `.git` under this iteration's tree), so there's no artifact-shape
   change available at this layer. The underlying point — that a
   non-trivial finding's author-context inference belongs in a commit
   body — is a delivery-pipeline concern (how these outputs get collected
   into a real repo), not something this pass over the test file and
   report can fix. Noting it here so it isn't silently dropped.

## Final status

- `go build ./...`: passes.
- `go vet ./...`: clean.
- `gofmt -l .`: clean.
- `golangci-lint run ./...`: 0 issues.
- `go test ./...`: **fails**, as intended. `TestTruncate` and
  `TestTailTreatsNegativeNAsZero` pass. `TestHeadPanicsWhereDocPromisesZero`
  panics on the `Head("hello", -1)` call: `slice bounds out of range
  [:-1]`. `Example_preview` is not reached by the default invocation for
  the reason spelled out above (verified separately via `go test -run
  Example_preview`, which passes). This is the drift, reported honestly
  rather than hidden.
