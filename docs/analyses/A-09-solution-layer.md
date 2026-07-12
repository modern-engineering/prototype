---
status: draft
since: 2026-07-06
refined-by:
  - D-12-statement-body-anatomy
---

# Solution Layer: From Definition to Reification

## Context

The analyses so far fix the application library: what a component is
([A-01 Scope]), how many of them share a program ([A-02 Architecture]), how
blueprints parameterize instances ([A-03 Parameterization]), and how components
consume their runtime surroundings ([A-08 Ambient Services]).
[D-06 Declarative Descriptor Values] already anticipates the layer above: a
solution-management layer describing how applications relate to one another and
to non-application dependencies.

This analysis opens that layer. Three roles divide the work:

1. Developers write applications and publish them as catalogues.
2. Solution engineers compose catalogued elements into solutions.
3. Platform engineers build the deployment systems that enact them.

A solution definition carries the intent of solution engineers; a deployment
system resolves everything else needed to run it. The analysis surveys four
boundary questions the layer must answer: what a solution is, what artifact
carries one, how a single statement addresses its several audiences, and how a
deployed solution changes over time. Verbs and keywords appearing in excerpts
(`solution`, `deploy`, `provision`, `extern`, `var`) are working names.

## Deployment Indifference

Four archetype environments recur wherever backend software is deployed:

- applications on Kubernetes, enacted by an operator;
- Linux services managed by systemd;
- a playground host serving an entire solution in a single process;
- an end-to-end testing harness, also single-process.

One company plausibly operates all four: a cloud product on Kubernetes,
on-premises installations on systemd hosts, demos and tests in single processes.
The applications behave the same everywhere; we call this deployment
indifference, and the library's explicit configuration surface
([A-03 Parameterization]) exists to preserve it.

One corollary shapes every survey below: whatever the solution layer produces
must be reifiable by all archetypes, including the single-process ones. A
definition expressible only as, say, Kubernetes objects is not a solution
definition. A later requirement will lock this litmus test.

## What Is a Solution?

INCOSE frames systems engineering as transforming stakeholder needs,
expectations, and constraints into a solution: a realized system, specific to
its purpose and context. The composition is the point — the same software
composed differently behaves differently, and so serves a different purpose.

Two shapes compete for what a solution definition denotes.

**A template.** The definition is parameterized and instantiated many times:
once per customer, once per environment. This is the shape of packaged charts
with per-site value files. Its forces: fleet concerns (which customers run which
versions where) migrate into the language; identity muddles ("the payments
solution" names a family, not a thing); and every construct eventually grows an
instantiation story.

**An instance.** The definition denotes one living, mutating thing, deployed at
most once. Per-customer variants are distinct solutions, authored above the
layer — nothing prevents a Go program from generating definitions the way code
generates code. Identity stays crisp: the definition names the deployed thing,
updates mutate it, and deleting it retires a purpose.

This analysis recommends the instance shape. It composes with the roles: a
solution becomes an isolation container anchored to a purpose. It holds
exclusive access carved out of shared services ([A-11 Substrate and Slices]
develops that carving), and it may later constrain co-location of its components
(open below). Operator-logical groupings larger than one purpose — a "mission
control" namespace served by several solutions — live above the layer, as does
fleet management.

## The Artifact

What does a solution engineer hand to a deployment system? Two paths.

**The authoring source is the artifact.** One document, probably YAML, validated
by a schema, shipped as-is. Its forces: because the source looks shippable, it
ships, and then the authoring surface can never change; the schema is not static
but a function of the catalogue the company's developers curate, so schema
generation and distribution become permanent infrastructure; and the layer
expects several authoring surfaces (a text notation now, a visual designer and a
code-first builder later), so canonizing one surface's format makes the others
second-class translators.

**A compiled intermediate representation.** Authoring surfaces are front-ends; a
compiler links definitions against the catalogue and emits an intermediate
language (IL) that deployment systems consume. C compilation is the intuition
pump:

| C                         | Solution layer                   |
| ------------------------- | -------------------------------- |
| translation units         | definition source files          |
| headers                   | the catalogue's registered types |
| linked image              | one IL document per solution     |
| undefined dynamic symbols | values the environment must bind |
| dynamic loader            | the deployment environment       |

Each definition statement compiles to one IL record, so any front-end that can
produce records is a peer of the text notation. The costs are real: a compiler
must exist, and the IL needs versioning discipline of its own.

This analysis recommends the compiled path; a decision record fixes the text
notation's syntax. An illustrative sketch:

```
solution sample

deploy Ping as Ping1 {
    target: "com.acme.Echo"
}

provision NATS slice as natsAccount {
    adminAccount: natsAdmin
}
```

When a value like `natsAdmin` binds — at compile time, at the deployment site,
or only once the provisioned account exists — is the subject of
[A-10 Value Binding].

## Statement Anatomy: Three Audiences

A single `deploy` statement speaks to three audiences at once:

- **the application:** parameters typed by the catalogue entry, destined for the
  component's configuration surface;
- **the environment's controller:** deployment intent — where to run, how many,
  how much — interpreted by the deployment system and never seen by the
  application;
- **nobody in particular:** opaque metadata carried through reification for
  whatever tooling cares to read it.

The split recurs across mature systems: Kubernetes separates `spec` from
`metadata`; Terraform separates resource arguments from meta-arguments; systemd
separates `[Service]` from `[Unit]` and `[Install]`. The audiences are a
fixture; how a grammar separates them is surveyed, not settled:

- **Reserved section words** inside one body (`on { ... }`, `metadata { ... }`),
  with catalogue registration rejecting colliding parameter names. Force
  against: a word reserved tomorrow breaks a catalogue registered yesterday.
- **Grammatical separation:** parameters always take a colon (`key: value`);
  sections are colon-less blocks. Future sections can never collide with
  parameters, at the price of hanging meaning on one character.
- **Explicit keyword blocks:** parameters move into their own block
  (`params { ... }`) alongside a `deployment { ... }` block. Maximum
  explicitness, ceremony on the most common content.
- **Controller-specific extensions:** whether the deployment-intent block admits
  specializations for controllers known ahead of time (a Kubernetes sub-block),
  or whether such tuning is confined to site-side configuration owned by
  platform engineers. Extensions in the definition strain the litmus test above;
  confinement keeps definitions portable but pushes legitimate knowledge out of
  the artifact.

[D-12 Statement-Body Anatomy] decides the separation: one compartment per
audience, with controller-specific extensions as named advisory stanzas that
unrecognizing controllers ignore.

The operator-facing channels those parameters ultimately travel — flags,
environment, files, signals — are the subject of [A-12 Operator I/O].

## Update Semantics

A solution is alive; its definition changes after deployment. Three paths for
what an update ships.

**Append/patch instructions.** An early sketch of this layer tried a `patch`
unit: a definition fragment linked into an already-shipped solution,
append-only. It was abandoned before it settled: "append" is an instruction
against presumed current state, and presuming state is exactly what breaks
idempotence. The IL would become a log to replay rather than a state to hold.

**Source overlays.** Per-site overlays patch a base definition (strategic-merge
and kustomization tooling are the precedent). Overlays answer real variance
needs, but they reintroduce the template shape through the back door, and the
artifact stops being one thing.

**A complete desired-state image, reconciled.** Every update ships the whole
solution again under the same identity; the deployment system reconciles:
observe what runs, diff against the image, converge, prune what disappeared.
Reifying the same image twice is a no-op. Instance names (`as Ping1`) become
reconciliation keys. Controllers take two shapes over one pure core (desired
state in, actions or artifacts out): live loops (the Kubernetes-operator shape)
and static emitters (a compiler from IL to systemd units, converging on the next
apply). Pruning makes guardrails mandatory — plan-before-apply, image
generations, delete protection on stateful holdings; a decision record fixes
them.

This analysis recommends the reconciled image.

## Open Questions

- **Compute isolation.** Which components may or must share a process or a host
  is expressed intent in production systems we studied; whether the layer
  carries co-location constraints (never assignments — the single-process
  archetypes forbid those) is open.
- **The fleet above.** Regions, blue/green colors, release channels, and
  per-customer rollout live above solution identity; how IL identity and
  generation compose with that layer is out of scope here and undesigned.
- **Naming.** Every keyword in the excerpts is provisional; the syntax decision
  record carries the live alternatives.

[A-01 Scope]: A-01-scope.md
[A-02 Architecture]: A-02-architecture.md
[A-03 Parameterization]: A-03-parameterization.md
[A-08 Ambient Services]: A-08-ambient-services.md
[A-10 Value Binding]: A-10-value-binding.md
[A-11 Substrate and Slices]: A-11-substrate-slices.md
[A-12 Operator I/O]: A-12-operator-io.md
[D-06 Declarative Descriptor Values]: ../adr/D-06-declarative-descriptor-values.md
[D-12 Statement-Body Anatomy]: ../adr/D-12-statement-body-anatomy.md
