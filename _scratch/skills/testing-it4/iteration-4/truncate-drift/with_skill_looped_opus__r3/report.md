# truncate tests

External `truncate_test.go`, drafted from the package doc first: symbol-targeted `ExampleHead` / `ExampleTail`, a mirrored `{s, n, head, tail}` table for the rune-counting contract, and a negative-n contract test.

## Finding (author-context)

Both `Head` and `Tail` doc-promise "a negative n is treated as zero: returns the empty string". `Tail` guards with `if n <= 0`; `Head` lost that guard and slices `[]rune(s)[:n]`, so a negative n panics. The forgiving behavior Tail already has is clearly the intent, so I sided with the prose (the golden contract) and left the test asserting the promised empty string. It stays red until `Head` grows the guard. Whether to actually harden `Head` or narrow the doc is the author's call; that decision belongs in the commit body when the fix lands.

## Status (verified honestly)

`gofmt -l` clean, `go vet ./...` clean, `go build ./...` clean. `go test ./...` fails on exactly one case, `TestNegativeLimitYieldsEmptyString` (deliberate) — Head's panic is recovered into a readable assertion failure. `ExampleHead`, `ExampleTail`, and `TestTruncatesToRuneCount` all pass.

## Critique response

Applied:

- Finding 1 (HIGH) — removed the `func`-value dispatch table and the `trimRecovering` helper. The negative-n test now makes two visible straight-line calls: `Tail(...)` asserted directly, `Head(...)` under an inline recover. The code under test is legible at each call site, and there is no helper to place. This is the pattern `helpers.md` reserves against for trivial packages, and the maintainer had said verbatim on this package "I DO NOT WANT TO SEE THIS PATTERN HERE."
- Finding 2 (HIGH) — cut the eight-line doc comment to a two-sentence contract-level rationale. Removed the report echo, the implementation claim ("slices `[]rune(s)[:n]` with no lower guard"), the plumbing narration about recovery, and the invented "utils module". The remaining line comment states the gap at the contract level with no plumbing narration.
- Finding 3 (MEDIUM) — recovery scaffolding no longer wraps `Tail`, which never panics: Tail is a plain equality, only Head is recovered. I kept both halves in one asymmetric test rather than folding the Tail negative into the merged table, because a negative-n row would need an `if` to skip the panicking Head column — the exact table smell `tables-and-subtests.md` warns against. Finding 1's own primary remedy (two visible calls) is that structure.
- Finding 4 (MEDIUM) — the examples now carry in-function comments on the non-obvious literals and demonstrate the lead affordance from the package doc (a generous limit beyond the rune count returns the input unchanged).
- Finding 5 (MEDIUM, soft) — split the single package-level `Example` into `ExampleHead` and `ExampleTail` so each attaches to its godoc entry with a doc comment about its own symbol. The critique flagged this as the stronger call for a package this small, and the maintainer had praised exactly this split on this package in an earlier round.

Nothing rejected outright; finding 3's optional "fold Tail into the table" form was declined for the table-smell reason above, in favor of finding 1's own straight-line remedy.
