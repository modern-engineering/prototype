---
name: testing
description: >-
  Guides writing and reviewing committed Go test files (_test.go) to the
  maintainer's standard. Works by pointing at canonical tests in the standard
  library and golang.org/x to read and imitate, and by carrying the
  maintainer's explicit guidelines on what committed tests are for.
when_to_use: >-
  Load on any explicit request to write, add, review, or fix Go tests, and on
  any question about test naming or structure. Load proactively at the
  earliest signal that a _test.go file will be created or modified in a Go
  repo, when a new package's testability is being designed, and the moment a
  test proves hard to write. Do not wait for the word "test": in a Go
  codebase, committed code without contract pins is unfinished work, so
  nearly every code-writing task ends here.
---

# Committed Go Tests

This skill sets the standard for `_test.go` files that get committed. Its
method is imitation: each section pairs the maintainer's guideline with
canonical tests to read and mimic. Read the matching canon before writing.

## Test freely; commit only the contract

There are many ways to test while developing: throwaway tests, tailored
executables, build-tagged experiments, exercising built artifacts as a user
or operator would. Do all of these, constantly. None of them gets committed.
The committed `_test.go` files carry one thing only: the package's contract
with its users. This lesson comes first because it dissolves most coverage
arguments: a check that helped you debug is not thereby worth keeping.

Committed tests exist so future maintainers can change the package knowing
that every committed promise to its users still holds; any breaking change
either fails the tests or amends them to state the updated contract. They
are not change-detectors. Tests that pin implementation detail are the worst
kind there is: they mark an inexperienced author and burden every future
change with false alarms. Nor are they bug hunts; unit tests exist to let
maintainers iterate rapidly against a stated contract, not to fish for bugs.

Much of the contract lives in prose: doc comments and the instructions
around function usage. A good assertion traces to a documented sentence. If
no sentence backs it, either the test pins an accident or the docs are
missing a promise; fix whichever is true.

## The unit is the package

Go's objects are packages, not files or symbols, unlike most popular
languages. Exercise each package as its user: usually other packages in the
same module, sometimes external users of the module. Never test a library
through a distributed CLI binary built from it; the CLI has its own corpus.

Aspire to fewer tests that cover more of the exported API. The best test
names describe several exported symbols working together, not one function
each, because users compose the API rather than call it piecemeal. The
standard library's hash package tests are the model: study their naming and
the single golden test that composes constructors, writes, and sums.

## Names and doc comments

Underscores in test names mean nothing except in example tests, where
go-doc reads them to attribute the example to an exported symbol.

Never open a test function's doc comment with the function's name. That
convention belongs to exported identifiers in ordinary Go code; on tests,
benchmarks, fuzz functions, and examples (runnable or not) it is noise.
Unlearn the habit.

Read the Go wiki's TestComments page for its conventions on failure
messages and test prose, and its TableDrivenTests page for the idiomatic
shape when many cases exercise one documented behavior.

## Lightweight forms

Trivial tests are often best captured as runnable examples: they document,
they execute, and they read as a user's own code, a balance many simple
contracts deserve.

For small dependencies, read io's multi and pipe tests: bespoke fakes of
three to eight lines, with the contract under test stated in prose above
the assertion. A fake you can read in full keeps the test honest in a way
no mock framework matches.

## Harnesses, fixtures, and concurrency

When a module offers an applicative layer, give it a helper testing package
that orchestrates scenarios respecting real user workflows.
net/http/httptest is the canon for that harness shape; testing/fstest and
testing/iotest show the complementary shape of conformance verifiers and
adversarial wrappers. These packages are bespoke; write one per applicative
layer the module provides.

Prefer testing/synctest (Go 1.25) for code with contexts and goroutines:
read it for virtual time and durably-blocking semantics, which make
concurrency tests deterministic instead of flaky and slow.

When a user-facing package does file I/O, keep input and expected output in
one reviewable fixture. The Go codebase uses txtar for this; cmd/go's
testdata/script corpus shows CLI transcripts with fixtures beside their
assertions, and x/tools' analysistest shows expectations annotated beside
fixtures. The mechanism can vary; the goal, one fixture a reviewer reads
whole, is what matters.

## Testability is a design signal

Design packages that play together in testable ways: prefer io interfaces
over concrete file structs, and accept readers and writers rather than
paths. When a test proves hard to write, treat the difficulty as a signal
to redesign, rethink, or try harder, never as a checklist item to get past
quickly. When no canon above fits, browse golang.org/x for inspiration.
