---
status: draft
since: 2026-11-11
derived-from:
  - M-01-business-mission-analysis
derives:
  - M-09-sales-led-demonstration
  - M-10-evolving-operational-solution
  - M-11-distributed-industrial-operation
  - M-12-controlled-disconnected-delivery
---

# Solution Management

Solution management builds on the backend toolkit described in [M-01-business-mission-analysis] to manage tailored services through delivery, use, change and retirement.

## Business context and opportunity

B2B software organisations often begin tailoring software services for individual customers with shared repositories, specialist knowledge and platform-specific automation.
As the number and variety of solutions grow, that mix becomes a constraint.
Responsibility blurs across teams, intended software becomes difficult to relate to what is running, and a change for one customer can affect another.
Routine delivery then depends on software and platform specialists who should be improving reusable capabilities rather than bridging the same gaps repeatedly.

The motivating case is a brownfield software business spanning cloud and local infrastructure, with dedicated applications and allocations in shared backing services.
Customers retain different versions and accept changes at different times.
The knowledge needed to provision, deploy and remove each customer's service is scattered across tools and people.

The business objective is to grow routine tailored delivery without proportional growth in specialist intervention.
People responsible for a customer's service should be able to select supported capabilities and carry routine changes through an established path with appropriately delegated authority and access.
The prevalence of this problem and the size of the improvement remain to be demonstrated.

## Mission and boundary

The solution-management mission is to preserve each solution's identity, intended design and delivery knowledge as work passes between people, tools, times and places.
It adds a common way to express which service is intended, carry it into operation and relate observations back to its design and history.
These outcomes require evidence beyond the toolkit's engineering benefits.

A **logical solution** is the continuing business and operational subject designed to satisfy a business beneficiary's needs.
It keeps its identity as its software, configuration or placement changes, and two solutions remain distinct even when their contents match.
Its **solution design** expresses the selected applications, settings and relationships; a **revision** distinguishes an intended edition of that design.
Files represent the design, while the solution's history connects that intent with what happened in operation.

The **business beneficiary** supplies the needs against which usefulness is judged.
It may be a customer or the adopting organisation operating a service for many customers; consumption alone does not make a tenant its beneficiary.
That relationship grants no deployment or retirement authority.
Cooperating parts at several sites can remain one logical solution.

Three responsibilities remain distinguishable:

- **Solution engineering** understands beneficiary needs, selects the intended design and validates the service's usefulness.
- **Software engineering** maintains and verifies the reusable applications and their supported configuration and integration.
- **Platform/service engineering** maintains backing capabilities and the means of provisioning, deploying and sustaining applications in their environments.

These are responsibilities rather than prescribed teams, and one actor may perform several kinds of work.
A delivery still depends on **specialist intervention** when its established path requires otherwise unavailable expertise, authority or access, or a change to a reusable capability.
Running a maintained procedure with delegated access can be routine delivery; changing that procedure is platform development.
Automation retains that dependency if a specialist must supply missing knowledge or credentials for each run.

The capability supplies shared meaning and reusable support within the adopting organisation's delivery system.
The operating arrangement allocates infrastructure ownership, execution, reconciliation and acceptance authority.
A vendor may operate the service itself, or support software and procedures that customer engineers execute in their own environment.
The customer then controls local changes, acceptance and the release of evidence.
Exchanging an artefact transfers none of those decisions.

## Concept of Operations

### Establish the intended service

A beneficiary need starts a solution or prompts its revision.
The solution engineer selects supported applications, versions, settings and relationships and records the intended revision.
The delivery arrangement establishes who can authorise the change, who will perform it and what technical and business results they will accept.
A request that existing capabilities cannot meet returns to the engineers responsible for extending them.

In [M-09 on a sales-led demonstration], the applications already support the prospect's evaluation need.
A bounded design surface and delegated access let the sales engineer deliver routine changes without editing shared production definitions or waiting for specialists.

### Carry the design into operation

The delivery path checks prerequisites and provisions required backing capabilities through maintained procedures.
It records which allocations belong to this solution and which shared services must remain available to others.
Deployment makes the selected applications operational using those dependencies and configuration.

When preparation and execution are separated, the handoff carries the solution's identity, intended revision and the supported knowledge needed to realise it.
In [M-12 on controlled disconnected delivery], Cisco prepares the material and customer engineers later perform the change at an isolated plant.
They need the applicable procedures, executables and dependencies locally available.
The same identity and design must retain their meaning across those tools and environments, even when their representations differ.

### Check and accept what is running

Operators check the actual applications, versions, configuration and backing capabilities against the intended revision, retaining the scope and result.
A prepared revision does not establish what has been installed, and deployment acceptance does not establish that the service meets the beneficiary's needs.
Customer use supplies evidence for that separate judgment.

A failed change may leave allocations, stopped processes or mixed application versions.
Authorised operators decide whether to pause, repair or restore a supported arrangement and record what remains for the next action.
In [M-10 on an evolving operational solution], operators recover from a failed application upgrade while preserving the configuration and capacity the merchant still needs.
The same solution's history connects the expansion, failure and recovery.

### Sustain and change the service

Customer use and observations guide support, repair and capacity planning.
Changed needs or an unsuitable design return to solution engineering; restoring the existing intent may require provision or deployment work without another design revision.
Support staff use retained versions, settings and customer-released evidence to reconstruct what they can investigate; other conditions still require operational evidence.

In [M-11 on distributed industrial operation], cloud and factory changes follow separate windows, with older and newer software intentionally present together.
Supported compatibility preserves the service during that overlap.
Operators retain intent and observations for each placement, distinguish deferred changes from failures and agree when to try again.
Delayed evidence remains evidence of the installation observed at that time; it does not confirm a later change.
The common contract leaves scheduling, coordination and reconciliation mechanisms to the delivery arrangement.

### Retire the solution

An authorised retirement ends the solution's operational life while retaining its identity and history without reassigning them.
The delivery path stops dedicated applications, revokes solution-specific access and releases allocations under the applicable data-retention and cleanup rules.
Shared services remain available to the other solutions that use them.
Ending one process or placement need not retire the whole solution.
The temporary demonstration exercises this cleanup, including accounts, routes and data that outlive its applications.

## Evidence and business acceptance

Adoption can begin with a bounded replacement in an existing delivery path and expand where the results justify it.
The organisation agrees the baseline, running result and acceptable resource effects before a trial.
Four levels of evidence support different decisions:

- **Deployment slice:** the delivery authority accepts an observable running service and its resource effects after a bounded replacement.
- **Full customer thread:** the function accountable for customer delivery accepts fulfilment of a need through design, delivery and use, with less routine specialist intervention.
- **Repeated fulfilment:** the adopting organisation's business authority assesses continuity of identity, design meaning and intended-to-running software traceability across different environments, alongside delivery growth and specialist effort.
- **Cross-organisation reuse:** the project sponsor assesses whether materially different organisations reuse the common contract and applicable reference tools within their own operating arrangements.

The comparison includes preparation, routine work, failed attempts, support and maintenance of reusable procedures under both the earlier and proposed paths.
Partial adoption demonstrates only the capabilities exercised.
The stories illustrate intended operation; they supply no measured business results.

## Risks and open questions

The capability may add integration and record-keeping work without reducing specialist intervention.
A common contract may be too weak to preserve meaning across environments or too restrictive to fit them without bespoke workarounds.
Shared repositories, credentials and backing services can still let one solution's change disrupt another; separate logical identities supply no security or failure isolation.
Incomplete history or observations can also create false confidence about what is running.

The stories leave the representation and partitioning of artefacts, compatibility and versioning rules, delegation mechanisms and coordination of concurrent changes open.
They do not establish sufficient coverage of adoption, catalogue discovery, relocation, standby or recovery arrangements.
Those questions need further operational evidence and later design work, alongside evidence of sustainable maintenance and support costs.

[M-01-business-mission-analysis]: M-01-business-mission-analysis.md
[M-09 on a sales-led demonstration]: M-09-sales-led-demonstration.md
[M-10 on an evolving operational solution]: M-10-evolving-operational-solution.md
[M-11 on distributed industrial operation]: M-11-distributed-industrial-operation.md
[M-12 on controlled disconnected delivery]: M-12-controlled-disconnected-delivery.md
