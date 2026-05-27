---
status: draft
date: 2026-05-27
---

# A3. Parameterization: Blueprints and Instances

## Context

The framework's defining need is **parameterization**: declaring that a
component has knobs and that a host process binds values to those knobs at
runtime. Replicas (the same component running multiple times with different
shard IDs, tenant scopes, port assignments) are one parameterization pattern;
database URLs, log levels, feature flags, and listener addresses are others.
This analysis frames the parameterization need, surveys ecosystem treatments,
and develops the working three-value model that subsequent analyses build on.

The model does not force where parameter values come from. That belongs to the
loader, framed in [A2](A2-architecture.md). This analysis stays focused on the
shape of the parameterized thing.

## Unix Framing

Unix gives precise terms that this analysis adopts:

- A **program** is the binary artifact, the executable file on disk.
- A **process** is one runtime instance of a program, with its own PID,
  memory, file descriptors.
- The `fork`/`exec` idiom expresses "one program, many processes."

The framework extends this with a third term:

- A **blueprint** (working name: `Descriptor`) is a compile-time template
  inside the program. It declares the component's name, its parameter slots,
  and its work function.
- An **instance** (working name: `Instance`) is a runtime parameterization of
  a blueprint. It pairs a blueprint with concrete values bound to its slots,
  plus identity (the instance's ID within the process).

The framework's contribution to the idiom: in addition to "one program, many
processes," a single process can host "many instances per blueprint."

## Survey of Parameterization in the Ecosystem

The need is universal; the carriers vary.

**12-factor (factor III, "Config").** Configuration lives in the environment.
The application reads named environment variables. Parameter slots are
implicit (the variables the app reads); the schema lives in documentation, not
in code.

**Kubernetes.** `ConfigMap` and `Secret` resources carry key-value data,
mounted as files or projected as environment variables. The Downward API
surfaces pod metadata (name, namespace, labels) as files or environment
variables. Container spec `args` and `env` declare what the container's main
process sees on its command line and in `os.Environ`.

**Helm and Terraform.** Parameterized templates with values files. Template
syntax declares slots; values files supply bindings. The template engine
renders concrete artifacts.

**Go's standard `flag` package.** Parameters declared in code
(`flag.StringVar(&s, "name", "default", "usage")`). The `FlagSet` is the
parameterization surface. `flag.Parse` binds values from `os.Args`.

**`peterbourgon/ff/v4`.** Layered sources (CLI flags, environment, config
files) with documented priority; per-`Command` `FlagSet`; `ff.Parse`
orchestrates the layering.

## Common Ground

Across the inspirations, three properties recur:

1. **Slot declarations are authored by the component author**, in code or in a
   template or in a CRD schema.
2. **Bindings come from outside the component**: environment, file, CLI,
   control plane.
3. **The framework or runtime is responsible for the binding**, not the
   component.

## Forces

The need plus the survey surface four forces this analysis records for the
requirements:

- Slot declarations must be **inspectable by the framework** so help, list,
  validation, and completion can read them ([A7](A7-documentation.md)).
- Bindings must be **late**: the loader chooses where values come from, after
  the component author has declared what the component needs.
- **Multiple instances** of one blueprint must be possible in one process,
  each with independent slot bindings (the [A1](A1-scope.md) scope demands
  this).
- Parameterization must **not require global mutable state**.
  `flag.CommandLine` and process-wide config singletons are disqualified by
  the multi-instance requirement.

## A Working Three-Value Model

The forces above admit one shape that subsequent analyses depend on (without
locking names):

1. **The blueprint** (`Descriptor`). A registered, immutable value carrying
   the component's name, its instantiation function, and metadata
   ([A6](A6-metadata.md)). One blueprint per registration call.

2. **The instance** (`Instance`). A per-replica value carrying identity (name
   copied from the blueprint, ID assigned by the loader) and the slot
   declarations (working representation: a `*flag.FlagSet` owned by this
   instance, not shared with any other). One instance per replica.

3. **The runner** (`Runner`). A value (or a function adapted to a
   single-method interface, developed in [A4](A4-component-contract.md))
   implementing `Run(ctx) error`. Constructed by an instantiation function
   that receives the instance, declares slots on `Instance.Flags`, binds
   closure-captured locals, and returns the runner.

### The Instantiation Function's Discipline

For the model to hold, the instantiation function must obey four constraints.
Each one is load-bearing for some other analysis.

- **Pure declaration.** Registers slot declarations, captures locals by
  closure, returns the runner. No I/O, no validation against bound values
  (because no values are bound at instantiation time).
- **Infallible.** Help rendering instantiates without intent to run; failure
  modes that depend on bound values surface at `Run`, not at instantiation
  ([A7](A7-documentation.md)).
- **Receives only the instance.** No context, no ambient services. The
  instance carries no I/O; the closure receives ambient services through
  `ctx` at `Run` time ([A8](A8-ambient-services.md)).
- **Safe to call multiple times.** Help may instantiate; tests may
  instantiate; the loader instantiates once per replica. Each call produces an
  independent runner with independent slot state.

## What the Model Does Not Force

Subsequent analyses and the requirements pick these up:

- **Where parameter values come from.** The loader chooses
  ([A2](A2-architecture.md)).
- **How values are presented to the user.** The loader chooses (CLI
  subcommand, env-only, file-driven).
- **How instances are enumerated.** The loader chooses (one per program, N
  from config, N from a command tree).
- **Validation timing.** The instantiation function does not validate.
  Validation is part of `Run`, or is performed by the loader between binding
  and `Run`.

## The Replica Special Case

Replicas (multiple parameterized instances of one blueprint) fall out of the
model: the loader creates N instances from one blueprint, binds each
instance's slots from a different input subtree, and runs each runner in a
separate goroutine. Nothing in the model is replica-specific. The replica use
case is what motivates the per-instance `FlagSet` (versus a shared
`flag.CommandLine`), but the same shape serves single-instance use as well.

## Open Questions

- Whether the slot declarations remain a `*flag.FlagSet` or whether a richer
  slot model (typed unmarshalers, structured values) supersedes it. The
  current direction keeps `*flag.FlagSet` as the substrate, with the
  `parameter` package's decorators (boolean, required, future JSON-aware)
  sitting on top. Future analyses or requirements may revisit.
- Whether instances carry baggage ([A6](A6-metadata.md)) of their own
  (loader-set, instance-specific values) in addition to the blueprint's
  baggage. Currently deferred; the same mechanism extends to the instance if
  a use case arises.
- Whether the framework offers helpers for late validation (a `Validate(ctx)`
  pass between binding and `Run`). Currently deferred; the runner's first
  statement can validate, but a framework-level hook may earn its place
  later.
