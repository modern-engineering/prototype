# Critique — limiter tests (with_skill_looped_opus__r1)

Reviewed as the `limiter` package maintainer, bound to `skill-v4/SKILL.md`, its
exemplar, and the package's exported prose (`go doc -all`). Source files other
than `_test.go` were not opened.

## Verdict

Would not merge as-is. The suite is competent Go on its surface — external
`limiter_test` package, `synctest` scenarios, exact virtual-time asserts, a
panic-recovery helper, a mandatory throttle example with the exact
Allow-then-Wait call pattern I keep asking for. But almost none of that is this
run's work: the file is my own `bucket_test.go` with `s/bucket/limiter/`, and
the two genuinely original edits both misfire. The single worst artifact — the
commented-out rate case — violates three separate doctrines at once.

## Exported contract (what the tests are held to)

`New` prose: "New panics if rate is not positive or if burst is negative." No
upper rate bound. Package doc: burst-0 stores no tokens, Allow always false,
Wait served only while already blocked. `Close` idempotent. `Wait` returns
`ErrClosed` after close, `ctx.Err()` on ctx-done.

---

## Findings (ranked by severity)

### 1. [HIGH — refuse] The committed test leaks the report and the authoring process

`limiter_test.go:152-159`. The withheld panic case carries a seven-line comment
that narrates a finding and, worse, points the reader at **`report.md`** — an
ephemeral hand-off artifact that does not exist in the repository. A committed
`_test.go` speaks only to maintainers and only about the contract. The
author-context here ("reinstate an upper-bound guard or document the ceiling")
is exactly the deep-in-mind rationale that belongs in a **commit body**, not
stapled into a struct literal with a cross-reference to a file the repo never
sees. This is the "LLM agent leaking its prompt" tell the skill's anti-slop rule
names outright: "IF YOU HAVE NOTHING GOOD TO SAY THEN DON'T SAY NOTHING." A
tight maintainer line ("large rates floor the interval to zero and crash the
refill goroutine — see commit message") would be defensible; a report pointer is
not.

### 2. [HIGH] Authored exemplar-first, not go-doc-first

`limiter_test.go:151-159`. limiter's `New` prose promises a panic only for
non-positive rate and negative burst; it defines **no upper bound and never
mentions 1e9**. A plan drafted from `go doc -all` alone — the skill's central
method, its whole reason to exist — could not have produced a
`{rate: 2_000_000_000, ...}` case or the "any rate above 1e9" threshold in the
comment. Both are lifted verbatim from the exemplar, whose `New` doc *did*
promise a 1-to-1e9 ceiling. The suite was copied and then patched, which inverts
the process the skill enforces. The withheld case is therefore not a drift
"surfaced" from the prose; it is a leftover from an implementation-shaped table
that limiter's contract never justified.

### 3. [MEDIUM-HIGH] Near-verbatim reproduction of the bundled exemplar

`limiter_test.go:17-108,143-177`. `Example_throttle`, `TestThrottleLifecycle`,
`TestNewPanicsOnInvalidArguments`, and `mustPanic` are the exemplar's functions
under a mechanical `bucket`→`limiter` rename — down to `mustPanic`'s comment
"reached only if the guard is broken" (line 176), a "guard" concept limiter's
doc no longer has. The exemplar is a teaching artifact whose authority, per the
intent corpus, "is not from being canon": imitation means a file that reads as
though it grew here, not a photocopy. Only `TestZeroBurstAdmitsOnlyWhileWaiting`
(lines 114-141) shows independent reasoning from limiter's own prose — and it is
the one solid contribution: all three of its assertions trace directly to the
burst-0 paragraph in the package doc.

### 4. [MEDIUM] The invalid input the skill names is left untested

`limiter_test.go:159`. The skill explicitly wants "technically-accepted invalid
inputs (zero-ticker rate)" exercised. This run comments that case out and defers
the decision instead. Granting the honest constraint — a case that crashes the
refill goroutine off a goroutine no caller can recover cannot ship as a passing
test — the committed suite still exercises **no large-rate behavior whatsoever**
and resolves nothing. The correct move is to ask the author, then either pin the
observed behavior once a guard exists or document the ceiling; parking it in a
comment is a half-finished hand-off, not a decision.

### 5. [MEDIUM] Secondary-goal miss: the contract under test has a prose defect

`go doc` output. `New`'s exported documentation reads "The bucket starts full"
for a type named `Limiter` in package `limiter` — copy residue bleeding through
the exported prose. The skill's secondary review goal is the package, and its
most sacred input is the prose. A go-doc-first author who actually drafted from
`go doc -all` would have tripped over "bucket" in a limiter's own contract.
`report.md` is silent on it. This reinforces finding 3: the copy went both ways,
and nobody read the prose closely enough to notice.

---

## report.md

Terse and mostly on-message — no "pinning" language, and the finding does carry
the author-facing inference the skill wants ("reinstate an upper-bound guard or
document the ceiling"). Two problems. First, that inference lives here instead of
in a commit body, and the source file points *back* at this report (finding 1).
Second, the report presents the withheld case as a discovered drift; per the
contract it is a foreign case the exemplar dragged in (finding 2). No evidence of
fine-grained commits carrying the non-trivial tests, which the intent corpus asks
for; not assessable from an outputs directory, noted only.

## What is genuinely good (so it survives a rewrite)

- The Allow-then-Wait throttle example with input-explaining call-site comments
  and the idempotent-Close demonstration is exactly the mandatory representative
  example, and its comments address users on pkgsite, not maintainers.
- `TestZeroBurstAdmitsOnlyWhileWaiting` is original, prose-derived, and correct.
- No statistical race-hunting, no leaked goroutines, no per-symbol grain — the
  concurrency and structure smells from earlier iterations are absent.
