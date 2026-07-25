# Stdlib doctrine — how the Go team commits tests

Distilled from a deep read of `hash/{crc32,crc64,fnv,maphash}`, `io`,
`strings`, and `time` (anchors are into the Go source tree, `src/`).

1. **A test name is a claim about the contract, not a pointer to a symbol.**
   Type-with-methods APIs get behavior sentences (`TestSeededHash`,
   `TestMultiReaderFinalEOF`, `TestNoonIs12PM`); pure-function APIs may name
   the function but delegate to a shared runner covering the whole family
   (`strings_test.go:2`: `TestIndex` is one line over `runIndexTests`).
2. **One test exercises the API the way a user composes it.**
   `crc32_test.go:282` `TestGolden` alone covers ChecksumIEEE + NewIEEE +
   MakeTable + New + Write + Sum32, including split/misaligned writes.
   `maphash_test.go:44` `TestHashGrouping` runs SEVEN write paths and demands
   one identical sum. The unit of testing is an invariant, not a function.
3. **Doc comments are the test plan.** hash.Hash's "It never returns an
   error" → a `WriteWithoutError` subtest; strings.Builder's "Do not copy" →
   `TestBuilderCopyPanic`; fnv's "big-endian byte order" → `testIntegrity`
   decodes with binary.BigEndian. When a test checks something non-obvious,
   the contract is restated in prose right above it (`io/io_test.go:104-107`
   explains WriterTo-over-ReaderFrom priority before asserting it).
4. **Ship conformance kits for interfaces you own.** `internal/testhash`:
   "TestHash performs a set of tests ... checking the documented requirements
   of Write, Sum, Reset, Size, and BlockSize" — every hash package buys the
   whole documented contract in one line. Public siblings:
   `testing/iotest.TestReader`, `testing/fstest.TestFS`. Floor for everyone:
   compile-time `var _ hash.Hash = &Hash{}`.
5. **Serialized representations users persist are contract.**
   `hash/marshal_test.go:5-7`: "…and lock in the current representations" —
   16 hashes' marshaled states pinned as hex goldens. The friction of editing
   a golden on a format change is the feature.
6. **Goldens come from an independent oracle** (`hash/test_gen.awk` pipes a
   shared corpus through a reference binary; `strings` cross-checks against a
   `simpleIndex` oracle), then live inline. Never recompute goldens with the
   code under test. Alternate implementations cross-check EACH OTHER on
   adversarial lengths (`crc32_test.go:99-101` clusters lengths on asm
   cutoffs) instead of getting separate golden sets.
7. **Tables for input→output; bespoke scenarios for interactions.**
   `io/multi_test.go` has no tables at all — each test is an interaction
   scenario with a purpose-built 3-8 line fake (`type byteAndEOFReader byte`;
   a Buffer embedding bytes.Buffer solely to HIDE its ReaderFrom and force
   the generic path, `io_test.go:19-24`). Heterogeneous behaviors put
   closures in the table (`TestBuilderCopyPanic`).
8. **Test from `package foo_test`; `export_test.go` is the peephole.**
   io re-exports error sentinels (3 lines); strings exports PrintTrie /
   DumpTables so the external test can assert which algorithm was SELECTED;
   time names its hooks `ResetLocalOnceForTest` / `ForceUSPacificForTesting`.
   Drop to the internal package only to cross-check implementation kernels
   or private state (maphash seed).
9. **Documented performance is contract**: `testing.AllocsPerRun` with exact
   counts, GC-liveness via runtime.AddCleanup
   (`TestMultiReaderFreesExhaustedReaders`), call-depth flattening via
   runtime.Callers. Pin panic/error TEXT only where users depend on it
   (`maphash_test.go:397` pins "hash of unhashable type []uint8").
10. **Regressions get promoted to property claims**; the issue number lives
    in a comment — "This used to yield bytes forever; issue 16795." — never
    in the test name.
11. **Deliberate non-testing is visible.** Symbols sharing a kernel go
    untested directly (crc64.Update — behavior coverage, not symbol
    coverage); properties that do NOT hold stay as commented-out assertions
    with the reason (strings' TestCaseConsistency on Unicode
    non-one-to-oneness); out-of-promise ranges are carved out in-line
    ("// not required to work"); hermetic fixtures exclude host state
    (`time/internal_test.go:13`: only the test GOROOT's tzdata); statistical
    batteries gate behind -short / build tags (maphash smhasher).
12. **Examples carry trivial assertions AND the hardest usage instruction**
    (crc32's ExampleMakeTable teaches reversed-polynomial notation).
    Their doc comments never open with the function name — they read "This
    example uses a Decoder to…". Package-level `Example()` narrates the
    zero-value contract.
13. **Randomness is disciplined**: log the seed for replay ("Deterministic
    RNG seed: 0x%x"), fixed seeds for corpora, `testing/quick` for
    round-trips with explicit carve-outs. Order-dependent init is declared:
    "First test, so that it can be the one to initialize castagnoliTable."
