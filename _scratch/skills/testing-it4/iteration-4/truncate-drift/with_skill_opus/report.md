# truncate tests

Wrote `truncate_test.go` from the package prose first: one `Example` (rune-safe display clip) plus a merged `TestHeadAndTail` table, since Head and Tail mirror each other.
Finding (drift): the doc promises a negative count is "treated as zero: returns the empty string". Tail honors this; Head slices `r[:n]` and panics. I sided with the prose and pinned the contract in `TestNegativeCountReturnsEmpty`, kept from aborting the run by a recover-guard.
Author question: did you intend Head to clamp negatives like Tail (a one-line fix), or should the doc drop that promise? I inferred the former, since Tail already does it.
Status: `go build ./...` and `go vet` clean; `go test ./...` FAILS only on that one deliberate drift test; Example and TestHeadAndTail pass.
