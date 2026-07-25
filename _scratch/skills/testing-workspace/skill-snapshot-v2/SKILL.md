---
name: testing
description: >-
  Sets the maintainer's standard for writing and reviewing committed Go test
  files (_test.go): canonical tests in the standard library and golang.org/x
  to imitate, plus explicit guidelines on what committed tests are for.
when_to_use: >-
  Load on any request to write, add, review, or fix Go tests, on any question
  about test naming or structure, at the earliest signal a _test.go file will
  change, when a package's testability is being designed, and the moment a
  test proves hard to write. Do not wait for the word "test": code without
  contract pins is unfinished, so nearly every Go code-writing task ends here.
---

# Committed Go Tests

This skill sets the standard for committed `_test.go` files. Its method is
imitation: each section pairs a guideline with canon to read and mimic first.

## Test freely; commit only the contract

There are many ways to test while developing: throwaway tests, tailored
executables, build-tagged experiments, exercising built artifacts as a user
or operator would. Do all of these, constantly. While work is uncommitted,
run the suite repeatedly, under constrained CPU if practical: a test that
flakes on a weak CI runner is a contract failure too. None of this gets
committed. The committed `_test.go` files carry one thing only: the
package's contract with its users. This lesson dissolves most coverage
arguments: a check that helped you debug is not thereby worth keeping.

Committed tests exist so future maintainers can change the package knowing
every committed promise still holds; any breaking change either fails the
tests or amends them to state the new contract. Tests that pin
implementation detail are the worst kind there is: they mark an
inexperienced author and burden every future change with false alarms. Nor
are unit tests bug hunts; they exist for rapid iteration against a contract.

Much of the contract lives in prose, in doc comments and usage instructions.
A good assertion traces to a documented sentence; if none backs it, the test
pins an accident or the docs miss a promise, so fix whichever is true.

## The unit is the package

Go's objects are packages, not files or symbols, unlike most popular
languages. Exercise each package as its user: usually other packages in the
same module, sometimes external users of the module. Never test a library
through a distributed CLI binary built from it; the CLI has its own corpus.

Aspire to fewer tests that cover more of the exported API, because users
compose the API rather than call it piecemeal. The standard library's hash
package tests are the model: study their naming and the single golden test
composing constructors, writes, and sums.

When asked to add a test for one symbol, first see whether it fits an
existing test as a sub-test or table row; usually it will not, and a
runnable example is the next reach. A request that is really a bug hunt
gets run, not committed (or committed provisionally, marked to be dropped).
A symbol extending the API as a sibling gets tested the way its siblings are.

## Names, bodies, and doc comments

A test name claims a property of the package in plain English, not a
symbol: TestWaitBlocksUntilRefill, not TestLimiterWaitBlocksUntilRefill;
TestInvalidNewLimiter, not TestLimiterNewPanics; TestMalformedLines, not
TestLoadRejectsMalformedLines. Receiver-type stutter and overly literal
names (TestTypedLookupRejectsUnparsableValues) are too much; a whole-package
scenario test named for the package, TestConfig for package config, is good.
Sub-test names built from description strings are not always worth it:
naming the inputs is often enough, and the best names complete a logged
sentence, as in "Load(%q) succeeded for a config with %s".

Test bodies are trivial, straight-line code read top to bottom: hard-coded
numbers, not algorithms, because that math is performed at coding time and
written as results; when the maths are delicate and unexpected, an adjacent
comment does the trick. No loops, no algos. A premature helper at this
trivial scale is a code smell that hides bigger misunderstandings.

Underscores mean nothing in test names except in example tests, where go-doc
reads them to attribute the example to a symbol. Never open a test function's
doc comment with the function's name: that convention belongs to exported
identifiers in ordinary code; on tests, benchmarks, fuzz functions, and
examples it is noise. Read the Go wiki's TestComments page for failure-message
conventions and TableDrivenTests for when many cases exercise one behavior.

## Lightweight forms

Trivial tests are often best captured as runnable examples: they document,
they execute, and they read as a user's own code. Package-level Example
functions, tied to no symbol, suit shallow APIs where one example walks the
typical flow; several small per-behavior examples can beat one big one and
improve the doc site. An ExampleLoad that just calls Load earns no commit.

For small dependencies, read io's multi and pipe tests: bespoke fakes of
three to eight lines, the contract under test stated in prose above the
assertion. A fake read in full keeps the test honest; no mock framework does.

Write benchmarks in the b.Loop form (testing.B.Loop, Go 1.24): the compiler
recognizes it and keeps the benchmarked call alive, where the old b.N
discard-loop can be optimized away entirely. Like TestComments, this canon
postdates most training data and deserves a deliberate read.

## Harnesses, fixtures, and concurrency

When a module offers an applicative layer, give it a helper testing package
that orchestrates scenarios respecting real user workflows: net/http/httptest
is the canon for that harness shape, and testing/fstest and testing/iotest
show the complementary shape of conformance verifiers and adversarial
wrappers. These packages are bespoke; write one per applicative layer.

Go's doc conventions (go.dev/doc/comment) presume a top-level function safe
for concurrent use and a type's methods not, unless the docs promise more.
Never commit a test pinning concurrency the docs do not promise; in review,
delete such tests rather than rewrite them.

Prefer testing/synctest (Go 1.25) for code with contexts and goroutines:
bubbles run on virtual time, so sleeping inside one is idiomatic and stable.
Use the least orchestration: synctest.Sleep(d) is exactly time.Sleep(d) then
Wait(), and an explicit Wait or extra goroutine earns its place only when
asserting effects of background goroutines you did not just sleep past.

When a user-facing package does file I/O, keep input and expected output in
one reviewable fixture, preferring testdata/ files over long string constants
(a const is fine at trivial scale). The Go codebase uses txtar; cmd/go's
testdata/script corpus and x/tools' analysistest show fixtures kept beside
their assertions. The goal is one fixture a reviewer reads whole.

## Reviewing an existing suite

Defective test structure is breakage: per-symbol grain, self-naming comments,
dev scaffolds, and change-detector tests re-asserting a constant fixture
contribute nothing beyond blunt coverage. Consolidate or delete them; "don't
break anything" means the package's contract, not the test file's shape.

## Testability is a design signal

Never assert error strings, least of all for errors this module controls:
return typed or sentinel errors inspected via errors.Is and errors.As. A
test that can only grep a message is a redesign signal aimed at the API.

Design packages that play together in testable ways: prefer io interfaces
over concrete file structs, and accept readers and writers rather than
paths. When a test proves hard to write, treat the difficulty as a signal
to redesign, rethink, or try harder, never as a checklist item to get past
quickly. When no canon above fits, browse golang.org/x for inspiration.
