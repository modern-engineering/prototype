Rewrote `config_test.go` as an external `config_test` package, folding `TestLoad`/`String`/`Int`/`Duration`/`Keys` into one `TestConfig` scenario (per-symbol tests were change-detector-shaped) and adding the missing coverage for Load's documented malformed-line error and for Int/Duration's "value present but unparseable" fallback.

`TestConcurrentUse` raced under `go test -race` (unsynchronized write in the goroutine vs. read in the main goroutine) and pinned a concurrency guarantee the docs never promise; deleted rather than fixed, per the docs' contract.

`TestDebugDump` was a dev scaffold (only `t.Logf`, no assertions, "check the dump manually" comment); deleted.

Judgment call: I test only that malformed input errors, not the exact "line N: ..." text, to avoid a brittle string-matching test; if that format is a real promise to callers, it deserves a typed/sentinel error so it can be tested via `errors.As` instead of grepping messages.

Final status: `go build ./...` and `go test ./...` both pass; suite is also clean under `-race`.
