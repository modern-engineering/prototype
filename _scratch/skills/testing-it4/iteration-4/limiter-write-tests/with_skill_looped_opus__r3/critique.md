# Maintainer critique — limiter-write-tests / with_skill_looped_opus__r3

Reviewed as the `limiter` package maintainer, blind to `limiter.go`, against
the exported surface (`go doc -all`), the skill, and its exemplar.

## Framing: most of the good here is borrowed

`Example_throttle`, `TestThrottleLifecycle`, `mustPanic`, and the panic table
are near-verbatim clones of the skill exemplar's `bucket_test.go` with
`bucket`→`limiter` renamed. That is the method — imitation is fine, and the
example's user-facing doc comment and in-function call-site comments are
exactly what I have been asking for. Credit where due. But it means the run's
*original* contributions are only three: `TestZeroBurstAdmitsOnlyWaitingCallers`
(good — straight from the package doc's burst-0 clause), the correct removal of
the `2e9` case from the panic table (the exemplar guards `rate>1e9`; this
package does not), and the bug test plus its report. The original work is
where this falls apart. Judge it there.

---

## limiter_test.go

### 1. (HIGHEST — refuse) The bug "test" never runs the package and can never pass

`TestNewRejectsRateThatFloorsRefillIntervalToZero` (lines 172-174) is a single
unconditional `t.Errorf(...)`. It never calls `New`, never exercises anything.
`go test` confirms: it fails 100% of the time regardless of the code. This is
not a test — it is a printed note wearing a test's clothes.

The report (report.md:5) says "add a `rate > 1e9` guard in `New` and it flips
green." That is false. Add the guard, delete the guard, rewrite `New`
entirely — this function still fails, because it asserts nothing about `New`.
A test that cannot validate the fix it demands is worthless as a regression
gate and actively misleading.

I *do* want the zero-ticker rate tested — it is invalid input the package
technically accepts, and surfacing it in the report was the right instinct.
But the correct shape is a real assertion that passes once the guard lands:
the huge-rate case belongs in the `New`-rejection table like the exemplar has
it. The genuine obstacle — calling `New(2e9,1)` crashes the whole suite via an
async `time.NewTicker(0)` panic in the refill goroutine, which no `recover`
can catch — is real, and it is exactly the case for "comment it out to see the
work through, restore at hand-off with a clear comment." A hard-coded
`t.Errorf` is not that. It is the lazy substitute.

### 2. The bug test's doc comment narrates the test itself — prompt-leak slop

The doc comment (lines 164-171) ends: "The test fails on purpose so the gap
stays visible: actually calling New would crash the suite, so it asserts the
contract we want rather than run the crash we have." That describes what the
test *function does* and why it is written the way it is — meta-narration
aimed at explaining the mechanism, the loser tell-sign of an agent leaking its
own reasoning into the source. A maintainer-facing rationale states the bug
and the author's judgment about it; it does not apologize for the test's
construction. Cut the last two sentences. The first half (the mechanism of the
floor-to-zero bug) is legitimate and can stay.

### 3. `elapsed != 900*time.Millisecond` pins a phase the prose never promises

Lines 83-85 assert the patient `Wait` returns after *exactly* 900ms. The
package doc promises only "refills at rate tokens per second" — nothing about
when the first tick lands relative to `New`, i.e. the refill *phase*. The 900ms
figure is derived from the exemplar's implementation (ticker started at `New`,
100ms already burned), not from this package's contract. Working go-doc-first,
you cannot know the phase; you imported it wholesale with the clone. Under
synctest it happens to be deterministic, and I wrote this exact line myself in
the exemplar, so I will not die on it — but understand that the assertion is
borrowed implementation knowledge, not a contract you read from the docs. If
the limiter's phase differs from the bucket's, this passes for the wrong
reason or fails for a reason the prose does not cover.

### 4. A top-level test is stranded behind the helper, severed from its table

File order runs: `Example_throttle`, `TestThrottleLifecycle`,
`TestZeroBurst...`, `TestNewPanicsOnInvalidArguments`, then the `mustPanic`
helper (151-162), then `TestNewRejectsRateThatFloorsRefillIntervalToZero`
(164-174). The huge-rate rejection is the *same concern* as the panic table
directly above the helper — it is a `New`-rejects-bad-rate case. Splitting it
off to the far side of the helper breaks "build complexity as we read" and
divorces it from the table it logically extends. If it survives at all (see
finding 1), it belongs as a row in / beside `TestNewPanicsOnInvalidArguments`,
not orphaned at the bottom.

---

## report.md

### 5. The finding is stated as a mechanical fix, with no author's judgment

report.md:5 reports the bug and prescribes "add a `rate > 1e9` guard... and it
flips green." That is the mechanical voice. The value I want from a non-trivial
finding is the author's inference: *do we actually want to promise rejecting
rate > 1e9, or is that hardening a contract irrelevant to any scenario a real
caller hits?* The exemplar's package chose to promise it ("between 1 and 1e9");
this package's prose is silent. Deciding whether to match that promise is the
package author's call, and that deliberation is precisely the deep context
that belongs in a fine-grained commit body for this one test — not a flat
"add the guard." A non-trivial, contract-shaping test like this should land as
its own commit carrying that reasoning; the report shows none of it.

---

## What I would accept as-is

- `Example_throttle`: audience is right, call pattern (Wait-after-Allow-false)
  is right, in-function comments repeat the non-Go contract at call sites. Good.
- `TestThrottleLifecycle`: one flow, end to end, no per-symbol scanning. Good.
- `TestZeroBurstAdmitsOnlyWaitingCallers`: drawn straight from the burst-0
  clause in the package doc; the "refill dropped, not banked" assertion is the
  right flow. Good — this is the one original test that lands.
- Removing the `2e9` row from the panic table: correct read of the contract
  gap between the exemplar and this package.

No goroutines leak (nothing spawns outside synctest or an un-deferred `Close`);
no error-string pinning; no statistical race-hunting. The structural hygiene is
fine. The failure is concentrated entirely in how the discovered bug was
turned into a committed artifact.
