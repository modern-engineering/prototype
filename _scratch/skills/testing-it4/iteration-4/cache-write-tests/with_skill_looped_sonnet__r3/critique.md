# Critique — cache-write-tests / with_skill_looped_sonnet__r3

Fresh-context review as the maintainer, bound to `skill-v4/SKILL.md` and its
`exemplar/`. Scope: `outputs/cache_test.go` and `report.md`, judged against
the exported surface only. `cache.go` was not opened. Exported surface, via
`go doc -all`:

```
package cache // import "example.invalid/cache"

Package cache provides an in-memory key-value store with per-entry expiry,
intended for short-lived values such as session tokens.

type Cache struct{ ... }
    Cache is an in-memory string store whose entries expire after a per-entry
    TTL. A background janitor goroutine removes expired entries, but expiry is
    enforced on read: an expired entry is invisible to Get and Len even before
    the janitor removes it.

    All methods are safe for concurrent use by multiple goroutines.

func New(janitorInterval time.Duration) *Cache
    New panics if janitorInterval is zero or negative.
func (c *Cache) Close()
    Close is idempotent. After Close, Set does nothing and Get reports a miss
    for every key.
func (c *Cache) Get(key string) (string, bool)
func (c *Cache) Len() int
func (c *Cache) Set(key, value string, ttl time.Duration)
    The entry expires ttl from now. If ttl is zero or negative, the entry is
    stored without an expiry.
```

`go build`, `go vet`, `go test -race`, `golangci-lint run`, and `gofmt -l`
all come back clean, confirmed independently.

## `cache_test.go`

### 1. [HIGH] `TestConcurrentAccessIsSafe` asserts nothing at all — it is a race-detector trip-wire, not a test

```go
func TestConcurrentAccessIsSafe(t *testing.T) {
	c := cache.New(time.Hour)
	defer c.Close()

	var wg sync.WaitGroup
	for n := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("session-%d", n)
			c.Set(key, "tok", time.Minute)
			for range 100 {
				c.Get(key)
				c.Len()
			}
		}()
	}
	wg.Wait()
}
```

`t` is the function parameter and is never touched again: no `t.Error`,
no `t.Fatal`, nothing. The doc's single strongest, most explicit contract
clause — "All methods are safe for concurrent use by multiple goroutines" —
gets exactly one test, and that test can only fail by panicking or by
`go test -race` catching a memory race. It pins zero observable behavior:
not that a goroutine's own `Get(key)` after its own `Set(key, ...)` returns
`"tok"`, not that `Len()` is 8 once every goroutine has set its key. Run it
without `-race` (or run it with `-race` against a correct-but-lucky
implementation) and it is a no-op dressed as coverage. `references/concurrency.md`
says to assert inside the goroutine rather than collect values for the main
goroutine to inspect — it does not say to do neither. This test does neither:
it neither collects nor asserts. It is the SKILL's own "coverage in the
nebulous sense spans the whole suite's code" doctrine used backwards — an
empty test cannot contribute to that coverage no matter how realistic its
call pattern looks. Two cheap, in-goroutine assertions (`Get` returns what
was just `Set`, plus a post-`wg.Wait()` `Len() == 8`) would have actually
pinned the promise instead of merely exercising it.

### 2. [HIGH] `Set`'s "expires ttl from now" clause — the part of the contract that matters most on replace — is never exercised

The doc is explicit: "Set stores value under key, replacing any existing
entry. The entry expires ttl from now." That "from now" is doing real work:
it says a replace resets the deadline to run from the replacement, not from
the original entry's creation, and not compounded with whatever was left of
the old entry's TTL. `TestCacheLifecycle` replaces exactly once:

```go
c.Set("session", "tok-v1", -time.Second) // no expiry
...
c.Set("session", "tok-v2", 0)            // still no expiry
```

Both the original and the replacement are no-expiry entries, so the test
demonstrates nothing about deadline semantics on replace — it would read
identically if `Set` ignored "from now" entirely and, say, kept whatever
expiry state the slot already had. A go-doc-first plan built from the
exported contract (which this run says it followed) should have replaced a
live, expiring entry with a new TTL and shown the new deadline honored on
its own clock, independent of how much of the old TTL was left. As written,
the suite is silent on a clause quoted verbatim in the package's own doc
comment.

### 3. [MEDIUM-HIGH] `mustPanic`'s doc comment opens with `mustPanic`, the exact bug TESTS.md names and SKILL.md restates

> `// mustPanic fails the test unless New(interval) panics; invalidity names
> what makes interval invalid in the failure message.`

SKILL.md: "no test comment ever opens with the function's name." TESTS.md,
verbatim: "BUG: agent tends to prefix the doc comment of example test with
the func's name... the doc comments of all test functions... must never open
with the function's name." This is not a matter of taste; it is the one
rule the corpus flags by name as the recurring agent failure, and this
comment trips it in the most literal way possible. Worth noting for the
loop, not as an excuse: `exemplar/bucket_test.go`'s own `mustPanic` comment
("mustPanic fails the test unless New(rate, burst) panics; invalidity
names...") commits the identical violation, so this was inherited by
imitating the bundled exemplar rather than authored fresh. The exemplar is
authoritative for shape, never for having every rule right by construction —
copying its defect along with its structure is exactly the failure mode the
loop's review pass exists to catch, and it didn't.

### 4. [MEDIUM] `Example_sessionTokens` leaves its one contract-bearing line uncommented

```go
c.Set("session-42", "tok-abc", 20*time.Millisecond)
if v, ok := c.Get("session-42"); ok {
	fmt.Println(v)
}

time.Sleep(40 * time.Millisecond)
_, ok := c.Get("session-42")
fmt.Println(ok)
```

The audience doctrine is explicit that in-function example comments are
prime real estate for repeating non-Go contract parts at their call sites.
The entire payoff of this example — that `Get` reports a miss once the TTL
has elapsed, ahead of any janitor sweep — rides entirely on the reader
noticing that 40ms follows a 20ms TTL, and that line carries no comment
tying the two numbers together. Contrast the bundled exemplar, which
comments nearly every semantically loaded line including exactly this kind
of timing relationship. The two comments this example does carry (the
`New(time.Hour)` aside and nothing on `defer c.Close()`, which is itself a
gap — the exemplar spells out "always defer Close" and this file drops that
line silently) show the writer knows the technique; it just wasn't applied
where it mattered most.

### 5. [LOW] The lone replace inside `TestCacheLifecycle` is a wasted test point

"Replacing the still-live session token updates it in place and resets its
TTL to none" replaces a no-expiry entry with another no-expiry entry — see
finding 2. Filed separately at lower severity because it's a design choice
inside an otherwise well-constructed lifecycle test, not an omission on its
own; folding it into finding 2's fix (replace a *live-TTL* entry instead)
resolves both at once.

## `report.md`

### 6. [MEDIUM-HIGH] "No drift... nothing here needed a call to you" overclaims completeness the suite doesn't back up

> "No drift between the package doc and the implementation: every promise
> in `go doc -all`... held up under test exactly as documented. Nothing here
> needed a call to you about the contract vs. the code; the tests assert the
> contract as written."

This isn't a false statement about drift (drift requires seeing the code,
which this review correctly didn't) — it's an overclaim about coverage. The
report lists `TestConcurrentAccessIsSafe` as covering "the doc's... promise,
exercised as the realistic pattern a session store actually sees," without
once mentioning it asserts nothing (finding 1), and it never surfaces that
"expires ttl from now" was never put under a replace of a live entry
(finding 2). The skill's own review doctrine asks findings to carry the
author's inference for the commit body ("promise it, or refuse to harden?")
precisely so gaps like these don't get closed out with a blanket "no drift."
A report this confident, over a suite with a functionally empty concurrency
test and an unexercised doc clause, reads as unearned.

## What's genuinely good here (for calibration, not padding the count)

`TestCacheLifecycle` is a real whole-package flow, not a symbol scan, and
gets the janitor-vs-read-time race right entirely inside a `synctest`
bubble. `Example_sessionTokens` is exactly the kind of representative,
doc-matching call pattern the skill makes mandatory. File order builds
complexity correctly (example, then typical flow, then concurrency, then
the panic table, helper placed after its only use). No comment narrates
conventions, ordering, or synctest mechanics anywhere in the file — the
anti-slop rule is respected throughout, which is not a small thing given
how often the corpus flags it as the default failure mode.
