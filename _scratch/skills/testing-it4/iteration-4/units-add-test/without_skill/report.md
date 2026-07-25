# ParseDurationOrDefault test

Added `TestParseDurationOrDefault` to `outputs/internal/units/units_test.go`, table-driven in the file's existing style, covering valid parses ("5m", "1h30m", "0s") plus the documented lenient fallbacks: empty string and garbage ("soon", bare "10") both return the default rather than an error, matching the author's stated intent.
One case worth the author's eye: "0s" parses to a zero duration and is honored as-is, not replaced by the default; the doc comment implies only parse failures fall back, so I pinned that behavior deliberately.
`go build ./...` and `go test ./...` both pass (ok, 0.346s), no failures.
Status: done, ready to commit.
