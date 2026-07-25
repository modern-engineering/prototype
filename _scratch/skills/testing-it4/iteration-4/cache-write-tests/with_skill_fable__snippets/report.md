# Cache tests — engineer's note

Wrote `cache_test.go` (external `cache_test` package) from the go-doc contract before reading the source; found no prose/code drift, including the boundary rule that expiry takes effect at the deadline instant.
Six tests plus a package Example run in `testing/synctest` bubbles on virtual time: the whole-package scenario uses a janitor interval longer than every TTL so misses prove read-side expiry, and each bubble structurally verifies Close releases the janitor goroutine.
Replacement adopts both new value and new TTL in both directions; non-positive TTLs survive a week of sweeps; Close is final and idempotent; New panics on non-positive intervals via a small mustPanic helper.
Janitor sweeps are unobservable through the API (reads enforce expiry first), so they are covered only via lifecycle and leave-live-entries-alone checks; worth noting if sweep metrics are ever wanted.
Status: `go build ./...`, `go vet ./...`, and `go test -race -count=2 ./...` all pass.
