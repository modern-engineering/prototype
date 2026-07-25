# Limiter tests: engineer's note

Wrote `limiter_test.go` as external `package limiter_test`, planned from the go-doc prose before reading bodies. The suite:

- A runnable gateway example: Allow on the hot path, Wait as the fallback, idempotent Close noted at the explicit call site.
- A synctest whole-package lifecycle scenario: opening burst, deadline-exceeded wait, exact 900ms refill wait, a lull capped at burst, ErrClosed beating leftover tokens.
- A burst-0 test for the package doc's stores-no-tokens paragraph: unattended refills evaporate, a parked waiter is served in exactly 500ms.
- A panic table with a mustPanic helper, covering only the panics the doc promises.
- `TestExtremeRateYieldsWorkingLimiter`, which fails today. See the finding below.

## Finding: doc-valid extreme rates crash the process

New's doc promises: "New panics if rate is not positive or if burst is negative." Nothing else is carved out, so a rate like 5e9 is doc-valid and must yield a working limiter. Today it does not: any rate above 1e9 floors the refill interval to zero, and `time.NewTicker(0)` panics inside the refill goroutine, crashing the whole process (verified with a throwaway program). Siding with the prose, `TestExtremeRateYieldsWorkingLimiter` asserts the promised behavior and is committed failing; it crashes `go test` deterministically. The case was parked during development to see the rest of the work through and is restored at hand-off. Your call on the fix: floor the interval at 1ns and keep the doc as is, or cap rate with a doc'd panic and amend the prose plus the test. Do not bless the panic silently; the doc as written makes no such promise.

## Finding: the constructor's bare ints are a package smell

Every `New` call site in the suite needed a translation: "Two tokens per second, and up to two held for a burst", "one token per second, two to start", and the burst-0 site leans on its test's doc comment for the same job. Tests repeatedly translating an API value mean production users need the translation too. Consider a rethink of the signature; a frequency- or interval-flavored constructor (`Every(time.Second)`, or a `rate.Limit`-style type) would erase all three comments.

## Questions for the author

- After Close, Wait refuses with ErrClosed but Allow keeps draining leftover tokens. The docs are silent on this asymmetry, so I pinned nothing. Promise it, or seal Allow too?
- Wait's doc says "every subsequent Wait call returns ErrClosed", which pointedly excludes a waiter already parked when Close lands. For a limiter whose typical caller falls back to Wait, that is the more user-visible open question. Released with ErrClosed, left to its context, or refused as a promise? The suite pins nothing there either.

Also unpinned: no concurrency the docs do not promise.

## Final status

- `gofmt -l` clean, `go vet ./...` clean, `go build ./...` passes.
- `go test ./...` FAILS, by design: `TestExtremeRateYieldsWorkingLimiter` crashes the process (the surfaced drift), reproduced 3/3 runs. Every other test passes before the crash lands.
- Excluding that one test, the suite is race-clean over `-race -count=5` and stable at GOMAXPROCS=1.

## Critique response (maintainer review, this round)

- Applied 1 (BLOCKER, parked case dead at hand-off): the commented-out case is gone; the input class is live again as its own test. A crash is a valid test failure, and the red suite is the honest hand-off shape.
- Applied 2 (BLOCKER, transplanted row pinning an unpromised panic): the restored form asserts the prose (a working limiter), not a panic this package never promised. The exemplar's `2_000_000_000` value and "one token per nanosecond" phrasing are dropped for a value and rationale derived from this doc alone, and the implementation internals plus the remediation menu now live only here and in the commit body, not in source comments.
- Applied 3 (NIT, zero-burst doc comment): trimmed to the scenario rationale; the in-body comments carry the phases.
- Applied 4 (MAJOR, unflagged package smell): the constructor finding above.
- Applied 5 (MODERATE, sharper prose silence): the Close-versus-parked-Wait question above.
- Applied minor: the drift finding now quotes New's panic promise precisely, including negative burst.
- Rejected: nothing. Every point traced back to the skill's own doctrine or the package's prose; none conflicted with the prose, so nothing earned a refusal.
