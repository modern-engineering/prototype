# truncate package: committed tests

## What I did

Drafted the test plan from `go doc -all` before opening `truncate.go`, per
the team's go-doc-first convention, then wrote `truncate_test.go` as an
external `truncate_test` package:

- `Example()` — the anticipated call pattern from the package doc's own
  motivating scenario (trimming a long identifier for a narrow display),
  exercising `Head` and `Tail` together.
- `TestCutsExactlyNRunes` — the core promise, one table driving both
  `Head` and `Tail` since they mirror each other, including the
  multi-byte case the package doc itself uses as its example.
- `TestGenerousLimitReturnsInputUnchanged` — the documented shortcut that
  a caller may pass n past the string's rune count without measuring it
  first.
- `TestNegativeNIsTreatedAsZero` — see finding below.

## Finding: `Head` does not honor its own doc for negative n

`Head`'s and `Tail`'s doc comments both promise: "A negative n is treated
as zero: ... returns the empty string." `Tail` implements this
(`if n <= 0 { return "" }`); `Head` has no such guard and slices
`r[:n]` directly, so a negative n panics with a slice-bounds error
instead of returning `""`.

Author-context read: this reads as an oversight, not a deliberate choice.
`Tail` was written with the negative-n guard front and center, as the
first line of the function. `Head`'s existing guard (`n > len(r)`) only
covers the "too large" side, not the "too small" side its own doc
promises. Nothing in the code suggests the prose was updated after a
deliberate behavior change; the more likely story is that `Tail`'s guard
was dropped when `Head` was adapted from it. Siding with the prose per
the maintainer's drift policy: I held `Head` to its documented contract
rather than silently encoding the panic, so `TestNegativeNIsTreatedAsZero`
fails and recovers cleanly (see the doc comment on that test) instead of
crashing the run. Recommend adding the same one-line guard `Tail` uses to
`Head`, rather than walking back the doc's negative-n promise: a panic on
this input contradicts the whole point of a "generous limit, don't
measure first" API.

## Final status

`go build ./...` — pass.
`go vet ./...` — pass.
`go test ./...` — 3 of 4 tests pass; `TestNegativeNIsTreatedAsZero` fails
by design, reporting the `Head` doc/code drift above. Not a flaky or
incidental failure: it is the one test in the suite written to fail until
someone resolves the drift (fix `Head`, or narrow the doc).
