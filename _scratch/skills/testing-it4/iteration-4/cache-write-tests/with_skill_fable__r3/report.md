# Cache test suite — engineer's note

Wrote `cache_test.go` as an external `package cache_test`: a runnable `Example_sessionTokens` showing the session-store call pattern, one synctest whole-lifecycle scenario (janitor sweeps hourly so every expiry observed is read-enforced, per the Cache doc), and a panic table for `New`.
Planned go-doc-first; found no prose/code drift, but one reading worth your confirmation: expiry is inclusive at the deadline (`!now.Before(deadline)`), and the test pins a miss at exactly t+ttl — if you meant exclusive, both test and doc need a word.
Also note the janitor's own deletions are unobservable through the exported API (Get/Len already hide expired entries), so it is covered only indirectly (it runs in the bubble and must exit on Close); intentional, or do you want that surfaced?
Final status: `go build ./...`, `go vet ./...`, and `go test -race -count=1 ./...` all pass; gofmt clean.
