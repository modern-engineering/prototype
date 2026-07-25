# cache tests

Planned from the package prose first, then wrote the flows a user runs: an `ExampleCache` round-trip, read-side expiry proving Get/Len hide an entry before the janitor sweeps, no-expiry TTL, Set replacement, an inert closed cache, and New's panic on a non-positive interval.
Time-based cases run in `synctest` bubbles on the virtual clock, so `time.Sleep` advances hours instantly and stays deterministic under `-race`.
The janitor's reclamation is invisible through Get/Len, so one internal-package test peeks at the entry map — the only contract clause the exported API can't observe.
`go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...` all pass; no drift found between prose and code.
