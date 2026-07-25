# Config test review

Rewrote the suite in an external `config_test` package: the old tests poked at `cfg.values` internals and tested one symbol each, so I consolidated them into one whole-package `TestConfig` scenario, added the missing coverage the docs promise (positioned "line N:" Load error; Int/Duration fallback on unparseable values), added a runnable package Example, and moved the benchmark to `b.Loop`.
Deleted two tests outright: `TestDebugDump` was a manual-inspection dev scaffold, and `TestConcurrentUse` pinned concurrency the docs never promise while itself being a data race ("green" only held without `-race`; it fails under `go test -race`).
Redesign signal for the maintainer: Load's positioned error is only a formatted string, so tests can do no better than grep the "line N:" prefix; a typed error carrying the line number (inspected via `errors.As`) would make the position a first-class contract. Judgment call: I asserted only that prefix, since the docs promise the form, and left `config.go` untouched.
Final status: `go build ./...` OK; `go test -race -count=3 ./...` PASS (includes the Example and both tests); gofmt and `go vet` clean; benchmark smoke-run passes.
