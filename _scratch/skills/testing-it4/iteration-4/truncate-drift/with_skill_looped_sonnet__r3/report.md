# truncate package: committed tests

## What I did

Copied the `truncate` module into `outputs/` and wrote `truncate_test.go` as an
external `truncate_test` package, following the go-doc-first method: the test
plan was drafted from `go doc -all` (package doc, `Head`, `Tail`) before
re-reading the implementation.

- `Example_abbreviateName`: the representative runnable example — abbreviating
  a display name by keeping both ends, the anticipated real call pattern for a
  package that ships both `Head` and `Tail`. The name carries an accented
  letter positioned so the cut lands mid-encoding under naive byte slicing,
  which puts the package's actual reason to exist (rune-safe truncation, per
  the package doc) directly in the one example pkgsite renders, instead of
  leaving it visible only in the maintainer-facing table.
- `TestHeadAndTailKeepWholeRunes`: a table shared by both functions (as the
  skill's own reference recommends for mirrored functions), covering plain
  truncation, the "shorter than n" passthrough, the exact-length boundary,
  multi-byte runes (`héllo`), and `n == 0`.
- `TestNegativeNIsTreatedAsZero`: isolates the one contract clause both doc
  comments make identically ("a negative n is treated as zero").

## Finding: prose/code drift on negative `n`

`Head`'s doc comment and `Tail`'s doc comment both promise the same thing for
negative `n`: the empty string. `Tail` implements a `n <= 0` guard and honors
it. `Head` has no such guard, so `n > len(r)` is false and `r[:n]` executes
with a negative index, which panics instead of returning `""`.

I sided with the prose, per the skill's drift rule (no owner to ask
mid-task): the test asserts `Head("hello", -1) == ""` and recovers the panic
so the failure is reported as a normal test failure rather than crashing the
binary. This is a deliberate, isolated failing sub-test
(`TestNegativeNIsTreatedAsZero/Head`), not a bug I fixed silently.

Author's inference for whoever owns this: either add `Head`'s missing
`n <= 0` guard to match `Tail` (my guess — the two functions read as
intentionally symmetric, and the doc was written for that symmetry), or, if
negative `n` is truly meant to be the caller's problem, weaken both doc
comments to stop promising graceful handling of it.

## Critique response (round 3 maintainer review)

The maintainer's critique (`../critique.md`) raised five points. All five
survived judgment against `skill-v4/SKILL.md`; none conflicted with the
skill's prose, so all five are applied. Nothing was rejected this round.

1. **`Example_shortHash` had zero in-function comments.** The skill calls
   in-function example comments "prime real estate" and requires a
   representative example; a doc comment followed by four bare lines of code
   failed that outright. Fixed by commenting the input and the call.
2. **The mandatory example never showed the package's actual selling
   point.** The package doc leads with rune-vs-byte safety and gives its own
   worked case (`Head("héllo", 2)`); a hex-string hash abbreviation never
   exercises that. Fixed by replacing the scenario: `Example_abbreviateName`
   now truncates a display name containing an accented letter positioned at
   the cut boundary, so the one example pkgsite shows demonstrates why this
   package exists over `s[:n]`, not just the ellipsis-abbreviation shape.
   The Head+Tail-together call pattern and the ellipsis idiom are unchanged;
   only the input data and framing moved.
3. **The first table case was lifted verbatim from the skill's own
   reference doc** (`references/tables-and-subtests.md`'s illustrative
   `{s: "hello", n: 3, head: "hel", tail: "llo"}`). That is the skill's
   pedagogical example, not a value chosen for this package, and it reads
   like the table was templated off the reference rather than authored here.
   Fixed by swapping the plain-ASCII cases to `banana`/`ba`/`na` so no case
   matches the reference text.
4. **`TestNegativeNIsTreatedAsZero`'s doc comment narrated the test's own
   machinery** (the recover-to-normal-failure clause). The skill is explicit
   that test comments speak only to maintainers about the package's contract,
   never about how the test itself is built ("nothing good to say, say
   nothing"). The drift statement is legitimate, non-trivial rationale and
   stays; the trailing clause explaining the recover is cut.
5. **`report.md` referenced a `utils` package that does not exist in this
   task** (this paragraph, previous revision). Concrete inaccuracy in the
   one artifact meant to carry trustworthy context forward. Fixed by
   removing the dangling reference; the inference stands without it.

Everything the critique already found in good shape is left untouched: the
external `truncate_test` package, the example-then-table-then-drift file
order, named struct fields on the four-field case, no global case slices,
and siding with prose on the drift itself.

## Final status

- `go build ./...`: **pass**.
- `go vet ./...`: **pass**.
- `gofmt -l .`: **clean**.
- `go test ./...`: **1 failing sub-test** —
  `TestNegativeNIsTreatedAsZero/Head`, for the reason above, left failing on
  purpose. Every other test (`TestHeadAndTailKeepWholeRunes`,
  `Example_abbreviateName`, `TestNegativeNIsTreatedAsZero/Tail`) passes.
