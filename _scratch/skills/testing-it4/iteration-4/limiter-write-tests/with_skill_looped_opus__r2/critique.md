# Maintainer critique — limiter-write-tests / with_skill_looped_opus__r2

Reviewed as the `limiter` package maintainer, bound to SKILL.md v4 + exemplar and
to the package prose (`go doc -all`). Read: `outputs/limiter_test.go`, `report.md`.
Did not open `limiter.go`.

Verdict up front: the bones are right — external `limiter_test`, a mandatory
`Example_throttle` with user-facing in-function comments, a synctest lifecycle
flow, least-orchestration (no gratuitous goroutines), a well-prose-derived
burst-0 test. But it ships two things I will not merge: banned "pinning" slop,
and committed comments that fabricate a repo neighbor and point at an ephemeral
hand-off file. And the whole file is my exemplar transcribed, which is how those
defects rode in.

Ranked, most severe first.

## limiter_test.go

### 1. (HIGH) "This pins that promise." — banned anti-slop comment (line 112)
The doc comment on `TestBurstZeroBanksNothing` ends "This pins that promise."
Every committed test pins a promise; saying so says nothing. I have rejected
this exact phrasing before ("DO NOT tell me that the test pins some contract
because that's true for all tests"; "IF YOU HAVE NOTHING GOOD TO SAY THEN DON'T
SAY NOTHING"). The first two sentences of that comment are good — they explain
why burst-0 is its own mode. Cut the last sentence. This alone is a refuse.

### 2. (HIGH) Committed comment fabricates a "sibling" and points at report.md (lines 154-158)
The commented-out panic case carries: "New should reject such a rate up front
the way **the package's sibling** does ... The case is kept here, commented, so
the fix has a home; **see report.md**."

Two non-repo references baked into permanent source:
- `go doc -all` shows `limiter` standing alone. There is no documented "sibling"
  package. That neighbor is the skill's `bucket` exemplar leaking into my
  package's committed record. Per the corpus, the exemplar is a style model, not
  canon and not part of my repo ("You must not learn from my code any more than
  you should from theirs"). A test comment must not assert a repo topology that
  does not exist.
- `report.md` is your hand-off artifact. It will never live in the repo. A source
  comment that says "see report.md" is the delivery process leaking into the
  code — the same disease as echoing the report/guidelines into comments, which
  I have already called out.

Rewrite the comment to speak only about `limiter`'s own contract, or drop it.

### 3. (MEDIUM-HIGH) The invalid input I explicitly asked to be tested ships disabled (lines 149-158)
The large-rate / zero-ticker case is commented out, so the delivered suite does
not exercise it — the one input the contract's silence makes most interesting.
Your analysis is actually correct: `New` returns normally, the panic fires in the
detached refill goroutine, and `mustPanic`'s recover cannot catch it, so the case
cannot be asserted in place. That is precisely a "test that proves hard to write
is a signal to redesign": the right artifact is a crisp finding that `New` must
validate `rate` synchronously (so a normal `mustPanic` row would then work),
not a dead table row justified by a phantom sibling and a pointer to report.md.
The skill's own guidance ("comment out to see the work through, restore at
hand-off, never drop silently") assumes a case that becomes runnable again; this
one never will until the API changes, so leaving it as a corpse is the wrong
shape. State the redesign ask; don't ship the commented row.

### 4. (MEDIUM) The file is the exemplar transcribed, not derived from limiter's prose
`Example_throttle`, `TestThrottleLifecycle`, `mustPanic`, and the panic table are
`exemplar/bucket_test.go` with `bucket`→`limiter` and `b`→`l`, byte for byte
(verified by diff). Only `TestBurstZeroBanksNothing` is genuinely written from
`limiter`'s own doc — and it is the strongest thing here, which proves the point.
Imitation of *style* is the method; wholesale copy is not. The tell is exactly
where the two contracts diverge: `bucket`'s doc promises a synchronous upper
guard ("rate ... between 1 and 1e9"), so its active 1e9 panic row is legitimate;
`limiter`'s doc promises only "not positive," so the transcribed row had to be
disabled and papered over with the invented "sibling." Go-doc-first was supposed
to surface this divergence cleanly; instead it was half-surfaced and half-hidden.
Re-derive the lifecycle from `limiter`'s prose so the file owns its own contract.

## report.md

### 5. (MEDIUM) The finding misreads the contract it cites
"`New` dropped the upper-rate guard your doc implies ('panics if rate is not
positive')." The doc implies no such guard. "Not positive" is `<= 0`; `2e9` is
positive, so the prose is simply *silent* on large rates — it does not imply a
guard and none was "dropped." The real defect is sound (silent acceptance leading
to an uncatchable async panic), but the stated premise is wrong, and the
recommendation is anchored to "the sibling" (the exemplar) rather than to
`limiter`'s own promises. State it as: the doc is silent here, the code accepts
the value and then crashes off-goroutine, so either the prose should bless the
value or `New` should reject it synchronously — and here is my inclination and
why. That authorial inference is what belongs in the finding (and in the commit
body), not a cross-reference to a package that is not in this repo.

## Credit where due (so the above is calibrated, not reflexive)
- External `limiter_test`; user seat enforced at compile time.
- `Example_throttle` is the mandatory representative flow (Allow on the hot path,
  Wait when dry), with in-function comments aimed at pkgsite readers, not
  maintainers — the audience doctrine is respected here.
- synctest lifecycle with least orchestration; no statistical race-hunting; no
  goroutine the main goroutine could run itself.
- `TestBurstZeroBanksNothing` is derived straight from the package doc's burst-0
  clause and asserts the exact-1s rendezvous; only its trailing "pins" clause is slop.
- Non-trivial tests carry rationale doc comments; the trivial `mustPanic`/panic
  table does not over-comment. The doc-comment discipline is right this round.
