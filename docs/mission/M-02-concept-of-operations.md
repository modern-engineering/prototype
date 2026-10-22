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
  - M-07-controlled-disconnected-delivery
---

# Concept of Operations

This document describes the intended use of the toolkit in developing, testing and operating distributed Go backends.
It follows the work of engineers from an application or system change through investigation, testing and use in production.
The accompanying stories examine particular operating conditions, including the delivery and sustainment of tailored customer solutions.

## Operational setting

[M-01-business-mission-analysis] establishes the mission: reduce engineering effort and make system testing that informs production decisions convenient to apply.
The toolkit sits within the adopting organisation's software engineering and operating practices.
Engineers continue to use their source repositories, Go development tools, build and deployment systems, operational hosts and observability tools.
The opportunity is to make the application code and the means of assembling and running it usable together through those practices.

The work may concern one application, cooperating programs or a selected part of a larger backend.
An engineer can begin with the part needed to develop a change or investigate a failure while other services remain in their existing environments.
Using the toolkit does not depend on creating a catalogue, assigning a logical solution identity or reorganising delivery responsibilities.

Applications remain responsible for their specialised work and communicate through the mechanisms chosen by their authors.
The toolkit gives the surrounding Go code access to their construction, configuration and execution.
A test harness or operational host can therefore use the application implementations directly while supplying the dependencies and execution policy appropriate to its setting.
The organisation still has to implement, maintain and verify those integrations.

## Working on a backend

### Assemble the system needed for the work

A feature change, an integration fault or a problem reported from operation gives an engineer a reason to run some part of the backend.
The engineer identifies the participating applications, the configuration and dependencies they need, and the behaviour to investigate.
The useful boundary depends on that question: it may include several cooperating programs without including the whole production estate.

Through Go code, the engineer constructs and configures the selected programs and supplies their dependencies to a host or harness.
The source makes it possible to inspect which construction code runs, where a dependency comes from and how it reaches a program.
Existing packaging and deployment arrangements can remain in use while this path is introduced.

Some dependencies may run under the harness's control; others may remain external services.
That distinction affects setup, failure handling, resource ownership and what a test can establish.
The engineer assembling the system coordinates deliberate sharing, including access to resources that survive an individual run.
Programs account for resources they acquire, while hosts account for resources they supply, as described in [M-01 § System boundary and operating model][M-01-boundary].

### Test behaviour through a complete run

A system test can use the assembled programs within the ordinary Go test workflow.
It interacts through the selected system boundary, using the applications' existing requests, messages or other interfaces.
The test author specifies the expected behaviour; the toolkit need not understand the protocol or the business meaning of an exchange.
Logs, metrics and traces help the engineer explain an observed result.

The run includes the lifecycle needed to obtain that result and leave the system in its intended condition.
The harness starts and supervises the selected programs, handles completion or failure according to its policy, requests cancellation when appropriate and observes whether the programs finish.
The programs must respond and complete their own cleanup.
Provisioning and deprovisioning procedures participate where the applications depend on them, with their resource effects included in the run's outcome.

A successful request is therefore only part of the evidence.
The test may also need to establish that background work has stopped, owned resources have been released and deliberately retained resources remain usable.
Repeating a run in the same process can expose retained state, a listener left bound or work that survives shutdown.
Running concurrent instances can reveal assumptions about exclusive resource use.
Such failures give the engineer concrete reasons to correct application lifecycle handling, make dependencies explicit or revise how the composition supplies and shares resources.

The useful result is a system whose behaviour and lifecycle can be investigated repeatedly.
Execution in a shared process is one means of obtaining that feedback.
It does not determine production placement or reproduce all the conditions of separate processes, networks and operational platforms.

### Carry the change into its operating environment

The engineers responsible for deployment use the application implementations through the intended operational hosts, packaging and delivery path.
Each host supplies the configuration, dependencies and supervision required in that environment.
Whether reuse reduces total effort depends on the integration work each host still requires.

Local test results inform the deployment decision, but the actual operating arrangement introduces conditions that need their own verification.
Process isolation, scheduling, dependency availability, network behaviour and platform shutdown policy can change the outcome.
The responsible engineers establish whether the programs start, cooperate, fail and stop as required through that arrangement.

An existing service may remain in use during the change.
Its operators judge the observed behaviour and act through the deployment and recovery procedures available in their setting.
A local harness does not take over those decisions or establish that an operating change is safe merely because the same application code passed its tests.

### Bring operational evidence back into engineering

A failure, an unexpected result or a changed workload can require further investigation after deployment.
Engineers use observations from operation to identify the relevant software, configuration, dependencies and conditions.
They then recreate the part they can investigate through supported construction and execution paths, subject to access to those versions, resources and evidence.

Reusing an implementation helps make that investigation practical; it does not make a local setup equivalent to the observed deployment.
The engineer distinguishes behaviour reproduced by the setup from conditions known only through operational observations.
A correction may concern application code, configuration, composition, a host integration or an operating procedure.
The relevant tests and deployment checks then provide evidence for accepting that correction.

Work can enter this cycle at any of these points.
A team adopting the toolkit for a support investigation need not first replace its delivery system, and a team improving system tests need not adopt every hosting capability.
The comparison with existing practice follows both the immediate work and the cost of maintaining the new path.
[M-01 § Evidence and acceptance][M-01-evidence] defines the evidence needed to judge the engineering benefit and the limits of production-confidence claims.

## Solution delivery as a further application

When an organisation delivers tailored services, the same program integration can support a longer operational thread.
Beneficiary needs inform a solution design; supported procedures make required backing capabilities usable, and deployment makes the selected applications available for use.
Use then produces evidence for sustainment or renewed design.
The logical solution supplies continuity across those changes, including retirement, through the lifecycle meanings in [M-01 § The durable handoff and the life of a solution][M-01-life].

That continuity matters when different people or tools prepare and perform the work, or when a deployment must remain understandable long after the original preparation.
A solution artefact can carry the identity, intent and supported delivery knowledge across those handoffs.
A revision records an intended edition without requiring every running part to reach it simultaneously.
Factory windows, staged rollouts and recovery may leave parts at different versions while they still serve the same logical solution.

Within this setting, solution engineering connects beneficiary needs to the intended design and its usefulness, software engineering maintains reusable capabilities, and platform/service engineering supports their delivery and operation.
These are the responsibilities described in [M-01 § Solution delivery responsibilities][M-01-delivery-responsibilities], rather than prescribed teams.
People and automation may perform work across them; authority to approve a change or accept its result follows the operating arrangement.

The existing stories include both vendor-operated services and a customer-controlled plant.
In the latter, the vendor supports the delivery means while the customer controls local execution, deployment acceptance and the release of evidence.
The plant's allocation is explicit in its story.
What remains unresolved is how the common responsibility model covers both vendor-operated and customer-operated settings, where work and authority fall to different organisations.
The toolkit itself supplies neither infrastructure ownership nor authority to make those decisions.

Solution management adds outcomes beyond easier program integration.
A delivery path must preserve the beneficiary's intended service and reduce the routine dependence on specialist knowledge, access or intervention.
Those outcomes require evidence from delivery and use, as set out in [M-01 § Downstream solution management][M-01-solution-management].

## Existing operational stories

The stories describe counterfactual uses in established product settings.
They make particular engineering and delivery consequences visible without establishing measured benefits or representative coverage.
Each retains its own technical conditions and allocation of responsibility.

### [M-03-sales-led-demonstration]

In a counterfactual Elastic, an online retailer investigates payment latency through a temporary vendor-operated demonstration using authorised production telemetry.
The sales engineer can revise and retire the demonstration through a supported delivery path while some backing infrastructure remains shared.
The episode connects resource ownership and deprovisioning to a complete operational life: stopping dedicated applications alone would not remove the demonstration's remaining accounts, routes and data.

### [M-04-evolving-operational-solution]

In a counterfactual Stripe, a merchant's need for broader payment classifications leads to changes in an existing service's configuration and capacity.
A later application upgrade fails, and operators restore the earlier revision while retaining the required configuration and capacity.
The episode distinguishes intended change from observed operation and shows why application behaviour, workload and recovery have to be judged in the running service.

### [M-05-distributed-industrial-operation]

In a counterfactual Siemens, one Big Hammer predictive-maintenance service spans cloud software and separately scheduled factories.
Supported upgrade software carries repeatable delivery knowledge across those windows while the service continues to provide maintenance value.
Different versions and interrupted connectivity are part of the operating conditions, and factory staff retain authority over production windows and local acceptance.

### [M-06-one-responsibility-model-across-platforms]

In a counterfactual Twilio Segment, a multi-tenant service receives customer events through serverless container environments in AWS, Azure and Google Cloud.
A validation change passes from local application checks through solution selection into three distinct deployment paths.
The episode connects reusable application code to several operational hosts while retaining the need to verify each integration and the organisation's chosen configuration.

### [M-07-controlled-disconnected-delivery]

In a counterfactual Cisco, North Basin Water extends Cyber Vision monitoring at an isolated treatment plant using material prepared in Cisco's connected environment.
Customer engineers perform the local change, and evidence from a later standby-pump alert returns through the customer to inform a correction.
The support investigation recreates the retained software configuration while relying on customer-supplied evidence for actual plant behaviour.
The episode connects repeatable execution to support across time and place, with customer control over local operation and evidence.

## Unresolved operational coverage

The existing stories mainly concern delivery and change in established services.
They provide useful conditions for the broader mission but do not yet give a detailed account of engineers assembling a modest distributed Go backend, repeatedly testing it through its external interfaces and correcting a lifecycle defect found that way.
The effort required to introduce that practice, the tooling used and its effect on later production decisions remain unexamined in a concrete story.

Greenfield adoption and discovery of independently supplied reusable programs also remain unexplored.
The present stories do not establish how much application refactoring is normally needed, how representative their workloads are or whether benefits recur across organisations.
A portable handoff and an accessible application implementation each help preserve part of the knowledge needed for execution; neither establishes that the available dependencies, data and operating conditions are sufficient for reproduction.

Testing APIs, host policies, artefact schemas, deployment mechanisms and compatibility rules remain open.
The operational concept requires those choices to support the work and evidence described here, without selecting a common implementation for all settings.
The document remains incomplete while the core engineering work and its costs lack that concrete operational account.

[M-01-business-mission-analysis]: M-01-business-mission-analysis.md
[M-01-boundary]: M-01-business-mission-analysis.md#system-boundary-and-operating-model
[M-01-evidence]: M-01-business-mission-analysis.md#evidence-and-acceptance
[M-01-solution-management]: M-01-business-mission-analysis.md#downstream-solution-management
[M-01-delivery-responsibilities]: M-01-business-mission-analysis.md#solution-delivery-responsibilities
[M-01-life]: M-01-business-mission-analysis.md#the-durable-handoff-and-the-life-of-a-solution
[M-03-sales-led-demonstration]: M-03-sales-led-demonstration.md
[M-04-evolving-operational-solution]: M-04-evolving-operational-solution.md
[M-05-distributed-industrial-operation]: M-05-distributed-industrial-operation.md
[M-06-one-responsibility-model-across-platforms]: M-06-one-responsibility-model-across-platforms.md
[M-07-controlled-disconnected-delivery]: M-07-controlled-disconnected-delivery.md
