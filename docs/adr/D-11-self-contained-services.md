---
status: proposed
since: 2026-07-09
supersedes:
  - D-06-declarative-descriptor-values
---

# Construct Application Instances as Self-Contained Services

## Context and Problem Statement

[D-06 Declarative Descriptor Values] settled how applications are identified:
descriptor values defined by filling exported fields, referenced directly,
registered nowhere. It also sketched how they are constructed — an instantiation
function receiving an instance remote collector that holds the per-instance flag set —
and that sketch has not survived contact with newer iterations. Building
toolchains against the model showed the remote collector earns nothing: every consumer
that held an instance immediately wanted the constructed unit, and every
constructed unit already had to know its own flags. A revised shape emerged in
practice; this record supersedes [D-06 Declarative Descriptor Values] to
re-decide identity and construction together, keeping what held and replacing
what proved over-engineered.

## Decision Drivers

- The dry-instantiation invariant carries the toolchain: help rendering, schema
  extraction, and solution compilation all need the parameter surface without
  running anything ([A-07 Component Documentation];
  [A-14 Staged Catalogue Compilation]).
- Multi-instance hosting must stay trivial: one application, many instances, no
  shared mutable state ([A-02 Architecture]).
- Fewer moving parts: each concept must pay for itself; remote collectors that only ferry
  data between two neighbours do not.
- Everything [D-06 Declarative Descriptor Values] got right must survive: values
  over registries, direct reference, explicit collection.

## Considered Options

- **Instantiation function over an instance remote collector.**
  `New(inst
  *Instance) Runner`; the remote collector holds identity and the
  per-instance flag set (the shape D-06 sketched).
- **Self-contained service values from a pure factory.** `Make() Service` where
  `Service` is a `Runner` that exposes its own `Flags() *flag.FlagSet`; the
  constructed value is the instance.
- **Runner-only descriptors with an external flag registry.** Construction
  returns a bare `Runner`; parameter surfaces live in a side table keyed by
  descriptor.

## Decision Outcome

Chosen option: **self-contained service values from a pure factory.**

A `Descriptor` remains a declarative value — identity fields plus constructors,
no `init()`, no registration, referenced directly:

- **`Make func() Service`** is the pure factory and the invariant remote collector: a
  fresh `Service` per call, flags declared, nothing else — no I/O, no failure,
  safe to call any number of times. Every catalogue citizen sets it. Dry
  instantiation is `Make().Flags()`.
- **`Service`** is a `Runner` with `Flags() *flag.FlagSet` (working name;
  `Application` is the candidate rename). The service value IS the instance: its
  flag set is its parameter surface, hosting N instances means calling `Make` N
  times, and no separate instance remote collector exists. Instance NAMES are not the
  application's concern — they belong to whoever composes instances (loaders,
  the solution layer's records).
- **`New func(context.Context) (Service, error)`** stays as the optional
  fallible, context-aware run-time construction path for applications whose real
  construction performs I/O; tooling never calls it.
- **Adapters** carry the common shapes: `MakeFunc` for
  `func() (Runner,
  *flag.FlagSet)` constructors, `MakeFor[T]` lifting a user
  type whose pointer implements `Service` (fresh zero value per call), `Main`
  for flagless functions.

Carried forward from [D-06 Declarative Descriptor Values] unchanged: the
definition is a package-level struct literal; relationships are ordinary Go
references; name-to-descriptor resolution happens only at an explicit `Set`
assembly, the single seam where relational invariants are checked. The sketched
`Requires` edges are dropped with the remote collector — no consumer materialized for
them — and return, if ever, through the Set.

### Consequences

- Good, the invariant that tooling leans on is one field's contract (`Make`),
  testable by a small citizenship harness.
- Good, instance-hood needs no vocabulary: fresh value per call gives isolation
  by construction, and the type system already enforces the surface (`MakeFor`
  compiles only for types that implement `Service`).
- Good, the compiler and the help system read the exact `flag.Value`
  implementations the running application parses with.
- Bad, a `Service` interface sits between `Runner` and every adapter; flagless
  runners must say so (`Main` returns a nil flag set) rather than simply being
  runners.
- Bad, superseding a proposed record this early costs a hop for readers of D-06;
  the alternative — silently diverging code — costs more.

## More Information

Open paths: the `Service`→`Application` rename waits for a requirement to lock
names; `Requires`-style structural edges return through the Set if a consumer
materializes; whether `New` earns its keep is judged when the first host rung
lands (its drop trigger: no loader calls it).

[A-02 Architecture]: ../analyses/A-02-architecture.md
[A-07 Component Documentation]: ../analyses/A-07-documentation.md
[A-14 Staged Catalogue Compilation]: ../analyses/A-14-staged-catalogue-compilation.md
[D-06 Declarative Descriptor Values]: D-06-declarative-descriptor-values.md
