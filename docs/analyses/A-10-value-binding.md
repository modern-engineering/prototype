---
status: draft
since: 2026-07-06
---

# Value Binding: Symbols Staged Across Compile, Site, and Reconcile Time

## Context

[A-09 Solution Layer] recommends compiling solution definitions to an
intermediate language (IL) that deployment systems reconcile. Definitions carry
values — an interval, a subject name, an account credential — and a value can
become known at three distinct moments:

1. **compile time**, when the definition is linked against the catalogue;
2. **site configuration**, when a deployment environment reifies the image;
3. **reconcile time**, when a provisioned resource actually comes to exist and
   can report its coordinates.

This analysis surveys how the IL should treat values whose binding moment lies
beyond compile time, who may rebind a value and when, how secrets ride along,
and how rich the value language itself should be. Keywords in excerpts (`var`,
`extern`) are working names.

## Inline at Compile, or References in the Image?

**Inline everything.** The compiler substitutes every value into every record —
macro expansion. The image is self-contained scalars; consumers need no
resolution machinery. The costs surface immediately: a value used by ten
statements becomes ten unrelated scalars, so a site cannot override it
uniformly; secret-ness is erased (a credential inlined into a record field is
indistinguishable from a hostname); a diff between two images shows ten changes
where one was made; and values that only exist at reconcile time cannot be
inlined at all, so a second mechanism appears anyway.

**References survive into the image.** Records carry symbolic references; the
image carries a symbol table; consumers resolve. One mechanism then serves
everything late-bound, and four properties fall out:

- a site override rebinds one symbol and reaches every use site uniformly;
- sensitivity is a property of the symbol and taints every field wired from it,
  so controllers know what they must not log or expose;
- image diffs are symbol-grained ("only the target subject changed");
- a future visual designer gets first-class knobs: the symbols are the
  definition's public tuning surface.

The cost is real: every consumer resolves references, and validation that
inlining would have forced to compile time can drift later. This analysis
recommends references, with the compiler still type-checking every reference
against the catalogue's schemas at link time.

## A Symbol Table with Linkage

C linkage is again the intuition pump. Three symbol classes cover the three
binding moments:

| Class            | Bound at        | May rebind      | C analogue               |
| ---------------- | --------------- | --------------- | ------------------------ |
| `var`            | compile default | the site (MAY)  | weak/interposable symbol |
| `extern`         | site config     | the site (MUST) | undefined dynamic symbol |
| instance outputs | reconcile time  | the controller  | loader-defined symbol    |

An illustrative sketch, wiring all three:

```
extern natsAdmin Secret        // the site must bind this

var echoSubject: "com.acme.Echo"   // compile default, site may override

provision NATS slice as natsAccount {
    adminAccount: natsAdmin
}

deploy Ping as Ping1 {
    target: echoSubject
    nats: natsAccount.config   // exists only once the account does
}
```

Reconcile-time outputs make the dependency graph explicit; the compiler can
build the DAG from references and reject cycles at link time. What a controller
does while a referenced output does not yet exist — hold the dependent record
back, or converge it degraded — is a controller-contract concern left open here.

## Override Governance

If a site may rebind `var` symbols, three postures compete:

- **All symbols open.** Any `var` is a site override target. Maximum operator
  convenience; the definition's behavior at a site is no longer readable from
  the definition alone.
- **All symbols sealed.** Overrides require a definition change. Maximum
  fidelity to the artifact; every site nudge becomes a recompile.
- **Per-symbol declaration.** The definition marks which symbols are open.
  Honest, at the cost of one more concept.

This analysis recommends starting open — all `var` symbols overridable — with a
sealing marker sketched for later (C's `static`, internal linkage, is the
analogue: a symbol invisible outside the image). Whatever the posture, one
guardrail is not optional: the controller records the effective value of every
symbol it resolved, or site overrides become an unauditable diagnosis gap.

## Secrets

Three paths for a value that must not leak:

- **Plaintext in the image.** Rejected without ceremony: images are copied,
  cached, and diffed.
- **Sealed into the image.** The value is encrypted at design time against a
  target environment's key. Attractive on paper; three forces push back.
  Rotating the credential now means recompiling and reshipping the solution (a
  secret rotation becomes a software release); rotating the environment's
  sealing key bricks rollback to older images; and one image can no longer serve
  environments that do not hold the key.
- **Extern-bound.** The image declares a sensitive symbol; the site binds it
  from its own secret store at reification. Rotation is an environment
  operation, the image stays portable, and nothing secret is ever inside the
  artifact.

This analysis recommends extern-bound secrets as the primary mode and defers
sealing (if ever, for demo and development convenience). Sensitivity is best
declared where the type is: a catalogue-level attribute on the symbol's type,
propagated as taint through references, so a record field wired from a sensitive
symbol is itself sensitive. Taint has a boundary worth stating: once the value
crosses into application code, enforcement ends; the application library's
discipline takes over from there.

## How Rich a Value Language?

Wherever declarative definitions carry values, pressure builds to compute them.
Four rungs on that ladder:

- **General expressions.** Infrastructure languages that started declarative and
  grew expressions demonstrate the end state: a programming language with worse
  tooling. Rejected as a starting point.
- **String interpolation.** A constrained middle ("`$(base).events`") that
  history shows grows rung by rung toward the first.
- **Whole-value references only.** A record field is a literal or a symbol
  reference, nothing else. Derivations live in application code: a component
  that subscribes to a wildcard below a base subject takes the base as its
  parameter and derives the wildcard internally, because the derivation is its
  own business.
- **A closed set of registered pure functions.** When a derivation is a
  convention _between_ components (a shared prefix scheme, say), hiding it in
  each component's code makes it an invisible contract. A small,
  catalogue-registered, compile-time-evaluable function set is the designed
  relief valve — added when pain demands, never general expressions.

This analysis recommends whole-value references now, with the function set as
the named escape hatch. The discriminator: derivations private to one component
belong in that component; conventions between components must be visible to the
compiler or they will drift apart silently.

## Open Questions

- **Naming.** `var` (Go's noun) versus a verb such as `define`, matching the
  directive grammar's verb-first style. Open, low stakes, revisited with the
  syntax.
- **Component-computed values.** A descriptor could expose functions, so one
  instance's parameter is computed by another component's logic ("Ping1's target
  is what Pong's naming function returns"). The idea is raw and risks a
  strictly-ordered initialization story; nothing in the reference model
  prohibits it — a function output is one more node in the DAG with a binding
  time — so it stays an open evolution, not a plan.
- **Interpolation demand.** If whole-value references prove too austere in
  practice, which rung to step to, and what evidence justifies the step.
- **Override audit.** Where the controller's record of effective values lives
  and how a site diff is presented to operators.

[A-09 Solution Layer]: A-09-solution-layer.md
