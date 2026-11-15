---
status: draft
since: 2026-08-13
derived-from:
  - M-01-business-mission-analysis
derives:
  - M-03-adopting-the-toolkit
  - M-04-debugging-the-status-gateway
  - M-05-testing-a-partial-recovery
  - M-06-probing-git-operations
  - M-07-one-responsibility-model-across-platforms
---

# Concept of Operations

This document describes how developers and coding agents use the toolkit to investigate changes, test cooperating programs and run applications in production.
It follows the work from a question about system behaviour to a repeatable way of checking it, and from observations in operation back to development.
[M-01-business-mission-analysis] establishes the mission and the evidence needed to judge its value; the stories here place that intended way of working in particular settings.

## Operational setting

A team may deploy its Go services independently and still need several of them running together to investigate a change.
The developer needs to choose those programs, supply their inputs and inspect what happens as requests or messages pass between them.
The team already has source repositories, development tools, build pipelines and operating environments.
The toolkit makes application construction, configuration and execution available through Go packages that the team can use within those practices.
Across teams and successive maintainers, it provides a common agreement about application inputs, execution and completion, with maintained tools that use that agreement.

Application authors decide what their programs do and how they communicate.
The toolkit supplies hosts and testing support that can run the applications, and an organisation can build its own hosts for its operating environments.
Applications receive their inputs and participate in cancellation and failure handling; their internal libraries and communication protocols remain the authors' choices.
Following those practices makes an application usable by the supplied tools and by other hosts built around the same agreement.
[M-01 § System boundary and operating model] describes that division of responsibility.

Adoption can begin with one useful investigation or test.
The team can adapt the programs it needs while retaining its existing packaging and deployment path.
As [M-03 on adopting the toolkit] illustrates, this can involve separating application setup from an executable entry point, making hidden inputs explicit and preserving the old deployment configuration through an adapter.
The benefit has to justify that work and the cost of maintaining the resulting Go code.

## Working on a backend

### Choose what to run

A requested feature, a failed test or a problem reported by a user gives the developer a reason to run part of the backend.
The expected result comes from what the system is meant to do for its users.
That question guides which programs and versions to select, which inputs to supply and which interactions to try.
The useful selection may be much smaller than the complete production system.

The developer uses the toolkit's executable tools to find applications and inspect their documented inputs.
For a server, the developer can see which inputs set its listening address and the backend it connects to.

The developer writes or changes application code in Go packages.
The developer also writes or adapts a Go composition that selects the programs, supplies their inputs and invokes the toolkit's supplied host or testing support.
The supplied tools construct and run the applications, coordinate cancellation and wait for completion.
Some dependencies can run locally under that code's control; others can remain in staging or another authorised environment.
The composition shows which addresses, settings and resources each program receives, including where two programs deliberately share a dependency.
The developer must still obtain the credentials, data and external services needed for the run.

### Investigate and repeat

The developer runs the selected programs from the checkout being changed and follows the interaction through their usual interfaces.
A debugger, logs, metrics or traces can show how those programs produced the observed result.
Inputs chosen for the investigation can make the interesting condition quick to reach, such as the small guest quota in [M-04 on debugging the status gateway].

After an edit, the developer can run the composition again and try the same interaction.
The host coordinates cancellation and waits for the programs to finish; each program completes the work and cleanup it owns.
The developer arranges fresh state where the experiment needs it and deliberately retains state where the question concerns continuity between runs.
Resources shared with other users need separate consideration: restarting a local program does not reset a remote service or make its data stable.

Running several programs in one process lets the developer inspect their code together and can expose assumptions about shared state and resource use.
It is one useful execution arrangement for this work.
Production placement follows the service's operating needs, and conditions that depend on separate processes or networks still need to be exercised in an appropriate environment.

### Retain a system check

An investigation can reveal a useful check that the team wants to repeat whenever the participating programs change.
The developer keeps the setup in a Go test alongside the application code.
The toolkit's testing support starts the applications, requests cancellation when the run ends and waits for completion.
The test sends requests or messages through the programs' existing interfaces and checks the result at the boundary relevant to the user-visible behaviour.
The test author chooses those interactions and expected results; the toolkit runs the programs without interpreting their protocols or business meaning.

The test can control selected dependencies to hold a condition steady or exercise a sequence that would be difficult to arrange in a shared environment.
For example, the partial-recovery test in [M-05 on testing a partial recovery] lets Git Operations recover while Actions remains impaired, then checks the status response read by the page.
A program's package tests can pass while this check fails because the cooperating programs produce the wrong result together.

The team retains the test for later changes and runs it through the ordinary Go test workflow, locally and in continuous integration.
A failure gives the next developer a starting point for investigation.
A passing run establishes the behaviour exercised with the selected implementations, inputs and dependencies.
It informs the release decision alongside the checks required in deployment.

### Carry the work between people and coding agents

Keeping the composition and tests with the applications lets a colleague check out a revision and see which programs run together and how they are configured.
The colleague can repeat the check or reuse the same packages for a different investigation.
Access to the required external services and data remains necessary, and changes in those dependencies can affect the result.
The retained Go code makes the setup inspectable without depending on the original developer to reconstruct it.

A coding agent can use the same starting point.
The application inputs and retained composition show what to supply, while the supplied tools provide a way to run the applications without building another harness.
Given the reason for a change and the behaviour expected by its users, it can run the affected programs, reproduce a failure, edit the code and check the result.
When an attempt fails, it can inspect the interaction and try again within the same working session.
A retained system test gives it feedback about consequences across programs that their separate tests may miss.

The reviewer can examine the code together with the failing and passing results and judge whether the change addresses the requested behaviour.
Questions about customer needs or conditions the setup cannot exercise still need further investigation.
The organisation decides how much work to delegate and who may approve or deploy the result.
The business hypothesis is that this runnable context lets agents carry more changes through verification with less human reconstruction and intervention, as described in [M-01 § Mission and opportunity].

### Use the applications in production

Engineers build and deploy the applications through the team's intended packaging and delivery path.
Each operational host supplies the inputs and supervision needed in its environment, using the same application packages exercised during development.
The host may also integrate operating controls specific to that environment, such as readiness reporting.
Engineers building it can reuse the execution packages behind the supplied host, keeping the environment's input handling and operating controls in their own code.
Applications explicitly opt into additional capabilities they support.
The adopting organisation decides whether a custom host uses fallback behaviour or rejects an application that lacks a capability.
Any capability required by a host is an explicit condition of using that host.
Those integrations require implementation and verification even when the application code is reused.

The engineers responsible for the release check the actual deployed arrangement.
Routing, credentials, process isolation, scheduling, network conditions and platform behaviour can affect what users receive.
A local test can support a decision to release a change without establishing that the intended versions are running or that those operating conditions are correct.
The organisation retains authority over deployment and recovery.

The toolkit can also host programs that perform ongoing operational work, including probes that exercise a service through its public interfaces.
In [M-06 on probing Git operations], the deployed prober observes a delayed fetch that the engineer's local run does not reproduce.
The local result helps narrow the investigation while the deployed prober continues to observe the route where the delay occurred.

### Bring observations back into development

A production problem can start another investigation.
Engineers use the observed requests, results, software versions and settings to identify the programs and conditions worth examining.
They reuse or adapt the Go composition to recreate the part they can run, subject to access to the necessary software, dependencies and data.
If that setup reproduces the fault, a system test can preserve the check for a correction and later changes.
If it does not, the engineer still has to investigate the conditions that differ in operation.

Work can enter at any point in this account.
A support investigation need not begin with a new deployment path, and a team improving its tests can retain the production arrangements it already has.
The comparison with existing practice includes the time to investigate and verify changes, the human help agents need, and the cost of adapting applications and maintaining the compositions.
[M-01 § Mission and opportunity] describes how the organisation can judge whether that work produces a worthwhile improvement.

## Engineering GitHub Status

These stories imagine how the team behind [GitHub Status] might develop, test and run the Go programs behind the page.

In [M-03 on adopting the toolkit], the team already has a pipeline that builds container images and supplies private test environments.
Ziv still needs control over the programs and settings used in an investigation, so she brings their construction and composition into reusable Go code.
The toolkit's supplied host and testing support can then run the selected applications.
Application inputs become explicit, while the toolkit coordinates cancellation and failures.
An optional custom host translates the team's existing environment variables into documented application inputs, preserving its deployment configuration.
The services keep their internal libraries and communication protocols.

In [M-04 on debugging the status gateway], a signed-in visitor hits the limit meant for guests.
The developer follows the request in a debugger on her laptop, where she can clear the guest count and try again using status reports from staging.

In [M-05 on testing a partial recovery], a coding agent changes a program to send shorter reports about the services listed on the page.
A Go test catches the page giving an all-clear when Git pushes work again but Actions workflows are still stuck.

In [M-06 on probing Git operations], a long-running application pushes a commit to a private repository and checks what a client can fetch.
The push succeeds, but a fresh fetch still gets the old commit a minute later.
The operator gives the Git engineer the branch and the two commit IDs to trace.

## Running the same application in several clouds

[M-07-one-responsibility-model-across-platforms] follows an engineer using the supplied host and testing support to prepare an event service for local checks, then a coding agent using those checks to develop a validation change.
The cloud integrations reuse the application package and the toolkit's execution support.
Each deployment still requires checks through its own route and operating controls.

## Other operating settings

[M-08 § Concept of Operations] follows tailored solutions through delivery, change, customer use and retirement, with stories about demonstrations, a merchant service, factories and a disconnected plant.
The solution-management mission and its additional business acceptance criteria are developed in that document.

## Open operational questions

The GitHub Status stories illustrate one team's incremental migration and the work it can then perform.
They do not establish the effort required across different codebases, the ongoing cost of maintaining compositions or how often coding agents can complete changes with less human help.
Those questions require experience from actual use.

Starting a new service and adopting reusable programs supplied by other teams also need further examination.
The stories do not establish which workloads are representative or what software, data and access must remain available to repeat every investigation.
Testing APIs, host policies and compatibility rules remain design questions to be informed by this operational work.

[GitHub Status]: https://www.githubstatus.com/
[M-01-business-mission-analysis]: M-01-business-mission-analysis.md
[M-01 § System boundary and operating model]: M-01-business-mission-analysis.md#system-boundary-and-operating-model
[M-01 § Mission and opportunity]: M-01-business-mission-analysis.md#mission-and-opportunity
[M-03 on adopting the toolkit]: M-03-adopting-the-toolkit.md
[M-04 on debugging the status gateway]: M-04-debugging-the-status-gateway.md
[M-05 on testing a partial recovery]: M-05-testing-a-partial-recovery.md
[M-06 on probing Git operations]: M-06-probing-git-operations.md
[M-07-one-responsibility-model-across-platforms]: M-07-one-responsibility-model-across-platforms.md
[M-08 § Concept of Operations]: M-08-solution-management.md#concept-of-operations
