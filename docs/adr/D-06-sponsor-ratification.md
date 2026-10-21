---
status: accepted
since: 2026-10-21
ratification: pending
refines:
  - D-04-status-lifecycle
---

# Record Sponsor Ratification Separately from Completeness

## Context and Problem Statement

A document can be complete enough to publish and use before the project sponsor has read it in full and endorsed its substantive content.
Keeping that document in `draft` would obscure the difference between an incomplete scaffold and a complete record awaiting personal review.
Treating its merge or technical review as sponsor endorsement would record approval that has not occurred.

[D-04 on document lifecycle][D-04-status-lifecycle] already separates admission to the mission corpus from completeness.
It defines `draft` as incomplete and `accepted` as complete for the document's purpose.
An independent record of sponsor ratification lets publication proceed without conflating those facts with personal endorsement.

## Decision Drivers

- Preserve the distinction between complete documents and incomplete scaffolds.
- Make sponsor ratification queryable across document families.
- Allow useful work to enter the corpus before the sponsor completes a full read.
- Record endorsement only when it is explicit, including when backfilling existing records.
- Keep meaning-preserving editorial and metadata work from invalidating an existing ratification.

## Considered Options

- **An independent `ratification` field** alongside the existing lifecycle metadata.
- **Additional lifecycle states** combining completeness and sponsor endorsement.
- **Pull request approval and merge history alone** as the record of endorsement.

## Decision Outcome

Chosen option: "an independent `ratification` field."
The field is available to numbered records in every document family and has two values:

- `pending` — the current substantive content awaits explicit sponsor ratification.
  The document may already be complete and technically reviewed.
- `ratified` — the project sponsor has read the substantive content in full and explicitly endorsed it.

An omitted field means ratification is unrecorded.
It must not be interpreted as either `pending` or `ratified`.
Existing lifecycle metadata, a merge, or a completed technical review is insufficient evidence for backfilling ratification.

New records carry `ratification: pending` unless the sponsor has explicitly ratified their substantive content.
Existing records receive the field when their ratification state is explicitly established.
Backfilling records the confirmed fact without inventing a historical ratification date.

For example, a complete record awaiting sponsor ratification can carry:

```yaml
status: accepted
since: 2026-10-21
ratification: pending
```

The three facts remain independent:

- Merging a mission document into `main` admits it to the canonical mission corpus.
- `status` records its lifecycle state, including whether it is complete for its purpose.
- `ratification` records the sponsor's endorsement of its substantive content.

A complete document can therefore be `accepted` while ratification is `pending`.
The sponsor can also ratify an intentionally incomplete scaffold without making its `draft` status inaccurate.
Ratification does not establish verification or operational proof; those depend on the relevant evidence.

### Recording and preserving ratification

The project sponsor holds ratification authority.
A contributor or automated tool may record `ratified` only on the sponsor's explicit instruction or confirmation that the substantive content has been read and endorsed.
Permission to publish or merge alone does not provide that confirmation.
Ratifying one record does not ratify the records it references or derives.

A change that alters the document's substantive meaning returns `ratification` to `pending` unless the sponsor explicitly ratifies the resulting content.
Such changes include revised outcomes, assumptions, constraints, responsibilities, boundaries or decisions.
Meaning-preserving editorial changes, formatting, indexing and relationship or lifecycle metadata changes retain the existing ratification.
Ratification therefore follows substantive meaning rather than requiring approval of each exact byte sequence.

`since` continues to date entry into the current lifecycle `status`, as defined by [D-04 on document lifecycle][D-04-status-lifecycle].
Changing ratification alone does not change `since`.
Git records the metadata change and its author; this decision introduces no additional date or reviewer fields.
Ratification metadata remains mutable under the lifecycle decision's allowance for frontmatter changes, while accepted decision bodies retain their existing immutability rule.

### Consequences

Completeness and sponsor endorsement can be inspected separately without delaying publication solely for a full sponsor read.
Maintainers must preserve that distinction when changing content and must obtain explicit evidence before marking ratification.
Legacy records can remain unrecorded until their state is established, avoiding an unsupported bulk inference from historical status or merge activity.

Additional lifecycle states were rejected because they would multiply combinations of independent facts.
Pull request history alone was rejected because permission to merge need not mean the sponsor has read and endorsed the whole document.

[D-04-status-lifecycle]: D-04-status-lifecycle.md
