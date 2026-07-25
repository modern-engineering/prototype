# Maintainer critique — cache tests (with_skill_looped_opus__r3)

Reviewed as the package maintainer, bound to `skill-v4/SKILL.md` and its
exemplar, against the prose from `go doc -all` (I did not open the non-test
`.go` sources). Blunt, per file, severity-ranked.

Verdict up front: this is a strong submission, close to commit-ready. The
exemplar imitation is real, not cosmetic. It sits in an external `cache_test`
package, tells a user-flow story instead of scanning symbols, ships the
mandatory runnable example, keeps its one internal peek honestly scoped, and
shapes the panic table exactly like `mustPanic`. `TestExpiryLifecycle` is the
headline: the re-issue at t=10m, checked alive at t=16m (past the *original*
deadline) and dead at t=26m (past the *refreshed* one), genuinely
discriminates "Set refreshes the deadline on replace" from "Set keeps the old
deadline." That is the hardest promise in the contract and the test actually
bites on it. Comment slop is absent, which most prior runs failed at. The
findings below are the gap between "good" and "what I'd commit."

Comment-rule audit (task-mandated): no test/example/bench/fuzz doc comment
opens with its own name. `Example_sessionTokens` ("A session server is..."),
`TestExpiryLifecycle` ("The whole session-token story..."),
`TestJanitorReclaimsExpiredEntries` ("Because expiry is enforced on read..."),
`TestNewPanicsOnNonPositiveInterval` (no comment, correctly trivial). Clean.
`mustPanic` (cache_test.go:135) opens with "mustPanic" — checked deliberately,
NOT a violation: the rule scopes to test/bench/fuzz/example functions, which
must not mimic classic Go doc style; `mustPanic` is an ordinary helper, and
name-first is correct Go convention. The exemplar writes it identically. A
mechanical critic flags this; a maintainer clears it.

---

## cache_test.go

### [MEDIUM] The negative-TTL no-expiry path is never exercised.

`Set`'s prose enumerates the branch explicitly: "If ttl is zero **or
negative**, the entry is stored without an expiry." Both the example
(`service-key`, ttl `0`, line 28) and the lifecycle (`service`, ttl `0`, line
63) only ever use `0`. The negative half of a two-value documented branch is
untested. This is the same gap the maintainer flagged on units-add-test, where
a distinct line was wanted for each documented fallback. The clean fix is one
more example line — a `Set(..., -1)` with a distinct comment that a negative
TTL behaves identically to zero — which also puts the corner where users read
it. Right now nothing holds the package to the "or negative" clause; a bug that
treated negative TTL as an immediate expiry would pass this suite.

### [LOW] The mandatory example ends at the miss, not the read-through loop.

`Example_sessionTokens` shows Set, hit, and miss, then prints "please sign
in". The archetypal cache call pattern is read-through: miss -> populate ->
hit. The maintainer was emphatic on the limiter that the mandatory example
must "mimic the call pattern we expect to see in real client" (Wait after
Allow-false), not tour the methods. For the *session-store* framing the
package doc chose, sign-in-on-miss is a defensible flow (auth issues the token
elsewhere, then Sets it), so this is a soft note, not a hard miss. But the loop
that makes a cache a cache is left implicit.

### [LOW] The stateful lifecycle uses no `t.Fatalf` for its preconditions.

Every assertion in `TestExpiryLifecycle` is non-fatal (`t.Error`/`t.Errorf`).
The exemplar's `TestThrottleLifecycle` deliberately mixes `t.Fatalf` where
continuation is meaningless (the Wait deadline checks, whose result the next
step's timing math depends on) with `t.Error` for independent observations.
Here the timeline is stateful: if the opening `Set`/`Get`/`Len` block (lines
62-73) is broken, the later checkpoints spray follow-on failures that share one
root cause. Each checkpoint reads mostly independent state, so the cascade is
mild rather than catastrophic — hence LOW — but a `t.Fatalf` on the "both
entries live" precondition would keep a failure report legible.

Positives worth keeping: file order builds complexity (example -> lifecycle ->
panic table -> helper); `mustPanic` sits after its first use; the panic case
struct names its fields; the `New(time.Minute)` comment (lines 16-17) correctly
disambiguates a genuinely confusable literal — sweep interval, not default TTL;
the idempotency demo on the explicit `c.Close()` (lines 42-44) speaks to users
at the call site exactly as the exemplar does. All aligned with the skill.

---

## janitor_test.go

### [LOW-MEDIUM] The peephole hardens an externally-invisible detail; the design signal was not surfaced.

The internal test reaches into `c.mu`/`c.entries` to prove the janitor
*physically* deletes, justified in its doc comment: read-time expiry already
hides the entry, so "the janitor's real contribution, reclaiming that entry's
memory, is invisible through the exported API." The peephole itself is
legitimate — the skill permits it "when required," and periodic removal *is* in
the prose ("removes expired entries every janitorInterval") — and the test
mechanics are correct and deterministic (entry expires at t=10s, sweep at t=60s,
`len(entries)==0` after a 61s sleep).

But this is exactly the secondary-goal moment the skill asks for ("testability
is a design signal"; "the secondary goal is the package"). A promise no user
can observe through the API is a candidate to either drop from the doc (do
users depend on the *interval*, or only on "expired entries are misses"?) or
expose via a hook. A maintainer-grade review states a lean and hands it back —
"keeping the interval promise because unbounded memory is a real concern," or
the opposite. The report instead declares the peephole "necessary" and moves
on. That is the mechanical move, one notch short of the author inference the
skill wants carried on a non-trivial test.

---

## report.md

### [LOW-MEDIUM] The concurrency justification is factually loose.

`go doc` affirms "All methods are safe for concurrent use by multiple
goroutines." The report defends shipping no concurrent scenario with: "the
janitor runs concurrently with the caller **in every synctest bubble**, so
`go test -race` exercises the lock." That is not true of the bubble that
matters. In `TestExpiryLifecycle` the janitor is set to `time.Hour` and the
test runs to virtual t=26m, so the janitor never fires and never touches the
map — only the main goroutine does. In `janitor_test` the single sweep lands
while the caller is parked in `Sleep`; the two never overlap. So `-race`
observes lock-ordered, non-overlapping accesses and proves little about
contention.

To be clear about what is NOT the defect: declining an explicit multi-caller
race harness is *correct* per revealed doctrine — do not hunt races
statistically, and suite-wide synctest scenarios are the concurrency testing.
The report even offers the explicit harness on request, which is the right
posture. The defect is narrow: don't state a mechanism ("runs concurrently in
every bubble") that the scenarios don't deliver. Either lead with the honest
"the affirmed concurrency promise is not directly exercised; here's why that's
an accepted call," or make one bubble actually interleave a caller with a sweep.

### [LOW] Findings ship as a "no drift" bundle without per-finding author inference.

The skill (feedback point 6) wants non-trivial findings carried with the
author's inference — "promise it, or refuse to harden?" — into
commit-shaped context. There are two live judgment calls here that warranted
it: the concurrency skip and the janitor peephole (the observability question
above). The report resolves both by assertion. "No drift" is accurate against
the prose I can see; it is not the same as surfacing the two calls back to the
owner.

---

## Considered and cleared (not findings)

- The refresh-on-replace claim IS exercised and discriminating (the strongest
  part of the suite); no complaint.
- Two-file split (external `cache_test` + internal `cache`) is forced by the
  package clauses; correct, and the report names it.
- Panic table with two cases plus `mustPanic`: two cases still earn the recover
  helper, blessed by helpers.md; placement after first use is correct.
- Triple `Close()` in the lifecycle is deliberate idempotency exercise, not
  redundancy.
- Dense single-sentence scenario doc comments on the flow tests match the
  exemplar's `TestThrottleLifecycle` shape; exemplar-blessed.
