---
name: testing
description: Designs committed Go test files as user-contract suites following Go team practice - fewer scenario tests covering more exported surface together, runnable examples, bespoke harnesses, testing/synctest for concurrency, and golden/script corpora with update flows. Distinguishes committed contract pins from throwaway development probes.
when_to_use: Triggers on explicit requests like "write tests", "add a test", "review these tests", "improve test coverage", or questions about test naming, structure, or what to test. Loads proactively at the EARLIEST signal that a _test.go file will be created or modified in a Go repo; when designing a new package whose testability is not yet settled (seams are cheapest before the API calcifies); when a test proves hard to write (that is a design signal this skill interprets); and when choosing between a unit test, an example, a harness, or a CLI script corpus. Do not wait for the word "test" - in a Go codebase, committed code without contract pins is unfinished work.
---

# Testing in Go

Committed tests exist so a future maintainer can change the package with a
guarantee: every contract the package holds with its USERS is upheld, and any
breaking change either fails a test or forces a visible test amendment. That
single purpose decides everything else — what to test, from where, how many,
and what never gets committed.

## Core principles

1. **Test from the user's seat.** The package's users are sibling packages in
   the module and external importers — not the CLI binary built from `cmd/`.
   Prefer `package foo_test`; punch the minimum peephole back in via
   `export_test.go`. Library contracts are pinned in the library's tests; the
   CLI's shell-observable contract is pinned in the CLI's own corpus.
2. **Fewer tests, each covering more.** The unit of testing is an invariant,
   not a function. One scenario test composes several exported symbols the way
   a user would (the `hash` model: constructors + writes + sums + marshaled
   state in one golden test). If you cannot say what breaks when a test fails,
   the test is misnamed or shouldn't exist.
3. **Contract, not bugs.** Unit tests assert the contract with callers so
   iteration stays fast. Part of that contract lives in doc comments — a
   subtle assertion should trace to a documented sentence, and vice versa.
   A regression test is promoted to the property it protects; the issue
   reference demotes to a comment.
4. **Many ways to test; few get committed.** Development probes, tailored
   executables, build-tagged experiments, running artifacts as an operator —
   do all of it while iterating, then throw it away. Commit only the contract
   suite. A committed scaffold is a contract nobody promised.
5. **Hard-to-test is a design signal.** When stating a contract requires disk,
   processes, sleeps, or reaching into internals, redesign the surface first:
   io/fs interfaces over concrete files, injected seams (writer, signal
   channel, listener), `run(ctx, in, out)` under the process skin. Disk
   appears in tests only when disk IS the contract. Treat this as
   redesign/rethink/try-harder — never a checklist item to get past.
6. **Trivial assertions become runnable Examples** — they test and document at
   once, and godoc shows them. Underscores are meaningless in test names
   except in Example names, where they attribute the example to a symbol.
7. **Doc comments of test functions never open with the function's own name.**
   This holds for tests, benchmarks, fuzz targets, and examples alike. Open
   with the contract clause: "Shutdown passes the caller's context to every
   Shutdowner, alive." — not "TestShutdownHonorsContext pins...". Models
   habitually get this wrong; re-read your comments before committing.
8. **Concurrency and time go through `testing/synctest`** (stable since Go
   1.25): fake clock, durably-blocking semantics, wait-before-read accessors.
   No sleeps, no poll loops, no real sockets in bubbles. Exactly one
   real-signal/real-socket kernel-integration pin per boundary is allowed,
   marked as such.

## Choosing the shape

| The contract is... | Commit this shape |
|---|---|
| Input→output mappings | One table under one runner; case ID in every failure message |
| Interacting symbols / lifecycle | Bespoke scenario test with 3-8 line fakes, prose contract stated above it |
| An interface others implement | Conformance kit: `func TestX(impl, ...) error`, no `*testing.T`, errors joined, subtests named after doc clauses |
| A repeated applicative layer | Bespoke harness (`httptest` shape): takes `testing.TB`, registers Cleanup, self-tested |
| Serialized bytes users persist | Byte-exact goldens pinned inline or via `-update` flow; changing the format must edit a golden |
| A CLI's shell-observable behavior | Script corpus (txtar transcripts): fixture sits beside its assertion |
| A trivial or teaching case | Runnable Example |

## Anti-patterns (each observed in the wild; evidence in references)

- Per-function test grain mirroring the API list instead of its use.
- The same contract clause pinned in several suites — a reword forces
  synchronized edits; pin once, at the layer that owns the wording.
- Sleep-based concurrency tests; readiness by delay instead of by signal.
- Tests reaching into internals for claims the exported surface can state.
- Committed dev scaffolds: kind sweeps, check-manually logs, assert-nothing
  benchmarks, placeholder TestNoop.
- Test doc comments that name their function; underscore names outside
  Examples.

## References

Canonical sources you already know — consult by name, no copy needed:
go.dev/wiki/TestComments, go.dev/wiki/TableDrivenTests, the stdlib `hash`,
`io`, `strings`, `time` test suites, `net/http/httptest`, `testing/fstest`,
`testing/iotest`, `testing/synctest`, cmd/go's `testdata/script`, and
x/tools' `analysistest`.

Surveyed evidence with file:line anchors (read when designing, not after):

| File | Contains | Load when |
|---|---|---|
| `references/stdlib-doctrine.md` | The 13-point doctrine distilled from hash/io/strings/time with anchors | Writing or reviewing any test file |
| `references/harnesses-and-seams.md` | Harness API shape, design seams, conformance kits, synctest patterns, txtar | Building a harness; testing concurrent/lifecycle code; a package resists testing |
| `references/cli-corpora.md` | Script-test anatomy, golden pairs, expectations-beside-fixtures, what stays Go | Testing a CLI or compiler-like layer with positioned diagnostics |
