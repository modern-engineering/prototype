---
status: draft
since: 2026-08-13
derived-from:
  - M-01-business-mission-analysis
derives:
  - M-03-sales-led-demonstration
  - M-04-evolving-operational-solution
  - M-05-distributed-industrial-operation
  - M-06-one-responsibility-model-across-platforms
---

# Concept of Operations

## A corpus about operation

This mission corpus describes the same endeavour at progressively more concrete operational viewpoints.
[M-01-business-mission-analysis] establishes why the toolkit exists, the enduring mission outcome and the boundaries within which an adopter remains accountable.
This document provides the common frame for seeing that mission in operation.
The individual operational stories follow solutions through the work of their adopting organisations, customers and users.

The parts are intended to remain read together.
The business mission is not repeated in every story, and one story is not promoted into the universal operating model.
Instead, each story places the common concepts in a particular sociotechnical delivery system, makes its authorities and material conditions visible, and shows a logical solution moving through time.
The contrasting settings reveal shared operational ideas and differences that remain significant to the people doing the work.

This common frame is accepted as part of the mission corpus but remains incomplete.
It is expected to change as the remaining storylines expose differences in boundaries, authorities and operational consequences.
It can be considered complete after the planned story portfolio is complete and its shared ideas and material differences have been integrated here.

## Operational viewpoint

The toolkit is viewed inside an adopter-owned sociotechnical delivery system, not as that delivery system in miniature.
People, policies, incumbent or proprietary tools, automation and platforms surround it and may continue to perform most operational work.
The adopter retains authority for its infrastructure, execution, reconciliation and deployment-acceptance decisions, as established by [M-01 § System boundary and operating model][M-01-boundary].

The identity-bearing subject of every operational story is the logical solution.
Its durable identity lets the story follow one business and operational subject while its intent, placement and realised contents change.
The story therefore follows what happens to the solution and who is accountable for each consequential transition, rather than treating files, commands or deployment machinery as the main character.

The responsibility domains in [M-01 § System boundary and operating model][M-01-boundary] provide the stable role vocabulary: solution engineering, software engineering, and platform/service engineering.
They describe concerns and accountabilities, not staffing or actor types.
A human, an ordinary automated tool or an AI agent may perform work in one or more domains, but automation does not erase responsibility or confer decision authority.
Each story makes the performing actors, accountable domain and authorising authority clear where the distinction affects the outcome.

## Common operational thread

The common thread follows the life of one logical solution through the activities and return paths defined in [M-01 § The durable handoff and the life of a solution][M-01-life].
It begins when beneficiary needs inform approved solution intent, crosses adopter-controlled handoffs into provisioning and deployment, and reaches customer use.
Operational evidence and changed needs may then drive sustainment or renewed design, provisioning or deployment, while retirement ends the operational life without reassigning the solution's identity.

This is a direction of travel, not a mandatory linear workflow.
An operational story may enter part-way through an existing lifecycle, revisit activities, combine roles or use incumbent work around the toolkit.
The solution's identity connects the intended changes, the work performed and the evidence returned from operation through those handoffs.

## Operational stories

The stories enter the solution's life at different points.
A temporary demonstration begins with a prospect's evaluation need and ends with complete retirement.
The sustained services already provide useful results when their software, configuration or delivery arrangements change.
Their beneficiaries continue to depend on those results while the adopting organisation introduces the change.

A solution revision describes an intended edition without requiring every running part to reach it simultaneously.
Factory windows, cloud rollouts and recovery from a failed upgrade can leave parts of the same solution running different versions or deliberately stopped.
Operators relate those observations to the solution's identity and intended revision, decide which differences are expected, and act through their organisation's operating procedures.
The toolkit supports that understanding while the adopting organisation retains responsibility for reconciliation and recovery.

The operational contribution varies with the setting.
The sales engineer gains the ability to carry routine demonstrations through their complete lifecycle, while Siemens engineers use supported software to carry more of the repeatable factory-upgrade work.
Stripe's engineers and operators can follow a merchant's intended solution through successive changes and recovery.
The event-intake company applies the same division of software, solution and platform responsibilities through three different cloud environments.
These contrasts preserve each operation's purpose and constraints while showing how the common responsibility model fits different work.

## Promoting commonality

An idea belongs in this common frame when it follows directly from the accepted business mission or when materially different operational stories corroborate it.
Direct derivation keeps the operational concept traceable to [M-01-business-mission-analysis].
Cross-story corroboration prevents one adopter's local practice from becoming an assumed product rule.

Promotion requires the stories to agree on meaning and operational consequence, not on implementation.
Where stories differ materially, the difference remains visible in the stories and their catalogue rather than being weakened into vague common prose.
If an apparent commonality conflicts with the accepted business mission, the mission must be reconsidered and accepted on its own terms before this frame can inherit the change.

## Story portfolio

### [M-03-sales-led-demonstration]

In a counterfactual Elastic, an online retailer uses authorised live production telemetry to investigate payment latency through a temporary vendor-operated demonstration.
A bounded path lets the sales engineer revise the same demonstration and remove it completely, even though some infrastructure remains shared.

### [M-04-evolving-operational-solution]

In a counterfactual Stripe, a merchant's need for broader payment classifications leads to changes in an existing solution's configuration and capacity.
A failed upgrade and the later removal of an experimental daemon show how the solution keeps its identity while its intended configuration changes.

### [M-05-distributed-industrial-operation]

In a counterfactual Siemens, one Big Hammer predictive-maintenance service spans cloud software and separately scheduled factories.
Supported upgrade software carries routine delivery knowledge across those windows while the customer retains its existing maintenance value and the decisions about production and local use.

### [M-06-one-responsibility-model-across-platforms]

In a counterfactual Twilio Segment, one multi-tenant event-intake solution operates through serverless container environments in AWS, Azure and Google Cloud.
A routine event-rule change passes from local application checks through solution selection into three distinct deployment paths using the toolkit's existing division of responsibility.

The portfolio is not a coverage claim.
Its value lies in the contrasts represented and the operational consequences each story makes visible.

## Method tailoring

This frame draws on public NASA and INCOSE systems-engineering material without claiming compliance with either body.
Its high-level operational viewpoint connects the system, users and operators through time-sequenced scenarios and selected nominal, change and recovery conditions.
[NASA's Concept of Operations annotated outline][NASA-Appendix-S] is treated as a source of concerns rather than a document template, while the [NASA Systems Engineering Handbook][NASA-SE-Handbook] and [INCOSE requirements-engineering material][INCOSE-RE] reinforce the separation between stakeholder-facing operational understanding and later requirements or implementation choices.

That tailoring is deliberate.
The project needs contrasting adopter stories before it can justify a richer common operating model.
Detailed physical-environment descriptions, command and data architectures, interface specifications, exhaustive mode and failure catalogues, training or staffing quantities, and implementation plans would add bulk without supporting the present decision.

## Deferred detail and known unknowns

This document does not select artefact schemas or carriers, APIs, deployment or provisioning mechanisms, execution topology, command structures, interface protocols or automation products.
It does not allocate requirements or prescribe how an adopter implements a responsibility domain.
Those decisions require evidence from operational stories and belong in later analyses, requirements and architecture.

The current portfolio covers established operations, from temporary demonstrations to sustained services on virtual machines, at factories and across clouds.
The current stories do not yet examine greenfield adoption, controlled disconnected delivery or catalogue discovery.
The stories also leave open how handoffs are partitioned among artefacts and tools, how provisioning authority is delegated beyond the arrangements shown, and what evidence would establish compatibility between independently supplied participants.

The portfolio does not yet establish which operational threads are representative or how many contrasting settings are sufficient.
Further operational evidence may expose needs that change the system boundary, role vocabulary or expected outcome.
The common concept remains incomplete while those material differences and their consequences are unresolved.

[M-01-business-mission-analysis]: M-01-business-mission-analysis.md
[M-01-boundary]: M-01-business-mission-analysis.md#system-boundary-and-operating-model
[M-01-life]: M-01-business-mission-analysis.md#the-durable-handoff-and-the-life-of-a-solution
[M-03-sales-led-demonstration]: M-03-sales-led-demonstration.md
[M-04-evolving-operational-solution]: M-04-evolving-operational-solution.md
[M-05-distributed-industrial-operation]: M-05-distributed-industrial-operation.md
[M-06-one-responsibility-model-across-platforms]: M-06-one-responsibility-model-across-platforms.md
[NASA-Appendix-S]: https://www.nasa.gov/reference/system-engineering-handbook-appendix/#hds-sidebar-nav-112
[NASA-SE-Handbook]: https://explorers.larc.nasa.gov/HPSMEX22/pdf_files/04_NASA_SystemsEngineeringHandbookRev2.pdf
[INCOSE-RE]: https://www.incose.org/wp-content/uploads/2026/01/requirements_engineering_part_1.pdf
