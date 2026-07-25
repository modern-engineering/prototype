# truncate package: committed tests

## What I did

Wrote `truncate_test.go` as an external `truncate_test` package, following
the go-doc-first method: drafted the plan from `go doc -all` before
re-reading `truncate.go`. It carries:

- `Example()` — the mandatory representative call pattern: shortening an
  accented display name (`"Bjørn Åström"`) in the middle for a narrow UI,
  which also exercises the package's core promise of not splitting
  multi-byte runes.
- `TestHeadAndTailCountRunes` — a single table driving both `Head` and
  `Tail`, since the doc holds them to the same contract (count runes, pass
  short input through unchanged, clamp to `""` at `n=0`). Includes the
  doc's own `"héllo"` example as a case.
- `TestNegativeNIsZero` — isolated from the table because `Head` and `Tail`
  diverge here (see finding below).

## Finding: Head does not honor its own doc for negative n

Both doc comments promise the same thing: "A negative n is treated as
zero: Head/Tail returns the empty string." `Tail` implements this with an
explicit `if n <= 0 { return "" }` guard. `Head` has no such guard, so
`Head(s, -1)` falls through to `r[:n]` with a negative index and panics
(`slice bounds out of range`) instead of returning `""`.

This reads as an oversight rather than a deliberate divergence: the two
functions are otherwise symmetric, and `Tail` already carries the guard
`Head` is missing, suggesting it was dropped rather than never intended.
Per the testing skill's rule (side with the prose when the code did not
move deliberately), `TestNegativeNIsZero` asserts the documented behavior
for both functions. The `Head` half wraps the call in a `recover` so the
drift surfaces as one reported test failure rather than a crashed test
binary that would also hide the `Tail` assertion and the `Example`.

I did not patch `truncate.go`: fixing the doc/code mismatch is the
package owner's call (add `Head`'s missing guard, or narrow the doc), not
something to absorb silently inside the test.

## Final status

- `go build ./...`: pass.
- `go test ./...`: **fails**, deliberately — `TestNegativeNIsZero` reports
  the `Head` drift above. `TestHeadAndTailCountRunes` and `Example` pass.
