---
name: testing
description: >-
  Sets the maintainer's standard for writing and reviewing committed Go
  tests (_test.go): the golden hierarchy of contract authority, the
  go-doc-first design and review mode, and standard-library canon to imitate.
when_to_use: >-
  Load on any request to write, add, review, design, or fix Go tests, on
  any question about test naming or structure, when a test suite or a
  package's testability is being designed or reviewed, at the earliest
  signal a _test.go file will change, and the moment a test proves hard to
  write; begin in go-doc-first mode, planning tests from the exported docs
  before opening source. Do not wait for the word "test": code without
  contract pins is unfinished, so nearly every Go code-writing task ends here.
---

# Committed Go Tests

The method is imitation: each section pairs guidelines with canon to read first.

## Test freely; commit only the contract

Test freely while developing: throwaway tests, tailored executables,
build-tagged experiments, exercising built artifacts as a user would. Do
all of these constantly, and run the uncommitted suite repeatedly under
constrained CPU when practical: a flake on a weak CI runner is a contract
failure too. None of it gets committed: committed `_test.go` files carry
only the package's contract with its users, so maintainers can change the
package knowing every promise still holds; a breaking change either fails
the tests or amends them to state the new contract. Coverage arguments
dissolve here: a check that helped you debug is not thereby worth keeping,
and unit tests are not bug hunts but rapid iteration against a contract.

## The golden hierarchy

The package's prose is the most sacred part of its contract: the package
doc and exported doc comments, what godoc publishes and users rely on.
Authority descends through the exported signatures, the code's behavior,
and unexported comments, down to the existing tests: the least golden thing
in the room, always the artifact under judgment, never the standard.

Prose-first separates contract tests from change-detectors, the worst tests
there are, burdening every change with false alarms: a test authored by
reading the implementation pins the implementation, bugs included, a
change-detector by construction, while one authored from the prose pins
what was promised. A prose-vs-code disagreement is a finding, not a silent
choice: ask the owner when there is one; unowned, side with prose,
deliberate promise-making where code drifts under maintenance, unless
history shows the code moved deliberately and the prose lagged, then side
with the code and fix the prose. Never silently absorb the code's current
behavior into a test.

The go-doc-first mode makes the hierarchy operational: draft the test plan
from go doc -all alone (prose and signatures, no bodies) before opening any
source file, so every collision between draft and code is surfaced drift.
It is the authoring-time counterpart of the external test package, which
enforces the user's seat at compile time; sub-agents can enforce the
blindness structurally, and the machine-generated input needs no curation.

## The unit is the package

Go's objects are packages, not files or symbols, unlike most popular
languages. Exercise each package as its user: usually other packages in the
same module, sometimes external users of the module. Never test a library
through a distributed CLI binary built from it; the CLI has its own corpus.

Aspire to fewer tests that cover more of the exported API, because users
compose the API rather than call it piecemeal; the standard library's hash
tests are the model, one golden test composing constructors, writes, and
sums. Placement tells a story too: foundational special-case contracts (the
zero-burst test) come first, verifying "contracts that we can later use for
more complex" proofs before other tests lean on those behaviors as technique.

When asked to add a test for one symbol, first see whether it fits an
existing test as a sub-test or table row; usually it will not, and a
runnable example is the next reach. A bug-hunt request gets run, not
committed. A symbol extending the API gets tested the way its siblings are.

## Names, doc comments, and bodies

A test name claims a property of the package in plain English, not a
symbol: TestWaitBlocksUntilRefill, not TestLimiterWaitBlocksUntilRefill; a
whole-package scenario test named for the package, TestConfig for package
config, is good. Underscores mean nothing in test names except in examples,
where go-doc reads them to attribute the example to a symbol. Sub-test
names built from descriptions are not always worth it: naming the inputs is
often enough, never repeated into log names the runner prints anyway, and
the best names complete a logged sentence ("Load(%q) succeeded for a config
with %s"). Panics take a table plus a recover helper whose failure message
names the inputs and their specific invalidity from the case struct.

Every test's doc comment gives the author's rationale: why the test is
needed, and why this mechanism when non-trivial, sometimes echoing the
prose contract, always in layman's terms, because "the reasoning and
rationale of the author is not" self-explanatory; don't be posh, since "an
LLM agent that feels presumptuous is the worst". A missing doc comment is
as wrong as opening one with the function's name, a convention that belongs
to exported identifiers, not tests. Line comments say what happens at that
line ("// ten refills against a bucket that holds two"); methodology
belongs in the function doc, not scattered inline. Failure messages follow
the Go wiki's TestComments; many cases of one behavior, TableDrivenTests.

Test bodies are trivial, straight-line code read top to bottom: hard-coded
numbers, not algorithms, because that math is done at coding time and
written as results; a delicate result takes an adjacent comment. "We prefer
more lines of code, if all these lines are meaningful": inline trivial
sub-tests into one linear test, and simplify cumbersome test code even at
the cost of repeated lines, since clarity over cleverness is the Go idiom.
Helpers need justification; a premature one saving three lines is a smell
that hides bigger misunderstandings.

## Lightweight forms

Source code speaks to maintainers; docs and examples speak to users; choose
the channel accordingly. Trivial tests are often best captured as runnable
examples, which document, execute, and read as a user's own code; several
small per-behavior examples can beat one big one, a package whose contract
is concurrent safety rather than orchestration is still a classic Example
fit with plain string outputs, and an ExampleLoad that just calls Load
earns no commit. Idempotence and linked contract aspects are tested in
usage, not isolation: a lone no-panic double-Close test proves little; an
Example that defers Close and also calls it explicitly proves it naturally.

For small dependencies, read io's multi and pipe tests: bespoke fakes of
three to eight lines, the contract under test stated in prose above the
assertion. A fake read in full keeps the test honest; no mock framework
does, and no new third-party test dependency (go-cmp included) enters
unless the module already carries it.

Two pieces of canon postdate most training data and deserve a deliberate
read. Write benchmarks in the b.Loop form (testing.B.Loop, Go 1.24): the
compiler recognizes it and keeps the benchmarked call alive, where the old
b.N discard-loop can be optimized away entirely. t.Context() (Go 1.24,
api/go1.24.txt, #36532) is the default context in tests: it is canceled
just before Cleanup runs, so context-shutdown resources drain in time.

## Harnesses, fixtures, and concurrency

When a module offers an applicative layer, give it a bespoke helper testing
package (one per layer) orchestrating scenarios that respect real user
workflows: net/http/httptest is the canon harness shape; testing/fstest
and testing/iotest show conformance verifiers and adversarial wrappers.

Go's doc conventions (go.dev/doc/comment) presume a top-level function safe
for concurrent use and a type's methods not, unless the docs promise more:
never commit a test pinning concurrency the docs do not promise; in review,
delete such tests rather than rewrite them. Prefer testing/synctest (Go
1.25) for contexts and goroutines: bubbles run on virtual time, so sleeping
inside one is idiomatic and stable. Use the least orchestration: never
spawn a goroutine whose only job the main goroutine could do directly
(waiting on a goroutine that calls Wait is calling Wait), and never contort
a test to dodge a hang, because unit tests complete faster than humans
notice, so "hang is a valid test failure".

When a user-facing package does file I/O, keep input and expected output in
one reviewable fixture, preferring testdata/ files over long string
constants; cmd/go's script corpus and x/tools' analysistest keep txtar
fixtures beside their assertions. The fixture file speaks: its comments
carry real information for both audiences ("Comments are the rest of the
line following the # sign", never "This is a comment"), with no warning
tone; trust the common-sense of contributors. Malformed-case files want
minimal scaffolding and elegant dogfooding: a separated format whose first
line states why the case should fail, printed when it unexpectedly passes.

## Review and redesign

Defective test structure is breakage: per-symbol grain, self-naming
comments, dev scaffolds, and change-detector tests re-asserting a constant
fixture contribute nothing beyond blunt coverage. Consolidate or delete
them; "don't break anything" means the package's contract, not the test
file's shape. Finish any authoring or review pass by re-reading the test
file top to bottom against the package's prose: one honest final pass
catches the recurring misses of self-naming or missing doc comments,
missing examples, and case-generating ceremony.

Testability is a design signal. Never assert error strings, least of all
for errors this module controls: return typed or sentinel errors inspected
via errors.Is and errors.As, because a test that can only grep a message is
a redesign signal aimed at the API. Prefer io interfaces over concrete file
structs and readers and writers over paths, so packages play together in
testable ways. A test that proves hard to write is a signal to redesign,
rethink, or try harder; when no canon above fits, browse golang.org/x.
