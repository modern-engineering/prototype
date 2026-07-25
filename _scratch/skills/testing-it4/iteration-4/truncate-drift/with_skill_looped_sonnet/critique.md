# Critique: truncate-drift, with_skill_looped_sonnet

Reviewed as the package maintainer, against `skill-v4/SKILL.md` + exemplar
and the intent corpus (`INTENT.md`, `feedback-3.json`, `TESTS.md`). Surface
checked with `go doc -all`; `truncate.go` itself was not opened. `go build`
passes; `go test` fails on one test, deliberately, matching `report.md`.

## `outputs/truncate_test.go`

### 1. Banned "pins the contract" phrasing survived into the committed comment (blocking)

```go
// ... so this test pins the documented contract rather than the
// implementation and reports the mismatch as a failure instead of
// crashing the whole test binary.
func TestHeadAndTailTreatNegativeNAsZero(t *testing.T) {
```

This is close to a verbatim hit on the one phrase the corpus calls out by
name as the example of a test comment to never write: *"this test pins some
contract" (true of all tests)*. Saying a test "pins the documented contract
rather than the implementation" is exactly that — true of every test in the
suite, so it tells the next maintainer nothing they didn't already know
from the fact that it's a test. The paragraph already does the real work
one sentence earlier ("Tail honors that promise; Head panics instead") —
that's the finding. The "pins... reports the mismatch as a failure instead
of crashing" clause is throat-clearing about test mechanics, not contract.
Cut it.

### 2. `Example_display` has zero in-function comments (blocking)

```go
func Example_display() {
	fmt.Println(truncate.Head("héllo", 2))
	fmt.Println(truncate.Tail("héllo", 2))
	// Output:
	// hé
	// lo
}
```

The doc comment does all the talking and the function body is bare. This
is the exact defect flagged on every prior example across every package in
this corpus: in-function comments on example tests are "prime real estate"
because pkgsite renders them next to the code, and they're the place to
carry the non-Go parts of the contract to the call site. Here there's a
genuinely non-obvious result going unexplained: `Tail("héllo", 2)` returns
`"lo"`, the last two *runes*, and a reader skimming the call site has no
signal that this is the point of the demonstration (as opposed to, say,
the last two bytes, which would have mangled the é and produced garbage
mid-character). The doc comment says "shows the package's motivating
scenario, straight from its doc comment" — restate at the call site
instead of only above the function; that's where a pkgsite reader's eye
actually lands.

### 3. Test name promises one property, table asserts four (should be split or renamed)

`TestHeadAndTailCountRunesNotBytes` reads as "this pins rune-vs-byte
counting." Only the first of its five rows does that:

```go
{s: "héllo", n: 2, head: "hé", tail: "lo"},   // the named property
{s: "hello", n: 0, head: "", tail: ""},        // zero n
{s: "hello", n: 5, head: "hello", tail: "hello"}, // n == rune count
{s: "hello", n: 8, head: "hello", tail: "hello"}, // n > rune count ("generous limit")
{s: "", n: 3, head: "", tail: ""},             // empty input
```

Rows 2–5 are all ASCII and pin the "generous limit"/zero/empty-input
contract from the doc comment, a completely different property than the
one in the name. I don't want it split into five functions — fewer tests
covering more surface is right — but the name has to own what it actually
locks down, or a future reader trusts the title and skips the file, never
learning this is where the boundary behavior lives. Something like
`TestHeadAndTailHonorLengthAndRuneBoundaries`, or pull the boundary rows
into a second table with their own name.

### 4. The rune-safety row and the boundary rows never intersect

Every multi-byte case (`"héllo"`) stops at partial truncation (n=2); every
"n exceeds available content" case is ASCII-only (n=5, n=8 on `"hello"`).
Nothing exercises "n exceeds available runes" *on the multi-byte string*.
`"héllo"` is 5 runes but 6 bytes — exactly the shape that would expose an
implementation comparing `n` against a byte length instead of a rune count
in the "fewer than n runes, return unchanged" fallback path the doc
promises. A row like `{s: "héllo", n: 10, head: "héllo", tail: "héllo"}`
costs one line and closes that gap; right now the suite would pass even if
that fallback silently used `len(s)` instead of `len([]rune(s))`.

### 5. Duplicated rationale across two adjacent comments

`TestHeadAndTailTreatNegativeNAsZero`'s comment and `callHead`'s comment
both explain, in near-identical words, that a panic gets turned into an
ordinary `t.Errorf` instead of crashing the binary. Say it once — the
helper's own doc comment ("turns an unexpected panic into a t.Errorf naming
the panic value") is the right place for it; drop the restatement from the
test's comment once finding #1 above is trimmed to just the contract
mismatch.

## `report.md`

No blocking issues. Correctly frames the finding as the owner's call
("promise it... or refuse to harden"), doesn't silently side with the
implementation, and its status section matches what `go build`/`go test`
actually report. Good discipline reusing the package doc's own "héllo"
sample rather than inventing a new one.

## What I'd actually block on

Findings 1 and 2 are the ones I'd bounce back before merge: one is a
verbatim instance of a phrase this project has explicitly banned from
committed test comments, the other is the single most repeated note in
every prior review round (example in-function comments are mandatory real
estate, not optional). 3 and 4 are real gaps in what the suite actually
proves and I'd ask for them in the same pass. 5 is a one-line trim.
