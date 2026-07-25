# units: pin ParseDurationOrDefault

Added `ExampleParseDurationOrDefault` mirroring the sibling `ExampleParseSizeOrDefault`: one well-formed duration, plus the empty-string and garbage fallbacks the doc comment promises; an example fits this trivial contract better than a table test and documents the leniency for users.
Planned from `go doc -all` before reading source; code matches the prose exactly, no drift.
Maintainer note: `ParseSize` trims surrounding whitespace but `ParseDurationOrDefault` does not, so `" 5s "` silently falls back to the default; the docs promise nothing about trimming, so the test does not pin it, but the asymmetry may surprise config authors.
Status: `go build ./...` and `go test ./...` both pass.
