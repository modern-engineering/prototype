---
status: draft
since: 2026-08-13
derived-from:
  - M-01-business-mission-analysis
derives:
  - M-03-sales-led-demonstration
  - M-04-evolving-operational-solution
  - M-05-distributed-industrial-operation
---

# Concept of Operations

## A corpus about operation

This mission corpus describes the same endeavour at progressively more concrete operational viewpoints.
[M-01-business-mission-analysis] establishes why the toolkit exists, the enduring mission outcome and the boundaries within which an adopter remains accountable.
This document provides the common frame for seeing that mission in operation.
Future mission documents will tell individual operational stories from the viewpoints of adopting organisations and greenfield settings.

The parts are intended to remain read together.
The business mission is not repeated in every story, and one story is not promoted into the universal operating model.
Instead, each story places the common concepts in a particular sociotechnical delivery system, makes its authorities and material conditions visible, and shows a logical solution moving through time.
As contrasting stories accumulate, they will reveal which operational ideas are genuinely shared and which differences the corpus must preserve.

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
It should still make the time sequence and consequential handoffs legible enough to show how identity, intent, authority and evidence survive the journey.

## Operational stories

Each future story will describe a bounded setting in which an adopter seeks an observable operational change.
It should be broad enough to expose a meaningful end-to-end thread and narrow enough that its assumptions, authorities and evidence can be examined.
The story should cover the following concerns in coherent prose rather than mechanically completing a template:

- the starting situation and the pressure or opportunity that makes adoption worth considering;
- the adoption boundary and intended outcome, including what remains with incumbent practices;
- the actors, responsibility domains and decision authorities that matter to the thread;
- the relevant before view, so that operational and organisational change can be judged;
- a time-sequenced nominal thread following the logical solution across the relevant lifecycle activities and handoffs;
- only those off-nominal, recovery or change threads that materially alter authority, continuity, evidence or outcome;
- sustainment, evolution and retirement where they are material to the setting;
- the evidence to be observed and the authority that accepts the result;
- positive and negative impacts on the adopter, its customer and affected operational work;
- assumptions and unknowns that bound what the story can support; and
- differences from this common concept or from other stories that must not be normalised away.

A story can combine responsibilities in one actor or distribute them across people, tools and organisations.
It can begin in a brownfield process or a greenfield setting.
Those choices belong in the story because they change operation, not because this frame prefers one realisation.

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

The portfolio is not a coverage claim.
Its value will come from the contrasts represented and the evidence each story contributes, not from the number of entries.

## Method tailoring

This frame draws on public NASA and INCOSE systems-engineering material without claiming compliance with either body.
It adopts the useful core of ConOps practice: describe the system, users and operators from a high-level operational viewpoint; use time-sequenced single-thread scenarios; include selected nominal and off-nominal conditions; and tailor depth to the system's size, complexity and current decisions.
[NASA's Concept of Operations annotated outline][NASA-Appendix-S] is treated as a source of concerns rather than a document template, while the [NASA Systems Engineering Handbook][NASA-SE-Handbook] and [INCOSE requirements-engineering material][INCOSE-RE] reinforce the separation between stakeholder-facing operational understanding and later requirements or implementation choices.

That tailoring is deliberate.
The project needs contrasting adopter stories before it can justify a richer common operating model.
Detailed physical-environment descriptions, command and data architectures, interface specifications, exhaustive mode and failure catalogues, training or staffing quantities, and implementation plans would add bulk without supporting the present decision.

## Deferred detail and known unknowns

This document does not select artefact schemas or carriers, APIs, deployment or provisioning mechanisms, execution topology, command structures, interface protocols or automation products.
It does not allocate requirements or prescribe how an adopter implements a responsibility domain.
Those decisions require evidence from operational stories and belong in later analyses, requirements and architecture.

The first stories need to test which contextual differences change the common concept.
Known candidates include brownfield versus greenfield adoption; connected versus air-gapped operation; local versus remote provisioning authority; toolkit, incumbent or proprietary front ends and execution tools; lightweight demonstration environments versus sustained customer operations; catalogue discovery and composition; and handoffs that cross activities separated by people, tools, time or place.
These are story discriminators, not product capabilities or architecture decisions.

The corpus also does not yet know which operational threads are representative, how many contrasting settings are sufficient, where authority patterns recur, or which off-nominal conditions expose new mission-level needs.
Early stories may challenge the system boundary, role vocabulary or expected outcome.
When they do, the corpus should preserve the contradiction, reconsider the affected accepted mission content, and promote only what the resulting evidence supports.

[M-01-business-mission-analysis]: M-01-business-mission-analysis.md
[M-01-boundary]: M-01-business-mission-analysis.md#system-boundary-and-operating-model
[M-01-life]: M-01-business-mission-analysis.md#the-durable-handoff-and-the-life-of-a-solution
[M-03-sales-led-demonstration]: M-03-sales-led-demonstration.md
[M-04-evolving-operational-solution]: M-04-evolving-operational-solution.md
[M-05-distributed-industrial-operation]: M-05-distributed-industrial-operation.md
[NASA-Appendix-S]: https://www.nasa.gov/reference/system-engineering-handbook-appendix/#hds-sidebar-nav-112
[NASA-SE-Handbook]: https://explorers.larc.nasa.gov/HPSMEX22/pdf_files/04_NASA_SystemsEngineeringHandbookRev2.pdf
[INCOSE-RE]: https://www.incose.org/wp-content/uploads/2026/01/requirements_engineering_part_1.pdf
