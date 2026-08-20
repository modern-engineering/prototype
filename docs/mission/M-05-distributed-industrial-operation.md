---
status: draft
since: 2026-08-21
derived-from:
  - M-02-concept-of-operations
---

# Distributed Industrial Operation

This counterfactual story applies the [Concept of Operations][M-02-concept-of-operations] to a distributed industrial service.
Public Siemens material establishes only that Siemens offers [cloud-based predictive-maintenance software across assets and sites][Siemens-Senseye], describes [hybrid edge-cloud arrangements for intelligent maintenance][Siemens-Industrial-Edge], and documents the [ingestion of industrial machine data][Siemens-machine-data].
Big Hammer Steelworks, the customer relationship, the toolkit adoption, the responsibilities and every event below are invented.

Big Hammer operates several factories where an unexpected equipment stoppage can halt production.
Siemens provides one customer-specific predictive-maintenance service that helps Big Hammer notice deterioration early enough to plan maintenance instead.
Plant-side software observes local conditions and remains useful when cloud connectivity is intermittent.
The cloud contribution retains the longer view across Big Hammer's factories, giving Siemens data scientists and Big Hammer's reliability team broader evidence for maintenance decisions.

Big Hammer experiences those contributions as one service.
For the toolkit, that customer service is one logical solution whose identity continues as its software changes.
The plant-side software is not a complete copy of the cloud service, and the cloud service cannot replace the timely local view.
Each factory needs the plant-side contribution for timely local action, while the cloud contribution reveals longer patterns across the factories.

Siemens already operates the service successfully.
Its engineers use mature internal tooling, maintain playbooks, augment scripts for particular customer sites and personally bridge the steps that the tooling does not carry.
That approach is workable for a company able to place skilled people around a growing customer estate, but routine changes continue to consume their attention.

The Siemens group responsible for Big Hammer's service chooses the toolkit as a different delivery foundation.
It wants supported software to carry more of the repeatable transition knowledge while people retain authority, operational judgement and the ability to intervene.
The group is not trying to rescue a failed service or discover predictive maintenance for the first time.
Its first obligation is to preserve the value Big Hammer already receives.

As operating data accumulates, Siemens data scientists see a way to improve the product's detection behaviour.
The change updates cloud software and data models together with related plant-side software.
It is an ordinary product change, but it must become one understandable change to Big Hammer's service even though its cooperating parts cannot all move at once.

Siemens identifies the intended edition as **solution revision R2**.
R2 belongs to the same logical solution and expresses the intended cloud-and-plant arrangement after the change.
It does not claim that every factory has already moved from the software specified by the previously accepted R1 revision.

Siemens verifies the new software and prepares a supported way to introduce it at the factories.
Supported software carries the repeatable upgrade know-how so that the employee executing the change does not have to interpret an arbitrary playbook or reproduce technical knowledge from memory.
A person still authorises and initiates the work, transports approved data where the environment requires it, judges the result and can depart from the nominal path when circumstances demand.

Siemens can introduce the cloud change centrally, but Big Hammer's factories have different production commitments.
Each factory manager chooses a window in which Siemens may change the plant-side software and retains the decision to return the factory to production.
The cloud therefore begins running the software specified by R2 while some factories still run software specified by R1.

At the first factory, a Siemens service engineer joins the agreed window and uses the supported upgrade capability to introduce the plant-side release.
Siemens accepts that its software is running as intended.
Big Hammer's factory manager separately confirms that production may resume.
Its reliability team confirms that the predictive-maintenance service is ready for local use.
None of those decisions substitutes for another.

As the factory resumes operation, the information it uploads carries the logical solution's durable identity, R2 and the application version that produced it.
When connectivity is interrupted, the plant-side contribution continues locally and retains that context until its information can reach the cloud.
Siemens' cloud application can then interpret the information according to the applicable revision and apply its own product logic when information from different revisions needs different handling.

The same context gives Siemens operations a view of the change across Big Hammer's footprint.
They can distinguish a factory that is intentionally waiting for its authorised window from one whose running software does not match the accepted result of a completed transition.
They decide which factory needs follow-up and when; the toolkit neither makes that decision nor forces the factories to converge.

The remaining factories move when their production schedules permit.
For a time, the cloud and some factories run software specified by R2 while other factories continue on software specified by R1.
That mixed state belongs to a change in one distributed customer service; it does not turn the cloud and factory contributions into separate services.

At one factory, the upgrade cannot complete before the authorised window closes.
Rather than improvise under production pressure, Siemens and Big Hammer return that factory to its previously accepted software and resume production.
Its uploaded information remains identifiable with R1 and the application version that produced it, while the software running elsewhere remains untouched.
Siemens schedules another attempt, and its engineers may intervene directly during the next window.

Once the intended cloud and factory transitions have been accepted, the Siemens service owner declares the new solution revision available across Big Hammer's intended footprint.
Big Hammer's reliability team continues to receive the alerts and maintenance insight on which its operation already depended.
That business parity is the required result: the toolkit-backed path has carried the same customer service through a distributed change without obscuring its intent, running software or lines of authority.

If Siemens later gains enough confidence in the supported path, it may allow Big Hammer's IT or OT staff to perform some routine plant-side upgrades with Siemens ready for escalation during the agreed window.
That would reduce the need for a Siemens specialist to enact every nominal step, but it is a possible later gain rather than the reason this adoption succeeds.

The case does not depend on a particular identifier format, data schema, transport, reconciliation mechanism, command interface or artefact partition.
Those choices remain with later design work.
What matters here is that one evolving customer service remains understandable while its cloud and factory contributions change through different tools, authorities and schedules.

[M-02-concept-of-operations]: M-02-concept-of-operations.md
[Siemens-Industrial-Edge]: https://www.siemens.com/en-gb/products/industrial-edge/
[Siemens-Senseye]: https://www.siemens.com/en-us/products/industrial-digitalization-services/senseye-cloud-application/
[Siemens-machine-data]: https://developer.siemens.com/senseye/machine/index.html
