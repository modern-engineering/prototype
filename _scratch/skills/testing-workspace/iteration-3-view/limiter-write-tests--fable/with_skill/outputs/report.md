# Limiter test suite

Wrote an external contract suite (`limiter_test.go`) from the exported docs before reading the source: zero-burst special case first, then constructor panics, bucket-start/refill-cap, Wait-until-refill, ctx cancellation, ErrClosed after Close, plus a runnable Example proving Close idempotence in usage; timing tests run on `testing/synctest` virtual time with exact elapsed assertions.
Findings for the maintainer: (1) behavior of a Wait already blocked when Close lands is undocumented and nondeterministic (the select races `done` vs `tokens`), worth a doc decision before the PR; (2) `time.Second / time.Duration(rate)` gives a zero ticker interval (panic in NewTicker) for rate > 1e9, unguarded by the panic checks; (3) the docs never promise concurrent use, which a gateway limiter surely needs, so no concurrency test was committed per convention: promise it in prose, then pin it.
No prose-vs-code drift found; the unbuffered-channel encoding of burst 0 matches the documented special case exactly.
Status: go build ./..., go vet ./..., go test ./... all pass; stable under -race -count=10 and GOMAXPROCS=1 -count=20.
