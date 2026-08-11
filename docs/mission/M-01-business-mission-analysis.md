---
status: draft
since: 2026-08-05
---

# Business Mission Analysis

## Executive summary

B2B software organisations often begin tailored customer delivery with shared
repositories, specialist knowledge and platform-specific automation. As the number and
variety of solutions grow, that mix becomes a constraint. Responsibility blurs across
teams, approved intent becomes difficult to relate to what is running, and a change for
one customer can affect another. Routine delivery then depends on development
specialists who should be improving reusable capabilities, not bridging the same gaps
repeatedly.

This project is building an opinionated, open-source Go toolkit for organisations that
need a clearer way to carry a tailored solution from approved intent into operation.
The toolkit makes the solution itself a durable, identity-bearing subject of delivery.
It gives solution, application and platform responsibilities a common way to exchange
the solution's identity and approved intent. Its reference tools can be adapted to
different platforms and automation systems.

The intended change is organisational as much as technical. Approved intent records
what a solution is. Inspectable and reproducible transition and deployment know-how
explains how the organisation puts that solution into operation in a particular
setting. Keeping both explicit allows different specialists, tools and locations to
participate without losing the solution's identity, history or line of accountability.

The current prototype is an early increment, not the final scope of the toolkit. Its
first useful proof is narrow: replace part of an incumbent delivery process and produce
an observable running service whose footprint is acceptable against criteria agreed in
advance. Broader claims require a complete customer thread, repetition within an
organisation, and eventually evidence across materially different organisations.

## Business context

The motivating case is a brownfield software business delivering tailored solutions
across development, staging, production and laboratory environments. Its estate spans
cloud and local infrastructure. Some backing services belong to one solution. Others
are shared services from which each solution receives an account, database or other
slice. Provisioning crosses declarative systems, operational platforms and manual
actions. Several application versions must coexist because customers do not all move
together.

The delivery system grew around those realities without retaining clear boundaries. A
shared environment repository allowed a change for one solution to affect another and
contributed to a cross-solution failure. Layers of configuration obscured which
software version was intended and whether the running service reflected the approved
solution. Knowledge of how to prepare a platform, obtain a shared-service slice and
connect it to an application lived in bespoke tooling and specialists' memories.

Routine customer work therefore crossed into application, catalogue or platform
development. The issue was not which department performed it, but that delivery
required someone to change a reusable capability or extend platform-specific machinery.
That dependence limited growth and drew specialists away from improving capabilities
shared by many solutions.

GitOps and automation are present in this story, but neither is the root problem.
Automation can repeat a poorly bounded process, and a repository can version files
without making the identity and history of each solution clear. The underlying failures
are blurred responsibility, weak version and provenance traceability, cross-solution
blast radius, tacit transition know-how and routine dependence on specialists to
connect the pieces.

This case does not prove that every organisation has the same estate or severity of
problem. It does expose a broader risk: local practices accumulate faster than the
organisation develops a durable way to say what each customer solution is, how it
should be realised and who is accountable for the work.

## Mission and opportunity

The mission is to establish a durable identity-and-intent handoff for each logical
solution. A logical solution is the composed whole that the organisation commits to
meet a customer need. It is one living business and operational subject with its own
identity, lifecycle, history and accountability, not a disposable template or
incidental collection of deployment files.

Approved intent states what the solution is: its selected capabilities, intended
configuration and relationships. The other half of the handoff is inspectable and
reproducible know-how for transition into operation. It covers how backing capabilities
are prepared or attached, target-specific forms are derived and intended applications
become running services. The toolkit connects these concerns without pretending that a
portable declaration can encode every local practice.

The business objective is to grow routine tailored delivery without proportional growth
in delivery-specific application or catalogue work and platform-extension work. Routine
tailoring, provisioning and deployment should use capabilities that already exist. Work
is classified by the responsibility it fulfils, not by a person's title or department.
A solution engineer changing a catalogue component is doing application/catalogue
engineering. A platform engineer performing ordinary provision work is not thereby
extending the platform.

This handoff makes the intended change boundary explicit, giving the organisation a
basis to reduce avoidable coordination, contain change, and relate running services to
approved intent and relevant software versions. It also exposes missing capability.
When routine delivery needs new catalogue or platform work, the organisation can treat
it as an explicit improvement rather than hide it inside customer fulfilment.

## System boundary and operating model

The toolkit is the system of interest within an adopting organisation's wider
sociotechnical delivery system. In this document, an **adopter** is a B2B,
software-centric organisation that incorporates the toolkit into the people, policies,
tools and platforms through which it delivers tailored solutions. The adopter and its
customer are distinct. The organisation owns the delivery system. The customer receives
and uses the operational solution to achieve a business outcome.

The boundary separates four related things. The toolkit supplies a semantic contract,
Go packages, supported extension mechanisms and reference tooling. The logical solution
is the identity-bearing subject whose intent and lifecycle move through that contract.
The platform is an organisation-owned operational capability on which solutions run.
The customer outcome arises when the customer uses the operational solution. The
toolkit can contribute to that outcome, but producing an artefact neither transfers
platform ownership nor guarantees business results.

The operating model divides delivery work into three enduring responsibility domains:

| Responsibility domain | Enduring accountability |
| --- | --- |
| **Solution engineering** | Understand the customer need, compose and approve the solution's declared intent, and validate that the operational solution fulfils that need. |
| **Application/catalogue engineering** | Develop and verify the reusable software capabilities from which solutions are composed, including their supported means of configuration and integration. |
| **Platform/service engineering** | Make platforms and backing services ready to receive solution work, and provide the organisation-specific means to provision, deploy and sustain solutions there. |

These are domains of responsibility, not prescribed teams. One person, team or
automated actor may wear several hats. Full mission fulfilment still requires the
accountabilities to remain distinguishable, so that routine delivery is not mistaken
for catalogue development and platform extension is not hidden inside deployment.

Customer use and organisational sustainment are also distinct. The customer uses the
operational solution. The organisation sustains it through monitoring, support, repair
and change, and capacity planning. Both may involve all three domains, but neither
creates a fourth responsibility domain.

A platform is ready when it can accept provision or deploy work. It need not be empty
or new. A brownfield platform may already run solutions and shared services. The
organisation retains infrastructure ownership and responsibility for authorising and
executing operational changes, reconciliation and deployment acceptance, even when it
uses toolkit-supplied code.

## The durable handoff and the life of a solution

A logical solution is non-fungible. Two solutions do not become the same because their
present contents happen to match, and a solution does not become a new one merely
because its contents change. Its stable identity carries its lifecycle, history and
accountability through those changes. This is the unit of **operational isolation** in
the mission: each solution can be changed, placed and retired on its own terms. The
term does not promise security, compute, network, data, resource or failure isolation.

An owning entity records business attribution. One customer, account or shared pool may
own several solutions, and a shared-pool solution may serve several customers.
Ownership does not by itself grant lifecycle or deployment acceptance authority. Those
remain matters for the organisation's governance.

A **Location** is an opaque, organisation-defined identity for a place where a solution
may be available. It may denote a site, cluster, namespace, machine, region or
something else. Simultaneous availability through placements in two Locations is
modelled as two logical solutions. Each logical solution carries its own lifecycle,
even when the two solutions have matching contents. Relocation is different: operation
ends at the old placement and the same solution identity is deployed at the new
Location. Ending the old placement is not retirement. Blue/green delivery, warm
standby, disaster recovery and overlapping relocation remain unresolved until later
operational work supplies better evidence.

The handoff uses artefacts without equating the solution with a file. A **solution
artefact** is the identity-bearing umbrella for material exchanged about a solution.
The Solution Exchange Format, or SEF, is the project's serialised exchange format and
one solution artefact. The **intermediate representation**, or IR, is a low-level
actionable description of the intended deployment footprint, understandable without
solution identity. IR is not the lifecycle record. Artefact schemas, containment,
carriers and custody remain open.

Portability is semantic. Solution identity and approved intent should retain meaning
across handoffs and materially different targets, even when local tools realise them
differently. The toolkit does not promise identical bytes, portable derivation lineage
or equivalent operational behaviour. Relevant software provenance records the identity
and version of catalogue software declared for the solution and observed in operation.
Traceability relates that declaration to those observations and does not require a
record of every source-to-runtime transformation.

The solution's life is described through recurring activities, not a one-way state
machine. **Design** composes and approves intent. **Provision** establishes or attaches
the backing capabilities needed by the solution. **Deploy** turns the intended
applications into observable running services. **Use** is the customer's use of the
operational solution. **Sustain** is the organisation's monitoring, support, repair,
change and capacity work that keeps it useful. Design, Provision, Deploy, Use and
Sustain may all recur as needs and circumstances change.

**Retire** is terminal: it ends the solution's operational life, while its identity and
history remain retained and are not reassigned. Detailed sequencing, triggers, retries,
concurrency and recovery belong to a later operational concept and architecture. The
mission requires the meaning of the activities and their handoffs, not a premature
execution design.

## Product posture and adoption

The product is an opinionated open-source Go toolkit, not a mandate for one delivery
platform. It offers a common contract, supported extension points and coherent
reference tools. Organisations can compose its packages with local code and replace a
reference tool where another implementation preserves the required semantics.
Compatibility is a specific interoperability claim at a handoff, not a blanket
assertion that tools or artefacts are interchangeable. Precise compatibility and
versioning contracts remain open.

Adoption can be cumulative. An organisation may begin by passing a solution artefact
into an incumbent process, replace one bounded step, and extend use only where evidence
shows value. Partial adoption is legitimate, but its claims are correspondingly narrow.
An organisation that omits lifecycle history or software provenance may still use part
of the toolkit. It cannot claim the traceability or full mission effect that the
omitted capability supports.

This posture fits brownfield reality better than imposing platform uniformity. One
platform can reduce variation, but it moves the product boundary into infrastructure
governance and may exclude obligations the organisation cannot standardise away. A
universal automation layer has a similar cost: it absorbs target-specific execution
policy and duplicates existing deployment systems.

The toolkit leaves local execution local. It is not infrastructure as code, a universal
deployment engine or a reconciler, and does not select a target mechanism. It provides
shared meaning while the organisation retains infrastructure ownership, execution,
reconciliation, deployment acceptance and the local mapping of responsibility.

## Evidence and business acceptance

Evidence must grow with the claim. Technical completion alone does not establish
business acceptance, and a result in one organisation does not establish reusable
product demand elsewhere.

| Claim | Required evidence | Accepting authority |
| --- | --- | --- |
| **Deployment slice** | A bounded part of an incumbent delivery process is replaced using the toolkit contract or a semantically compatible replacement. An approved solution artefact produces an observable running service whose footprint is judged against the baseline, scope and local equivalence criteria agreed before the slice. | The organisation's delivery authority. |
| **Full customer thread** | A customer need is composed into approved solution intent, transitioned into operation and validated through customer use, with less routine dependence on delivery-specific application/catalogue or platform-extension work. | The function accountable for customer delivery. |
| **Repeated fulfilment in one adopter context** | Repeated solution changes and deliveries across materially heterogeneous environments demonstrate continuity of identity, semantic portability, and traceability from runtime to approved intent and relevant software provenance. They also demonstrate inspectable, reproducible transition know-how, distinguishable accountability and delivery growth without proportional specialist intervention. | Adopter leadership or its accountable business authority. |
| **Cross-adopter product hypothesis** | Materially different organisations reuse the same semantic contract and applicable reference tooling while retaining their own execution authority and local realisation. Compatible replacement proves compatibility where exercised, but not reuse of the replaced reference tool. | The project sponsor. |

The deployment slice is the first milestone because it can test the handoff without
claiming to replace a whole delivery system. The full customer thread follows because
customer use is needed to validate business fulfilment. Repetition strengthens the
organisation-level claim. Only accumulated evidence across different organisations can
support the product hypothesis, and even that does not by itself prove sustainable
support economics.

## Constraints, risks and uncertainty

The mission is shaped by a small number of enduring constraints:

- The toolkit is open-source and implemented in Go.
- Actionable intent must cross work separated by people, tools, time or place in
  serialised form.
- Full mission evidence must include materially heterogeneous delivery environments.
  One organisation-specific execution path is insufficient.
- The organisation retains infrastructure, execution, reconciliation and
  deployment-acceptance authority.

The principal risks are equally direct:

- Local extensions may drift until nominally shared intent has different meanings in
  different settings.
- Unbounded customisation may recreate bespoke delivery behind toolkit-shaped
  interfaces.
- Existing incentives may resist distinct accountabilities even when the same people
  continue doing the work.
- Shared repositories, platforms or services may preserve cross-solution blast radius
  despite adoption of the toolkit.
- Weak retention of identity, history or provenance may make a partial implementation
  appear to fulfil the whole mission.

Product reuse remains a hypothesis rather than proven market need. Maintenance and
support economics, and therefore long-term sustainability, are unknown. Workload
breadth, useful evidence thresholds, air-gapped operation, unresolved placement
patterns, semantic compatibility, and the design of artefacts, security and execution
remain open. Detailed choices belong to later requirements, the operational concept and
architecture.

These are known unknowns, not an exhaustive boundary around uncertainty. Early slices
and customer threads may reveal unknown unknowns that challenge the system boundary,
the operating model or the value claimed. The mission should remain durable in purpose
while being revised when evidence shows that its assumptions are wrong.
