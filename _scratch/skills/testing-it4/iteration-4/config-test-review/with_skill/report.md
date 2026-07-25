Rewrote config_test.go as an external `package config_test` user of the package: the old suite was green but only shape-deep — per-symbol tests re-loading one fixture, self-naming comments, a committed debug dump, and TestConcurrentUse, which failed `go test -race` outright (unsynchronized `cfg` write in a goroutine) while pinning concurrency the docs never promise, so it was deleted rather than repaired.

The new hire clearly worked from the implementation, not the doc comments: nothing covered the prose promises (whitespace trimming, blank/comment lines, fallback on unparseable values, positioned "line N:" errors), and the internal package let tests grep `cfg.values` instead of sitting in the user's seat. Consolidated into one whole-package scenario test plus a malformed-input table, added the mandatory runnable example of the load-then-typed-reads pattern, and moved the benchmark to `b.Loop`.

One design finding for the owner: Load's positioned errors are string-shaped, so holding the documented "line N:" form requires matching `err.Error()`; a typed error carrying the line number would let callers and tests check position properly.

Final status: `go build ./...` and `go test ./...` pass; `go test -race ./...` also passes (it did not before); gofmt and go vet clean.
