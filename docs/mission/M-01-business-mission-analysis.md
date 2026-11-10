---
status: draft
since: 2026-10-22
derives:
  - M-02-concept-of-operations
---

# Business Mission Analysis

This document sets out the mission of a Go toolkit for distributed backends and the evidence needed to assess its value.
Its operating model defines the intended scope and the responsibilities of the toolkit and its users.

## Executive summary

Distributed backend systems written in Go consist of specialised programs running concurrently and communicating through interfaces chosen by their authors.
The work required to assemble, configure, package and manage those programs extends from development into production.
When that work depends on separately maintained executable entry points, scripts and deployment configuration, engineers must repeatedly bridge the application code and the means of running it as a system.

The mission is to reduce the effort required to develop, test and operate distributed backend systems written in Go, while making system testing practices that build confidence in production behaviour convenient to apply.
The project is an open-source toolkit for long-lived backends whose application code is written entirely in Go.
Engineers writing a program decide what it does and how others interact with it.
Engineers assembling a system choose which programs run together and supply what each needs.
The project's working hypothesis is that making construction, configuration and execution accessible through reusable Go packages can reduce surrounding integration work and expose design weaknesses earlier.
Running cooperating programs in one process is a useful technique for investigating that hypothesis.

The intended value is lower engineering effort across development and operation, together with better evidence about system behaviour.
Keeping the code that assembles and runs the system alongside the applications may reduce how much developers and coding agents have to reconstruct before they can contribute.
This could make an unfamiliar codebase easier to understand and work in.

System-level testing is a critical enabler of greater autonomy for coding agents.
It lets agents check whether a change has broken critical flows across the system.
The results help stakeholders judge how much confidence to place in the changes.

Managing the delivery and ongoing operation of tailored software solutions is a downstream application of the approach, with its own acceptance criteria.
The magnitude of these benefits remains to be demonstrated.

## Engineering context and problem

Backend applications perform specialised work and communicate through networks, message brokers, files and other mechanisms selected for their purpose.
Their authors already implement those behaviours and interactions in Go.
To make the applications useful together, engineers also need to select a composition, supply configuration and dependencies, control execution and account for the resources used over its lifetime.
The same concerns arise when developing a change, testing a system and operating it in production.

Much existing packaging and deployment tooling controls software through executable entry points and external configuration.
The knowledge needed to construct and manage a program is consequently distributed across application code, executable wrappers and surrounding tooling.
A test harness that works with Go code cannot necessarily reuse the construction and lifecycle control available through those external arrangements.
Engineers then maintain additional paths for local execution, system testing and production, with changes to an application requiring corresponding changes in several places.

That separation raises the cost of obtaining useful feedback.
A component can pass its unit tests while the composed system remains expensive to configure, run and investigate.
Testing may depend on an independently deployed environment whose setup and state are outside the test's control.
In production, the same fragmentation makes it harder to understand which code supplies a dependency, applies configuration or owns a resource when the system starts, changes or stops.

Lifecycle defects make this more than a convenience problem.
A successful startup and request do not show that a program stops its background work, releases the resources it owns or permits a subsequent run to begin cleanly.
Implicit global state, fixed resource assumptions and incomplete cleanup can remain unnoticed when each run receives a fresh process or environment.
Repeated tests of a composed system through its external interfaces can expose these defects before they disrupt operation.

AI coding agents encounter the same separation between application code and the means of exercising the system.
An organisation can give an agent the source code and a description of the change it wants, while the setup needed to try that change remains in a separate deployment repository or workflow.
The agent has to establish which programs run together, how they are configured and where it can observe their behaviour.
It may need to follow a build and deployment process before it can try the change, even when the behaviour it needs to check could be exercised with a small part of the system.

For an organisation, more capacity to write code creates a corresponding need to evaluate the changes being proposed.
The changes still need to be tried as part of the system, their effects measured and their usefulness assessed against the needs that prompted them.
If this work remains difficult, increasing the number of agents can increase the amount of work still waiting to be tested and assessed.
The difficulty of exercising the system can therefore limit how far an organisation can scale its use of coding agents.

The motivating experience is engineering distributed Go backends whose programs are already implemented but remain difficult to assemble and test together repeatedly.
It supports investigating a better development and operating model; it does not establish the prevalence of the problem or the size of the improvement across other organisations.

## Mission and opportunity

The opportunity is for software organisations to make more useful changes with less effort spent assembling, running and testing their systems.
For engineers, this means shorter feedback cycles, less code and tooling to maintain around the applications, and better ways to check the effects of a change.

For organisations using AI coding agents, the further opportunity is to give an agent more of the work of developing and assessing a change.
The business hypothesis is that agents can carry more of this work through to completion without a proportional increase in the human effort needed to guide and review it.
This requires giving an agent both the reasons for the change and a practical way to run the system and check the result.

The project pursues these opportunities by making the code that assembles and runs the system part of the system's Go codebase.
That code selects the programs, supplies their configuration and dependencies, and starts and stops them.
Developers and agents can read it to see how the programs fit together, adapt it to run a useful subset and reuse it in tests or operational hosts.
A host is software that starts and supervises programs, supplying the dependencies and execution policy appropriate to its setting.

The focus on systems whose application programs are written in Go is deliberate.
A tool intended to support applications written in many languages has to accommodate their different build tools and runtimes.
Here, the code that runs the programs together can call their Go packages directly and reuse how the applications are constructed and configured.

That code can use the same dependency management, build tools and testing tools as the applications.
Recorded module versions help reproduce the selection of dependencies, and the Go toolchain supports building for different operating systems.
Scripts used for the same work may require a particular shell, locally installed commands or container tools before they can run.
Keeping this work in Go reduces the separate setup needed around the codebase.

With the system's assembly and execution available in code, a developer or agent can try a change without first reconstructing how to run the affected programs.
They can run those programs together, send requests or messages through their interfaces and measure the resulting behaviour.
The feature request, requirements and what matters to the customer then give them reasons to choose particular checks and a basis for judging the results.
They can verify whether the system does what was requested and gather evidence about whether that behaviour meets the customer's needs.
If a new request calls for a different test, they can change the Go code that sets up the run and retain that test for later work.
If the results leave questions unanswered, they can decide what else to try or what more they need to learn about customer use.

The toolkit is intended to make modern engineering practices easier to adopt and make it harder to leave basic design problems unresolved.
Running cooperating programs or several instances within one process can expose assumptions about shared state, resource use and cleanup that a fresh process would hide.
Making an application work reliably in these tests can take extra effort.
That effort goes into making dependencies explicit, establishing which program owns each resource and ensuring that a program finishes its work and releases what it owns.
The tests bring these problems into development, where the resulting corrections can improve the application before it is deployed.
Production placement remains a decision about the system's operational needs.

These benefits remain hypotheses to evaluate against actual engineering work.
Adoption must account for the work required to make existing applications usable in this way, as well as any additional tooling and maintenance.
An abstraction that merely relocates the same setup and maintenance burden would not achieve the intended improvement.
Users of the resulting services benefit when those improvements preserve dependable service behaviour.

## System boundary and operating model

The toolkit is the system of interest within an organisation's wider software engineering and operating practices.
It supplies Go packages, supported extension mechanisms and reference tooling through which engineers can construct, configure and run programs.
Distributed Go backends are the primary target; individual applications and selected subsets of a system are also useful units of development and testing.
Adoption does not depend on a particular company structure, customer-delivery model or deployment platform.

Application authors retain control of behaviour, communication and resource access.
The toolkit does not require these to be expressed through toolkit-defined abstractions.
Its integration boundary concerns construction, configuration and execution, including requesting cancellation and reporting completion or failure.
The programs remain responsible for responding to those requests and completing the work and cleanup they own.

Engineers assembling a system choose its constituent programs, provide configuration and arrange dependencies.
Dependencies can be supplied explicitly through ordinary Go code, making the source of a runtime resource and the wiring that supplies it available for inspection.
Application authors expose the configuration needed to use their programs together.
Engineers assembling a system coordinate deliberate sharing.
The toolkit does not infer resource identity, exclusivity or safe sharing from parameter names or values alone.

Hosts control how programs are started and supervised in their setting and how completion, failure and cancellation affect the rest of the composition.
They also account for resources supplied to the programs, while programs account for resources they acquire themselves.
This division must remain intelligible when programs are run repeatedly, concurrently or in different process arrangements.
It is a division of responsibilities in the software and its use, not a prescription for separate teams.

Applications are the principal operational components.
Provisioning, deprovisioning and other finite procedures enter the scope where the applications depend on them over their operational lives.
Their construction, configuration, execution and resource effects need the same engineering attention as the applications they support.
No universal ordering of these procedures or execution policy is established here.

System tests observe behaviour at the selected system boundary using the applications' existing interfaces and communication mechanisms.
Test authors determine the interactions and outcomes relevant to that system.
The toolkit provides access and lifecycle control without needing to interpret the application's protocols or business behaviour.
Logs, metrics, traces and other observations can help explain a result; the adopting organisation chooses and interprets the evidence needed for its decisions.

The adopting organisation retains ownership and authority over infrastructure, execution, reconciliation and deployment acceptance.
Its operational hosts integrate the programs with the environments in which they run.
Local tests contribute evidence about application and system behaviour, while deployment verification establishes the behaviour and operating conditions of the actual production arrangement.
Process isolation, scheduling, network conditions and platform behaviour can materially affect that result.

## Product posture and adoption

The product is an open-source Go toolkit intended to support modern engineering practices for distributed backends.
It combines reusable program integration with tools and practices for working on whole systems.
Engineers can compose its packages with their own Go code and develop specialised hosts and harnesses for their needs.
The choice of a testing API, authoring language, deployment mechanism or resource model follows from those needs and the evidence obtained.

Adoption can begin with a useful subset of an existing system or with newly written applications.
An organisation can retain its incumbent packaging and deployment arrangements while evaluating how the toolkit changes development, testing or operational work.
Migration effort, additional maintenance and the work required to make implicit dependencies explicit all count towards the cost of adoption.
A smaller local setup is valuable to the extent that it reduces total engineering effort or produces evidence that justifies its cost.

Using a common program implementation in several hosts provides a basis for reuse.
It does not by itself establish equivalent behaviour across those hosts or eliminate the need to verify the integrations with each environment.
Claims about portability, confidence and saved effort are limited to the conditions actually evaluated.

## Evidence and acceptance

The first useful proof is a bounded system drawn from real engineering work whose components can be assembled, run and tested repeatedly using the toolkit.
The evaluating engineers identify the work they currently perform, the behaviours that matter and the conditions under which the result will be used.
That baseline makes it possible to judge whether the toolkit improves their work instead of merely demonstrating that its interfaces function.

### Engineering effort

The engineers responsible for developing and maintaining the system evaluate the effort required to establish a run and to change its composition or configuration.
Evidence includes the manual steps, elapsed feedback time and bespoke integration code that must be written and maintained, together with adoption and tooling costs.
The comparison follows the work through the relevant development and operational paths so that savings in one place do not conceal extra work elsewhere.
An improvement is accepted when the people doing and maintaining that work can identify a repeatable benefit against the baseline.

Where coding agents take part in the work, evaluation follows a change beyond producing the implementation.
It considers whether the agent could exercise the affected system, obtain relevant observations and use them to assess the change against its requirements and purpose.
Human effort spent preparing the work, deciding what to run, reviewing the result and correcting mistakes is included in the comparison.
Repeated work on comparable changes can then test whether agents can complete more of this work with less human direction while the resulting changes meet the organisation's needs.

### Confidence in system behaviour

The evidence concerns the composed system's externally observable behaviour and lifecycle.
It includes repeated runs, completion or failure of relevant work, shutdown and the expected disposition of owned and shared resources.
The responsible engineers establish that tests can expose relevant defects, including known or deliberately introduced lifecycle faults, and distinguish corrected behaviour from a merely successful startup.
They identify which conditions the tests represent and which remain outside their evidence.

Execution within one process may contribute to this evidence by revealing interference, retained state or incomplete cleanup.
Passing such tests is useful only for the behaviours and conditions they cover.
Additional evidence is needed for failures and operating conditions specific to distributed production execution.

### Use in production

The engineers responsible for operating the system evaluate the same application implementations through the intended operational hosts and deployment path.
They establish that configuration, dependencies, supervision and lifecycle handling produce the required behaviour in that environment.
Acceptance includes relevant observations during startup, ongoing operation, failure and shutdown, and verifies resource release or retention according to the system's intended ownership.
The adopting organisation's deployment authority decides whether that evidence is sufficient to accept the change into service.

### Reuse and broader value

Repeated use across changes and materially different systems tests whether the benefit survives beyond the initial example.
Results in different organisations can then test whether the toolkit and practices are reusable without reproducing the same integration burden in local extensions.
The project sponsor judges broader product claims from that accumulated evidence, including the cost of maintaining the toolkit itself.
Useful numerical thresholds and representative workloads remain to be established for each evaluation.

## Downstream solution management

An organisation delivering tailored software services can build solution-management capabilities on the same program and composition foundations.
In this setting, recurring customer delivery can depend on specialist knowledge, shared repositories and platform-specific automation whose effects are difficult to relate to one customer solution.
Reusable application and provisioning code can support a more explicit delivery path, while solution identity and intended design connect that work to the beneficiary's needs.
Reducing routine dependence on specialists is an additional business objective of this application of the toolkit.

The operational concept in [M-02-concept-of-operations] and its stories examine this delivery setting.
They retain value for understanding solution change, shared backing services, distributed placement and accountability.
Their delivery-specific concepts do not constitute prerequisites for using the toolkit to develop, test or operate a backend system.
Solution-management outcomes require their own evidence beyond the core engineering benefit.

### Solution delivery responsibilities

Within that setting, solution engineering understands beneficiary needs, curates the intended solution and validates its usefulness.
Software engineering maintains the reusable applications, and platform/service engineering maintains the backing capabilities and means of provisioning, deploying and sustaining them.
These responsibilities may be performed by the same people or tools; the adopting organisation retains the authority for consequential operational decisions.

Specialist intervention occurs when a particular delivery still requires specialist-held expertise, authority or access, or a change to reusable software or platform capability.
Running a maintained provisioning procedure with appropriately delegated authority can be routine delivery.
Changing that procedure, extending an application or relying on a specialist to supply otherwise unavailable access remains specialist intervention.
Acceptance of a solution-management capability therefore requires evidence that routine delivery depends less on those interventions while the beneficiary continues to receive the intended service.

### The durable handoff and the life of a solution

A logical solution is the durable identity-bearing subject intended to satisfy a business beneficiary's needs.
Its solution design expresses the selected software, configuration and relationships among its parts; revisions distinguish intended editions while the identity continues.
Two logical solutions remain distinct even when their present contents match.
A business beneficiary supplies the needs against which that solution is designed and validated, and is distinct from a user who merely consumes the service.
An adopting organisation may itself be the beneficiary of a service that it offers to many customers.

Design establishes or revises that intent, Provision makes required backing capabilities usable, and Deploy renders the applications operational.
Operate is the customer's use of the solution, while Sustain keeps it fit for that use.
Changed needs can lead to renewed Design; operational needs can require further Provision or Deploy work without changing the intended design.
Retire ends the solution's operational life while retaining its identity and history without reassigning them.
Changes to one component or placement do not by themselves retire the whole solution.

An adopter-defined Location identifies where all or part of a solution is available.
Cooperating parts at several Locations can remain one logical solution, and relocation can preserve that identity.
Solution artefacts carry identity and representations of design across people, tools, time or place.
The durable handoff preserves those meanings and the know-how needed to realise the design, while the organisation's tools perform and accept the operational work.
That continuity does not itself supply resource isolation or guarantee equivalent behaviour across environments.

## Risks and uncertainty

The principal product risk is that reusable program integration adds another layer of configuration and lifecycle machinery without reducing total engineering effort.
Exposing implicit assumptions may require substantial changes to existing applications, and a convenient initial example may conceal the cost of adoption and maintenance.
The project must evaluate those costs alongside the benefits.

The principal evidence risk is confidence that exceeds what the tests establish.
Repeated execution can reveal lifecycle defects, but a shared process cannot reproduce every failure or isolation boundary of a distributed deployment.
External dependencies, shared resources and host-specific behaviour can also make an otherwise repeatable test depend on uncontrolled state.
Evaluation must keep these conditions visible and relate test outcomes to the production decisions they support.

Resource ownership and cooperative lifecycle handling remain responsibilities that authors and hosts must fulfil.
Declared configuration cannot make an arbitrary implementation safe to combine, and cancellation cannot ensure that a program releases its resources correctly.
The toolkit's value depends partly on how effectively its practices help engineers discover and correct such shortcomings.

The workload breadth, useful evidence thresholds, adoption effort and sustainable maintenance cost remain uncertain.
Detailed system-testing practices, host policies and extensions for particular environments require further operational evidence.
Solution management adds separate questions about identity, handoff compatibility, authority and delivery economics.
Those questions can be investigated without making their resolution a condition for evaluating the core mission.

[M-02-concept-of-operations]: M-02-concept-of-operations.md
