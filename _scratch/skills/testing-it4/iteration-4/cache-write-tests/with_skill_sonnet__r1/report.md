# cache package: committed tests

Added `cache_test.go` (external `package cache_test`): `Example_sessionToken` for the intended session-token call pattern, `TestCacheLifecycle` (a `synctest`-driven walk through Set/Get/Len, TTL expiry enforced on read, a zero-TTL entry that never expires, an overwrite that resets value and expiry, and Close making the cache empty and Set a no-op, checked for idempotency), and `TestNewPanicsOnInvalidJanitorInterval` with a `mustPanic` helper for the documented panic on a non-positive interval. Also added `cache_internal_test.go` (white-box `package cache`) with one test peeking at the unexported `entries` map: Get/Len already hide expired entries on every read regardless of the janitor, so that peek is the only way to confirm the janitor doc-comment's promise ("removes expired entries") is real rather than masked forever by the read-time filter. Flagging this as the one place the tests reach past the public API, per the maintainer's peephole exception.

No bugs found; behavior matches the doc comments exactly (TTL semantics, zero/negative TTL meaning no expiry, overwrite resetting expiry, Close discarding everything and making Set a no-op, Close idempotency, and the panic guard on `janitorInterval`).

Final status: `go build ./...` and `go vet ./...` clean; `gofmt -l .` clean; `go test ./...` and `go test -race -count=5 ./...` both pass.
