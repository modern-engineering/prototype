# Critique: `limiter_test.go` (with_skill_looped_sonnet__r2)

Reviewed against `skill-v4/SKILL.md`, its bundled `exemplar/bucket_test.go`, and the
package's own prose (`go doc -all` output, reproduced below for reference). No
non-test source file was opened; the one behavioral claim below was verified by
running the built package through a throwaway external program, not by reading
`limiter.go`.

```
package limiter // import "example.invalid/limiter"

Package limiter provides a token-bucket rate limiter driven by a background
refill goroutine.

A limiter created with burst 0 stores no tokens: Allow always reports false, and
Wait proceeds only when a refill arrives while the caller is already waiting.

var ErrClosed = errors.New("limiter: closed")

type Limiter struct{ ... }
func New(rate, burst int) *Limiter
    New panics if rate is not positive or if burst is negative.
func (l *Limiter) Allow() bool
func (l *Limiter) Close()
func (l *Limiter) Wait(ctx context.Context) error
```

Findings, most severe first.

## 1. A known, reproduced process-crash is left untested — and the author already knew the right call

`report.md`'s "Findings" section states the doc's "rate is not positive" panic
guard has a gap: a `rate` at or above roughly `1e9` makes
`time.Second/time.Duration(rate)` floor to `0`, and `time.NewTicker(0)` then
panics **inside the background refill goroutine**, which the caller cannot
recover from — it takes the whole process down. I reproduced this independently
(external program importing the built package via a `replace` directive, not by
reading `limiter.go`):

```
$ go run .   # calls limiter.New(2_000_000_000, 1) with a recover() in main
New succeeded without panic: true
panic: non-positive interval for NewTicker
...
created by example.invalid/limiter.New in goroutine 1
```

`main`'s `recover()` does nothing — the panic is on a different goroutine, exactly
as reported. This is real and it's ugly.

The suite ships with this drift merely narrated in `report.md` ("happy to add it
if you want it before review") rather than tested, guarded, or even asked as a
blocking question. That would be a defensible call for *novel* drift, but it
isn't novel here: the maintainer's own review corpus already settled this exact
question for this exact package (`feedback-3.json`,
`limiter-write-tests--fable-with_skill`): *"a large enough rate gives a zero
ticker, but let it slide untested. **Definitely test this!** It's part of the
invalid input, even if the package technically accepts it."* SKILL.md's golden
hierarchy backs this up too: "Drift is a finding, never silently absorbed: ask,
else side with prose." Parking a known crash bug in report prose, again, on the
same package that was already corrected on this point, is not something I'd
merge without at least a `t.Fatalf`-worthy regression test or an explicit
sign-off before the commit lands.

## 2. `TestZeroBurstAdmitsOnlyLiveRefills` spawns a goroutine to do what the caller could do directly — for a method literally named `Wait`

```go
errc := make(chan error, 1)
go func() {
    errc <- l.Wait(t.Context())
}()
synctest.Wait()
time.Sleep(time.Second)
if err := <-errc; err != nil {
    t.Fatalf("Wait() = %v, want nil once a live refill lands", err)
}
```
(lines 131–139)

SKILL.md is explicit: "never spawn a goroutine whose only job the main goroutine
could do directly (**waiting on a goroutine that calls Wait is calling Wait**)."
That sentence describes this code almost word for word. Inside the `synctest`
bubble, the main test goroutine blocking on `l.Wait(t.Context())` is itself a
durable block that lets virtual time advance; the ticker's send only succeeds
via synchronous rendezvous with whatever goroutine is parked on `<-l.tokens`
(that's the whole point of the "caller already waiting" promise being tested).
There is nothing the spawned goroutine buys here — replacing the whole block
with

```go
if err := l.Wait(t.Context()); err != nil {
    t.Fatalf("Wait() = %v, want nil once a live refill lands", err)
}
```

exercises the identical property. The extra goroutine, channel, `synctest.Wait()`
call, and a second `time.Sleep` are pure orchestration overhead in a test suite
whose skill explicitly asks for "the least orchestration."

## 3. The test also collects the goroutine's result through a channel instead of asserting inside it

Compounding #2: even granting the goroutine, TESTS.md/the intent corpus are
explicit that "*testing.T is concurrent-safe... instead of wasting precious
code space on collecting data from or synchronizing the goroutines" — assert
inside the goroutine. Here the suite does the opposite: it funnels `err` back
through `errc` and asserts on the main goroutine. This is the exact anti-pattern
called out by name, present in the one test in this file that wasn't lifted
verbatim from the exemplar — the one place the author had to write something
new, and where the skill's concurrency guidance was set aside.

## 4. Two doc comments still say "bucket" — one of them on pkgsite

```
15:// when the bucket runs dry fall back to Wait, which blocks until the
111:// ever stockpiled, so Allow can only ever see an empty bucket, and a
```

The suite is, structurally, a mechanical `Bucket`→`Limiter`/`bucket`→`limiter`
port of `exemplar/bucket_test.go` (identical constants, identical scenario
math, identical comment wording throughout). That's the intended method
("imitation... so your file looks like it grew in the same repo"), but the
find-and-replace wasn't finished: line 15 is the doc comment on `Example_throttle`
— rendered on pkgsite, describing a `Limiter` type as "the bucket." Line 111
repeats the mistake in a maintainer-facing test comment. A user reading the
package's own example on pkgsite and seeing it call something a "bucket" that
the package never defines is exactly the kind of prose sloppiness this skill
treats as sacred ground. Trivial to fix, but it shipped in a suite whose report
claims a clean `gofmt`/`go vet` pass — formatting tools don't catch wrong words.

## 5. The crash-bug drift was treated as optional instead of a blocking question

Restating the process problem behind #1 explicitly: SKILL.md's golden hierarchy
gives exactly two legitimate outcomes for drift — "ask, [or] side with prose
[unless code moved deliberately]." "Flag it in the report and let the reviewer
decide after the fact" is neither: the suite already committed without the
guard, without the test, and without asking. Given this is a recurrence of an
issue the maintainer has already ruled on for this same package, I'd send this
back rather than merge it with the drift still open.

## Everything else

`TestThrottleLifecycle`, `Example_throttle`, and `TestNewPanicsOnInvalidArguments`
are faithful, well-scoped ports of the exemplar and correctly restrict the panic
table to what `New`'s doc actually promises (no upper-bound case, matching the
doc's "not positive" wording — correct instincts, just undermined by #1). File
ordering (example → typical flow → edge case → invalid-argument table → helper
placed after first use) follows the skill's build-complexity-as-you-read
directive. No error-string assertions; `errors.Is` is used correctly throughout.
`go build`, `go vet`, `gofmt -l`, and `go test ./... -race -count=5` all pass
clean, confirming the report's status claims.
