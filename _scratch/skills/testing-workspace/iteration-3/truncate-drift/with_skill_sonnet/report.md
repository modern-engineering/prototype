Wrote `truncate_test.go` (table tests `TestHead`/`TestTail` plus `ExampleHead`/`ExampleTail`) from the package doc, go-doc-first, before trusting the source.
Finding: the doc promises "a negative n is treated as zero" for both `Head` and `Tail`, but only `Tail` guards `n <= 0`; `Head` slices `r[:n]` directly and panics on negative n.
Judgment call: sided with the doc (unowned, no history of deliberate drift) and kept the negative-n row in `TestHead`, wrapped in a recover helper so the known panic fails only that row instead of crashing the whole run.
Final status: `go build ./...` passes; `go vet ./...` passes; `go test ./...` fails as expected/intended on exactly the one documented `TestHead` case (`n=-1` panics instead of returning `""`) — `TestTail` and both examples pass.
Maintainer action needed: fix `Head` to guard negative n like `Tail` does, or narrow the doc comment; do not silence the test to match the current panic.
