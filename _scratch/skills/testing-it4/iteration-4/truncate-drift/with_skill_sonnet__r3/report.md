# Engineer's note: truncate package tests

## What I did

Copied the `truncate` module into `outputs/` and wrote `truncate_test.go`
as an external `truncate_test` package: one runnable `Example()` for the
anticipated call pattern (shortening a UI label without splitting a
multi-byte rune), a table-driven `TestHeadAndTailStayRuneSafe` covering
the mirrored contract both functions' docs make (rune-safe cuts,
pass-through on a generous limit, zero limit), and
`TestNegativeNIsTreatedAsZero` for the negative-n contract both docs
state verbatim.

## Finding (author-context framing)

While drafting the negative-n test from the doc comments (go-doc-first,
per the team's testing standard), I found a drift between prose and
code, not a bug I went looking for: `Head`'s doc comment promises "A
negative n is treated as zero" (same wording as `Tail`'s), but only
`Tail`'s implementation has the `n <= 0` guard; `Head` has no such guard,
so `Head(s, negative)` panics (slice bounds out of range) instead of
returning `""`.

Per the maintainer's golden hierarchy, doc prose outranks implementation,
so the test asserts the documented contract rather than the current
behavior, and I did not patch `truncate.go` myself since this task was
scoped to writing tests. `TestNegativeNIsTreatedAsZero` therefore panics
on the `Head` half of the assertion. This is a deliberate failing test
with a stated reason, not an oversight: it is the honest way to hold
`Head` to its own doc comment. Before this ships in `utils`, the owner
should pick one fix: add the same `n <= 0` guard to `Head` that `Tail`
already has, or soften Head's doc comment.

## Final status

- `go build ./...`: passes.
- `go test ./...`: fails. `TestHeadAndTailStayRuneSafe` and the package
  `Example` pass; `TestNegativeNIsTreatedAsZero` panics on `Head`'s
  negative-n case for the reason above.
