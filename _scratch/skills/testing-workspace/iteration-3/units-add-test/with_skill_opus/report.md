# ParseDurationOrDefault test

Added `ExampleParseDurationOrDefault` to `internal/units/units_test.go`: happy path plus the empty-string and garbage fallbacks you flagged. Chose a runnable example to mirror its sibling `ExampleParseSizeOrDefault` and because the leniency is user-facing contract, not a bug hunt.
No drift: source matches the doc comment exactly; the doc already frames the garbage fallback as deliberate leniency, so no owner question needed.
`go build ./...`, `go vet ./...`, `go test ./...`: all pass. Good to commit.
