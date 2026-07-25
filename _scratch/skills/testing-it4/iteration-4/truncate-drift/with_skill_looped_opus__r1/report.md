# truncate: committed tests (revised after critique)

## What the file contains now

`outputs/truncate_test.go` (external `truncate_test`), file order building complexity:

- `ExampleHead` and `ExampleTail`: two symbol-attributed runnable examples, each rendering its own function's contract on pkgsite. Each shows the rune-not-byte cut and the width-past-the-end clamp, with call-site comments repeating the non-Go contract (runes vs bytes, unchanged on overshoot).
- `TestTruncatesOnRuneBoundaries`: one table of shared `{s, n, head, tail}` inputs driving Head and Tail together (ASCII, multi-byte, exact length, overshoot, zero). Plain `for _, c := range cases` loop, no sub-tests.
- `TestNegativeCountYieldsEmptyString`: the negative-count contract, split into a `Tail` sub-test that asserts `""` directly and a `Head` sub-test that guards the call with an inline recover. No shared helper.

## Drift surfaced (deliberately red, not blessed)

Both functions document that a negative count is treated as zero and returns the empty string. Tail honors it; Head lacks Tail's `n <= 0` guard, so `Head(s, -1)` slices `r[:-1]` and panics. I sided with the prose (the golden contract) and left the Head case red. I did not touch `truncate.go`.

This is not a one-way fix. Two live resolutions exist, and the choice is the author's, not the test's:

1. Add the `n <= 0` guard to Head so it matches its own documentation and Tail's behavior.
2. Decide that panicking on a negative bound is intentional, and amend Head's doc comment to drop the negative-is-empty promise.

Which one is right is the deep-in-mind author context (would the author promise no-panic for existing callers, or refuse to harden a case that never arises in the anticipated flows?). That inference belongs in the fine-grained commit body that lands this deliberately-red test, not as a prescription in the test or the report. The revised test comment states the divergence factually and stops short of prescribing.

## Status (honest)

- `gofmt -l -d .`: clean.
- `go vet ./...`: clean.
- `go test ./...`: FAILs only on `TestNegativeCountYieldsEmptyString/Head` (recovered panic reported as a failure). `ExampleHead`, `ExampleTail`, `TestTruncatesOnRuneBoundaries`, and `TestNegativeCountYieldsEmptyString/Tail` all pass.

## Critique disposition

- Finding 1 (HIGH, scaffolding leak `exemplar/'s mustPanic`): APPLIED, fully. Removing the shared `callWithoutPanic` helper (see finding 4) deleted the leaking doc comment outright rather than trimming it.
- Finding 2 (redundant sub-tests over identical rune-table code): APPLIED. Dropped the `name` field and the `t.Run` wrapper; the failure messages already print the inputs (`Head(%q, %d)`), so naming the inputs is enough. It is now a plain loop over the merged case struct.
- Finding 3 (one package Example where two symbol-attributed ones belong): APPLIED. Split `Example` into `ExampleHead` and `ExampleTail`. Doc comments now speak about the attributed symbol to users, not a tour of the body. This aligns with the skill (examples speak about the attributed symbol) and the per-function examples this package had been praised for; splitting does not conflict with the package prose, which documents Head and Tail separately.
- Finding 4 (function-value dispatch routes non-panicking Tail through a panic guard): APPLIED. Replaced the uniform `funcs` table with two sub-tests: Tail asserts `""` directly, Head guards its own call with an inline recover. Head and Tail genuinely diverge here, so the recover machinery guards only the panicking case. This also removed the helper indirection and, with it, the finding-1 leak.
- Finding 5 (report prescribes the fix instead of surfacing the author's choice): APPLIED. The drift write-up above now presents both live options and locates the author-context inference in the commit body rather than closing with "should clear once its guard matches Tail's."

Nothing was rejected. The two sub-tests in `TestNegativeCountYieldsEmptyString` (Head, Tail) are not in tension with finding 2's removal of sub-tests from the rune table: there the cases were identical code under input-category names, here the bodies genuinely differ (direct assertion vs recover) and the names are the two functions.
