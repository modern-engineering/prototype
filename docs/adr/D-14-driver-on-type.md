---
status: proposed
since: 2026-07-13
refines:
  - A-11-substrate-slices
  - D-11-self-contained-services
---

# Tie Provisioning Drivers to the Declaring Type

## Context and Problem Statement

[A-11 Substrate and Slices] gave provisioning its two kinds — slices owned,
attachments verified — and put the output scheme on the provisioning type, but
left "where drivers live" an open question. [D-13 Two-Phase Enactment] gives
provisioning its moment: a PROVISION phase every host runs before any
application starts. What no record fixes is the code. Until now a provision type
was declaration alone — a bare function filling a flag set for the compiler to
validate against, an output list of names — with nothing executable behind
either: a host reaching a provision record had nothing to call, and nothing said
what a reported output was worth to the parameter it feeds.

The two gaps are one decision. The driver produces the outputs, so where the
driver lives fixes where the output contract can be declared, enforced, and
tested. This record answers both with the move [D-11 Self-Contained Services]
already made for applications: the declaring value carries its own execution,
and the contract around it — parameters in, typed outputs out — is data on the
same value.

## Decision Drivers

- Declaration and execution drift when they live apart: a driver maintained away
  from its type validates against yesterday's parameter surface, and the
  divergence surfaces only at a site, at enactment.
- The dry-instantiation invariant carries the toolchain
  ([D-11 Self-Contained Services]): the compiler needs the parameter surface
  without running anything, the host needs the same surface wet, and what checks
  dry must be what runs wet ([A-14 Staged Catalogue Compilation]).
- No registries: identity is a value referenced directly, registered nowhere
  ([D-06 Declarative Descriptor Values], upheld by
  [D-11 Self-Contained Services]); the driver seam must not reintroduce a
  name-keyed lookup table.
- Outputs feed typed parameter slots; a wrong output should fail at the boundary
  that produced it, not surface later as an application's parse error at bind
  time.
- The single-process litmus ([A-09 Solution Layer]): the playground host calls
  drivers in-process, so a seam demanding external executables or sidecars is
  disqualified.

## Considered Options

- **Site-supplied outputs, no driver.** Provisioning executes nothing; site
  configuration supplies every output value, and provision records are
  bookkeeping. An early host sketch of this layer did exactly this.
- **A host-side driver registry.** Each host maps type identity to driver code —
  an `init()`-registered or hand-maintained table curated by the platform team.
- **Drivers as catalogue citizens.** Provisioning code enters the catalogue as
  components, and provisioning becomes deploying a special application.
- **The driver rides the declaring type.** The provision type value carries a
  pure factory for its driver, the exact shape [D-11 Self-Contained Services]
  gave applications.

## Decision Outcome

Chosen option: **the driver rides the declaring type**, with the output contract
typed alongside it. The seam, as a sketch:

```go
type ProvisionType struct {
    Doc     string
    Make    func() Provisioner // pure factory: fresh value, flags declared, no I/O
    Outputs []Output           // named, documented, typed
    Kinds   Kinds
}

type Provisioner interface {
    Flags() *flag.FlagSet                              // the parameter surface
    Attach(ctx context.Context, w *OutputWriter) error // v0: the only act
}
```

- **`Make` mirrors [D-11 Self-Contained Services] exactly.** A fresh provisioner
  per call, flags declared, nothing else — no I/O, no failure. The provisioner
  value is the provisioning instance: the compiler dry-instantiates
  `Make().Flags()` for its validation surface, and the host wet-binds the same
  flags before calling `Attach`, so the checking schema and the running schema
  stay one piece of code ([A-14 Staged Catalogue Compilation]'s doctrine, now on
  the provisioning side). A nil `Make` is legal — a parameterless, driverless
  declaration — and is refused at plan time ("declares no driver",
  [D-13 Two-Phase Enactment]), so a type may enter the catalogue before its
  driver exists and solutions still compile against it.
- **`Attach` is the whole verb set, v0.** The driver verifies and adopts what
  already exists and reports its coordinates — [A-11 Substrate and Slices]'s
  attachment, made runnable. Slice records still compile (the declaration is
  meaningful and the DAG is checked), but enactment refuses them until lifecycle
  verbs exist; a substrate type whose real driver has not landed may keep an
  `Attach` that fails with a teaching error, staying compilable while
  unenactable.
- **Outputs are typed scalars, pinned into the image.** An output declares its
  type from a small vocabulary — string, int, bool, duration; empty means
  string, the permissive default — and the compiler pins that type into the
  image's output schema, so consumers learn it without the live catalogue.
  Outputs remain [A-10 Value Binding]'s reconcile-time symbols; the type is what
  makes them checkable at every boundary they cross.
- **The host mediates output writing through a typed boundary.** `Attach`
  receives a writer constructed from the type's declared outputs; writing an
  undeclared name, the wrong kind, or the same output twice fails at the write
  (the flag-set precedent: fail at the boundary, not downstream), and after
  `Attach` returns, every declared output must have been written. The host reads
  typed values back and renders the flag-ready string when a parameter consumes
  the output.
- **The link checks reference sites where it honestly can.** Where an output
  reference feeds a parameter slot the linker can positively type — today that
  is boolean slots only, the one kind the standard flag machinery itself
  advertises — a mismatched output type is a positioned link error. Every other
  mismatch defers to wet binding, where the same `flag.Value` that would have
  parsed a literal parses the rendered output. The narrowness is deliberate:
  flags advertise no richer kind information, and inventing a parallel type
  declaration for parameters is the transcription drift
  [D-10 Generate-Compile-Run] compiles to avoid. The limit is a door below.

## Pros and Cons of the Options

### Site-Supplied Outputs, No Driver

- Good, no execution model at all: hosts stay pure binders, and any substrate
  nobody has automated can be stood in for by hand.
- Bad, nothing verifies anything: [A-11 Substrate and Slices]'s attachment
  promise is _verified_ access, and a copied-through site value verifies nothing
  — the first wrong endpoint is discovered by the application that fails to
  connect.
- Bad, outputs stop being reconcile-time facts and become site configuration,
  collapsing [A-10 Value Binding]'s third binding moment into its second; the
  slice kind can never appear (slices exist precisely because code must run), so
  the mechanism dead-ends at attachments.

### A Host-Side Driver Registry

- Good, platform teams hold an explicit gate over exactly what code touches
  their substrate.
- Bad, it is the name-keyed lookup [D-06 Declarative Descriptor Values] retired:
  a name resolving differently per host, collision on the name, and
  driver-against-type version skew invisible until a site enacts.
- Bad, every archetype maintains its own table, so a type usable under one
  controller silently lacks its driver under another — the per-archetype drift
  [A-13 Mission Analysis]'s coverage measure exists to catch.

### Drivers as Catalogue Citizens

- Good, one execution model for everything, and drivers inherit the library's
  full contract.
- Bad, the contract is wrong-shaped: provisioning is run-to-completion work
  against substrate, not a long-running service; termination and health
  capabilities are ballast, and [D-09 Catalogue Citizenship] scoped the
  catalogue precisely so its guarantees stay meaningful.
- Bad, provisioning-as-deployment erases the boundary [D-13 Two-Phase Enactment]
  just fixed: driver instances would appear in images beside the applications,
  and the PROVISION/DEPLOY order would need expressing as inter-record
  dependencies instead of a phase.

### The Driver Rides the Declaring Type

- Good, declaration and execution cannot drift: one package, one value, one
  release; where the type goes its driver goes, on every archetype alike.
- Good, the toolchain's invariant extends unchanged: the same
  citizenship-harness pattern that checks descriptors checks provision types —
  fresh distinct provisioners, matching flag schemas, valid typed outputs —
  keeping [D-09 Catalogue Citizenship]'s full-contract guarantee a failing test
  rather than a hope.
- Good, in-process by construction: the single-process host calls `Attach` as a
  method call, which is the in-memory injection posture [A-12 Operator I/O]
  gives the playground archetype.
- Bad, the declaring package links its driver's dependencies (an admin client, a
  database driver) into every consumer of the catalogue package, the generated
  compiler included; heavy drivers may eventually want a factoring the dry path
  need not pay for.
- Bad, the platform veto becomes coarse: hosts run whatever the types in their
  catalogue carry, so the gate is which packages a host is built against
  (package-grained, at host build), not per-driver policy.

### Consequences

- The local stand-in [A-13 Mission Analysis]'s single-process scenario relies on
  becomes an ordinary catalogue value: an attach-only type whose driver writes
  fixed coordinates, so the demo path exercises the same seam production will.
- Real substrate types may precede their drivers as compile-only citizens whose
  `Attach` explains itself — catalogues and definitions lead, drivers follow.
- The output vocabulary is the image's first value typing beyond the binding
  kinds; plumbing and future designers can render and validate outputs without
  the live catalogue.

## More Information

Doors this decision leaves open, each with its reopening trigger:

- **Slice lifecycle.** `Attach` is the only act; the slice kind's create,
  mutate, and destroy — with pruning and the delete protection
  [D-08 Desired-State Image] requires before anything destructive — enter as
  further `Provisioner` capabilities. Trigger: the acceptance spec below
  becoming due.
- **The first real slice driver.** Specified before it is built: the NATS
  slice's driver creates the solution's account against the guaranteed cluster
  when absent and verifies it when present. That spec is this door's acceptance
  test. Trigger: the first solution whose isolation outgrows the local stand-in.
- **Structured outputs.** The vocabulary is scalar, so a connection bundle
  (endpoint, credential, subject prefix) flattens into several outputs. This is
  the same composite-value gap [D-12 Statement-Body Anatomy] records for
  parameters and stanzas — one shared door, reopened by the first value that
  will not flatten.
- **Static typing beyond booleans.** The link-time ref-site check covers boolean
  slots only. Widening it needs parameter slots that advertise their kinds — a
  typed-slot surface grown on the library's flag conventions, never a parallel
  schema. Trigger: the first recurring class of wet-bind mismatches that a
  positioned link error would have caught.
- **Guardrails on destructive acts.** Delete protection and prune policy have no
  home yet because nothing can destroy; where they ride (the record, the type,
  the image's provenance) is decided with the lifecycle door. Trigger: the first
  destructive verb.

[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
[A-11 Substrate and Slices]: ../analyses/A-11-substrate-slices.md
[A-12 Operator I/O]: ../analyses/A-12-operator-io.md
[A-13 Mission Analysis]: ../analyses/A-13-mission-analysis.md
[A-14 Staged Catalogue Compilation]: ../analyses/A-14-staged-catalogue-compilation.md
[D-06 Declarative Descriptor Values]: D-06-declarative-descriptor-values.md
[D-08 Desired-State Image]: D-08-desired-state-image.md
[D-09 Catalogue Citizenship]: D-09-catalogue-citizenship.md
[D-10 Generate-Compile-Run]: D-10-generate-compile-run.md
[D-11 Self-Contained Services]: D-11-self-contained-services.md
[D-12 Statement-Body Anatomy]: D-12-statement-body-anatomy.md
[D-13 Two-Phase Enactment]: D-13-two-phase-enactment.md
