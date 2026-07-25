# truncate tests — engineer's note (revised after maintainer critique)

`truncate_test.go` (external `package truncate_test`), planned go-doc-first: per-function `ExampleHead` and `ExampleTail`, a composed `Example_display` (head-of-preview / tail-of-path call pattern), one merged table (`TestRunesNotBytes`) driving Head and Tail over the rune-counting and boundary promises, and `TestNegativeCountMeansEmpty` for the negative-n promise.

Finding (doc/code drift): the docs promise a negative n is treated as zero. Tail keeps the promise; Head panics. The observed panic is `runtime error: slice bounds out of range [:0] with capacity 32`, and the traceback points at truncate.go:23. Tail carries an explicit `n <= 0` guard that Head lacks, which suggests the author intended the promise package-wide and dropped the check in Head. The test asserts the documented contract, so it stays red until Head honors it. Owner call needed: add the guard to Head, or amend the prose. This inference belongs here and in the eventual commit body, not in the committed test file.

Status: `go build ./...` clean; `go test ./...` FAILS on `TestNegativeCountMeansEmpty` alone (deliberate, per the drift above; it sits last in the file so its panic cannot mask the other results); all examples and `TestRunesNotBytes` pass; gofmt and `go vet` clean.

## Critique disposition

All five findings were verified and applied; the non-blocking suggestion was applied too. Nothing was rejected.

- Finding 1 (blocker), applied: the `Example_display` comment claimed a byte cut at index 5 would split the é. Verified false: `s[:5]` is "héll", valid UTF-8. The comment now says the é spans bytes 1-2 and a cut at byte index 2 splits it; the 12-rune/14-byte counts stay.
- Finding 2 (blocker), applied: the drift test's doc comment is trimmed to the promise, the observed failure, and the owner question. The ordering narration, the "pinning" phrasing, and the implementation narration are gone; the placement judgment stands without being narrated.
- Finding 3 (major), applied: the comment and the previous report claimed Head "panics slicing r[:n]". Re-ran the suite and confirmed the observed message contradicts that phrasing. The committed comment now says only that Head panics; this report records the observed message verbatim, which is defensible from the user's seat.
- Finding 4 (minor), applied: renamed `TestTruncate` to `TestRunesNotBytes`. The old name read like a symbol scan of the package name; the new one claims the property the table proves, and matches the name the maintainer's round-3 feedback used for exactly this merged Head/Tail struct.
- Finding 5 (minor), applied: "pinning" phrasing removed from this report. The sibling-guard inference stays here, its sanctioned home, with the corrected mechanism.
- Non-blocking thought (per-function examples), applied: added `ExampleHead` and `ExampleTail` so each function's pkgsite page shows its own contract. The skill bars examples that merely call the symbol, so each demonstrates two contract facets (the rune-counted cut and the unchanged-when-short promise) with call-site comments; the maintainer's round-3 feedback praises per-function examples that repeat contract comments per page. Neither example touches the negative-n promise: that clause is the disputed drift, and a rendered example should not entrench either side before the owner rules.
