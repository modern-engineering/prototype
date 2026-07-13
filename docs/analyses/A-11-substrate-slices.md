---
status: draft
since: 2026-07-06
refined-by:
  - D-14-driver-on-type
---

# Substrate, Slices, and Attachments

## Context

Backend solutions lean on services they do not implement: message brokers,
databases, caches, identity providers. [A-08 Ambient Services] settles how a
component consumes such a dependency in-process — explicit imports, fail-early
accessors, never magical injection. This analysis surveys the other side of the
process boundary: where the dependency comes from, who guarantees it exists, and
what a solution may claim of it.

[A-09 Solution Layer] framed a solution as an isolation container anchored to a
purpose. The isolation it holds is mostly data isolation, and data isolation is
carved out of exactly these shared services. Keywords in excerpts (`provision`,
`slice`, `attach`) are working names.

## Whose Job Is Availability?

Three paths for who makes the backing service exist.

**The solution deploys its own.** Every solution ships its broker and its
database alongside its components. A multi-tenant production system we studied
does this for its relational store — a dedicated server instance per customer —
and the costs are instructive: per-customer capacity planning, version drift
across the fleet, backup and upgrade procedures multiplied by the customer
count. The path works; it does not scale as engineering.

**The catalogue admits third-party software.** Wrap the broker, the identity
provider, and the dashboard server as pseudo-components behind shims, and let
solutions deploy them like anything else. The shim's cost is owning someone
else's packaging: the component contract ([A-04 Component Contract]) and the
configuration surface ([A-03 Parameterization]) mean nothing to software that
never consumed the library, so every guarantee the catalogue implies must be
re-established by hand, per shim, per upgrade.

**Substrate, guaranteed by the platform.** Availability of shared services is
the platform team's business, outside any one solution's scope. A solution
_carves access_ out of the guaranteed substrate: an account on the shared
broker, a schema in the shared database. The definition then never says "run a
broker"; it says "I hold this partition of the broker you guarantee."

This analysis recommends the substrate path; a decision record fixes the
catalogue's citizenship accordingly.

## Isolation Levels

For one relational database, the access a solution holds could be:

- a dedicated server instance (the heaviest, the field case above);
- a dedicated database within a shared server;
- a dedicated role and grants within a shared database;
- a dedicated schema or table set;
- no partition at all — one shared database where isolation is enforced
  per-request by application-level authorization.

Each level is different provisioning code, a different blast radius, and a
different amount of human coordination (the heavier levels involve DBAs and
capacity owners; the lighter ones are automatable). The level is a
solution-design choice, and it is expressed by _which provisioning type the
developers registered in the catalogue_ — not by deployment mechanics. The
degenerate last level is a legitimate design: a user-facing service whose
authorization arrives with each request holds no partition of its own.

## Slices: Owned Partitions

A slice is code that runs at provisioning time against guaranteed substrate,
carving a partition the solution owns. Ownership means lifecycle: the slice's
driver creates the partition, mutates it on definition changes, and — only with
delete protection satisfied — destroys it when the statement disappears from the
image. A slice produces outputs that components consume:

```
provision NATS slice as natsAccount {
    cluster: natsCluster           // substrate, bound by the site
    adminAccount: natsAdmin
}

deploy Ping as Ping1 {
    nats: natsAccount.config       // output, bound at reconcile time
}
```

The output's scheme is registered on the provisioning _type_ in the catalogue,
which matters for what follows.

## Attachments: Verified Access Without Ownership

Not all substrate is modern enough to slice. The per-customer server instances
from the field case exist today; a solution moving onto this layer must plug
into one without pretending to own it. An attachment is the second kind of
provisioning: its driver _verifies_ at reconcile time that the named substrate
exists and is compatible, and emits outputs — often echoing its verified inputs
— in the _same scheme the type's slice kind would emit_:

```
provision Postgres attach as pgLegacy {
    server: pgServer               // extern: the legacy instance
}
```

Because the output scheme belongs to the type and not the kind, downstream
wiring cannot tell a slice from an attachment, and migrating off legacy
substrate is a one-line change from `attach` to `slice`. Ownership draws the
other lines too: attachments are never pruned, never destroyed, and a failed
re-verification degrades the solution's status rather than deleting anything.

## Open Questions

- **More kinds.** `slice` and `attach` are unlikely to exhaust provisioning; a
  one-time initialization job at reification has been floated. Whether new needs
  become kinds under `provision` or verbs of their own is open.
- **Adoption.** Migrating `attach` to `slice` over live data needs a
  take-ownership story (and stateful renames need a moved-equivalent); neither
  is designed.
- **Driver authorship.** Whether slice drivers are written by platform teams
  (who own the substrate) or developers (who own the catalogue) is undecided.
  Where they live no longer is: [D-14 Driver-on-Type] puts the driver on the
  declaring type itself, so it ships with whichever team publishes the package.

[A-03 Parameterization]: A-03-parameterization.md
[A-04 Component Contract]: A-04-component-contract.md
[A-08 Ambient Services]: A-08-ambient-services.md
[A-09 Solution Layer]: A-09-solution-layer.md
[D-14 Driver-on-Type]: ../adr/D-14-driver-on-type.md
