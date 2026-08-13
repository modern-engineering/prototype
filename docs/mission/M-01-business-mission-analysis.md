---
status: draft
since: 2026-08-05
---

# Business Mission Analysis

## Executive summary

B2B software organisations often begin tailored customer delivery with shared repositories, specialist knowledge and platform-specific automation.
As the number and variety of solutions grow, that mix becomes a constraint.
Responsibility blurs across teams, approved intent becomes difficult to relate to what is running, and a change for one customer can affect another.
Routine delivery then depends on development specialists who should be improving reusable capabilities, not bridging the same gaps repeatedly.

This project is building an opinionated, open-source Go toolkit for organisations that need a clearer way to carry a tailored solution from approved intent into operation.
The toolkit makes the solution itself a durable, identity-bearing subject of delivery.
It gives solution, application and platform responsibilities a common way to exchange the solution's identity and approved intent.
Its reference tools can be adapted to different platforms and automation systems.

The intended change is organisational as much as technical.
Approved intent records what a solution is.
Inspectable and reproducible transition and deployment know-how explains how the organisation puts that solution into operation in a particular setting.
Keeping both explicit allows different specialists, tools and locations to participate without losing the solution's identity, history or line of accountability.

The current prototype is an early increment, not the final scope of the toolkit.
Its first useful proof is narrow: replace part of an incumbent delivery process and produce an observable running service whose footprint is acceptable against criteria agreed in advance.
Evidence of broader value requires a complete customer thread, repetition within an organisation, and eventually results across materially different organisations.

## Business context

The motivating case is a brownfield software business delivering tailored solutions across development, staging, production and laboratory environments.
Its estate spans cloud and local infrastructure.
Some backing services belong to one solution.
Others are shared services from which each solution receives an account, database or other slice.
Provisioning crosses declarative systems, operational platforms and manual actions.
Several application versions must coexist because customers do not all move together.

The delivery system grew around those realities without retaining clear boundaries.
A shared environment repository allowed a change for one solution to affect another and contributed to a cross-solution failure.
Layers of configuration obscured which software version was intended and whether the running service reflected the approved solution.
Knowledge of how to prepare a platform, obtain a shared-service slice and connect it to an application lived in bespoke tooling and specialists' memories.

Routine customer work therefore crossed into changes to reusable applications and platform capabilities.
The issue was not which department performed it, but that delivery required someone to change a reusable capability or extend platform-specific machinery.
That dependence limited growth and drew specialists away from improving capabilities shared by many solutions.

GitOps and automation are present in this story, but neither is the root problem.
Automation can repeat a poorly bounded process, and a repository can version files without making the identity and history of each solution clear.
The underlying failures are blurred responsibility, weak version and provenance traceability, cross-solution blast radius, tacit transition know-how and routine dependence on specialists to connect the pieces.

This case does not prove that every organisation has the same estate or severity of problem.
It does expose a broader risk: local practices accumulate faster than the organisation develops a durable way to say what each customer solution is, how it should be realised and who is accountable for the work.

## Mission and opportunity

The mission is to establish a durable identity-and-intent handoff for each logical solution.
A logical solution is the composed whole that the organisation commits to meet a customer need.
It is one living business and operational subject with its own identity, lifecycle, history and accountability, not a disposable template or incidental collection of deployment files.

Approved intent states what the solution is: its selected capabilities, intended configuration and relationships.
The other half of the handoff is inspectable and reproducible know-how for transition into operation.
It covers how backing capabilities are prepared or attached, target-specific forms are derived and intended applications become running services.
The handoff standardises what must retain its meaning across environments, while target-specific procedures remain explicit in the adopter's tooling.

The business objective is to grow routine tailored delivery without proportional growth in specialist intervention.
Here, specialist intervention means delivery-specific changes to reusable software or platform capabilities.
Routine tailoring, provisioning and deployment should use capabilities that already exist.
The nature of the work, not the person's title, determines whether it is specialist intervention.
Extending a reusable application for one delivery is software development.
Running an established provisioning procedure is routine delivery; changing that procedure is platform development.

The handoff makes the change boundary visible.
Teams can see which solution a change belongs to and relate the running software to its approved intent and version.
It also reveals capability gaps.
When a delivery needs new reusable software or new platform behaviour, the organisation can plan that improvement explicitly instead of concealing it within customer fulfilment.

## System boundary and operating model

The toolkit is the system of interest within an adopting organisation's wider sociotechnical delivery system.
In this document, an **adopter** is a B2B, software-centric organisation that incorporates the toolkit into the people, policies, tools and platforms through which it delivers tailored solutions.
The adopter and its customer are distinct.
The organisation owns the delivery system.
The customer receives and uses the operational solution to achieve a business outcome.

The boundary separates four related things.
The toolkit supplies a semantic contract, Go packages, supported extension mechanisms and reference tooling.
The logical solution is the identity-bearing subject whose intent and lifecycle move through that contract.
The platform is an organisation-owned operational capability on which solutions run.
The customer outcome arises when the customer uses the operational solution.
The toolkit can contribute to that outcome, but producing an artefact neither transfers platform ownership nor guarantees business results.

The operating model divides delivery work into three enduring responsibility domains:

| Responsibility domain | Primary concern | Enduring accountability |
| --- | --- | --- |
| **Solution engineering** | Approved solution intent | Understand the customer need, compose and approve the solution's declared intent, and validate that the operational solution fulfils that need. |
| **Software engineering** | Reusable application catalogue | Develop and verify the reusable software capabilities from which solutions are composed, including their supported means of configuration and integration. |
| **Platform/service engineering** | Ready platform estate and backing-service catalogue | Make platforms and backing services ready to receive solution work, and provide the organisation-specific means to provision, deploy and sustain solutions there. |

These are domains of responsibility, not prescribed teams.
One person, team or automated actor may wear several hats.
Full mission fulfilment still requires the accountabilities to remain distinguishable, so that routine delivery is not mistaken for software development and changes to platform automation are not hidden inside ordinary deployment.

Customer use and organisational sustainment are also distinct.
The customer uses the operational solution.
The organisation sustains it through monitoring, support, repair and change, and capacity planning.
Both may involve all three domains, but neither creates a fourth responsibility domain.

A platform is ready when it can accept provision or deploy work.
It need not be empty or new.
A brownfield platform may already run solutions and shared services.
The organisation retains infrastructure ownership and responsibility for authorising and executing operational changes, reconciliation and deployment acceptance, even when it uses toolkit-supplied code.

## The durable handoff and the life of a solution

A logical solution has a durable identity.
Two solutions remain distinct even when their present contents match, and a solution retains that identity as its contents change.
The identity carries the solution's lineage, lifecycle, history and accountability through those changes.
This is the unit of **operational isolation** in the mission: each solution can be changed, placed and retired on its own terms.
The term does not promise security, compute, network, data, resource or failure isolation.

An owning entity records business attribution.
One customer, account or shared pool may own several solutions, and a shared-pool solution may serve several customers.
Ownership does not by itself grant lifecycle or deployment acceptance authority.
Those remain matters for the organisation's governance.

A **Location** is an opaque, organisation-defined identity for a place where a solution may be available.
It may denote a site, cluster, namespace, machine, region or something else.
Simultaneous availability through placements in two Locations is modelled as two logical solutions.
Each logical solution carries its own lifecycle, even when the two solutions have matching contents.
Relocation is different: operation ends at the old placement and the same solution identity is deployed at the new Location.
Ending the old placement is not retirement.
Blue/green delivery, warm standby, disaster recovery and overlapping relocation remain unresolved until later operational work supplies better evidence.

The handoff uses artefacts without equating the solution with a file.
A **solution artefact** is the identity-bearing umbrella for material exchanged about a solution.
The Solution Exchange Format, or SEF, is the project's serialised exchange format and one solution artefact.
The **intermediate representation**, or IR, is a low-level actionable description of the intended deployment footprint, understandable without solution identity.
IR is not the lifecycle record.
Artefact schemas, containment, carriers and custody remain open.

**Semantic portability** applies to meaning, not implementation.
A solution's identity and approved intent must still mean the same thing after crossing a handoff or moving to a materially different target.
Local tools may represent and act on that meaning differently.
The toolkit does not promise byte-for-byte artefacts, a portable record of every transformation or equivalent operational behaviour.

Relevant software provenance is narrower.
It identifies the reusable application and version approved for the solution, and the application and version observed in operation.
Traceability connects those two points; it need not reconstruct every transformation between them.

The solution's activities have a direction: their purpose is to put a solution into customer use and keep it useful:

- **Design** brings the logical solution into being by establishing its identity and approved intent.
  Later **Design** work can revise that intent without changing the solution's identity.
- **Provision** renders the required backing capabilities usable by the solution.
- **Deploy** renders the intended applications operational.
- **Operate** is the customer's use of the solution to meet its needs.
  The resulting experience and outcomes reveal whether the solution continues to satisfy those needs.
- **Sustain** is the adopter's work to keep the operational solution fit for use.
  It includes routine monitoring and maintenance, operational support and repair, and capacity planning.
- **Retire** is terminal.
  It ends the solution's operational life, while its identity and history remain retained and are not reassigned.

```mermaid
flowchart TD
    subgraph T["Transition into operation"]
        D["Design"]
        P["Provision"]
        DP["Deploy"]
        D --> P --> DP
    end

    subgraph O["Operational life"]
        direction LR
        U["Operate"]
        S["Sustain"]
        U -- "feedback" --> S -- "support" --> U
    end

    R["Retire"]

    DP --> U
    U -. "needs change" .-> D
    S -..-> P
    S -..-> DP
    U --> R
    S --> R
```

The dotted return arrows distinguish two reasons to repeat work:

- A new or changed customer need, or evidence during **Operate** that the solution no longer satisfies that need, returns the solution to **Design**.
  The approved intent may change, but the solution retains its identity.
- Operational needs identified during **Sustain** may return the solution to **Provision** to restore, replace or scale backing capabilities, or to **Deploy** to restore the intended applications to operation.
  This work leaves the approved intent unchanged, provided that the intent still reflects the customer's need.

When an activity's result changes, downstream results must be reassessed; an unaffected result may be reaffirmed without repeating the activity that produced it.
Detailed triggers, retries, concurrency and recovery belong to the later operational concept and architecture.

## Product posture and adoption

The product is an opinionated open-source Go toolkit, not a mandate for one delivery platform.
It offers a common contract, supported extension points and coherent reference tools.
Organisations can compose its packages with local code and replace a reference tool where another implementation preserves the required semantics.
Compatibility means that two participants preserve the required meaning at a particular handoff.
It does not make their tools or artefacts interchangeable.
Precise compatibility and versioning contracts remain open.

Adoption can be cumulative.
An organisation may begin by passing a solution artefact into an incumbent process, replace one bounded step, and extend use only where evidence shows value.
Partial adoption is legitimate but demonstrates only the capabilities actually used.
An organisation that omits lifecycle history or software provenance may still use part of the toolkit, but has not established runtime traceability or the full mission outcome.

The toolkit's boundary ends before target-specific provisioning and execution.
After the handoff, adopter-owned tools carry out that work.
For example, Terraform may provision a cloud database required by a solution, and Kubernetes may run the solution's containerised applications.
The toolkit carries the solution's identity and approved intent into those steps; the adopter configures, authorises and operates both tools.

Accordingly, the toolkit is neither infrastructure as code nor a universal deployment engine or reconciler.
It does not select target mechanisms.
It provides shared meaning while the organisation retains infrastructure ownership, execution, reconciliation, deployment acceptance and the local mapping of responsibility.

## Evidence and business acceptance

Each step below answers a different question and is accepted by a different authority.
Technical completion alone does not establish business acceptance, and a result in one organisation does not establish that others need or can reuse the product.

| Evidence level | What must be demonstrated | Accepted by |
| --- | --- | --- |
| **Deployment slice** | A bounded part of an incumbent delivery process is replaced using the toolkit contract or a semantically compatible replacement. An approved solution artefact produces an observable running service whose footprint is judged against the baseline, scope and local equivalence criteria agreed before the slice. | The organisation's delivery authority. |
| **Full customer thread** | A customer need is composed into approved solution intent, transitioned into operation and validated through customer use, with less routine dependence on specialist intervention. | The function accountable for customer delivery. |
| **Repeated fulfilment in one adopter context** | Repeated solution changes and deliveries across materially heterogeneous environments demonstrate continuity of identity, semantic portability, and traceability from runtime to approved intent and relevant software provenance. They also demonstrate inspectable, reproducible transition know-how, distinguishable accountability and delivery growth without proportional specialist intervention. | Adopter leadership or its accountable business authority. |
| **Cross-adopter product hypothesis** | Materially different organisations reuse the same semantic contract and applicable reference tooling while retaining their own execution authority and local realisation. Compatible replacement proves compatibility where exercised, but not reuse of the replaced reference tool. | The project sponsor. |

The deployment slice is the first milestone because it can test the handoff without replacing a whole delivery system.
The full customer thread follows because customer use is needed to validate business fulfilment.
Repetition shows whether the result persists across solution changes and heterogeneous environments.
Only accumulated evidence across different organisations can support the product hypothesis, and even that does not by itself prove sustainable support economics.

## Constraints, risks and uncertainty

The mission is shaped by a small number of enduring constraints:

- The toolkit is open-source and implemented in Go.
- Actionable intent must cross work separated by people, tools, time or place in serialised form.
- Full mission evidence must include materially heterogeneous delivery environments.
  One organisation-specific execution path is insufficient.
- The organisation retains infrastructure, execution, reconciliation and deployment-acceptance authority.

### Product and mission risks

- The common contract may be too weak to preserve meaning across heterogeneous environments, or too restrictive to fit them without bespoke workarounds.
  Either outcome would **defeat semantic portability**.
- The toolkit may introduce new integration and coordination work without reducing specialist intervention.
  Technical adoption would then fail to achieve the business objective.
- The contract and reference tooling may prove reusable in only one adopter context.
  The result would be a local solution rather than a **reusable open-source product**.

### Adoption risks

- Local extensions may assign different meanings to the same shared concepts.
  The same solution artefact could then pass validation in two environments but be interpreted differently, breaking semantic portability.
- Extension points may allow every adopter to build a different, incompatible delivery model.
  The toolkit would then add another interface without reducing bespoke integration or enabling reuse.
- Existing authority or incentives may conflict with the responsibility model.
  Routine delivery could remain dependent on software and platform specialists even after the toolkit is adopted.
- The toolkit distinguishes logical solutions, but cannot isolate every shared implementation.
  Shared repositories, platforms, backing services, credentials, or workflows may still allow a change for one solution to disrupt another.
- If identity, history or provenance are not retained, the adopter cannot reliably relate a running service to its approved solution.
  A partial implementation may still look complete, creating false confidence in its traceability and accountability.

### Known and unknown uncertainty

The **known unknowns** include whether materially different organisations will adopt the same contract and whether the project can sustain maintenance and support.
Workload breadth, useful evidence thresholds, air-gapped operation, concurrent placement, semantic compatibility, and the design of artefacts, security and execution also remain open.
These decisions belong to later requirements, the operational concept and architecture.

Deployment slices and customer threads may expose **unknown unknowns**: needs or constraints not anticipated here.
They may require us to change the system boundary, operating model or expected value.
The mission's purpose should remain stable, but the analysis must change when its assumptions are wrong.
