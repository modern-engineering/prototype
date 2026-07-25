# Critique — cache-write-tests / with_skill_looped_sonnet__r1

Reviewed strictly against `skill-v4/SKILL.md` + its bundled exemplar, and
against the package's exported prose (`go doc -all` below). No non-test
`.go` source was opened.

```
package cache // import "example.invalid/cache"

Package cache provides an in-memory key-value store with per-entry expiry...

type Cache struct{ ... }
    All methods are safe for concurrent use by multiple goroutines.

func New(janitorInterval time.Duration) *Cache
    ... New panics if janitorInterval is zero or negative.
func (c *Cache) Close()
func (c *Cache) Get(key string) (string, bool)
func (c *Cache) Len() int
func (c *Cache) Set(key, value string, ttl time.Duration)
```

## outputs/cache_test.go

### 1 — Critical: the type's headline promise is never exercised, and the report's excuse for skipping it answers the wrong question

The doc comment says, in so many words, "All methods are safe for
concurrent use by multiple goroutines" — a promise the skill's own base
doctrine says goes *beyond* the Go default and therefore must be honored,
not the trimmed-down "docs omit concurrency" case the reference file
actually talks about. `TestEntryLifecycle` runs one caller goroutine racing
the package's internal janitor inside a synctest bubble. That is real
concurrency, but it is the package's *own* internal concurrency, present
whether or not the docs said a word about it — it is not a test of what the
docs actually promise: that several *caller* goroutines may call
`Set`/`Get`/`Close` at once. For a session-token cache, concurrent callers
are the primary anticipated flow, not an edge case.

report.md defends the omission by invoking "the skill's guidance against
statistical race-hunting." That guidance targets contrived misuse hunts
against packages whose docs are silent on concurrency (four goroutines
hammering `Close`, hoping the detector fires). It was never meant to license
skipping the one concurrency claim the docs make explicitly. A real test
here is exactly the shape the maintainer's own notes describe: goroutines
outside synctest, a `sync.WaitGroup` so they don't leak, assertions made
*inside* each goroutine (`*testing.T` is concurrent-safe) instead of
ferrying values back.

### 2 — High: the flagship test's own doc comment overclaims what its body proves

`TestEntryLifecycle`'s doc comment and its closing in-body comment both say
"no matter how many [entries] are left" / "remain" at Close. Trace the
actual state at that point: every prior entry (`token`, `config`) has
already expired and `Len() == 0` is asserted immediately before `late` is
set. Exactly one entry exists when `Close` fires. The exemplar's analogous
scenario earns its "no matter how many tokens are left" line honestly — it
lets two tokens accumulate during the lull so Close is proven to discard a
non-trivial pile, not a singleton. Here the comment asserts a property in
plain English that the test never bothered to set up. Either seed two or
three live entries before the final Close, or stop claiming "how many."

### 3 — High: half of a documented either/or branch is untested

`Set`'s doc: "If ttl is zero or negative, the entry is stored without an
expiry." Only `ttl: 0` is exercised (the `config` entry). A negative ttl
never appears anywhere in the file — a real gap, and a careless one, because
the very same file already reaches for the "zero or negative" pairing two
functions down, in `TestNewPanicsOnInvalidJanitorInterval`'s table (`{rate:
0}, {rate: -1}` — er, `{interval: 0}, {interval: -time.Second}`). The author
demonstrably knows to test both halves of that idiom and simply didn't
carry it over to `Set`.

### 4 — Medium: `mustPanic`'s doc comment opens with its own name

"mustPanic fails the test unless New(interval) panics..." — this is the
exact shape the skill rules out categorically: no test comment ever opens
with the function's own name. It happens to match the bundled exemplar's
`mustPanic` comment almost verbatim (`refill`→`janitor`, `rate, burst`→
`interval`). That is not cover: the corpus is explicit that the exemplar
embodies the canon rather than being the canon, so a lapse riding in on
imitation is still a lapse. Reword to lead with what the helper checks, not
its own name — e.g. "Fails the test unless New(interval) panics; ...".

### 5 — Medium: the most basic miss case in a key-value store's contract has zero coverage

Every `Get` call in the suite targets a key that was set and then expired,
or set and then wiped by `Close`. Not one call targets a key that was never
set at all in a live cache — the plainest possible miss, and the first
thing any caller of a KV store needs to trust. The closest candidate,
`Get("ignored")` after `Close`, doesn't count: post-`Close` the doc already
guarantees every `Get` misses regardless of whether the key was ever
stored, so it tests the `Close` override, not a genuine absent-key miss.

## report.md

Minor, not part of the top findings: the report tells the *human*
maintainer that a decision was made "per the skill's guidance against
statistical race-hunting." The maintainer reading this report has no reason
to know what "the skill" is — it's the agent's own scaffolding leaking into
a document meant to read as ordinary engineer-to-maintainer communication.
Restate the same reasoning without naming the apparatus that produced it.

## What's actually fine, for the record

File order (Example → flagship lifecycle test → panic table → helper),
`mustPanic`'s placement after its only call site, the panic table's field
naming, `gofmt`/`golangci-lint`/`go vet` all clean, and `go test -race`
stable across five runs — none of that is in question here.
