---
status: proposed
since: 2026-07-06
refines:
  - A-09-solution-layer
---

# Express Solution Definitions as Verb-First Directives

## Context and Problem Statement

[A-09 Solution Layer] recommends compiling solution definitions to an
intermediate language (IL), with several authoring front-ends over time — a text
notation now, a visual designer and a code-first builder later. The text
notation is needed immediately: it is the medium the design itself is being
prototyped in, so its fluency is on the critical path. The notation must stay
stable while company catalogues grow (registering a new component type must
never touch the grammar), and, being scaffolding, it must not attract tooling
investment the layer's compiler deserves instead.

## Decision Drivers

- Every statement should map onto one IL record, so all front-ends are peers
  producing the same artifact.
- The grammar must be closed under catalogue growth: types and their parameter
  schemes live in the catalogue, never in the syntax.
- Prototype-grade longevity: hand-rolled parsing must stay cheap, and aesthetics
  matter because tangible mockups drive the design work.
- The notation should feel like Go tooling (the go.mod directive family),
  without pretending to be Go code.

## Considered Options

- **A YAML document with a generated schema.** Fixed top-level keys (components,
  services, secrets, variables); an external schema, generated from the
  catalogue, validates typed bodies.
- **Go-flavoured assignments inside verb blocks.**
  `deploy { Ping1 =
  Ping{...} }` — composite-literal bodies bound to names
  with `=`.
- **Verb-first directive statements.** `deploy Ping as Ping1 { ... }` — a flat
  list of statements, each a verb, a catalogue type, an optional kind word, an
  instance name, and a body.
- **Adopt HCL.** Terraform's block grammar (`deploy "Ping" "Ping1" {...}`) with
  its existing parser, formatter, and schema-driven decoding.

## Decision Outcome

Chosen option: **verb-first directive statements**, in the go.mod flavour. What
this record fixes is the statement shape, not the fine grammar:

- a solution is a directory of source files, each opening with the same
  `solution <name>` clause, the way Go files repeat a package clause;
- statements are verb-first (`deploy`, `provision`, `extern`, `var`, `default`),
  with go.mod-style factoring of same-verb statements into parenthesized blocks;
- instances are named at declaration (`as Ping1`), and a kind word between type
  and name (`provision NATS slice as ...`) disambiguates when a type registers
  several kinds, like the type in a Go `var` declaration;
- quoted text is data, bare identifiers are symbol references
  ([A-10 Value Binding] gives references their meaning);
- bodies carry catalogue-typed parameters and, per [A-09 Solution Layer]'s
  statement anatomy, non-parameter compartments for deployment intent and
  metadata.

An illustrative sketch:

```
solution sample

extern natsAdmin Secret

default Ping {
    interval: 1s
    count: -1
}

deploy Ping as Ping1 {
    target: "com.acme.Echo"
}

provision NATS slice as natsAccount {
    adminAccount: natsAdmin
}
```

The compiler resolves `Ping` against the catalogue and checks the body against
the registered scheme, so all type knowledge stays out of the grammar; safety
comes from compiling, not from syntax.

## Pros and Cons of the Options

### A YAML Document with a Generated Schema

- Good, parses everywhere; no parser to write; IDE validation via schema.
- Bad, the schema is a function of the catalogue, so generation and distribution
  become permanent infrastructure, and validation errors are structural rather
  than semantic.
- Bad, the source looks like a shippable artifact, inviting exactly the
  surface/artifact conflation [A-09 Solution Layer] warns against.

### Go-Flavoured Assignments

- Good, composite-literal bodies read familiarly to Go developers.
- Bad, `=` binding reads as an expression language and invites composition
  semantics (value reuse, arithmetic) that a declaration notation should not pay
  for; [A-10 Value Binding] rejects that ladder.
- Bad, pseudo-Go (`interval: 1s` is not Go) sits in an uncanny valley next to
  the real code-first builder; ironically, go.mod's own lesson is verb-first
  directive lines, which is the chosen option.

### Verb-First Directive Statements

- Good, statement-to-record isomorphism: verbs are the growth axis, and a new
  verb never disturbs existing grammar.
- Good, flat, order-insensitive, and addressable — friendly to designer
  round-trips and to linking with duplicate-symbol errors.
- Bad, a parser, however small, must be written and maintained; no external
  tooling understands the notation.

### Adopt HCL

- Good, parser, positioned diagnostics, formatter, and schema-driven decoding
  exist today; the provider-schema model mirrors the catalogue.
- Bad, the aesthetic is not the one that keeps the mockups fluent, and the
  dependency's weight is not justified while the notation is scaffolding. HCL
  remains the comparable language to raid for ideas; the trigger to revisit is
  the first time the hand-rolled toolchain needs positioned diagnostics or
  round-trip editing at quality.

### Consequences

- Tooling investment goes to the compiler, linker, and IL; the notation gets a
  small hand-rolled parser and no language server.
- The visual designer and code-first builder target the IL record shape, not
  this notation.
- Grammar details remain deliberately unstable while prototyping; breaking
  syntax between iterations is accepted and cheap.

## More Information

Open grammar paths, kept live on purpose and revisited as the mockups evolve:

- **Body compartments.** Colon-distinguished sections versus an explicit
  `params { ... }` keyword block versus a `deployment { ... }` block; whether
  controller-specific extensions belong inside the deployment compartment when
  target controllers are known ahead of time ([A-09 Solution Layer] surveys the
  forces).
- **Section naming.** `on` is provisional and reads oddly; `deployment` is a
  candidate.
- **Symbol declaration.** `var` (Go's noun) versus a verb such as `define`.
- **Defaults addressing.** `default <verb>` versus `default <kind>` for
  statement-level defaults (`default deploy` versus `default component`).
- **Declaration order.** `deploy Ping as Ping1` versus Go-style name-first
  `deploy Ping1 Ping`; kind words as optional disambiguators either way.

The go.mod file syntax (directive lines, parenthesized factoring, comment
preservation) is the parsing reference; its published implementation shows the
directive layer is a few hundred lines.

[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
