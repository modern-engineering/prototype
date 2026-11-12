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
The toolkit codifies established engineering practices and supplies maintained hosts, testing support and packages that make those practices useful across applications and teams.
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
For example, a team may deploy each of its microservices independently yet still need several of them running together to check what a user sees after a change to one service.

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

These concerns recur across an organisation as different people develop and maintain its applications over time.
Teams can agree to make configuration explicit and handle shutdown carefully, yet their implementations can drift as deadlines, maintainers and dependencies change.
Each team may also build its own local runner, test harness and process-signal handling around those implementations.
The opportunity is to give those teams a common agreement backed by maintained tools, so adopting the practices also gives them useful capabilities without rebuilding that support for each application.

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

### Automation and developer control

An organisation can automate the path to trying a system by building container images from a developer's branch, deploying a private environment and returning an address to use.
This lowers the effort of obtaining an environment whose composition and configuration have already been prepared.
Trying a different subset of programs or changing how they are connected can still require changes to deployment configuration or the automation that interprets it.
The developer can submit code and inspect the deployed result, while the choices about assembling the system remain outside the ordinary edit, run and debug cycle.

Running containers locally can leave the same difficulty in place.
For example, a Docker Compose file can tell Docker which service and worker images to run and how to connect their containers.
A Go test can use that file to start those containers, but the file supplies no Go functions that the test can call to construct the worker with different inputs or reuse its setup in another test.
The developer still has to maintain a separate path for constructing the application from Go.

The project's motivating concern is that automation at the wrong layer often removes developers from the work they need to control.
The opportunity is to let them select the relevant programs, supply their inputs, pause execution and repeat the experiment through their development tools, with that assembly expressed in reusable Go code.

The project adopts the principle behind [the Go team's work on reproducible toolchains][Go-reproducible-toolchains]: The build tools should eliminate unintended dependencies on the build machine and make the source and other necessary inputs explicit.
Rebuilding from those inputs should produce the same executable without requiring the organisation to preserve a particular build environment.

The toolkit aims to offer a similar experience for assembling and running applications together.
Engineers should be able to check out a revision, supply its configuration and dependencies, and use the recorded Go composition to construct and run the selected programs again.
That composition can also be adapted for a different investigation or reused in a test.
The working hypothesis is that this gives engineers a repeatable way to exercise the system without depending on a previously prepared environment remaining available.
The external services and data needed for a particular run must still be supplied, and the deployed arrangement still requires thought.

## Mission and opportunity

The opportunity is for software organisations to make more useful changes with less effort spent assembling, running and testing their systems.
For engineers, this means shorter feedback cycles, less code and tooling to maintain around the applications, and better ways to check the effects of a change.

For organisations using AI coding agents, the further opportunity is to give an agent more of the work of developing and assessing a change.
**The business hypothesis is that agents can carry more of this work through to completion without a proportional increase in the human effort needed to guide and review it.**
This requires giving an agent both the reasons for the change and a practical way to run the system and check the result.

The project pursues these opportunities by making the code that assembles and runs the system part of the system's Go codebase.
That code selects the programs, supplies their configuration and dependencies, and starts and stops them.
Developers and agents can read it to see how the programs fit together, adapt it to run a useful subset and reuse it in tests or operational hosts.
A host is software that starts and supervises programs, supplying the dependencies and execution policy appropriate to its setting.

The toolkit brings together established practices for explicit configuration, cancellation, resource ownership and optional capabilities.
Application authors follow a common agreement about inputs, execution and completion, and implement how their programs do their work and stop safely.
Once applications follow that agreement, engineers can select them and supply their inputs to the toolkit's supplied host to run them together locally.
They can use the toolkit's testing support to run the same applications in Go tests.
Engineers building a custom host can reuse the packages behind the supplied host for execution coordination while implementing their environment's configuration and operating controls.

The toolkit's maintainers sustain those shared packages and tools as the practices evolve.
An adopting organisation still maintains its applications, compositions and environment integrations.
The intended gain is less repeated work implementing and maintaining runners and lifecycle coordination across teams, with more ways to exercise applications that already follow the agreement.
For a coding agent, documented application inputs, supplied execution support and retained tests provide a starting point for trying a change without a person first reconstructing each application's setup.

The focus on systems whose application programs are written in Go is deliberate.
A tool intended to support applications written in many languages has to accommodate their different build tools and runtimes.
Here, the code that runs the programs together can call their Go packages directly and reuse how the applications are constructed and configured.

Engineers can build and test the packages that assemble the system with the Go tools they already use for the applications.
Go modules record the dependency versions, and the Go toolchain can build the resulting program for different operating systems.
If a script starts the applications, each developer may first need to install the shell and commands that the script expects.
Calling the application packages from Go can spare the team that setup.

A system test written in Go records which programs the developer chose to run together and how they were connected.
When the test is committed alongside the applications, a colleague investigating an old fault can check out the affected revision and run the same programs together.
Suppose a test runs a service that accepts jobs together with the worker that processes them, but the fault was reported by someone waiting for a notification.
The colleague can use the same packages to start the service and worker, bring in the program that sends notifications and follow a job all the way through.

A developer or agent checking out a changed revision can start with one of these setups and send requests or messages through the programs' usual interfaces.
The reason the customer asked for the change guides which interactions to try and what to look for in the result.
They can verify whether the system does what was requested and gather evidence about whether that behaviour meets the customer's needs.
If a new request calls for a different test, they can change the Go code that sets up the run and retain that test for later work.
If the results leave questions unanswered, they can decide what else to try or what more they need to learn about customer use.

The toolkit is intended to make modern engineering practices easier to adopt and make it harder to avoid confronting basic design pitfalls.
Running cooperating programs or several instances within one process can expose assumptions about shared state, resource use and cleanup that a fresh process would hide.
Making an application work reliably in these tests can take extra effort.
Engineers have to make some assumptions explicit and configurable.
They also have to decide what graceful termination means for the work a program has taken on.
A worker asked to stop halfway through a job may need to finish it or leave it where another worker can pick it up; closing its connection to the queue is only part of stopping safely.

In a local test a developer can request shutdown while the worker holds a job, then repeat the failed run under a debugger to see where the job was lost.
The same failure is harder to diagnose after deployment, when the worker is running on a remote host and access is usually restricted.
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
The programs remain responsible for honouring those signals and completing the work and cleanup they own.

A program opts into an additional capability by implementing a Go interface for it.
For example, a service can implement an interface that signals when it is ready to receive traffic, and a Kubernetes host can report that signal to the platform.
When the interface is absent, the host follows documented default behaviour that respects the program's choice not to support that capability.
The basic agreement about inputs, execution and completion still applies.
A specialised host can require readiness reporting from the programs it runs, but that requirement must be explicit.
A program running in another host can use the toolkit without having to implement readiness reporting.

Application authors decide what their programs need in order to run.
For example, an author whose program writes files can let the caller supply the directory.
An engineer who wants a second program to read those files can pass the same directory to both in the Go code that starts them.
If instead the two programs use files with the same names for their own temporary work, the engineer gives each a separate directory.
Anyone investigating a file collision can follow those calls back to where the directory was chosen and see which programs received the same path.
Whether they should share it depends on what each program does with the files.

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

## What success looks like

Consider an organisation whose process for delivering a change has grown across several teams.
Each team checks the part it owns, yet faults in the way those parts work together have reached production.
Even after the teams fix such a fault, they have struggled to turn it into a regression test.

An early success would be a coding agent using the incident report and the team's Go code to run the affected programs together and reproduce the failure.
A reviewer could see the resulting test fail on the faulty revision and pass with the fix.
The longer-term gain comes when another team changes one of those programs and the test catches the same fault before it reaches users again.
What the organisation learned from the incident then protects later changes without depending on the engineers who investigated it.

## Downstream solution management

An organisation delivering tailored software services can build solution-management capabilities on the same program and composition foundations.
In this setting, recurring customer delivery can depend on specialist knowledge, shared repositories and platform-specific automation whose effects are difficult to relate to one customer solution.
Reusable application and provisioning code can support a more explicit delivery path, while solution identity and intended design connect that work to the beneficiary's needs.
Reducing routine dependence on specialists is an additional business objective of this application of the toolkit.

The solution-delivery stories linked from [M-02-concept-of-operations] examine this setting.
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
Declared configuration cannot make an arbitrary implementation safe to combine, and asking a program to stop cannot ensure that it will terminate gracefully.
The toolkit's value depends partly on how effectively its practices help engineers discover and correct such shortcomings.

The workload breadth, useful evidence thresholds, adoption effort and sustainable maintenance cost remain uncertain.
Detailed system-testing practices, host policies and extensions for particular environments require further operational evidence.
Solution management adds separate questions about identity, handoff compatibility, authority and delivery economics.
Those questions can be investigated without making their resolution a condition for evaluating the core mission.

[M-02-concept-of-operations]: M-02-concept-of-operations.md
[Go-reproducible-toolchains]: https://go.dev/blog/rebuild#making-a-build-reproducible
