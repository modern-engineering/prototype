# limiter tests — engineer's note

Wrote external-package tests (`outputs/limiter_test.go`) pinning the godoc contract: full-bucket start + non-blocking Allow + idempotent Close (`ExampleLimiter`), the burst-0 special case, Wait-blocks-until-refill (its return time also pins the 1/rate refill rate), context-deadline precedence, ErrClosed after Close, and New's panic guard; `testing/synctest` runs the ticker goroutine on a virtual clock so the timing assertions are exact and instant.
Finding 1: the docs never promise method-level concurrency, yet the implementation is fully channel-based and safe, and a gateway limiter will be called from many goroutines — add a "safe for concurrent use" line so a test can pin it (I did not, per Go doc convention).
Finding 2: Close also unblocks an already-waiting Wait with ErrClosed, but the prose only promises *subsequent* Wait calls return ErrClosed, so I left the in-flight behavior unpinned rather than write a change-detector.
Judgment calls: proved "starts full / Allow drains it" through the runnable Example instead of a separate test, and did not assert the panic *message* strings (only that New panics).
Status: `go build ./...` OK; `go test ./...` OK — also green under `-race -count=5` and `GOMAXPROCS=1 -count=20`; `ExampleLimiter` executes and passes.
