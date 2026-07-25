# Maintainer critique: cache committed tests (with_skill_looped_fable__r1)

Reviewed as the package's maintainer, bound to skill-v4 and its exemplar.
Contract obtained from `go doc -all` only; implementation source not opened.
The report's mechanical claims verified independently: `gofmt -l` clean,
`go vet` clean, `go test -race -count=1` passes.

The skeleton is right and clearly grew from the exemplar: example first,
one whole-package flow in a synctest bubble, a panic table whose recover
helper sits after its first use, external `package cache_test`, hard-coded
times computed at coding time, wiki-form failure messages, no sub-tests
where prose names would rot. That inheritance is also the file's weakness:
it copied the exemplar's shape onto a package whose prose makes one promise
the exemplar's package never made, and that promise is the one thing the
suite never touches.

## cache_test.go

### MAJOR — the concurrency promise is the one untested clause

The type's prose says, in its own paragraph: "All methods are safe for
concurrent use by multiple goroutines." That is the loudest sentence in
the doc and the only contract clause with zero exercise. Every call in
this suite happens on a single goroutine; the only second goroutine alive
is the janitor the package spawns for itself. The report then advertises
`go test -race` passing, which is vacuous evidence when no two goroutines
ever touch the cache.

The concurrency reference's carve-out ("a suite whose scenarios run inside
synctest bubbles is the concurrency testing") covers docs that *omit*
concurrency. These docs affirmatively promise it. The rule against pinning
unpromised concurrency cuts the other way here: promised and untested.

The fix is not a statistical race hunt, which I would delete on sight. A
session store's anticipated caller *is* concurrent request handlers — the
example's own narration ("each request looks it up") says so. A handful of
request goroutines doing Get/Set on their own keys, asserting inside the
goroutines (*testing.T is concurrent-safe), then a Len tally, is the
anticipated flow made literal, deterministic in or out of a bubble
(WaitGroup if outside). Alternatively, refuse to harden and say so in the
report with the author's inference. The report says neither; see below.

### MEDIUM — the "either way" comment is false (lines 79–82, 87–89)

The drift-flagging comment is the sanctioned pattern (ask the owner, note
it in the report) and I credit it. But its last clause is wrong: "either
way Len stops counting hours before the janitor's first sweep" sits on a
`Len() = 2` assertion taken at the *same contested instant* (exactly ttl)
as the Get probe. If I rule the boundary half-open the other way, Len is 3
at that instant and this assertion fails right along with the Get one.
The read-time-enforcement claim is hostage to the very question the
comment says it is independent of. A committed comment that tells the
maintainer something false is worse than no comment.

Fix: keep the Get probe as the one deliberate boundary assertion carrying
the owner-pending note, sleep one more virtual nanosecond, and take the
Len reading there with an honest comment.

### MEDIUM — the example cheaps out on the package's headline

Per-entry expiry is the package's reason to exist, and the example only
alludes to it inside a comment ("or one past its TTL"). Real-time sleeps
are rightly off the table in a runnable example. But the zero/negative-TTL
no-expiry form is a distinct contract clause ("lives until it is replaced
or the cache is closed") and demonstrable with zero waiting: one more Set
with a distinct comment saying zero (or negative) ttl stores the entry
without an expiry. That is prime pkgsite real estate left empty. Don't
shorten examples; a reader who got this far should see the contract.

### MINOR — "lives until it is replaced" is half untested

Replacement is exercised against a live TTL'd entry (the refresh) and a
dead one (the revival). No test ever replaces a *no-expiry* entry, so the
"replaced" arm of the no-expiry clause's "until it is replaced or the
cache is closed" disjunction goes unexercised. Cheap probe inside the
existing flow: after the lull, Set `signing-key` with a short ttl and
watch the formerly immortal entry expire, proving the replacement took,
deadline and all.

## report.md

### MINOR — pinning-language, and an over-claim

"so the test pins the boundary as a miss": findings speak as the user
("at exactly ttl the token is already a miss"), not in pinning mechanics;
pinning-talk is how suites go mechanical. The boundary finding itself has
exactly the right shape (asserted-as-read, owner asked, loosening offered).
But "No prose/code drift found otherwise" over-claims: the report never
mentions the concurrency promise at all, neither exercising it nor
declining to. A finding the author decided not to act on still belongs in
the report with the inference (promise kept elsewhere? refuse to harden?).

## go.mod

Nothing to say. Minimal, matches the exemplar's shape.

## Comment-rule audit

Every doc comment checked against the no-name-opening rule:

- `Example_sessionTokens` — "A session store is the typical caller…":
  clear, and correctly voiced at users about the scenario, not at
  maintainers, not about the example function itself.
- `TestSessionTokenLifecycle` — "The store's whole day in one sitting…":
  clear; the closing sentence (janitor sweeps hourly, so pre-lull expiry
  is read-time enforcement) is genuine rationale, not narration.
- `TestNewPanicsOnNonPositiveInterval` — no doc comment on a trivial
  table: correct; a comment there would be slop.
- `mustPanic` — opens with its own name. Ruled compliant: the rule ("no
  test comment ever opens with the function's name") scopes to test
  functions — tests, benchmarks, fuzz, examples — and mustPanic is a
  plain helper, where standard Go doc convention requires name-first; the
  skill's own exemplar carries the identical comment on the identical
  helper. Flagging it would indict the exemplar. If the letter of
  SKILL.md is meant to cover helpers in _test.go files, that is a skill
  edit to request, not a defect in this file.

No slop anywhere: nothing explains synctest mechanics, function ordering,
or conventions; no "this test pins the contract". The in-example comments
repeat non-Go contract at call sites and explain the non-meaningful
literals (`New(time.Minute)` gets its sweep-cadence comment). That part is
exactly right.
