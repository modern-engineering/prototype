# truncate: committed tests

Wrote `outputs/truncate_test.go` (external `truncate_test`): a whole-package `Example` showing rune-safe display truncation, and `TestTruncate`, one `{s, n, head, tail}` table driving both Head and Tail across the documented boundaries (empty, generous limit, exact length, zero, and a real cut on a 4-byte-rune boundary).
Drift found: both doc comments promise "a negative n is treated as zero," and your Tail delivers it, but Head omits Tail's `n <= 0` guard, so `Head(s, -1)` panics (slice bounds out of range) instead of returning "". I sided with the prose and pinned it in `TestNegativeCountYieldsEmptyString`, which recovers the panic into a clean failure so the rest of the suite still reports.
`go build ./...` and `go vet ./...` pass; `go test ./...` FAILS on that one deliberate test. Fix: add `if n <= 0 { return "" }` to Head, or amend its doc if the panic is intended.
