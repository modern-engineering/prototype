# truncate: committed tests

Wrote `truncate_test.go` (external `truncate_test` pkg) from the go doc contract first: `ExampleHead`/`ExampleTail` plus two tables pinning rune-safe cuts, the generous-limit passthrough, and the n==0 / n==rune-count edges.
Drift found: `Head` breaks its own doc ("a negative n is treated as zero: Head returns the empty string") by panicking on negative n; `Tail` honors the identical promise via its `n <= 0` guard. Fix is one line: give `Head` the same guard.
Judgment call: sided with the prose (no owner/history to consult), so `TestHeadTreatsNegativeCountAsZero` pins the promised empty string and fails deliberately rather than absorbing the panic; source left untouched since the ask was tests. Also noted: module path is `example.invalid/truncate`, not the "utils module" the request mentioned.
Status: `go build ./...` and `go vet ./...` pass; `go test ./...` FAILS only on that one deliberate contract test (recover turns the panic into a legible failure); all other tests and both examples pass.
