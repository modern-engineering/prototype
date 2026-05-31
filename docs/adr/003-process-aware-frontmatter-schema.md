---
status: accepted
since: 2026-05-31
supersedes: 002
---

# Adopt a Process-Aware Frontmatter Schema

## Context and Problem Statement

[ADR-002 MADR format][adr-002] fixed the frontmatter at `status` and `date`, the
minimum the corpus needed before it had a process. Now that the corpus runs as a
connected workflow (analyses survey forces, requirements quote them, ADRs record
choices), the metadata can carry that process instead of merely labelling files.
Two weaknesses surfaced:

- `date` has no defined meaning. Read as "last modified" it duplicates git and
  drifts; read as "authored" it is also already in git. Neither reading earns a
  hand-maintained field.
- `status` is an open string. The lifecycle differs by family, and the corpus
  already needs states a boolean cannot express (a superseded ADR, a rejected
  option).

## Decision Drivers

- Frontmatter exists for tooling and queries (the reason [ADR-002][adr-002]
  chose it). Every field must serve a consumer and must not duplicate an
  authoritative source: git, the filename, or the H1 title.
- Traceability is the backbone of the process; "every artifact traces back"
  should be queryable, not only prose.
- The lifecycle vocabulary differs per family; the schema must admit that
  without inventing states a family does not need.

## Considered Options

- **Process-aware schema** — semantic `since`, per-family `status` enums, and
  machine-readable traceability fields.
- **A `draft` boolean instead of `status`** — one stable/unstable flag.
- **Keep `status` plus `date`** — the [ADR-002][adr-002] minimum.

## Decision Outcome

Chosen option: "Process-aware schema." It retains MADR 4.0.0 (inherited from
[ADR-002][adr-002]) and replaces the two free-form fields with:

| Field                          | Families     | Meaning                                                                                                        |
| ------------------------------ | ------------ | -------------------------------------------------------------------------------------------------------------- |
| `status`                       | all          | Lifecycle state; see the lifecycle shape below.                                                                |
| `since`                        | all          | Date (`YYYY-MM-DD`) the document entered its current `status`; moves on a transition, never on a content edit. |
| `supersedes` / `superseded-by` | all          | The record this one replaces, or that replaces it.                                                             |
| `derived-from`                 | requirements | The analyses the requirement quotes.                                                                           |

Traceability values (`derived-from`, `supersedes`, `superseded-by`) may be
written as Markdown links so they resolve, for example
`[A3 Parameterization](../analysis/A3-parameterization.md)`. The body's
reference-style rule does not bind frontmatter.

### The Lifecycle Shape

`status` follows a shared shape, not a spec'd state machine, and the wording
bends per family where the domain reads better:

- A **preliminary** state (`draft` for analyses and requirements, `proposed` for
  ADRs) marks a document merged into `main` while still unstable, so work can
  land and circulate before it settles. A document already settled at merge time
  may enter as `accepted` directly; being quoted is not a required gate.
- **`accepted`** is the green, settled state every family converges on: the
  load-bearing version other documents may rely on.
- A preliminary document that does not make it is abandoned (`rejected`, in ADR
  terms).
- An `accepted` document that ages out is `deprecated` or `superseded` by a
  named successor.

The `draft` boolean was rejected: it collapses this shape to two states and
cannot express the off-ramps (abandoned, `rejected`) or the end-of-life states
(`deprecated`, `superseded`) that [ADR-001 decision records][adr-001] already
mandates.

### Consequences

- `date` becomes `since`, redefined as a status-transition date; authoring and
  last-modified dates remain git's responsibility.
- A requirement's link to its analyses becomes machine-readable through
  `derived-from`; the inverse link is derived, not stored.
- This record supersedes [ADR-002][adr-002], whose status moves to `superseded`.
  Its body stays intact, per the immutability rule in [ADR-001][adr-001].
- Existing documents still carrying `date` migrate to `since` in a follow-up.

## More Information

- Verification and validation is a separate axis from `status`: a requirement
  can be `accepted` yet unverified. A `verification` field is recognised and
  deferred until requirements are verified in practice.
- [MADR 4.0.0][madr]

[adr-001]: 001-use-architectural-decision-records.md
[adr-002]: 002-madr-format-with-frontmatter.md
[madr]: https://adr.github.io/madr/
