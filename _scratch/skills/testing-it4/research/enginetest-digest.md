# enginetest digest — the maintainer's harness taste in production

Source: `github.com/go-digitaltwin/go-digitaltwin` (singular; the task's
"go-digitaltwins" is a misnomer). Cloned at `research/go-digitaltwins/`
(HEAD 7aba933). `enginetest` landed 2025-12-11 (30e9460, "guarantee graph
engines conform to the expected behaviour"), so it POSTDATES TESTS.md's seed
notes but PREDATES the 2026-07 audience-doctrine rulings. All paths below are
repo-relative.

## Two harnesses, two altitudes

The module ships two testing packages that compose:

- `enginetest/` — EXPORTED conformance suite at the CONTRACT altitude. It
  exercises implementations only through the `digitaltwin.Applier` and
  `digitaltwin.WhatChangeder` interfaces. Public because third-party engine
  authors are the audience (the httptest/analysistest peer TESTS.md names).
- `internal/dbtest/` — INTERNAL infrastructure harness (testcontainers Neo4j).
  Internal because only this module's tests consume it. This is the
  "test helpers require tests themselves → internal testing package" line
  from INTENT ruling 2, drawn in production.

Consumer proof: `neo4jengine/engine_test.go` is 25 lines total — an `init`
registering the harness's fixture nodes, then
`driver := dbtest.SetupNeo4j(t); engine, err := NewEngine(...); enginetest.Run(t, engine, engine)`.
A whole engine is certified in one screen.

## enginetest API shape (enginetest/enginetest.go, checks.go)

- ONE exported entry point: `Run(t *testing.T, applier, changeder)` (l.248).
  First line `t.Helper()`. Takes `*testing.T`, not `testing.TB` — the suite is
  test-only. The package doc (l.9-18) shows the canonical call pattern in a
  code block: "Call enginetest.Run in its own test".
- Exported FIXTURE TYPES `NodeA..NodeD` (l.49-52), gob-registered in the
  harness's own `init`. Exported so consumers can pre-process them for their
  backend (`neo4jengine/engine_test.go:11-16` calls `Register(enginetest.NodeA{})`).
- NO t.Run subtests. Cases are a package-level ordered `var cases []testCase`
  run strictly in sequence; each case's graph is the next case's starting
  state (`lastGraph` threading, l.262-296). `Apply`/`WhatChanged` failures are
  `t.Fatalf` (later cases are meaningless); check mismatches are `t.Errorf`.
  Case names ("retract-nonexistent-node", "connect-tree") are kebab-case
  labels in failure messages, not subtest names.
- FLOWS OVER PINNING, embodied: the case list reads as one graph's biography —
  new-node → delete-node → connect-tree → extend-tree → split-tree →
  change-root → merge-trees → assert-edge → retract-edges. Run's doc comment:
  sequential execution is "akin to the real-world use of an engine over time"
  (l.243-247).
- `locateSource()` (l.324) captures file:line via `runtime.Caller` per case;
  Run logs "Read the source for test-case %v at %v" (l.267) because failures
  surface in the CONSUMER's test run but the case lives in the HARNESS's
  source. Comment above: "We'd put a lot of effort into making this suite
  readable and understandable" (l.264-266).
- Checks are functional values: `type check func(digitaltwin.GraphChanged) (problem string)`
  (checks.go:13). They RETURN problems; only Run talks to `*testing.T`.
  Variadic constructors `created(...)/updated(...)/removed(...)` double as
  emptiness assertions when called bare: `created()` means "nothing created".
- `snapshot.Checks(before)` (checks.go:114) derives continuity checks
  (GraphBefore/GraphAfter forest hashes, non-zero Timestamp) so every case
  gets whole-state verification beyond its explicit checks.
- Failure text is Go-expression style with go-cmp:
  `len(.Created) = %v, want %v`; `Created mismatch (-want +got):\n%v`.
- Documented DELIBERATE non-features: no ctx parameter ("neutral conditions...
  the intention is to test the correctness of operations, not their
  performance", l.237-241); package doc closes with scope: engines "are
  encouraged to perform additional tests which are specific to the underlying
  graph engine" (l.25-26) — the harness owns the shared contract only.

## dbtest API shape (internal/dbtest/)

- `SetupNeo4j(t *testing.T) neo4j.DriverWithContext` (neo4j.go:48): t.Helper();
  `testing.Short()` skip BEFORE `t.Parallel()` (ordering was its own commit,
  e70ac73); `t.Cleanup` for container termination AND driver close (LIFO);
  bounded connectivity retries with logged attempts.
- Doc comment defines a "standard" instance and warns: depending on
  customisation means "you depend on a deployment detail no-longer considered
  'standard' and thus may break in production too" (neo4j.go:43-47).
- Developer-experience flag `-dbtest.inspect` (flags.go:16): on failure the
  container stays up, browser + bolt URLs are logged, teardown waits for
  Ctrl+C. Harnesses serve debugging humans, not just CI.
- Package doc: "intended to be used in tests only. It is not suitable for
  production use" (doc.go:16-17). Lower-level util takes `testing.TB`
  (options.go:13) while the entry point takes `*testing.T`.

## Committed suites vs current INTENT rulings

CONFIRMS (production code matching 2026-07 standards):
- Field=Value subtest naming: `contentaddress_test.go:35-56`
  ("types=same,values=different"); machine-derived names
  `t.Run(typ.String(), ...)` (l.239).
- One continuous test with in-function `t.Helper()` closures instead of
  forced subtests: `TestContentAddress_reflectionOrder` (l.154-176).
- Loud drift, never silent: known bug kept visible as a passing case with
  `// TODO: left == right; because ContentAddress does not consider the type
  of the inner field` (contentaddress_test.go:69); honest skips
  `t.Skip("Skip until @danielorbach figures out...")` + `// TODO: ... WTF?!?`
  (builder_test.go:11,23); `t.Skip("JSON marshalling is not supported (yet?)")`
  (digitaltwin_test.go:112).
- Stdlib as exemplar: `// based on stdlib strings/builder_test.go`
  (builder_test.go:9,29).
- Interrelated const/table values get interplay comments:
  `{"tiny", 128}, // increase load by factor of 64` ladder
  (contentaddress_test.go:363-369).
- Examples as package spec: every public package has example_test.go;
  scenario-suffixed names (`ExampleCompilation_anonymous`,
  `ExampleRecorder_relationshipAssertions`, unadorned `Example()` in assert).
  Shared exported example fixtures (`printApplier`, `dummyNode`, `Person`/`Dog`)
  with user-voiced in-function comments ("Always embed this type to implement
  Value", example_test.go:19).
- Reuse over purpose-built: ONE global `marshalTests` table feeds
  TestGobMarshalling, TestJSONMarshalling, AND BenchmarkGobMarshalling
  (digitaltwin_test.go:12,89,134) — globals earn their keep via cross-function
  reuse.
- Input visualization: ASCII tree diagram opens TestInspect (walk_test.go:11-23).

PREDATES (own code violating own later rulings — cite as "standards are
refinements, not retrofits"):
- `// ExampleDisassembler an example [component.Descriptor]...`
  (disassembler_test.go:68) opens with a (stale, mismatched) function name —
  the exact TESTS.md BUG note.
- Example doc comments narrate the example function ("We demonstrate how to
  use the Recorder...", compilation/example_test.go:13-17; "This example
  illustrates... It shows the process of initializing...",
  attributemap_test.go:183-186) — the round-3 audience anti-pattern.
- `TestBuilderCopyPanic` collects panics over a channel from goroutines
  (builder_test.go:99-106) instead of asserting in-goroutine — inherited from
  stdlib, predates the concurrent-*testing.T refinement; benchmarks use
  ResetTimer/RunParallel, no b.Loop (pre-1.24 vintage).
- checks.go triples a near-identical body (created/updated/removed) and its
  comment typo ("Slices are not friendly to compare by maps are") three times.

## Canon quotes for a harness reference file

1. "Call enginetest.Run in its own test" — one entry point, one line of glue.
2. "Read the source for test-case %v at %v" — a harness must route foreign
   developers to the failing case's source (runtime.Caller altitudes).
3. "The intention is to test the correctness of operations, not their
   performance" — harness doc-comments its deliberate non-features.
4. "Akin to the real-world use of an engine over time" — conformance cases
   are a user flow, sequential and stateful, not symbol pins.
5. "Engines are encouraged to perform additional tests" — the harness owns
   the shared contract; consumers own their specifics.
6. dbtest's "standard instance" warning — harness defaults mirror production;
   depending on details beyond "standard" predicts production breakage.
