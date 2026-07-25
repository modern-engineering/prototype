# Critique — cache-write-tests / with_skill_looped_opus__r2

Reviewed as the `cache` maintainer, bound to skill-v4 and its exemplar, against
the exported surface from `go doc -all`. Build/vet clean; `go test -race
-count=3` green; the janitor test survived `-count=500`.

The external suite is genuinely good: one mandatory package-level `Example`, one
whole-package scenario test on synctest's virtual clock, and a panic table that
copies the exemplar's `mustPanic` shape faithfully. It exercises the package as
a user and does *not* fall into the statistical race-hunt trap despite "All
methods are safe for concurrent use" — the janitor goroutine running inside the
synctest scenarios is the concurrency coverage, exactly as the doctrine wants.
The damage is concentrated in the internal test and in a couple of prose slips.

Severity-ranked, per file.

## janitor_internal_test.go

### 1. HIGH — a white-box change-detector, sold as a contract test

`TestJanitorReclaimsUnreadExpiredEntries` locks `c.mu` and reads
`len(c.entries)`. That is the implementation, not the contract. The golden
hierarchy is explicit that a test authored from the implementation "mirrors it,
bugs included"; this one is welded to two private symbols and to the locking
discipline. Reshape the janitor (sharded maps, `sync.Map`, a heap, a renamed
field) and the test breaks while every promise in the go-doc still holds. That
is the definition of a change-detector.

The deeper problem is what the test *reveals* and the review *ignored*. The
package promises "A background janitor goroutine removes expired entries every
janitorInterval," yet that promise is invisible through the exported surface:
Get and Len enforce expiry on read, so a swept entry and an unread-expired one
are indistinguishable to any user. A documented behavior that can only be
verified by locking a private map is a testability/design signal — precisely
the "secondary goal is the package" call the skill demands. The right maintainer
move is to flag it to the owner: either the janitor's reclamation does not belong
in user-facing prose (it is an internal memory-management detail), or the package
should expose something observable. Instead the submission silently absorbed the
signal by writing a peephole test. The doc comment argues the peephole is
"required," and it argues it well — but "required to observe a promise" is the
smell, not the justification.

The test's own doc comment is clean (opens with "The janitor's distinct
promise…", not the function name; states maintainer rationale). The defect is
that the test exists in this shape at all without the design flag.

### 5. LOW (PLAUSIBLE) — sleeping onto a janitor tick boundary

The test does `New(time.Second)` then `time.Sleep(11 * time.Second)` and reads
the map. 11s lands exactly on a janitor tick, and the entry's 10s TTL means
removal happens at either the t=10 or t=11 sweep. If the source removes only at
t=11 (a strictly-after expiry comparison), the sleeper's wake and the ticker's
fire are scheduled at the same virtual instant with no ordering guarantee, and
the main goroutine could read `len(entries)` before the sweep runs. Empirically
500 runs pass, so removal is happening at t=10 and this is not a live flake —
but the exemplar deliberately sleeps to *off-boundary* instants (4500ms, 2500ms)
so the woken goroutine is strictly after the event it inspects, independent of
any boundary comparison. Imitate that: sleep to 11.5s. Cheap robustness the
exemplar already models.

## cache_test.go

### 2. MEDIUM — the `Example` comment leaks test-timing reasoning to pkgsite

> `// A short-lived token is a miss once its TTL elapses; time only moves
> forward, so the 10ms wait always clears the 1ms deadline.`

The first clause is a proper user-facing contract statement. The second clause
("time only moves forward, so the 10ms wait always clears the 1ms deadline") is
the author reassuring themselves that the real-time sleep is deterministic. That
is maintainer-facing test-robustness reasoning, and it renders on pkgsite to a
reader who does not know or care that this is a test. It is the same species as
the "explain synctest mechanics" slop the audience doctrine forbids, just moved
into a runnable example. Cut the clause; the "miss once its TTL elapses" half is
what the reader needs. (The 10ms wall-clock sleep itself is a defensible
pragmatic choice — examples cannot capture Output inside a synctest bubble — so
the sleep stays; only the comment leaks.)

### 3. MEDIUM — a documented flow is never exercised: replace resets the TTL

Set's contract: "Set stores value under key, replacing any existing entry. The
entry expires ttl from now." `TestSessionLifecycle` tests replacement only for a
no-expiry entry (`service` at ttl=0 → `rotated` at ttl=0), so the TTL never
transitions on a replace. The user flow that the contract actually describes —
replace a live, soon-to-expire entry with a longer TTL and confirm it now
survives past the original deadline (or replace a no-expiry entry with a TTL and
confirm it now expires) — is untested. This is a flow, not a symbol scan, so it
belongs in the lifecycle test: after the 30s token is set, re-`Set` it with a
fresh TTL and confirm the clock restarted.

### Cleared — `mustPanic` opening with its own name is NOT a violation

Flagging this would be a false positive. The skill's rule scopes to *test*
comments ("no test comment ever opens with the function's name"), and TESTS.md
enumerates "tests, bench, fuzz, example, runnable example" — not helpers.
`mustPanic` is an ordinary unexported function, Go convention wants its doc to
open with the identifier, and the exemplar's own `mustPanic` opens exactly the
same way ("mustPanic fails the test unless New(rate, burst) panics…"). Imitation
is the method; this imitates correctly. No test/example function in the suite
opens its comment with its name — checked all five.

## report.md

### 4. MEDIUM — self-contradiction and no author inference

The report declares: "your prose, signatures, and behavior agree, so the tests
hold you to the doc rather than to the implementation." One of its four tests is
explicitly implementation-bound (finding 1). The report cannot both ship a
lock-and-read-the-private-map test and claim the suite avoids binding to the
implementation.

Second, the maintainer wants non-trivial tests to carry the author's inference —
the deep-in-mind context that belongs in a fine-grained commit body and in the
report's findings ("promise it? refuse to harden?"). The janitor peephole is the
single non-trivial judgment call in this submission, and the report says nothing
about it beyond naming the file: no argument for why the private access is
warranted, no question of whether the reclamation promise is worth guaranteeing
at all. "No drift found" is not the same as "no design signal found" — and there
was a design signal (finding 1).

## Net

Delete or redesign the internal test, or at minimum land it with a commit body
and report note that raises the unobservable-promise design question to the
owner. Fix the two prose slips. The external suite underneath is close to what
the exemplar models.
