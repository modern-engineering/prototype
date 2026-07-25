# Critique — truncate-drift / with_skill_looped_sonnet__r3

Reviewed as the maintainer, bound to `skill-v4/SKILL.md` and its exemplar
(`skill-v4/exemplar/bucket.go` / `bucket_test.go`), against the exported
surface only (`go doc -all` in `outputs/`), `outputs/truncate_test.go`, and
`report.md`. `go build ./...`, `go vet ./...`, and `gofmt -l .` are all clean.
`go test ./...` has one failing sub-test, `TestNegativeNIsTreatedAsZero/Head`,
left failing on purpose — that part I accept outright: the skill says drift
is sided with prose and never silently absorbed, and I already said in an
earlier round that blessing a panic like that would have been a mistake. That
is not a finding here.

Ranked by what I'd stop the merge over first.

## 1. `truncate_test.go` — `Example_shortHash` has zero in-function comments

This is the file's biggest problem. The skill is explicit that in-function
example comments are "prime real estate" and a representative example is
"mandatory" — and I have said the same thing in almost every review round
this corpus has collected. `Example_shortHash` has a fine doc comment above
it, then four lines of naked code: no comment on why 4 characters from each
end, none on why `+ "…" +`, nothing. Compare it to the bundled exemplar's
`Example_throttle`, which comments nearly every call. I already told a
sibling submission of this exact package "Both examples look really as if
taken from the standard library — good job... targeting each function and
repeating comments and code, that's the way to go." This submission reads
like the doc comment was written and the body was an afterthought.

## 2. `truncate_test.go` — the one mandatory example never shows the package's actual selling point

The package doc (visible in `go doc -all`) leads with the whole reason this
package exists: byte-slicing splits multi-byte UTF-8, so these functions
count runes instead, and it gives its own worked example, `Head("héllo", 2)`
→ `"hé"`. `Example_shortHash` truncates a plain ASCII hex string. A pkgsite
reader gets a nice abbreviation story but never sees the one behavior that
justifies the package's existence over `s[:n]`. The rune-safety case only
shows up in a maintainer-facing table (`TestHeadAndTailKeepWholeRunes`),
which nobody reading go doc ever sees. One example is enough per the
skill's letter, but it has to be the representative one, and the
multi-byte case is the representative one for this package, not a
hash-abbreviation aside.

## 3. `truncate_test.go` — first table case is lifted verbatim from the skill's own text

`{s: "hello", n: 3, head: "hel", tail: "llo"}` is not just similar to, it is
character-for-character identical to the illustrative snippet in
`skill-v4/references/tables-and-subtests.md` ("merge their expectations into
one case struct, `{s: "hello", n: 3, head: "hel", tail: "llo"}`"). That's the
skill's own pedagogical example, not a value derived from this package's
actual domain (hashes, abbreviations, whatever `Example_shortHash` chose).
It reads like the table was templated off the reference doc rather than
authored for this package, the same "leaking the prompt" smell the skill
condemns for comments, just showing up in data instead.

## 4. `truncate_test.go` — `TestNegativeNIsTreatedAsZero`'s doc comment explains the test's own machinery

Most of this comment is legitimate, non-trivial rationale I want: it states
the drift (both docs promise zero-treatment, only `Tail` honors it) and I
have no complaint about that half. But the tail end — "so it is wrapped in a
recover to turn that crash into a normal test failure rather than taking
down the rest of the suite" — is explaining how the test itself is built,
not the package's contract. That's the same species of thing I've flagged
before as a tell: don't narrate the test's own mechanics or ordering to me,
"nothing good to say, say nothing." Cut that clause; the drift statement
alone earns the comment.

## 5. `report.md` — "before it ships in `utils`" refers to nothing in this task

The hand-off note reads: "Author's inference for whoever owns this before it
ships in `utils`: either add Head's missing n <= 0 guard...". There is no
package, module, or directory called `utils` anywhere in this task — the
module is `example.invalid/truncate`, full stop. Either this is bleed from
another run in the loop or an invented detail, but either way it's a
concrete, checkable inaccuracy in the one artifact meant to carry trustworthy
context to whoever picks this up next. I'd stop and ask "what utils?" before
reading another word of the report.

---

Everything else is in good shape and I want that on record too: external
`truncate_test` package, file order builds from example → typical table →
special-case drift test exactly as the skill asks, named struct fields on a
four-field case, no global case slices, the drift itself correctly sided
with prose and left red instead of silently patched or blessed. The fixes
above are all real but they're finish-work, not a redesign.
