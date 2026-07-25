# truncate tests

Wrote external `truncate_test.go`: a runnable `Example_ellipsis` showing the Head+Tail display pattern, one merged table pinning both functions per cut (rune counting, unchanged-when-short, zero, empty), and `TestNegativeCountKeepsNothing` for the negative-n promise.
Finding: both doc comments promise negative n is treated as zero; Tail honors it, but Head slices `r[:n]` unguarded and panics. Tail's guard plus the matching prose read as a deliberate promise, so Head's panic looks like an oversight; the test sides with the prose rather than absorbing the drift, and the owner should either add Head's guard or amend both docs.
Status: `go build ./...`, `gofmt -l`, and `go vet ./...` are clean; `go test ./...` fails on the one deliberate test above (panic recovered into an ordinary failure), everything else passes.
