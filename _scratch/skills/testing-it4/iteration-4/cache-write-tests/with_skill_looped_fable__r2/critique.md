# Critique: cache committed tests (with_skill_looped_fable__r2)

Reviewed as the package's maintainer, bound to skill-v4 and the package's
prose (`go doc -all`; no non-test source opened). Verification claims in
report.md reproduce: `gofmt -l` clean, `go vet` clean, `go test -race
-count=1` passes.

Overall: the shape is right and the arithmetic is honest. External
`package cache_test`, example first, one whole-package lifecycle under
synctest, panic table last, helper after first use, no error-string
asserts, no change-detectors, no per-symbol grain. I checked the time
math (the 44-minutes-to-first-sweep claim and the two-dozen sweeps over
the day are both exact). What remains are contract gaps: one tier-one
prose promise with no test in its seat, one replacement direction never
run, and the sharpest clause of Set's prose held only by the report.

## outputs/cache_test.go

### HIGH — the explicit concurrency promise never gets a user in its seat

The Cache doc promises, in so many words: "All methods are safe for
concurrent use by multiple goroutines." That is the most sacred layer of
the contract, and no test ever has two goroutines touch the API. The
doctrine's latitude ("a suite whose scenarios run inside synctest bubbles
is the concurrency testing") covers docs that *omit* concurrency; it does
not absorb an explicit promise attached to a suite whose one scenario is
strictly sequential. The only second goroutine here is the package's own
janitor, which under virtual time runs almost exclusively while the test
goroutine is durably asleep, so the report's appeal to `-race` covers the
sweep path and nothing else. I am not asking for a race hunt. The
anticipated caller of a session-token cache is a fleet of request
handlers: a few goroutines doing Get/Set on shared and distinct keys
inside the existing bubble, WaitGroup, assertions made directly on the
concurrent-safe `*testing.T`, is the promised seat with the least
orchestration. If the author decides against it, record that refusal as
an author decision; do not let the report imply coverage (see below).

### MEDIUM — Set's replacement semantics are pinned in one direction only

The refresh scene proves TTL-over-TTL restarts the clock: good, and the
9-minutes-later probe that outlives the original deadline is exactly how
to prove it. But "replacing any existing entry" crosses expiry classes
too, and neither crossing ever runs: "pinned" and "archive" are set once
and only read. Failure scenario the suite cannot catch: an implementation
that fails to clear the old deadline when the new ttl is zero would
expire a "made immortal" entry at the stale deadline, violating "stored
without an expiry ... lives until it is replaced or the cache is closed."
One Set of a short-TTL entry over "pinned" (or a ttl=0 Set over
"session") plus one probe past the old deadline buys the whole clause
inside the lifecycle that already exists.

### MEDIUM — the expiry instant is verified in the report, not in a test

Set's prose: "The entry expires ttl from now; from then on Get misses
it." Under synctest the exact deadline is deterministic and free, yet the
test brackets it with one-minute margins (live at 14m, missed at 16m) and
never lands on it. report.md says the `now >= deadline` boundary was
checked against the source; that check dies with the report, and the
golden hierarchy wants the prose promise held by something committed.
Either pin the instant (prose says the miss begins *at* the deadline) or
surface the author's inference that pinning it would over-harden; today
nothing committed decides it.

### LOW — the example drops Get's ok on the refresh lookup

`tok, _ := c.Get("user-42")` after the refresh, three lines below a
lookup that guards ok properly. In the package's most-copied artifact,
for the one API where handling the miss is the entire idiom, the
inconsistency teaches the misuse. Guard it and print through the same
`if` shape as the first lookup.

### LOW — expiry is invisible in the mandatory example

The package doc leads with "per-entry expiry, intended for short-lived
values", and the example's login/refresh/logout story never shows or
mentions a token dying: as rendered on pkgsite it reads like a map with
Close. Not demonstrating expiry in real time is a defensible flake
trade-off, and the doc comment honestly promises only what the body
shows, so this is judgment, not breakage. But the in-function comments
are prime real estate: one clause on the `time.Hour` TTL ("an hour from
now this lookup starts missing") carries the non-Go contract to users
for free.

### Comment audit (clean)

No test or example doc comment opens with its function's name. The one
name-opening comment is `mustPanic`'s, a plain helper where ordinary
go.dev/doc/comment convention, and the exemplar's own `mustPanic`, demand
exactly that: not a violation. The trivial panic table carries no doc
comment, correctly. No synctest narration, no ordering commentary, no
"pins the contract" filler; the janitor-interval rationale comment and
the "zero and negative TTLs mean no expiry, not instant expiry" call-site
comment are the non-slop kind I want. The panic helper's failure message
names each case's specific invalidity.

## outputs/report.md

### MEDIUM — the concurrency coverage claim oversells

"The synctest scenarios plus `-race` exercise it without a statistical
race hunt": "scenarios" is plural for a suite with exactly one, and
`-race` against a janitor that runs while the test sleeps is thin
evidence for a multi-client promise. Say what was and was not exercised;
as written, a reader signs off believing the promise is covered.

### Good

The no-drift verdict is stated against specific prose clauses, the
janitor-unobservability finding carries the right author-facing inference
("reads as deliberate, so nothing pins them"), the stale-file
housekeeping is disclosed, and every final-status claim reproduced under
my re-run. That is the report shape I want.
