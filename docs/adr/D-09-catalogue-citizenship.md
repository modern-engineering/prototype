---
status: proposed
since: 2026-07-06
refines:
  - A-01-scope
  - A-11-substrate-slices
---

# Scope the Catalogue to Application-Library Citizens

## Context and Problem Statement

Real deployment estates are half third-party: brokers, databases, identity
providers, dashboards run beside the software a company writes. If solution
engineers are to compose everything they operate, pressure builds to admit that
software into the catalogue — and every guarantee the catalogue implies (the
component contract, the explicit configuration surface, the operator ABI of
[A-12 Operator I/O]) means nothing to a binary that never consumed the
application library. [A-11 Substrate and Slices] surveys who makes backing
services exist; this record fixes what the catalogue may contain.

## Decision Drivers

- The application library is the core of this project; the catalogue's value is
  the guarantees library citizenship carries.
- [A-01 Scope] already constrains the framework to long-running Go applications;
  the catalogue should not quietly widen that scope.
- Backing services still must be reachable — solutions depend on them — but
  reachability and membership are different relations.
- The problem is hard enough; the prototype earns the right to widen scope later
  ([A-11 Substrate and Slices] keeps the shim path surveyed, not buried).

## Considered Options

- **Citizens only.** The catalogue admits only components built on the
  application library; everything else is substrate reached through
  provisioning.
- **Shims from day one.** Third-party software enters the catalogue behind
  adapter components that fake the contract.
- **Open catalogue.** Any runnable artifact registers with a reduced,
  best-effort scheme.

## Decision Outcome

Chosen option: **citizens only**. A catalogue entry is a component built on the
application library, full stop. Third-party and legacy software is not deployed
by solutions at all: its availability is the platform's guarantee, and solutions
reach it as substrate — an owned slice or a verified attachment per
[A-11 Substrate and Slices] — whose drivers are ordinary code run by
controllers, not catalogue components.

### Consequences

- Every catalogue entry honors the whole contract: configuration surface,
  termination and health capabilities, the operator ABI. Downstream tooling
  (compilers, controllers, designers) may rely on it without per-entry caveats.
- A workload that cannot be substrate and cannot consume the library is out of
  scope for now — named plainly rather than half-supported. The revisit trigger:
  the first solution that genuinely cannot ship without deploying such a
  workload.
- The Go-centric constraint of [A-01 Scope] extends up the stack: this layer
  serves Go-centric backend estates first, on the conviction that the problem is
  hard enough constrained.

## More Information

The scope may widen by evolution rather than exception: if shims ever enter,
they enter as a declared second citizenship class with reduced guarantees, never
as pretend citizens. [A-12 Operator I/O]'s ambient-surface advertisement is the
likely vehicle — a shim is, in effect, all ambience and no contract.

[A-01 Scope]: ../analyses/A-01-scope.md
[A-11 Substrate and Slices]: ../analyses/A-11-substrate-slices.md
[A-12 Operator I/O]: ../analyses/A-12-operator-io.md
