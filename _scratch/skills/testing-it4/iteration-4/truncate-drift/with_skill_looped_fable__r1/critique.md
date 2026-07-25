# Critique — truncate tests (fresh-context maintainer review, bound to skill v4)

Verified before reviewing: `go vet` clean; `go test ./...` fails exactly one
test, deliberately, as the report claims (`Head("héllo", -1)` panics). The
drift finding itself is real and correctly sided with prose. That part earns
the commit. The shape of what gets committed does not, yet.

Reviewed against `go doc -all` (the only source surface I opened), the skill,
its exemplar, and `references/helpers.md`, `tables-and-subtests.md`,
`concurrency.md`.

## truncate_test.go

### 1. The drift is not carried by the committed artifact (lines 58–62) — SEVERE

The skill: "Drift is a finding, never silently absorbed" and "Findings
deserve the author's inferences; non-trivial tests land as fine-grained
commits carrying them." The inference (Tail guards, Head doesn't, looks like
accidental omission, owner's call) lives only in report.md — a hand-off note,
not a committed artifact. The doc comment on TestNegativeCountActsAsZero
reads timeless, as if written before the bug was found: "a panic here would
take the caller down" hints, but never states that Head is the violator today
while Tail already complies, nor that an owner's decision is pending. A
maintainer who merges this gets a permanently red suite whose only in-repo
explanation is a runtime failure message. The finding and its inference must
live where they will be found: this test's doc comment, and the commit body
when it lands. One sentence fixes it. As committed, I refuse this.

### 2. Example_display has an empty-of-comments body (lines 12–18) — SEVERE

The skill is explicit: "Example comments, doc and in-function alike, render
on pkgsite ... repeating non-Go contract parts at call sites." The exemplar's
Example_throttle comments nearly every call. Here the body is two bare
Println calls. `5` and `10` are rune budgets — non-meaningful literals of
exactly the `New(1, 2)` kind that demand a call-site comment — and the one
non-Go contract part worth repeating (the budget counts runes, not bytes, so
"héllo wörld" cuts cleanly) appears nowhere a pkgsite reader will look. The
doc comment above the example is good: it speaks to users about the scenario,
not the code. The body wastes the prime real estate the doc comment earned.

### 3. Func-valued table + name-string wiring in the negative test (lines 63–73, 75–88)

`references/helpers.md` sanctions "a table plus a recover helper" for
expected panics — where the table varies *inputs*, as the exemplar's
mustPanic does with `bucket.New` called visibly inside the helper. This table
varies the *function under test* through a `cut func(string, int) string`
field, and reconstructs the callee in failure messages from a `name` string.
That is the exact signature-sharing indirection helpers.md warns about:
"failure messages become awkward to wire, masking the actual test code
executed." Two static cases do not earn a table, a func field, and a name
field. tables-and-subtests.md: simplify "even at the cost of repeated lines."
Drop the `cuts` slice; call the helper twice directly with Head and Tail
named at the call site, or write two sub-tests ("Head"/"Tail" read like Go
symbol names, so sub-tests are appropriate here) with the recover inline.

### 4. The mandatory example skips the documented call pattern (lines 12–18)

The package prose headlines its anticipated pattern twice: "callers may pass
a generous limit without measuring the input first." That reassurance is the
contract's most user-facing affordance and the example never shows it. The
skill mandates "a representative runnable example of the anticipated call
pattern"; an example that only shows the happy cut, without the
no-need-to-measure line (one more Println, one comment), undersells the
package on pkgsite. Do not shorten examples.

### 5. TestTruncate's doc comment narrates structure and leaks guideline vocabulary (lines 20–22)

"One table drives both cuts" describes what the reader sees three lines
down — structure narration, which the skill bans ("never explain
conventions"). "Every expectation is a hard-coded rune count" is the skill's
own vocabulary leaking into source: hard-coded values are a convention,
conventions are just well known, and telling maintainers you followed one is
the prompt-leak tell. The salvageable rationale is the second half: a
byte-index implementation fails the multi-byte cases instead of silently
mangling them. Keep that sentence, delete the rest.

## report.md

Good: verification status is precise and honest, the drift is stated with the
Head/Tail asymmetry, an inference (accidental omission vs. deliberate), and a
concrete owner's call. This is the report shape the skill asks for. Its flaw
is only what finding 1 says: the report is the sole carrier of the finding.

## What passes

- External `truncate_test` package; go-doc-first plan is evident.
- TestTruncate's merged case struct `{s, n, head, tail}` with one loop and
  two subsequent calls is exactly the blessed shape; fields named; rows
  hard-coded; input-naming row comments; the documented `Head("héllo", 2)`
  example is a row.
- Helper placed after first use, at the file's end; file builds complexity:
  example, typical table, special case, machinery last.
- Recover on an expected panic is the sanctioned exception, and siding with
  prose over Tail's-guard-but-not-Head's is the correct golden-hierarchy
  call.
- Failing the suite deliberately rather than blessing the panic is right.

## Verdict

The analysis is right; the artifact is not merge-ready. Fix findings 1–3
before this lands: record the drift in the file, comment the example body,
and unwind the func-field table. Findings 4–5 go in the same pass.
