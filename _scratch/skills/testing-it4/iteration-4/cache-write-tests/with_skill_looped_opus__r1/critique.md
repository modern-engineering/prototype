# Maintainer critique — `cache` committed tests

Reviewed as the package maintainer, bound to `skill-v4/SKILL.md` and its
`exemplar/`, against the exported surface from `go doc -all` (I did not open
`cache.go`). Suite builds, `go vet` is clean, `go test` and `-race` pass. Four
functions: `Example`, `TestCacheLifecycle`, `TestNewPanicsOnNonPositiveInterval`
(with a `mustPanic` helper), and the white-box `TestJanitorReclaimsExpiredEntries`.

Verdict up front: this is a strong submission that clearly imitated the
exemplar — flows-first names, a mandatory representative `Example`, a synctest
lifecycle, a panic table with the helper placed after its use, and no comment
slop. The known "example doc comment prefixed with its own function name" bug is
absent; every test opens on a scenario. What follows is where it still misses,
ranked. None of it is a compile or logic bug; the two real weaknesses are about
honesty, not mechanics.

---

## `janitor_test.go`

### [HIGH] The one white-box test asserts an implementation detail dressed as a contract

`TestJanitorReclaimsExpiredEntries` locks `c.mu` and reads `len(c.entries)`
(lines 25-30) to prove the sweep "deletes, not merely hides." Read the go-doc
again: expiry is enforced on read, so `Get` and `Len` hide an expired entry
*whether or not it is still in the map*. The user-visible behavior is therefore
**identical** in both worlds. The only genuine stake behind "removes expired
entries" is memory growth over a long-lived cache — and `len(c.entries) == 0`
read through a private field is a poor proxy for that: it measures a map's
length, not memory, and it hard-couples the test to today's storage
representation. Swap the map for a `sync.Map`, shard it, or change the field
name, and this test breaks while the contract holds intact. That is the
change-detector shape the golden hierarchy warns against: a test authored from
the implementation, mirroring it.

The doc comment argues the peephole is the one promise "the public API cannot
show," and the skill does permit a peephole "when required." So this is a
judgment call, not a flat error — the prose *does* say "a background janitor
goroutine removes expired entries." But it is exactly the judgment ruling #6
wants surfaced: *do we promise reclamation and want it hardened, or is the
janitor an implementation note and memory a soak/benchmark concern?* The author
made that call silently and the report calls it "nothing to flag." At minimum
this is the suite's single most fragile, most debatable piece, and it should be
raised with the owner (expose a size metric? downgrade the prose to
best-effort?) rather than committed as settled contract.

Mechanically the test is fine: maintainer-facing doc comment that does not open
with the function name, correct lock discipline, `synctest.Wait()` used to let
the sweep land.

---

## `cache_test.go`

### [MED] TestCacheLifecycle comment claims a proof the assertion cannot make

Lines 86-92: after `time.Sleep(time.Hour)` + `synctest.Wait()`, the test checks
`Get("perm")` still returns `"updated"`, under the comment "The no-expiry entry
survives it, confirming the janitor only reclaims entries that have expired."

It confirms no such thing. `perm` has no expiry, so it survives whether the
janitor ran, ran and correctly spared it, or never ran at all — the assertion is
blind to the difference. There is no positive control in this test that the
janitor deleted anything (that lives only in the white-box test). So the comment
asserts a selectivity the code does not demonstrate. This is the anti-slop rule
inverted: the comment says something confident and untrue rather than saying
nothing. Drop the "confirming..." clause, or accept that the only way to earn it
here is another peephole — which is itself the tell that this property doesn't
belong in a black-box lifecycle test.

### [MED] The headline concurrency promise is never a deliberate target

The type doc's most emphatic sentence is "All methods are safe for concurrent
use by multiple goroutines." No test runs two callers against the cache, and
under synctest the janitor and the main goroutine never actually contend — the
bubble advances virtual time only while the caller is durably blocked (asleep),
so the two never touch the map at the same instant. `go test -race` passing is
therefore near-zero evidence for this clause; the suite exercises it only
incidentally, if at all.

I am tolerant of not hunting races with contrived goroutine lotteries (I said as
much on the limiter, where concurrency was merely omitted from the prose). This
is different: it is an explicit, headline promise for a use case — session
tokens across concurrent HTTP handlers — where multi-caller access *is* the
anticipated flow, not an unrealistic pattern. One representative concurrent flow
is warranted: a handful of goroutines doing `Set`/`Get`, a `sync.WaitGroup`
against leaks, assertions made **inside** the goroutines (`*testing.T` is
concurrent-safe, so no value collection). That is a user flow, not a race hunt.

### [LOW-MED] Stated contract clauses left under-covered

- **Negative TTL is never tested.** The prose says "If ttl is zero or negative,
  the entry is stored without an expiry." Only `ttl == 0` is exercised (lines
  71, 78). The exemplar's panic table deliberately covered both zero and
  negative; the "or negative" branch here is asserted nowhere. One extra case in
  the existing lifecycle costs nothing, and it is exactly the
  "technically-accepted invalid input" category the skill says to cover.
- **The `Example` teaches less contract than it cheaply could.** The runnable
  Example covers set / hit / unknown-key-miss well, but omits the non-obvious
  "ttl ≤ 0 never expires" clause — precisely the fallback-style edge I keep
  asking examples to surface at the call site with a distinct comment (units and
  truncate rounds). One `c.Set("config", "on", 0)` line with a one-clause
  comment would put that footgun in front of pkgsite readers, where it belongs;
  today it lives only inside `TestCacheLifecycle`'s maintainer-only comments.
  Expiry proper can't sit in a runnable Example (no meaningful sleep), so the
  lifecycle test carrying it is right — but the no-expiry case has no such
  excuse.

### Cleared (so these aren't mistaken for misses later)

- `mustPanic`'s doc comment opens with "mustPanic" — **not** a violation. The
  name-first rule is scoped to *test/example functions* (Test, Benchmark, Fuzz,
  Example); a plain helper follows ordinary Go doc convention, and the
  exemplar's own `mustPanic` opens with its name identically. Correct as written.
- No comment narrates conventions, ordering, or synctest mechanics. `Example`,
  `TestCacheLifecycle`, and the white-box test all open on scenario prose, not
  on their symbol. The `New(time.Minute)` argument gets the right kind of
  call-site comment (its meaning is non-obvious), and `defer c.Close()` mirrors
  the exemplar's phrasing.
- File and complexity order is right: `Example` first, then the lifecycle flow,
  then the panic table, helper after first use; the internal test isolated in
  its own `package cache` file. Good soft landing.
- The `Example` is representative (a real session-token story with a deliberate
  unknown-key miss), not an `ExampleNew`-calls-`New` placeholder. Deterministic
  output.

---

## `report.md`

### [MED] Over-claims completeness and carries no author inferences

"Nothing to flag: the code matched its prose on every point I tested" reads as a
completeness claim the suite has not earned. The concurrent-safety clause was
not a point tested; the negative-TTL branch was not tested; and the white-box
test is a live judgment call, not a clean pass. "On every point I tested" is
technically self-protecting, but in a hand-off it misleads — it invites the next
reader to believe the contract is fully pinned when the type's loudest promise
never was.

More to the point of ruling #6: a report should hand the maintainer the author's
*inferences*, and the two judgment calls here (peek-for-reclamation; treat the
concurrency promise as covered) are exactly the deep-context notes that belong
in fine-grained commit bodies, not a flat "nothing to flag." As written, this
suite reads like it would land as a single commit with no contextual body. It
should land as several, with the white-box test's rationale — *why we believe
reclamation is a promise worth hardening* — written where the next maintainer
will find it.

The one genuinely good observation is "Close's synctest requirement also proves
the janitor goroutine is released" — that is real and worth keeping.
