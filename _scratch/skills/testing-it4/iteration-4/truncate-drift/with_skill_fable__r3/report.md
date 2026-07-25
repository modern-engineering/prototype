# truncate tests — engineer's note

Wrote external `truncate_test.go` planned go-doc-first: a runnable `Example` showing the display call pattern (rune-safe cut, Tail suffix, generous limit), plus one merged table (`TestTruncate`) driving Head and Tail from shared cases including 2-, 3-, and 4-byte runes.
Finding (drift): both doc comments promise "a negative n is treated as zero" returning ""; Tail guards `n <= 0`, but Head slices `r[:n]` unguarded and panics on negative n. Tail's parallel guard reads as one intent, so the prose looks right and Head is missing its guard; I did not patch the source since the package is about to ship and that call is the owner's.
`TestNegativeCountMeansEmptyString` asserts the documented behavior and fails deliberately (panic in Head) until the guard lands or the docs are weakened; it sits last in the file so the rest of the suite reports first. (Context: the module here is `example.invalid/truncate`, not a utils module, so the test's import path needs a mechanical update when it moves.)
Status: `go build ./...` PASS; `go test ./...` FAIL, solely the deliberate drift test above; `Example` and `TestTruncate` pass, gofmt and go vet clean.
