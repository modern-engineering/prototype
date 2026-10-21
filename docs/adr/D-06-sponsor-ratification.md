---
status: accepted
since: 2026-10-21
refines:
  - D-04-status-lifecycle
---

# Record Sponsor Ratification by Identity

## Context and Problem Statement

A document can be complete enough to publish and use before the project sponsor has read it in full and endorsed its substantive content.
Keeping that document in `draft` would obscure the difference between an incomplete scaffold and a complete record awaiting personal review.
Treating its merge or technical review as sponsor endorsement would record approval that has not occurred.

[D-04 on document lifecycle][D-04-status-lifecycle] already separates admission to the mission corpus from completeness.
It defines `draft` as incomplete and `accepted` as complete for the document's purpose.
Personal ratification needs its own attribution without requiring every unratified document to declare another state.

## Decision Drivers

- Preserve the distinction between complete documents and incomplete scaffolds.
- Record who personally ratified the current substantive content.
- Give omission the same useful meaning for existing and future records.
- Allow useful work to enter the corpus before the sponsor completes a full read.
- Record endorsement only when it is explicit.
- Keep meaning-preserving editorial and metadata work from invalidating an existing ratification.

## Considered Options

- **An optional scalar `ratified-by` field** recording the sponsor's identity.
- **A ratification-state enum** requiring a separate value for unratified records.
- **Additional lifecycle states** combining completeness and sponsor endorsement.

## Decision Outcome

Chosen option: "an optional scalar `ratified-by` field."
The field is available to numbered records in every document family.
Omission is the zero value: the current substantive content is unratified.
This meaning applies equally to old and new records, without asserting rejection or a scheduled review.

A nonempty value identifies the project sponsor who has read the current substantive content in full and explicitly endorsed it.
Record that identity as a GitHub login, for example:

```yaml
ratified-by: danielorbach
```

There is one sponsor ratification authority, so one scalar identity is sufficient.
This is person attribution, not a document relationship.
The named ratifier can differ from the contributor or tool that records the metadata.
Only the sponsor's explicit ratification authorises that attribution; a merge, technical review or Git author identity does not establish it.
Each record is ratified independently, including a refining or superseding decision.

### Relationship to lifecycle status

Merging a mission document into `main` admits it to the canonical corpus.
Its lifecycle `status` continues to describe completeness and disposition under [D-04 on document lifecycle][D-04-status-lifecycle].
Ratification records personal endorsement of the content independently:

| Status     | `ratified-by`  | Meaning                          |
| ---------- | -------------- | -------------------------------- |
| `draft`    | omitted        | Incomplete and unratified        |
| `accepted` | omitted        | Complete and unratified          |
| `draft`    | `danielorbach` | Personally endorsed scaffold     |
| `accepted` | `danielorbach` | Complete and personally ratified |

The same separation applies to the preliminary `proposed` state used by decision records.
For `rejected`, `deprecated` or `superseded` records, attribution remains about the retained content; status still determines its disposition and applicability.
Ratification cannot adopt a rejected option or restore a superseded decision.
Technical review, verification and operational proof remain separate matters of evidence.

### Recording and preserving ratification

Record attribution only for explicit sponsor ratification, including when backfilling existing documents.
Other records remain unratified by omission, without a special migration state.

A change to the endorsed substantive meaning removes `ratified-by` unless the sponsor explicitly ratifies the resulting content.
Meaning-preserving changes retain the attribution.
That distinction applies to metadata too: a relationship or scope change can be material, while a lifecycle transition alone need not erase endorsement of the retained content.

`since` continues to date entry into the current lifecycle `status`, as defined by [D-04 on document lifecycle][D-04-status-lifecycle].
Changing ratification alone does not change `since`.
Git records the metadata change and its recorder; backfilling attribution does not invent a historical ratification date.

### Consequences

Unratified documents need no ratification metadata, including complete documents ready for publication.
An explicit endorsement is both queryable and attributable without a second enum or a review queue.
The enum was rejected because absence of endorsement already has a useful representation; additional lifecycle states were rejected because they combine independent facts.

[D-04-status-lifecycle]: D-04-status-lifecycle.md
