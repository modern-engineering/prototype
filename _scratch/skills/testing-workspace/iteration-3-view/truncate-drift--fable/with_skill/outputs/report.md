# truncate: committed tests

Added `truncate_test.go` (external package): a negative-n special-case test, one rune-counting table per function, and a package Example on the doc's own "héllo" string.

Finding for the maintainer: `Head` contradicts its doc comment. The prose promises a negative n is treated as zero, but `Head` lacks `Tail`'s `n <= 0` guard, so `Head("héllo", -1)` panics with a slice bounds error. I sided with the prose and left the test failing deliberately; the fix is copying Tail's guard into Head before this ships.

Status: `go build ./...` passes; `go test ./...` fails on `TestNegativeCountIsTreatedAsZero` (the drift above, panic converted to a named failure), with the other three tests passing.
