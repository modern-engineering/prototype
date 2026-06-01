---
status: accepted
since: 2026-05-31
---

# Record Traceability in Frontmatter

## Context and Problem Statement

Traceability is the backbone of the process: every artifact should trace back to
the one that motivated it. Stated only in prose, that chain cannot be queried —
a tool cannot list what a requirement derives from, or which record supersedes
another, without reading every body.

## Decision Drivers

- "Every artifact traces back" should be machine-readable, not only legible.
- A field must not store what can be derived from its inverse.
- A value must resolve to a document and survive YAML without coercion.

## Decision Outcome

Add three traceability fields, written as [family-prefixed identifiers][d-03]:

| Field                          | Families     | Meaning                                            |
| ------------------------------ | ------------ | -------------------------------------------------- |
| `supersedes` / `superseded-by` | all          | The record this one replaces, or that replaces it. |
| `derived-from`                 | requirements | The analyses the requirement quotes.               |

A value is a full identifier, for example `derived-from: A-03-parameterization`,
so it names the target for a reader and resolves to a file. The body's
reference-style rule does not bind frontmatter.

### Consequences

- A requirement's link to its analyses becomes queryable through `derived-from`;
  the inverse link is derived, not stored.
- A supersession names its successor on the retired record and its predecessor
  on the replacement, so either reads as a complete statement of the relation.

## More Information

- [MADR 4.0.0][madr]

[d-03]: D-03-document-naming-convention.md
[madr]: https://adr.github.io/madr/
