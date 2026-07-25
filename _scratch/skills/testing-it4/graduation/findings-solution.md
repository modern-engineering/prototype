# Findings — solution/, solution/image, examples/, root main (ranked)

1. Diagnostics expectations hand-count file:line:col across `\n`-escaped Go
   string literals for ~45 cases (`solution/compile_test.go:1420`, reused at
   `solution/scheme_test.go:101`, `solution/vocab_test.go:61`): this corpus
   earns the go/parser harness — SDL fixtures in testdata with the expected
   diagnostic marked inline at the offending token, so nobody re-counts
   columns at helper-call distance every time a message moves.
2. The 330-line golden image lives as an in-file const with no testdata
   pairing, no `-update` flag, and a failure that dumps both whole blobs
   instead of a diff (`solution/compile_test.go:286,633`): the gofmt golden
   discipline — fixture beside golden, `-update`, unified diff — is the shape.
3. Self-naming doc comments are systemic — nearly every non-trivial test
   opens with its own function's name ("TestUnpack proves...",
   "TestOutputWriterRoundTrip drives...", "TestBuildRoundTrip pins..."), and
   `ExampleMainCompile`'s (`solution/example_test.go:15`) renders on pkgsite;
   no test or example doc comment ever opens with its own name.
4. solution/image ships no runnable example although its package doc sells
   struct-literal composition as a peer frontend; `composedPingpong`
   (`solution/image/compose_test.go:221`) IS the anticipated call pattern
   (compose, Canonicalize, Validate, Encode) hidden from pkgsite — the
   mandatory example is missing.
5. Equal's documented clause "images holding the same content in a
   non-canonical order compare unequal" has no test — `disorderedImage` and
   `canonicalImage` (`solution/image/compose_test.go:22,108`) already exist
   one file over; `Equal(disorderedImage(), canonicalImage())` pins it in one
   line (`solution/image/equal_test.go`).
6. Grep-only error tests indict the API: `TestOutputWriterBoundary` exact-
   matches nine module-controlled `err.Error()` strings
   (`solution/outputs_test.go:132`), and the image suite matches
   Validate/Decode faults by substring throughout
   (`solution/image/compose_test.go:626`); nothing typed or sentinel exists
   to check with errors.Is — flag the error surface for a rethink.
7. CheckProvisionType's doc-promised "identical flag schema every time"
   check exists (`solution/provisiontest.go:82-90`) but its suite never
   exercises it (`solution/provisiontest_test.go:87`) — the scheme twin
   tests "unstable schema" (`solution/schemetest_test.go:73`); a harness's
   self-test is its usage documentation, and this clause is silent.
8. The root main package is a committed scaffold: `main.go:3` is an empty
   `func main() {}` and `TestNoop` (`main_test.go:5`) asserts nothing —
   delete the no-op test or give the package a contract worth one.
9. `TestVocabularyWiresLinker` rebuilds its expected stderr by joining the
   vocabulary at run time (`solution/vocab_test.go:54-59`) — the same
   rendering the linker performs, so its bugs pass; the comment even
   celebrates "never hard-coded". Write the lines as authored results; a
   vocabulary change then amends the stated contract visibly.
10. Sentence-length sub-test names read as prose, not test names:
    "duplicate symbols anchor at the first declaration"
    (`solution/compile_test.go:1505`), "verb default faults surface once per
    element" (`:2052`), "set the untyped output as non-string"
    (`solution/outputs_test.go:96`) — name the malformed input or Field=Value,
    or inline the case.
11. `composedPingpong` claims to mirror examples/pingpong's SDL and copies
    examples/ff doc strings verbatim (`solution/image/compose_test.go:216`),
    but nothing links them — the mirror will drift silently; either compile
    the real example and compare Equal, or drop the mirror claim.
12. `TestUnpack` spends t.Run ceremony on five one-line cases
    (`solution/catalogue_test.go:33`); a plain loop whose failure message
    names the case covers the same contract with less machinery.
