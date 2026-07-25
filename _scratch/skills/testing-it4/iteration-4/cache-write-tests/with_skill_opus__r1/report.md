# Cache tests — engineer's note

Wrote the committed suite go-doc-first from the exported contract: a runnable `Example` for the session-token pattern, one whole-package `TestCacheLifecycle` on synctest virtual time (store/replace/expiry-on-read-before-any-sweep/no-TTL-survives-sweeps/Close empties + no-ops + idempotent), and a table-driven panic check for non-positive intervals.
Findings: the janitor's removal is unobservable through the API (Get/Len already hide expired entries), so I test its two visible promises — expiry-on-read and no-TTL survival across sweeps — and let synctest's clean bubble exit prove Close actually stops the goroutine; I skipped a race-hunt stressor since the docs' concurrency promise is covered by running the scenario under `-race`.
Heads-up: your output dir already held a `cache_test.go` (not in the fixture module), a leftover from a prior run; I replaced it with this authored suite.
Status: `go build ./...` OK, `go vet` clean, `gofmt` clean, `go test ./...` and `go test -race` both PASS.
