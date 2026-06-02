---
status: accepted
since: 2026-05-31
refines:
  - D-01-use-architectural-decision-records
---

# Define a Per-Family Status Lifecycle

## Context and Problem Statement

[D-02 MADR format][d-02] fixed `status` as a free string and paired it with a
`date` whose meaning was never defined. A free string cannot express the states
the corpus needs — a superseded decision, a rejected option — and the lifecycle
those states form differs by family. A hand-maintained `date`, read as "last
modified" or "authored", only duplicates what git already records.

## Decision Drivers

- The lifecycle should be queryable, not only legible in prose.
- Its vocabulary differs per family; the schema must admit that without
  inventing states a family does not need.
- A status change is worth dating, but only with a meaning git does not own.

## Considered Options

- **A per-family `status` enum with a `since` transition date.**
- **A `draft` boolean** — one stable/unstable flag.
- **Keep the free `status` plus `date`** — the [D-02 minimum][d-02].

## Decision Outcome

Chosen option: "per-family `status` enum with `since`." `status` follows a
shared shape — not a spec'd state machine — whose wording bends per family where
the domain reads better:

- A **preliminary** state (`draft` for analyses and requirements, `proposed` for
  decision records) marks a document merged while still unstable, so work can
  land and circulate before it settles. A document already settled at merge time
  may enter as `accepted` directly.
- **`accepted`** is the settled state every family converges on: the
  load-bearing version other documents may rely on.
- A preliminary document that does not make it is abandoned (`rejected`).
- An `accepted` document that ages out is `deprecated`, or `superseded` by a
  named successor — the end-of-life states [D-01 decision records][d-01]
  anticipates.

`since` records the date (`YYYY-MM-DD`) a document entered its current `status`.
It moves on a transition and never on a content edit, so it states what git does
not: when the status last changed. Authoring and last-modified dates remain
git's responsibility.

Recording a transition edits the frontmatter of an accepted record. This refines
[D-01 decision records][d-01]: its immutability governs the decision body, while
lifecycle and relationship metadata stay mutable so the corpus stays navigable.

The `draft` boolean was rejected: it collapses this shape to two states and
cannot express the off-ramp (`rejected`) or the end-of-life states.

### Consequences

- `date` gives way to `since`; documents still carrying `date` migrate to it.
- A family uses only the states its domain needs, drawn from the shared shape.

## More Information

- Verification and validation is a separate axis from `status`: a document can
  be `accepted` yet unverified. A `verification` field is recognised and
  deferred until requirements are verified in practice.
- [MADR 4.0.0][madr]

[d-01]: D-01-use-architectural-decision-records.md
[d-02]: D-02-madr-format-with-frontmatter.md
[madr]: https://adr.github.io/madr/
