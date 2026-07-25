# Maintainer critique — `limiter-write-tests` / `with_skill_looped_sonnet__r3`

Scope: `outputs/limiter_test.go` and `report.md`, judged against `skill-v4/SKILL.md`
+ `skill-v4/exemplar/` and the package's exported surface (`go doc -all` below). I did
not open `outputs/limiter.go`.

```
package limiter // import "example.invalid/limiter"

Package limiter provides a token-bucket rate limiter driven by a background
refill goroutine.

A limiter created with burst 0 stores no tokens: Allow always reports false, and
Wait proceeds only when a refill arrives while the caller is already waiting.

var ErrClosed = errors.New("limiter: closed")
type Limiter struct{ ... }
func New(rate, burst int) *Limiter        // panics if rate is not positive or burst is negative
func (l *Limiter) Allow() bool
func (l *Limiter) Close()
func (l *Limiter) Wait(ctx context.Context) error
```

Verified independently: `go build ./...`, `go vet ./...`, `gofmt -l .`, and
`go test -race ./...` all pass as claimed in `report.md`.

## `outputs/limiter_test.go`

### 1. A skill-mandated test is missing at hand-off — the crashing case is left commented out (blocking)

`references/concurrency.md` states this exact scenario as canon, almost word for
word: *"Test invalid inputs the package technically accepts: a rate large enough to
floor the refill interval to zero is invalid even though the constructor lets it
through."* It also gives the explicit workflow for a bug-panic that blocks progress:
*"comment the triggering case out to see the work through, and restore it at
hand-off, never dropping it silently."*

`report.md` finds exactly this bug — `New` only guards `rate <= 0`, so
`rate: 2_000_000_000` sails through, floors `time.Second/time.Duration(rate)` to
`0`, and crashes the process in the unrecovered refill goroutine via
`time.NewTicker(0)`. But the case sits commented out in the committed table
(lines ~144–151), and no follow-up test replaces it. This is the hand-off state the
reference explicitly forbids: the case was never restored, in any form.

A real fix exists without touching `limiter.go`: re-exec the test binary as a
subprocess (the standard idiom stdlib itself uses for testing fatal exits/panics
that would otherwise take the whole binary down — guard on an env var, have the
child call `New(2_000_000_000, 1)`, and assert on the parent's observed crash).
Leaving a comment and a report footnote is not equivalent to testing it; it ships a
known crash bug with no regression coverage.

### 2. The two headline tests are a search-and-replace of the bundled exemplar, not an authored test for this package (severe)

Diffing `skill-v4/exemplar/bucket_test.go` (renamed `Bucket`→`Limiter`,
`bucket`→`limiter`) against this file:

```
diff "$TMPDIR/bucket_as_limiter.go" outputs/limiter_test.go
```

turns up **only** identifier renames (`b`→`l`) for `Example_throttle`,
`TestThrottleLifecycle`, and `TestNewPanicsOnInvalidArguments`: every doc comment,
every in-function comment, and every scenario number (rate 2/burst 2, rate 1/burst
2, the 100ms timeout, the "arrives 900ms from now" math, the 4500ms lull, the
2500ms shutdown gap) is carried over verbatim. Only
`TestZeroBurstOnlyAdmitsAnAlreadyWaitingCaller` and the commented-out panic case are
new work.

The corpus is explicit that the exemplar's authority is in embodying canon and
intent, never in being a source to lift from ("You must not learn from my code any
more than you should from theirs" — the same epistemology SKILL.md states for its
own bundled exemplar). "Imitation is the method" means the file should read like it
grew in this repo, not like the reference file with two words replaced. It also
turns out to matter here, not just as a matter of principle: `limiter.New` and
`bucket.New` are *not* identical (limiter drops the upper-bound guard, see #1
above), so the 900ms/4500ms/2500ms math was reused without any evidence it was
independently re-derived against *this* package's implementation. It happens to
still hold; next time this habit may not be so lucky.

### 3. Dead code committed into the table literal (moderate, compounds #1)

Independent of the reference violation, a permanently commented-out case with a
five-line comment explaining why it's disabled is exactly the kind of cruft
SKILL.md's "trivial straight-line" bodies rule warns against. Either the case gets
fixed and restored, or it gets deleted and the finding lives solely in the
handoff note — not both parked in the source as a landmine for the next reader.

## `report.md`

### 4. The finding is real and well-diagnosed, but framed as a footnote instead of the blocking issue it is (moderate)

"One thing worth a look before you send the PR" undersells a discovered crash bug
with no test coverage. The golden hierarchy calls drift/gaps like this out as
something to surface prominently, and the reference above makes covering this
input mandatory, not optional polish. This should be the lead item, not an
aside after the summary line.

### 5. The recommendation is stated as a decision, not offered as a choice (minor)

"An upper-bound guard mirroring the existing rate/burst checks would close it" reads
as the agent picking the fix rather than surfacing the fork the maintainer actually
faces: harden `New` with an upper bound (changing the panic contract), or accept
today's unbounded-rate behavior as a known, documented limitation and test that it
doesn't get quietly worse. The corpus is explicit that this kind of inference
belongs to the maintainer to make ("promise it, or refuse to harden?"); the report
should pose it that way rather than pre-select an answer.

## Bottom line

Would not merge as-is. #1 is a hard blocker: a real crash bug was found and then
shipped with the regression test disabled, which is the one thing the loaded
concurrency reference says never to do. #2 is close behind — regardless of whether
the numbers hold up, a test file that is 95% a renamed copy of the skill's sample
package does not demonstrate that anyone verified this package's own contract.
`TestZeroBurstOnlyAdmitsAnAlreadyWaitingCaller` is the one place this run did the
actual job the task asked for, and it is good work: grounded in the package's own
doc sentence, correctly reasons about the non-blocking send/rendezvous semantics,
named and commented for a maintainer audience.
