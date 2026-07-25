# Expiring-cache tests

Wrote the committed suite go-doc-first: `Example` (session-token call pattern), `TestSessionTokenLifecycle` (whole-package scenario in a synctest bubble covering TTL vs. permanent entries, replacement, read-expiry before any sweep, Len, and post-Close semantics), `TestNewPanicsOnNonPositiveInterval`, plus one internal peephole `TestJanitorReclaimsExpiredEntries`.
Findings: none contradicting the docs; the code matches its prose. I added the internal peephole because the janitor's one distinct promise (reclaiming map slots) is invisible through Get/Len, which already hide expired entries — leaving it external would have left a janitor that never deletes passing every test.
Status: `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race -count=1` all PASS; gofmt clean.
