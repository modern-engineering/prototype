# Maintainer critique — truncate-drift / with_skill_looped_opus__r1

Reviewed as the `truncate` package maintainer, bound to skill-v4 and the
package prose (`go doc -all`). Verified: `go vet` clean, `gofmt` clean,
`go test` red only on the deliberate `TestNegativeCountYieldsEmptyString/Head`
panic — every other case passes. The drift call (side with prose, keep the
test red, do not touch `truncate.go`) is exactly right and I would keep it.

Files: `outputs/truncate_test.go`, `report.md`.

## outputs/truncate_test.go

### 1. Skill scaffolding leaks into the committed source (HIGH — refuse)

Line 81, `callWithoutPanic` doc comment:

> it is the inverse of exemplar/'s mustPanic, asserting a documented no-panic contract.

There is no `exemplar/` and no `mustPanic` in the `truncate` repo — that is
the skill's teaching package. This is a dangling cross-reference to my prompt
scaffolding baked into a source file a real maintainer will read forever. It
is precisely the "LLM leaking its prompt" tell I called a loser sign. Cut the
clause to the ground: the comment's first half ("turns a panic into a plain
failure that names the offending call rather than crashing the test binary")
already says everything a maintainer needs. "asserting a documented no-panic
contract" is filler on top of the leak. This alone blocks the merge.

### 2. Redundant sub-tests over identical code in the rune-boundary table (MEDIUM)

Lines 33-54. Every case runs the *same* two assertions, yet each is wrapped in
`t.Run(c.name, ...)` keyed by a `name` field ("ASCII", "MultiByte",
"ExactLength", "LimitOvershoots", "Zero"). Two problems, both in the doctrine:

- The names label input *categories*, not behaviors, and the failure messages
  already print the inputs (`Head(%q, %d)`). Naming the inputs is enough; the
  sub-test name is a log name the runner prints anyway. The `name` field earns
  its keep only to feed `t.Run` — drop both and this is a plain `for _, c` loop.
- My standing ruling on this exact Head/Tail table is "I would not use
  sub-tests." The merged `{s, n, head, tail}` case struct is right and welcome;
  the sub-test wrapper around it is the part I'd strike.

### 3. One package-level Example where two symbol-attributed ones belong (MEDIUM)

Lines 14-25. `Example()` is attributed to the whole package and exercises Head
and Tail together. It is competent and the in-function call-site comments
(runes vs bytes at lines 17-19) are the good kind. But for *this* package my
preference is `ExampleHead` and `ExampleTail`, so each function's contract
renders under its own symbol on pkgsite where a user looking up `Head` will
find it. The doc comment (lines 10-13) also drifts toward walking the three
calls ("shorten... keep... pass a generous limit...") rather than telling one
user story; it reads as a tour of the body more than the exemplar's
`Example_throttle` scenario does. Not a blocker, but it is a step back from the
per-function examples this package had earned praise for.

### 4. Function-value dispatch routes non-panicking Tail through a panic guard (MEDIUM-LOW)

Lines 62-90. `TestNegativeCountYieldsEmptyString` builds a `funcs` table of
`func(string,int) string` values and threads *both* Head and Tail through
`callWithoutPanic`. Only Head can panic; Tail never does, yet it is run through
a recover wrapper it does not need, which clouds what is actually under test.
Head and Tail genuinely *diverge* here (one panics, one returns ""), and my
rule is that divergence means the recover machinery guards the panicking case,
not a uniform function-pointer loop smoothing the two into one shape. Two
explicit sub-tests — Tail asserting `""` directly, Head asserting no-panic —
would say the same thing without the indirection.

## report.md

### 5. Report prescribes the fix instead of surfacing the author's choice (LOW-MEDIUM)

The drift write-up is clear and correctly refuses to touch `truncate.go`. But
it closes with "should clear once its guard matches Tail's," presuming the
resolution is "add the guard to Head." That is one of two live options; the
other is to decide `Head` panicking on a negative bound is intentional and fix
the *prose* instead. The context I want here — would the author promise the
no-panic behavior, or refuse to harden it? — is the deep-in-mind inference that
belongs in the fine-grained commit body for this non-trivial, deliberately-red
test, not a one-line prescription in the report.

## Credit where due

- File order is right: Example first, typical table, edge/drift test, helper
  last (after first use). No convoluted helper opening the file.
- The merged `{s, n, head, tail}` case with named fields is the shape I asked
  for; the Head/Tail contracts are stated together.
- Keeping the suite red on the documented-but-broken `Head(-1)` is the correct
  call; blessing the panic would have been the mistake.
- Correctly identified the failure as Head-only (Tail keeps the promise).
- No invented concurrency: pure functions, no synctest, nothing pinned that the
  docs do not promise.
