# units: test for ParseDurationOrDefault

Added `ExampleParseDurationOrDefault` to `internal/units/units_test.go`, mirroring the sibling `ExampleParseSizeOrDefault`: one runnable example covering a valid duration ("90s" -> 1m30s), empty string -> default, and garbage -> default, since the doc comment names the leniency as deliberate.
One author-context note: unlike `ParseSize`, `time.ParseDuration` does not trim whitespace, so a padded value like " 5s " silently falls back to the default; the prose permits this ("a string that does not parse also returns def"), but if trimming was intended the helper should trim before parsing — I did not pin the current behavior either way.
Final status: `go build ./...` clean, `go test ./...` passes (ok, 0.341s).
