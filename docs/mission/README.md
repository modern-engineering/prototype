# Mission Documents

Mission documents establish why the project exists, whose needs it serves, and the enduring outcomes and boundaries that frame downstream work.
They preserve material uncertainty instead of forcing early answers.

[M-01-business-mission-analysis] frames the engineering of distributed Go backends across development, testing and production.
[M-02-concept-of-operations] describes how developers and coding agents investigate changes, keep system tests and use production observations to guide later work.
[M-08-solution-management] develops the separate business mission and operational concept for tailored delivery, supported by its own stories.
The GitHub Status stories begin with the team's adoption of the toolkit, then follow engineers debugging a gateway on a laptop, testing what the page reports during an incident and probing the Git service in production.

Every mission document merged into `main` is accepted into the canonical corpus.
The `draft` status marks an accepted but incomplete scaffold that is expected to evolve, while `accepted` marks a document considered complete at its current revision.
If later evidence reopens a completed document, it returns to `draft` until the corpus completes it again.

Completeness and sponsor ratification are independent under [D-06-sponsor-ratification].
A complete document can be `accepted` with `ratified-by` omitted, meaning complete and unratified.
The optional field names the sponsor who personally ratified the content; merging alone does not supply that endorsement.

| Record | Mission document |
| --- | --- |
| [M-01-business-mission-analysis] | Business Mission Analysis |
| [M-02-concept-of-operations] | Concept of Operations |
| [M-03-adopting-the-toolkit] | ConOps Story: Adopting the Toolkit |
| [M-04-debugging-the-status-gateway] | ConOps Story: Debugging the Status Gateway |
| [M-05-testing-a-partial-recovery] | ConOps Story: Testing a Partial Recovery |
| [M-06-probing-git-operations] | ConOps Story: Probing Git Operations |
| [M-07-one-responsibility-model-across-platforms] | ConOps Story: Running the Same Application in Three Clouds |
| [M-08-solution-management] | Solution Management: Business Mission and Concept of Operations |
| [M-09-sales-led-demonstration] | ConOps Story: Sales-Led Demonstration |
| [M-10-evolving-operational-solution] | ConOps Story: Evolving an Operational Solution |
| [M-11-distributed-industrial-operation] | ConOps Story: Distributed Industrial Operation |
| [M-12-controlled-disconnected-delivery] | ConOps Story: Controlled Disconnected Delivery |

[M-01-business-mission-analysis]: M-01-business-mission-analysis.md
[M-02-concept-of-operations]: M-02-concept-of-operations.md
[M-03-adopting-the-toolkit]: M-03-adopting-the-toolkit.md
[M-04-debugging-the-status-gateway]: M-04-debugging-the-status-gateway.md
[M-05-testing-a-partial-recovery]: M-05-testing-a-partial-recovery.md
[M-06-probing-git-operations]: M-06-probing-git-operations.md
[M-09-sales-led-demonstration]: M-09-sales-led-demonstration.md
[M-10-evolving-operational-solution]: M-10-evolving-operational-solution.md
[M-11-distributed-industrial-operation]: M-11-distributed-industrial-operation.md
[M-07-one-responsibility-model-across-platforms]: M-07-one-responsibility-model-across-platforms.md
[M-12-controlled-disconnected-delivery]: M-12-controlled-disconnected-delivery.md
[D-06-sponsor-ratification]: ../adr/D-06-sponsor-ratification.md
[M-08-solution-management]: M-08-solution-management.md
