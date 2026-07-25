# Critique: cache-write-tests / with_skill_looped_fable__r3 (regenerated outputs)

Reviewed as the package maintainer, bound to skill-v4 and the package's
go-doc prose only. Verified the report's status claims myself: gofmt
silent, `go vet` clean, `go test -race` passes. The lifecycle timeline
arithmetic is correct end to end (blip deadline at 0.5s, sweeps at 1s..4s,
replacement at 1.5s arming a 3.5s deadline, negative-TTL replacement at
2.5s, Len audited at 4.5s after four sweeps). Two findings from the
previous loop round were fixed: negative TTL is now exercised, and the
post-Close Set is now discriminated through Len instead of a Get that is
contractually blind. Credit taken. What follows is what still stands.

## cache_test.go

### HIGH: the type's loudest promise is still never exercised, and still undisclosed

"All methods are safe for concurrent use by multiple goroutines" sits in
Cache's doc comment, the most sacred tier of the golden hierarchy. Not one
test spawns a user goroutine. The only interleaving the suite ever
produces is incidental: the janitor firing while the test goroutine
sleeps. The concurrency reference's "the synctest scenarios are the
concurrency testing" clause answers docs that OMIT concurrency; these docs
PROMISE it, by name, for every method, and a single-goroutine scenario
does not touch that clause.

This is not a call for statistical race hunting; that stays banned. The
package doc names session tokens, so the anticipated flow is concurrent by
nature: request handlers calling Get while a login calls Set. A synctest
bubble with a handful of goroutines asserting inline (*testing.T is
concurrent-safe) is realistic, cheap, and exactly the flow a user
composes. Worse: the previous round's critique ranked this finding first,
the regeneration fixed the two easier findings below it, and report.md
still does not surface the omission even as a decision declined. A
promised clause with zero deliberate exercise and zero disclosure is the
worst kind of gap: nobody downstream knows it is open.

### MEDIUM: mustPanic's doc comment opens with its function's name (line 135)

"mustPanic fails the test unless New(interval) panics." The skill's rule
is absolute: no test comment ever opens with the function's name. Yes, the
bundled exemplar's mustPanic carries the identical shape; the exemplar's
authority derives from embodying the canon, not from being it, and where
they collide the prose rule wins. Reword ("Fails the test unless
New(interval) panics.") and file the exemplar defect upstream rather than
propagating it.

### MEDIUM: Set's replacement promise is only half-exercised

Set's prose: "Set stores value under key, replacing any existing entry.
The entry expires ttl from now." The lifecycle covers TTL-to-TTL renewal
and TTL-to-immortal demotion, never immortal-to-TTL. A regression that
keeps the old no-expiry state when replacing an entry stored with ttl<=0
sails through green. One more Set on a fourth key (or re-arming blip)
plus one existing sleep closes it; "refresh" itself must stay immortal
because the Close leg leans on it.

### MEDIUM-LOW: the exact-deadline assertion pins an instant no real caller can observe (lines 82-84)

The test sleeps exactly 500ms and asserts blip is already a miss at t ==
deadline. Only virtual time can land on that instant; the assertion
hardens the implementation's comparison-operator choice (>= versus >)
into the suite, and a harmless flip breaks the tests while no real-time
user could tell the difference. The report does raise the boundary with
the right owner framing ("say so explicitly if intended, amend if not"),
which is the doctrine working. But the committed artifact outlives the
report: assert strictly past the deadline and let the exact instant live
as the finding until the owner rules. The in-body comment ("the miss
below is the reader's own doing, not the janitor's") is good and survives
either way.

### LOW: the example never teaches Close idempotence

Close's prose promises "calling it again is safe and does nothing", and
the exemplar's Example_throttle shows the pkgsite-facing way to teach it:
an explicit Close after the deferred one, with a call-site comment. The
lifecycle test exercises the second Close, so maintainers are covered,
but users reading Example_sessionTokens never see it, and in-function
example comments are prime real estate for repeating non-Go contract
parts. Don't cheap out on the example.

### NITS

- The panic loop drops the per-case invalidity naming that helpers.md
  prescribes; "non-positive janitor interval" covers both cases and the
  printed inputs (0s, -1m0s) carry the distinction, so this passes, but
  the exemplar's case-struct shape is the stated standard.
- The lifecycle doc comment's "on one timeline" gestures at the synctest
  machinery, same as last round's "on one clock". It stops short of
  mechanics, so it survives; the sentence loses nothing without it.

## report.md

### MEDIUM-LOW: "no prose/code drift" and a drift-shaped finding in the same breath

The findings paragraph opens "no prose/code drift" and immediately raises
a contested reading of "from then on Get misses it" that the tests now
assert. Pick one: either the boundary is a drift-class ambiguity (it is)
or the file has no findings. The self-contradiction teaches the next
reader to skim. Also "the test now pins that reading": pinning vocabulary
again; say what the user observes. And "(honest: all green, first run)"
leaks the loop's instructions into the artifact; that aside is a tell,
drop it.

## Checked and cleared

- External `package cache_test`; example first, lifecycle next, panic
  machinery last; helper after its first use. Exemplar-faithful order.
- Example and test doc comments hit their audiences: the example speaks to
  users about the scenario, the lifecycle comment gives maintainers the
  story, neither opens with its function's name, no convention or
  ordering narration anywhere.
- Flow-shaped throughout: one representative example of the anticipated
  call pattern, one whole-story scenario, fewer tests covering more. No
  change-detectors, no dev scaffolds, no error-string assertions, no
  global case slices, hard-coded results, trivial straight-line bodies.
- Negative TTL exercised; post-Close Set discriminated via Len with the
  rationale commented; Len-after-Close asserted and its implied-only
  status correctly escalated to the owner in the report.
