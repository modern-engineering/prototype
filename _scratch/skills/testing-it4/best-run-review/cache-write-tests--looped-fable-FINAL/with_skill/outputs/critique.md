# Critique — cache_test.go and report.md

Reviewed as the package maintainer, bound to skill v4: whole exemplar corpus
read first, exported surface taken from `go doc -all`, no non-test source
opened. Verified locally: `gofmt -l` clean, `go vet` clean, `go test` and
`go test -race -count=1` pass on go 1.25.

The suite is flow-grained, go-doc-first, and honest — no per-symbol scan, no
change-detectors, no self-naming comments, bodies are straight-line hard-coded
code. The findings below are ranked; the first one matters, the rest are
polish.

## cache_test.go

### 1. HIGH — the doc's one explicit concurrency promise is never exercised

The package doc promises outright: "All methods are safe for concurrent use by
multiple goroutines." No test calls the API from more than one goroutine.
Worse, the appearance of concurrency here is an illusion of virtual time:

- `TestCache` parks the janitor on purpose (1h interval, ~11m of virtual
  time) — its own comment says so.
- `TestEntryWithoutTTLNeverExpires` rides 1440 sweeps, but synctest advances
  virtual time only while every goroutine in the bubble is durably blocked.
  Each sweep runs while the test goroutine is parked in `time.Sleep`. Janitor
  and caller never overlap; the executions are strictly sequential.

So `-race` passing proves nothing about this promise — the detector was never
shown two goroutines touching the cache. This is not a call for statistical
race-hunting (banned, and rightly): one realistic flow suffices. Several
handler goroutines each storing and looking up their own session, WaitGroup
against leaks, assertions inside the goroutines (*testing.T is
concurrent-safe). That is the anticipated call pattern for a session-token
cache — concurrent access is why the type carries the promise at all.

### 2. MEDIUM — ExampleCache closes the file instead of opening it

Doctrine: the file builds complexity as it reads — examples first (when not in
a standalone example_test.go), then the typical call patterns, the special
cases and their machinery last. This file reads scenario → property test →
panic table → example. Move `ExampleCache` to the top of the file or into
`example_test.go`; the panic table stays last either way.

### 3. MEDIUM — the example's comment promises what its code never shows

The doc comment says "a miss covers both 'never stored' and 'expired'", but
the code demonstrates only the never-stored miss. Expiry is this package's
entire reason to exist and it is absent from the one artifact users see on
pkgsite. Either demonstrate the expired miss (a short real TTL and a real
sleep is the honest route; weigh the weak-runner flake risk) or cut the
"expired" clause from the comment so it stops claiming coverage the code
doesn't have. A comment on the example must not out-promise the example.

### 4. LOW — TestEntryWithoutTTLNeverExpires drops the values

Both `Get("forever")` and `Get("negative")` check only `ok`. Asserting the
stored `"v"` costs nothing and would catch value corruption on the no-expiry
path, which no other test crosses. Every other Get in the suite checks the
value; these two should too.

### 5. NIT — the panic table's failure message names the input, not the sin

`New(%v) did not panic` identifies the case; the reference shape also wants
the specific invalidity stated from the case. With two self-explanatory
durations and the property in the test name, this is defensible — inline
recover over a mustPanic helper is the right call at this size. Noting for the
record, not demanding a change.

## report.md

### 1. MEDIUM — "exercised (and race-checked)" overstates the janitor coverage

The report claims the sweep "is exercised (and race-checked)". Exercised, yes,
1440 times. Race-checked, no: under virtual time the sweeps run only while the
sole other goroutine is durably blocked (see finding 1). The sentence would
lead a reader to believe the concurrency promise has evidence behind it. It
does not.

### For the record, done right

The boundary-instant question (expiry at exactly t+ttl) was deferred to the
owner with the exact tradeoff stated instead of silently pinning
implementation behavior. That is the correct move, and the offer of a one-line
case on request is exactly the shape I want these reports to take. No drift
findings were missed: the tests hold every clause the prose actually makes,
except the concurrent-use clause above.

## Package smell (secondary goal)

None found. No test needed a translation comment for an API value; the
TTL-in-durations surface reads directly. Nothing to flag for rethink.
