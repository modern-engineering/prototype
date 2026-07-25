# Critique — cache-write-tests / with_skill_looped_sonnet__r2 (round 2)

Reviewed strictly against `skill-v4/SKILL.md` + its bundled `exemplar/`, and
against the package's own prose (`go doc -all` on `outputs/`). I did not
open `cache.go`. Files under review: `outputs/cache_test.go`, `report.md`.

This directory already contained a round-1 critique.md and grading.json
from before the file's last edit (cache_test.go's mtime is hours after
both). I diffed the current file against what those two documents quote,
so this is a critique of round 2, not a cold read — where a round-1 defect
survives unfixed, that's noted explicitly, because it's the sharpest signal
this review can give: not "here's a bug" but "here's a bug you already
knew about."

Verdict up front: round 2 is a genuine, substantial improvement. Three of
round 1's four content findings and the one report.md finding are fixed
cleanly — the negative-TTL branch is now exercised, the refresh timeline
now carries explicit interplay arithmetic in comments, the Example now
explains its non-obvious literals, and report.md dropped the "pins"
language and the re-opened non-question. What's left is one finding round 1
raised and round 2 ignored outright, plus one defect neither round-1
critique.md nor round-2 authoring caught, that this run's own grading.json
did.

## `outputs/cache_test.go`

### 1. `Example_sessionToken` still races a real wall clock against a 50ms TTL, entirely outside `synctest` — HIGH

```go
func Example_sessionToken() {
	c := cache.New(time.Hour) // a janitor slower than the TTL below on purpose
	defer c.Close()

	c.Set("session-a1", "dana", 50*time.Millisecond)

	if user, ok := c.Get("session-a1"); ok {
		fmt.Println("authenticated as", user)
	}

	time.Sleep(100 * time.Millisecond) // twice the TTL: past any doubt

	if _, ok := c.Get("session-a1"); !ok {
		fmt.Println("session-a1 expired")
	}
```

`Example` functions take no `*testing.T`, so this can't be wrapped in
`synctest.Test` — but that's exactly why the skill's stability doctrine
matters here, not a reason it doesn't apply: "bubbles run on virtual time,
so sleeping inside one is idiomatic and stable," and "a weak-runner flake
is a contract failure too." This function bets its exact-match `Output`
block on a real 2x margin: if the scheduler stalls the goroutine for more
than 50ms between `Set` and the first `Get` (a GC pause, a loaded CI box,
a noisy neighbor), the first line flips from "authenticated as dana" to
nothing printed, and the test fails for a reason that has nothing to do
with the package.

This is not a hypothetical I'm inventing under fresh eyes: this run's own
`grading.json` already failed exactly this expectation — "Example_
sessionToken calls time.Sleep(100 * time.Millisecond) at cache_test.go:29
outside any bubble... the sleep does not sit inside a bubble" — against
the round-1 file (`cache.New(time.Minute)` there, per its quoted line).
Round 2 changed the janitor interval to `time.Hour` and added the
in-function comments, which fixed round 1's *other* Example finding
(missing comments) but left the actual race untouched: same 50ms TTL, same
bare 100ms `time.Sleep`, same missing bubble. Loosen the margin substantially
or, better, don't make the mandatory example's `Output:` block depend on
winning a real-time race at all.

### 2. `mustPanic`'s doc comment still opens with its own name — round 1 flagged this by name, and round 2 didn't touch it — HIGH

```go
// mustPanic fails the test unless New(interval) panics; invalidity names
// what makes interval invalid in the failure message.
func mustPanic(t *testing.T, interval time.Duration, invalidity string) {
```

SKILL.md: "no test comment ever opens with the function's name." This
exact function, this exact defect, was round 1's finding #1 in this same
directory's own critique.md, word for word. Round 2 fixed the Example's
comments, the negative-TTL gap, and report.md's phrasing — all genuinely
addressed — and then left this one line character-for-character identical
to what was already called out. It happens to be a verbatim reskin of the
bundled exemplar's own `mustPanic` comment, which has the same defect —
so the failure mode here isn't "didn't know the rule," it's "cited the
exemplar's shape without checking it against the rule the exemplar itself
sometimes violates." The golden hierarchy exists so a flawed worked
example doesn't outrank the stated doctrine; this is the case it's for.

### 3. The one concurrency test never contends on a shared key — MEDIUM

`TestConcurrentAccess` spawns 8 goroutines, each `Set`ting and `Get`ting a
*distinct* key (`session-0`..`session-7`). That's real concurrent access,
and it would catch a missing lock around a bare Go map (concurrent writes
to *any* keys of an unprotected map fatal-panic on their own) — but it's
the weakest possible reading of the doc's actual promise, "All methods are
safe for concurrent use by multiple goroutines." The scenario that
promise exists for is two callers touching the *same* session — a refresh
racing a read, or two refreshes racing each other — and nothing in this
suite ever puts two goroutines on one key.

### 4. The mandatory example never shows the one idempotence fact the exemplar bothered to show — LOW

The bundled `Example_throttle` doesn't stop at `defer b.Close()`; it also
calls `b.Close()` a second time, non-deferred, with a comment explaining
that the deferred call becomes a no-op — because Close's idempotence is a
documented, user-relevant fact, and the mandatory example is exactly where
pkgsite readers see it modeled. `cache`'s doc makes the identical promise
("Close is idempotent: calling it again is safe and does nothing"), and
`TestCacheLifecycle` demonstrates it — for maintainers only.
`Example_sessionToken` never does, so the one audience that reads pkgsite
never sees it. Imitating the exemplar's file *shape* (Example → lifecycle
→ concurrency → panic table → helper, all correct) isn't the same as
imitating its specific content choices.

### Noted, not ranked: no plain never-set-key miss is ever exercised

Every miss in the suite is a miss via expiry or via `Close`. `Get`'s own
doc — "returns the value under key and whether key held a live entry" —
implies a baseline behavior (a key nobody ever `Set` is a miss) that the
suite never establishes before layering expiry and shutdown semantics on
top of it. Minor; flagging rather than silently passing.

### What round 2 fixed cleanly (for the record, so the above doesn't read as ingratitude)

- Negative TTL is now exercised (`"pinned"`, `-500*time.Millisecond`)
  alongside zero TTL, closing round 1's finding #3 outright.
- The refresh timeline now carries the interplay arithmetic in comments at
  each sleep (the 300/900/400ms breakdown against the 800→1200ms deadline
  math), closing round 1's finding #4; this is now genuinely auditable.
- `Example_sessionToken` gained in-function comments explaining the
  janitor/TTL relationship and the sleep margin, closing most of round 1's
  finding #2 (the 50ms TTL's own `Set` call is still bare, but the
  adjoining sleep comment makes it inferable).

## `outputs/report.md`

No new finding here. Round 1's finding #5 (the report reaching for "pins"
language and re-opening a question the prose had already settled) is
fixed: the current text is a plain, accurate description of what was
written, and its one stated finding — "the doc's 'zero or negative' TTL
clause was previously untested for the negative branch — now both are
exercised" — checks out exactly against round 1's actual state (confirmed
by round 1's own critique.md finding #3) rather than being a report-slop
sentence invented to look substantive. Build/vet/race status line is
accurate; I reran `go build ./...`, `go vet ./...`, and `go test -race
./...` myself.
