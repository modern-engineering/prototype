# units: test ParseDurationOrDefault

Added `TestParseDurationOrDefault` to `internal/units/units_test.go`, table-driven in the file's existing style: valid/compound/zero/negative durations parse, while empty, garbage, bare-number, and whitespace-padded inputs fall back to the default (documented leniency, asserted as behavior, not error).
Judgment call: a valid `"0s"` must return 0, not the default; the test pins that so leniency never conflates "parsed zero" with "fell back".
Maintainer's eyes: unlike `ParseSize`, `ParseDurationOrDefault` does not trim whitespace, so `" 1s "` silently becomes the default — inconsistent with the size path and a likely silent-config-typo trap; the test documents current behavior rather than guessing intent.
Status: `go build ./...` and `go test ./...` both pass (all 8 new subtests green).
