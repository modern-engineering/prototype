# Maintainer critique — truncate tests (with_skill_looped_fable__r3)

Reviewed from the user's seat: `go doc -all`, the test file, and report.md.
No source files opened. Suite verified: gofmt and vet clean, `go test` fails
on `TestNegativeCountMeansEmpty` alone by panic, exactly as reported.

The shape of this file is largely right: external test package, one
mandatory scenario example with the audience aimed correctly, a merged
head/tail case struct driven by one loop with two calls (the exact form I
asked for), named struct fields, no global case slices, no helper in sight,
and the drift surfaced instead of blessed. That is why the findings below
sting: they are all prose failures, and prose is the most sacred part.

## truncate_test.go

### 1. BLOCKER — Example_display teaches users something false (line 16)

> "héllo, wörld" is 12 runes but 14 bytes; a byte-based cut at
> index 5 would split the é.

It would not. `s[:5]` of that string is `68 c3 a9 6c 6c` — "héll", valid
UTF-8; the é sits at bytes 1–2, so only a cut at byte index 2 splits it. I
verified the bytes. This comment renders on pkgsite; it is the single most
visible sentence in the whole file, and it is wrong. A user who trusts it
walks away with a broken mental model of UTF-8, which is precisely the
misunderstanding this package exists to spare them. Refuse to merge until
the claim is true: say a cut at byte 2 splits the é, or demonstrate on the
CJK string where nearly every byte index lands mid-rune. The rune/byte
counts (12 and 14) are correct; keep those.

### 2. BLOCKER — the drift test's doc comment mixes rationale with slop (lines 56–62)

The good part is genuinely good: state the documented promise, note Tail
keeps it and Head breaks it, side with the prose, put the owner call at the
hand-off. That is the drift discipline working. Now delete the rest:

- "it sits last in the file so the panic cannot mask the other results" —
  never explain ordering inside the source. This is the prompt-leak
  tell-sign. The placement judgment is correct; making it is enough,
  narrating it is not.
- "rather than pinning the panic" — pinning-speak. Every test pins; the
  word says nothing and breeds the mechanical mindset. "asserts the
  documented contract" already carries the meaning.
- "Head is missing Tail's n <= 0 guard and panics slicing r[:n]" —
  implementation narration in a committed comment. Maintainers can read
  the code; the comment will rot the moment the guard lands. The
  sibling-guard inference is commit-body and report material, where it
  already lives.

Trimmed to the promise, the observed failure, and the owner question, this
comment is exactly what a non-trivial test deserves.

### 3. MAJOR — the narrated mechanism is not just misplaced, it is uncorroborated (line 58, and report.md)

The comment and the report both assert Head "panics slicing r[:n]". The
actual panic reads `slice bounds out of range [:0] with capacity 32`;
`r[:n]` with n = -1 panics `[:-1]`. Whatever line 23 of truncate.go really
does, the observable evidence contradicts the stated mechanism. From the
user's seat the only defensible committed claim is "Head panics on a
negative n" — which is also all the test needs to say. This is the concrete
cost of guessing at internals: the guess shipped wrong. It also suggests
the test plan was not drafted purely go-doc-first, or the source peek was
misread; either way the committed artifact must not carry it.

### 4. MINOR — TestTruncate claims no property (line 32)

A test name claims a property in plain English. The package-name allowance
covers whole-package scenario tests; this is not a scenario, it is a
property table — runes-not-bytes plus the boundary promises. The merged
case struct and single loop are the right shape; the name should say what
the table proves: `TestRunesNotBytes` or kin. As written, "TestTruncate"
reads like a symbol scan of the package name.

## report.md

### 5. MINOR — report-speak and misplaced inference

"Rather than pinning the panic" again: reports speak in contract and flow
terms, not pinning. The sibling-guard inference (author probably intended
the promise and dropped the check) is the right kind of thinking and the
report is the right first home for it — but it is duplicated into the
committed test comment (finding 2) and states the wrong mechanism (finding
3). Keep it here and in the eventual fine-grained commit body; strip it
from the _test.go file. Otherwise the report is honest and complete: the
deliberate red test is declared, status is accurate, and the drift is the
headline, as it should be.

## Not blocking, worth a thought

Head and Tail render on pkgsite with prose only. Example_display covers the
composed call pattern, which satisfies the mandate; per-function doc
examples (ExampleHead, ExampleTail) would let each function's page show its
own contract at once. A judgment call, not a demand.

## Verdict

Not mergeable as-is: finding 1 is a false statement to users, finding 2 is
committed slop. Both are comment edits away from a merge; the test bodies,
case selection, and the drift handling need no change.
