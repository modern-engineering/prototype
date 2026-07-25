# truncate tests — engineer's note

Wrote external-package tests go-doc-first: a composed Example (Head+Tail middle-ellipsis for display), one mirror-image table (`TestTruncate`) driving both functions over rune-counting, pass-through, zero, and empty cases, and `TestNegativeCountMeansEmpty` for the negative-n promise.
Finding: both doc comments promise negative n is treated as zero, and Tail has the `n <= 0` guard, but Head lacks it and panics on `r[:n]`; given the functions are otherwise mirror images, this reads as an oversight rather than a deliberate change, so the test sides with the prose (a recover converts the panic into a plain failure). Owner call: add the one-line guard to Head, or amend the docs.
Status: `go build ./...` and `go vet ./...` pass; `go test ./...` fails deliberately on `TestNegativeCountMeansEmpty` (Head drift above) while `TestTruncate` and `Example` pass. Source left untouched: fixing exported behavior is the owner's decision.
