# Engineer's note: `./truncate` committed tests

Wrote `truncate_test.go` (external `truncate_test` package) from `go doc -all`
first: a package-level `Example_display` reusing the doc's own "héllo" sample,
a table covering both `Head` and `Tail` for rune-safety and length edge cases,
and a dedicated test for the documented negative-`n` behavior.

**Finding (drift, not silently absorbed):** both doc comments promise "a
negative n is treated as zero" returning `""`. `Tail` guards `n <= 0` and
honors this; `Head` has no such guard and instead panics
(`slice bounds out of range [:-1]`) on `n < 0`. `Tail`'s explicit guard next
to `Head`'s bare `if n > len(r)` reads as an oversight, not a deliberate
change, so the test sides with the prose rather than the implementation.
This needs the package owner's call: promise it (add the guard) or refuse
to harden and fix the doc comment instead.

## Status

`go build ./...`: **PASS**. `go test ./...`: **FAIL** — deliberately, one
test (`TestHeadAndTailTreatNegativeNAsZero`) pins the documented contract for
`Head` and fails against the current panic, per the finding above; it
recovers the panic so the rest of the suite still runs and reports cleanly.

## Response to maintainer critique

The critique bounced two items as blocking, flagged two as real coverage
gaps, and one as a one-line trim. All five held up against the skill
prose and are applied:

- **Banned phrase, cut.** `TestHeadAndTailTreatNegativeNAsZero`'s comment
  said the test "pins the documented contract rather than the
  implementation... instead of crashing the whole test binary" — a
  near-verbatim hit on the corpus's named example of a comment that says
  nothing a reader doesn't already know from the fact that it's a test.
  Trimmed to the actual finding: "Tail honors that promise; Head panics
  instead."
- **`Example_display` got in-function comments.** The doc comment carried
  all the rationale and the body was bare, the exact defect this skill's
  audience doctrine calls out (in-function example comments are prime
  real estate because pkgsite renders them at the call site). Added a
  one-line comment per call explaining what's non-obvious: `Head` keeps
  the é intact because it counts runes, and `Tail`'s result is the last
  two *runes*, not the last two bytes, which would have split the é.
- **Table renamed, not split.** `TestHeadAndTailCountRunesNotBytes` had
  four of five rows pinning length/zero/empty-input behavior, unrelated
  to the rune-counting property in its name. Renamed to
  `TestHeadAndTailHonorLengthAndRuneBoundaries` rather than splitting into
  two functions: the skill favors fewer tests covering more of the
  exported surface, and a name that owns the full table serves that
  better than a second small function would.
- **Added the missing multi-byte boundary row.** No case exercised "n
  exceeds the rune count" on a multi-byte string; every rune-safety row
  stopped at partial truncation and every over-length row was ASCII-only.
  Added `{s: "héllo", n: 10, head: "héllo", tail: "héllo"}`, which would
  fail if the fallback path ever compared `n` against a byte count
  instead of `len([]rune(s))`.
- **Duplicated rationale, resolved as a side effect.** Once the negative-n
  test's comment lost the panic-recovery aside (previous point), it no
  longer duplicated `callHead`'s own doc comment. Nothing further to
  change there.

Nothing from the critique was rejected; all five points matched both the
skill's audience doctrine and its "fewer tests, more surface" preference.
`go build ./...` still passes and `go test ./...` still fails on exactly
the same, deliberate test, for the same documented reason.
