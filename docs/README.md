# Documentation

The corpus begins with mission documents, which establish the enduring frame for later work.
Analyses investigate bounded questions within that frame, while requirements and decisions trace to the records that motivate them.
[The working vocabulary][working-vocabulary] is a non-normative scratchpad for terms used across those families.
[D-03] defines naming, [D-04] defines status lifecycles, and [D-05] defines document relationships.
Each family README indexes its current records.

[D-06-sponsor-ratification] records sponsor endorsement separately from completeness.
The `ratification` field is available across document families: `pending` awaits explicit ratification, `ratified` records it, and omission means unrecorded.
New records start pending unless the sponsor has explicitly ratified their substantive content.

| Family                 | Description                                                   |
| ---------------------- | ------------------------------------------------------------- |
| [mission/](mission/)   | Mission documents that establish the project's enduring frame |
| [analyses/](analyses/) | Problem-space analyses within that frame                      |
| [reqs/](reqs/)         | Requirements derived from mission documents and analyses      |
| [adr/](adr/)           | Architectural decision records (MADR 4.0.0)                   |

[D-03]: adr/D-03-document-naming-convention.md
[D-04]: adr/D-04-status-lifecycle.md
[D-05]: adr/D-05-document-relationships.md
[working-vocabulary]: vocabulary.md
[D-06-sponsor-ratification]: adr/D-06-sponsor-ratification.md
