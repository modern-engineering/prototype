---
status: accepted
since: 2026-05-31
---

# Record Document Relationships in Frontmatter

## Context and Problem Statement

As the corpus grows, a reader cannot guess which documents bear on one another.
The relationships that matter — one record derives from another, replaces it, or
refines its intent — should be navigable from the frontmatter, not reconstructed
by reading every body.

## Decision Drivers

- A named relationship should be queryable and lead a reader straight to the
  related document.
- Either end of a relationship should resolve, so a reader arriving at one
  document finds the other.
- A value must resolve to a document and survive YAML without coercion.

## Decision Outcome

Record relationships as pairs of [family-prefixed identifiers][d-03], one field
per direction, stored on both records:

| Forward      | Inverse         | Families                | Meaning                     |
| ------------ | --------------- | ----------------------- | --------------------------- |
| `supersedes` | `superseded-by` | all                     | replaces / is replaced by   |
| `derives`    | `derived-from`  | analyses / requirements | gives rise to / quotes      |
| `refines`    | `refined-by`    | all                     | qualifies / is qualified by |

Every relationship field holds a list of full identifiers, for example
`A-03-parameterization`, each naming a target for a reader and resolving to a
file. A single target is written as a one-element list too, so a tool reads one
shape and never guesses between a scalar and a sequence. Both edges are written
and kept in step by hand: editing frontmatter after the fact is welcome, not a
cost, because it is what keeps the corpus navigable.

`refines` carries the subtler links a review tends to surface — a later record
qualifying an earlier rule rather than replacing it — which would otherwise live
only in prose and be lost to a reader of the earlier record.

A relationship with no name in the table above is not a field. It belongs in
prose, written as a reference-style link, where the surrounding sentence gives
the context a bare pointer cannot.

### Consequences

- Traceability is not a field or an index. It is a property the corpus has when
  these relationships, taken together, let a reader move between related
  documents; the fields record relationships, and traceability is what they add
  up to.
- Each new relationship type is a pair of fields, added here, never a lone
  forward pointer whose inverse a reader has to compute.

## More Information

- [MADR 4.0.0][madr]

[d-03]: D-03-document-naming-convention.md
[madr]: https://adr.github.io/madr/
