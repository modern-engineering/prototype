# limiter tests

`limiter_test.go` is an external `limiter_test` package:

- `Example_throttle`: the representative example, the Allow-false-then-Wait
  gateway pattern with deferred Close.
- `TestThrottleLifecycle`: one synctest flow through the typical client
  story: starting burst, deadline-expired Wait, refill-served Wait at exact
  virtual times, a lull capped at burst, Close winning over leftover tokens.
- `TestZeroBurstStoresNoTokens`: the burst-0 flow. Polling never succeeds,
  unclaimed refills evaporate, and a blocked caller is served only by a
  refill that arrives after it started waiting.
- `TestNewPanicsOnInvalidArguments`: the panic table, including one
  deliberately failing row (finding 1).

## Method

The file imitates the skill's bundled exemplar closely. This package is
structurally the exemplar's twin, so most lines match the exemplar with the
receiver renamed; the comment prose and failure strings are transplanted,
not reconstructed from `go doc`. Each transplanted assertion was checked
against this package's own published prose before keeping it: the shutdown
assertion rides on Wait's "Once Close has returned, every subsequent Wait
call returns ErrClosed", "the bucket starts full" is New's own sentence,
and the lull cap is the Limiter doc's "up to a maximum of burst". The two
original pieces are the burst-0 test and the 2e9 panic row.

## Findings for you (the author)

1. Rate drift, unresolved. New's docs accept any positive rate, but a rate
   above 1e9 floors `time.Second / rate` to zero and `time.NewTicker`
   panics inside the refill goroutine, crashing the process where no caller
   can recover. The `rate: 2_000_000_000` table row fails deliberately; its
   comment states the drift and stops at the question. Guard it in New
   (panic like the other invalid arguments, and amend the doc), or support
   such rates via a different refill scheme?
2. The rate parameter itself. Five comments in this one test file translate
   the integer rate into refill cadence ("two tokens per second", "the
   first refill lands at the 1s mark", "due at the 3s mark 500ms from
   now"). If the tests need the translation, production users need it too.
   A duration-shaped constructor, say `Every(time.Second)` plus a burst,
   would need none of those comments and would dissolve finding 1 with it,
   since a validated positive duration cannot floor to zero the way the
   integer division can. Either rework the parameter, or keep the integer
   and record why (perhaps requests-per-second is the number your callers
   already hold); if it stays, finding 1 still needs its own ruling.
3. Concurrency is promised nowhere. The docs never say whether one Limiter
   is safe for concurrent goroutines, and a gateway limiter surely serves
   many. Either promise it explicitly on the type, or decide the omission
   is deliberate and record why. The suite exercises the documented flows
   and passes under `-race`; it does not hunt for races beyond that.
4. Allow after Close still drains leftover tokens. The docs neither promise
   nor forbid it, so no test asserts it. Same fork: promise it, or leave it
   unspecified on purpose.

## Critique disposition (round 2 review, applied in this revision)

- Applied 1 (the 2e9 row pre-committed the ruling): the row's comment now
  states the observed drift, carries the breadcrumb from the raw
  "panic: non-positive interval for NewTicker" crash back to the row, and
  ends at the open question instead of "New itself must reject the rate".
  The row still fails deliberately at hand-off.
- Applied 2 (provenance-narrating doc comment): dropped "The package doc
  singles out burst 0:" and kept only the rationale.
- Applied 3 (the report misrepresented the method): the Method section
  above replaces the "from the go-doc prose" claim with what actually
  happened.
- Applied 4 (missed the secondary review goal): finding 2 now flags the
  rate parameter for a rethink.
- Applied 5 (pinning vocabulary; findings without the fork): descriptions
  rewritten as flows, and findings 2-4 hand you both branches.
- Rejected: nothing. No critique point conflicted with the package's
  published prose.

## Status

- `gofmt -l` clean, `go vet ./...` clean, `go build ./...` passes.
- `go test ./...` fails deliberately on the 2e9 panic-table row: the refill
  goroutine's NewTicker panic kills the binary with no test-name
  attribution, exactly as the row's comment describes. Panics are valid
  test failures; the row stays until finding 1 is ruled on.
- Everything else passes under `-race`: the example and both synctest flows
  verified in the committed tree, and the three passing panic rows verified
  in a scratch copy with the deliberate row removed (not committed).
