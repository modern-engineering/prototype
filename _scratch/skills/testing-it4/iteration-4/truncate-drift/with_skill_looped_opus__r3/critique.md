# Maintainer critique — truncate-drift / with_skill_looped_opus__r3

Reviewed as the `truncate` maintainer, bound to skill-v4 and the package
prose from `go doc -all` (source file not opened). Build clean, `go vet`
clean, `gofmt` clean; `go test` fails on the one deliberately-drifted case
(`Head("héllo", -1)` panics — slice bounds out of range). The drift was
correctly surfaced, not blessed. That part is right.

But two things I would refuse to merge, and three I would send back.

## truncate_test.go

### 1. HIGH — `TestNegativeLimitYieldsEmptyString` dispatches the code under test through a `func` value. Forbidden here.

The test builds `funcs []struct{ name string; fn func(string,int) string }`
and drives everything through `trimRecovering(t, f.name, f.fn, "héllo", -1)`.
This is the func-value / method-on-case indirection that `helpers.md`
reserves for "packages that are much harder to test" and forbids "while
plain straight-line bodies suffice." `truncate` is two pure functions —
straight-line suffices. I have said, verbatim, on this exact package:
"I DO NOT WANT TO SEE THIS PATTERN HERE."

Compare the exemplar's `mustPanic(t, rate, burst, invalidity)`: it calls
`bucket.New(rate, burst)` *directly* inside the helper, so the code under
test is visible at the call site. `trimRecovering` hides `truncate.Head` /
`truncate.Tail` behind `f.fn`; the reader cannot see which function ran.

The masking is not theoretical. The test now reports "the same" assertion
through two different message shapes: the panic path prints
`%s(%q,%d) panicked (%v)` from inside the helper, while a wrong non-panic
value would print `%s(%q,-1) = %q, want %q` from the loop. That is exactly
the "error strings become awkward to wire, masking the actual test code
executed" failure I warned about. Unwind it: assert `Tail("héllo", -1) == ""`
straight, and handle `Head`'s panic on its own (recover, or comment it out
to work through and restore at hand-off). Two visible calls, not a dispatch
table of two.

### 2. HIGH — the doc comment on `TestNegativeLimitYieldsEmptyString` echoes the report, narrates plumbing, and invents a module.

Eight lines, and most of them should never reach source:

- It reproduces `report.md`'s finding almost verbatim. Findings and
  author-context are commit-body material, not source comments. I have
  dinged this before as "echoes the report, guidelines that should never
  make it to the source file."
- "it slices `[]rune(s)[:n]` with no lower guard" states an internal
  expression as fact. A go-doc-first contract test states the gap at the
  contract level ("Head panics on a negative n; the doc promises the empty
  string") and lets the panic evidence it. This sentence could only be
  written by reading the implementation.
- "The panic is recovered so the gap reads as one failure, not a crash
  that hides the passing Tail case" narrates test plumbing. If you have
  nothing good to say, say nothing.
- "which the utils module wants before it ships" — there is no "utils"
  module. `go.mod` says `module example.invalid/truncate`. An invented
  reference plus a shipping-timeline narration, both unsupported, in a
  committed comment.

A non-trivial test earns a rationale, so keep one or two sentences: the
doc promises empty string on negative n, Tail honors it, Head does not,
side with the prose. Delete the rest.

### 3. MEDIUM — the recovery scaffolding is wrapped around `Tail`, which never panics.

The drift is Head-only. `Tail` guards with `n <= 0` and returns `""`, so
running it through `trimRecovering` builds panic-recovery machinery around a
call that cannot panic. The negative case is asymmetric by nature and should
read that way: the Head half is the failing, recovered, drift-documenting
half; the Tail half is a plain equality. Better still, fold the Tail
negative check in beside its siblings (a row/sub-test near
`TestTruncatesToRuneCount`) and keep only Head's panic as the standalone
drift marker. The uniform loop pretends the two halves are the same test;
they are not.

### 4. MEDIUM — the `Example` under-sells the documented contract and carries no call-site comments.

The doc comment is genuinely good: it speaks to users, about the runes-not-
bytes promise, and says why it matters (a broken glyph). Keep it.

But the body demonstrates only exact-length truncation. The package doc
leads with "callers may pass a generous limit without measuring the input
first" — the single most reusable affordance — and it is nowhere in the
example. Show `Head(s, n)` with an `n` beyond the rune count returning `s`
unchanged, the way I asked the units example to show its fallbacks.

And there are zero in-function comments. `Head("héllo", 2)` / `Tail("café", 3)`
render on pkgsite with the code; the `2` and `3` and their outputs go
unannotated. In-function example comments are prime real estate. The
combined doc comment carries the runes point, so this is a miss rather than
a reject, but it is a missed opportunity on the one artifact users read.

### 5. MEDIUM (soft) — a single package-level `Example()` leaves neither `Head` nor `Tail` with a symbol-attributed example.

Two exported functions, each with its own doc-promise, and neither gets a
runnable example attached to its godoc entry; `Example()` renders only under
the package. `ExampleHead` / `ExampleTail` would attach comment-bearing,
symbol-targeted examples where a user actually lands. I will not hard-block
this — the exemplar's mandatory example is itself one package-level
`Example_throttle`, and one package example satisfies the skill minimum —
but for a package this small, symbol-targeted examples are the stronger
call, and I would ask for them.

### What is right (so this reads fairly)

- File order is correct: example, then the typical table, then the special
  drift case, helper last — complexity builds as I read.
- The merged `{s, n, head, tail}` table is exactly the shape I want: one row
  states both halves of a cut, fields are named, no global slice, no `if`
  specializing rows. This is the good work.
- No invented concurrency, no synctest theatre for pure functions, no
  error-string assertions. The drift is kept failing rather than blessed.

## report.md

The report is concise and does the right job: it names the drift, states
the golden-hierarchy decision (side with prose), and flags the deliberate
failure. That is the artifact where the author-context belongs. The defect
is not the report — it is that its content was *also* copied into the source
doc comment (finding 2). The report even stays honest about the module name;
the "utils module" invention is the test comment's alone, which makes it
worse, not better.

## Verdict

Changes requested. Findings 1 and 2 block the merge. Rewrite the negative-n
test as visible straight-line calls, trim the doc comment to a two-sentence
rationale, and the rest is a good, lean file.
