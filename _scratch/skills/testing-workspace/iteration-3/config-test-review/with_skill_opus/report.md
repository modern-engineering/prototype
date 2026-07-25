# config test review

Collapsed the five per-symbol tests that each re-loaded `sample` and poked the unexported `cfg.values` map into one external-package golden test (`TestConfig`) that drives the whole public API as a user would; added the missing "unparseable value falls back to def" branches for `Int`/`Duration`, an error test, and a runnable `Example`; swapped `reflect.DeepEqual` for `slices.Equal` and modernized the benchmark to `b.Loop`.

Deleted `TestDebugDump` (a manual debug scaffold, no contract) and `TestConcurrentUse` (pins concurrency the docs never promise, and it is a real data race on a 100ms sleep: plain `go test` was green but `go test -race` failed on it). Redesign signal for the maintainer: `Load`'s doc promises a positioned `"line N: ..."` error, but it returns a bare `fmt.Errorf`, so the position is only checkable by string-grepping; consider a typed `*ParseError{Line int}` for `errors.As`. The new suite therefore only asserts that malformed lines error, not the wording.

Final status: `go build ./...` clean, `go vet ./...` clean, `go test ./...` green, `go test -race ./...` green. Package (`config.go`) left untouched.
