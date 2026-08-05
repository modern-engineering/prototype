---
status: draft
since: 2026-08-05
---

# Business Mission Analysis

This document establishes the enduring product mission for the toolkit. The
prototype is the first genuine increment toward that mission, following a
period of useful but disconnected playgrounds; it is not the mission's time
horizon or its boundary. This analysis defines why the product should exist,
whose work it should improve, the outcomes that matter, and the boundaries and
uncertainties that downstream work must respect. It is not an operational
concept, requirements specification, architecture, roadmap, or implementation
plan.

The sponsor is the sole human authority for the mission. The rest of the
project team consists of LLM agents working through loop-engineering practices,
but the quality of that development process is evidence about how the product
is built, not evidence that adopters need the product. INCOSE Business/Mission
Analysis supplies useful vocabulary and themes; the project is not adopting an
INCOSE process wholesale. This document remains a draft until the sponsor
accepts its exact revision. Operational-concept work begins only after that
acceptance.

## The business problem

The intended adopter is a B2B, software-centric engineering organization that
delivers tailored software solutions to customer entities. Such an organization
often wants to serve more customers without growing a small R&D team in direct
proportion to customer count. The adopting engineering organization is the
product customer; this is not a B2C delivery model.

A representative brownfield organization tried to make all delivery GitOps.
Its development clusters run on GCP, staging and production run on AWS, and lab
VMs simulate edge clusters. Some customer environments are on-premises or
air-gapped. Some backing services, such as Kafka and Postgres, are redeployed
per solution or namespace. Others, such as NATS and Keycloak, are shared
products from which a solution receives a slice. Some provisioning is expressed
as Helm and ArgoCD configuration; other provisioning requires operator commands
against services that are already running.

That process does not distinguish operational layers or their ownership.
Solution engineers depend on development for every customer delivery. A shared
environment repository permits accidental edits across solutions and has
already enabled a serious failure. Multiple application versions must coexist,
yet their provenance is buried in Helm values and layers of overrides; inspecting
a running pod is often easier than recovering committed intent. On-premises and
air-gapped delivery magnify all of these weaknesses.

GitOps is not the underlying problem. Deployment intent is authored and acted
on by different people and tools, in different places and at different times,
without an explicit, durable handoff or a useful separation of concerns. The
organization has configuration, automation, and repositories, but it lacks a
stable conversation about what a solution is meant to be and which operational
layer is responsible for making each part real.

## Product mission

The product mission is to enable a software-centric organization to transform
deployment from bespoke, error-prone management of environment-specific
configuration into a conversation among key stakeholders about intent and
operational layering. Frontends express actionable solution intent in a shared
intermediate representation; backends act on that intent later and elsewhere.
This allows complex, multi-component services to be provisioned and deployed
across heterogeneous environments without ad hoc adjustment for every target.

The mission is fulfilled through business outcomes, not through the mere
existence of an artifact format or a collection of tools:

- tailored B2B customer delivery can scale without scarce R&D involvement
  scaling linearly with it;
- intent is portable between heterogeneous deployment environments;
- a running process can be traced to a stable solution identity, its intended
  configuration, and the relevant software provenance;
- deployment knowledge becomes explicit and composable instead of remaining
  trapped in bespoke tools or individual people; and
- responsibilities can be separated cleanly even when one person performs
  several of them.

The product changes how an adopter communicates and hands off work. It may
leave the resulting deployment footprint exactly as it was, or consciously
equivalent to it. Preserving every existing process is neither expected nor
desirable when those interactions are the source of the problem.

## The system of interest in context

Three contexts must remain distinct when discussing the mission:

| Context | Meaning in this analysis |
| ------- | ------------------------ |
| **Project-owned toolkit** | The open-source framework contracts, Go packages, and reference tools maintained by this project. |
| **Adopter's sociotechnical delivery system** | The people, internal software, policies, infrastructure, third-party products, and toolkit components that an adopter composes to deliver and operate solutions. |
| **Logical solution** | An identity-bearing, non-fungible object managed through that delivery system over its lifecycle. |

The toolkit is the product under development, but its value appears only inside
the adopter's delivery system. A logical solution is neither the toolkit nor
the delivery system: it is the durable object whose intent and history those
systems manage. Confusing these contexts would cause the project either to
claim control over an adopter's whole organization or to reduce a solution to
one generated deployment.

## People whose work must improve

The mission uses roles to expose responsibilities, not to demand a particular
org chart. One human may perform several roles.

**Solution engineers** tailor the software offering to a customer's needs
before deployment and validate in operation that those needs were met. They
need clear control of solution intent and independence from development/R&D
for routine customer delivery.

**Backend engineers** build long-running Go programs and make them available
as composable catalogue entries, rather than only as standalone `main`
packages or container images. They need lightweight ways to instantiate,
configure, run, and verify their software before deployment, followed by
production evidence. The detailed Go interfaces that might support this are
design hypotheses, not mission commitments.

**Platform engineers** keep locations, infrastructure, and backing services
ready. They expose finite provisioning jobs, service slices, or attachments to
applications, then monitor backing services, service levels, and capacity.
Platform practice varies more than the other concerns and may remain partly
manual. The toolkit does not seek to formalize or automate all infrastructure
work.

**Release and operations personnel** need deterministic provenance and
confidence at each handoff. Explicit solution boundaries can also let
compliance personnel find deployed vulnerable dependencies and let finance
attribute resources to solutions or customers. These are examples of questions
the product could enable, not mandatory initial scope.

AI agents may participate at layers an adopter selects. Explicit boundaries can
make intent authoring, extension development, operation, and verification more
tractable for agents as well as humans. AI participation is enabled by the
product posture; it is not a dependency an adopter must accept.

## The durable handoff across the lifecycle

The working black-box lifecycle vocabulary is:

**Design -> Provision -> Deploy -> Operate -> Retire**

- **Design** is where a solution engineer tailors a solution design and a
  frontend compiles it into a Solution Exchange Format (SEF) artifact.
  Compilation is an activity within Design, not an additional lifecycle phase.
- **Provision** executes finite jobs against backing services and wires their
  outputs into application inputs.
- **Deploy** turns intended applications into running services. It is the point
  at which intent becomes a running service.
- **Operate** covers monitoring, verification, support, capacity, change, and
  continued responsibility until retirement.
- **Retire** ends operation while preserving the solution's identity and
  history.

These phases are deliberately described from the outside. Their detailed
stories, participants, triggers, and exception paths belong to later work.
What matters to the mission is that the handoffs can be disjoint in time and
place. Actionable intent therefore has to cross them in serialized form rather
than depend on shared memory, a particular workstation, or an operator
reconstructing it from target-specific configuration.

Provisioning work is finite, noninteractive, cancellable, and reports a clear
terminal failure. Retry, compensation guidance, and state reconciliation may
become optional capabilities, but this mission does not settle them. In
contrast, deployed applications are initially expected to be long-running
services; other workload modes remain open.

## Concepts that frame the mission

### Solution identity and isolation

A solution always has an explicit, stable identity. It is a non-fungible object
whose contents may change over time and the atomic unit of lifecycle and
operational isolation. Atomic does not imply dedicated compute, network, or
data isolation.

An owning entity—a customer, account, or shared pool—may own multiple
solutions, including multiple solutions at one location. Two customers normally
have separate solutions even when their current configurations match. A
deliberately shared-pool solution may serve several customers with looser
isolation.

Simultaneous availability across two locations is represented as two
solutions. By contrast, relocating a solution from EKS to `systemd` preserves
the solution's identity: the old placement operation ends and the same
solution is deployed in the new environment, with both events retained in its
audit history. Whether
blue/green concurrency represents one or multiple operational instances or
revisions under one solution identity is unresolved and carries enough risk to
require expert treatment later.

### Locations, artifacts, and intent

A Location is an opaque, organization-defined identity. An adopter may map one
to a site, cluster, namespace, VM, serverless region or zone, or something else.
The toolkit carries constraints concerning a location but assigns no universal
semantics to it.

A solution artifact carries the solution's stable identity; the term is an
umbrella rather than a single format. A SEF is an exchange-format file and is
definitely one of a solution's artifacts. The intermediate representation (IR)
is the low-level, actionable description of intended deployment footprint and
can be understood independently of the solution's identity. Its actionable
content must be serializable, but the exact SEF contents, the IR's shape, and
the relationship between supporting artifacts remain open.

This mission does not assume a formal pipeline of layered or derived SEFs.
Advanced plumbing may edit SEF internals, while outputs such as Kubernetes
manifests are generated projections; neither fact establishes a universal
derivation model.

### Platform readiness and execution authority

A platform's operational idle state means behavioral readiness: it can accept
provisioning and deployment work as a ready server can accept requests. A
brownfield estate need not literally contain no solutions if it provides the
same capabilities. A reference zero-solution state may have reachable
locations and minimally operating shared services, but no solution-owned
resources.

The project will not build a reconciliation controller. Mature adopters can
integrate products such as ArgoCD; reconciliation authority and policy remain
part of the adopter's delivery system.

Air-gapped packaging is likewise specific to a customer and organization. A
SEF is desirable at the final target for traceability, but need not be present
there when an accepted bundle contains generated manifests and executables or
images. The mission does not prescribe one universal bundle.

## Product posture and adoption

The intended product is an opinionated open-source toolkit with supported
extension points and a reference operating model. Adopters are expected to
maintain proprietary Go extensions for their specific needs. Making that
specialization straightforward is preferable to endlessly parameterizing one
universal tool.

An adopter may replace the open-source tooling at every layer. Replacement
tools should preserve interoperability through compatible artifacts. An adopter
may also choose to abandon a capability such as provenance; the product does
not prevent that choice, but the adopter bears its operational and business
cost.

Adoption is cumulative and branching rather than all-or-nothing. A common
early stem is to emit intent through a solution artifact into an existing
proprietary deployment process. From there, an adopter can use more toolkit or
replacement tooling above or below that handoff where it creates value. This
posture does not promise that every toolkit layer is independently useful or
that every unusual workflow will be supported.

The first credible completion point for adoption is replacing at least one
complete slice of an existing deployment or continuous-delivery process with a
solution-artifact-based handoff. The result must produce the same deployed
solution footprint, or a consciously accepted equivalent, and be accepted by the
organizational function responsible for tailored customer delivery—typically
solution engineering or operations.

## How progress will be judged

Product and mission evidence must show useful portability of intent between
deployment environments without unconsciously changing the deployed solution
footprint. It should also show whether the handoff improves the responsible
function's ability to deliver a complete slice without bespoke R&D
intervention.

Two other evidence classes are valuable but must not be mistaken for customer
outcomes:

- **Architectural learning** includes abilities that emerge from composable
  catalogue, IR, and tool boundaries. Emergence is a desired architectural
  quality and a learning signal, not something a customer buys by itself.
- **Delivery-process quality** includes LLM agents producing fine-grained,
  reviewable increments with strong GitHub issue, pull-request, and code
  quality. It judges the project's execution, not its business mission.

The experimental `compiler-twoshot`, `k8s-deployments/k8s-deployment`, and
`snippets` branches are proof-of-concept evidence only. Their mechanisms are
not product or architecture decisions. The first end-to-end prototype value
stream and demo remain deliberately undecided until the mission and later
operational work provide a sound basis for selecting them.

No numeric targets are asserted yet. Baselines and measurable thresholds must
be learned rather than invented.

## Constraints, preferences, and hypotheses

The strength of each statement matters:

| Strength | Current position |
| -------- | ---------------- |
| **Fixed constraint** | The toolkit implementation language is Go. |
| **Necessary constraint** | The actionable IR is serializable across disjoint handoffs. |
| **Current technical constraint, pending later justification** | Solution exchange retains enough Go module provenance to resolve or detect the intended catalogue code version. The sponsor currently envisions embedding `go.mod` and `go.sum`. |
| **Current preferences** | Use the standard-library `flag` package; express configuration as strings or string-parseable values; generate one executable per major phase; focus first on long-running backend applications. |
| **Hypotheses to test** | Distinguish variables, secrets, and ordinary values; allow sealed secrets inside exchanged artifacts. |
| **Undecided** | The CLI name. |

None of these statements authorizes this analysis to design interfaces,
schemas, secret handling, command names, or packaging.

## Outside the mission

The product is:

- not another infrastructure-as-code system;
- not a universal deployment engine or a new reconciler;
- not an attempt to automate every immature or manual platform operation;
- not a means of preserving every adopter's existing process unchanged;
- not a promise that every layer is useful in isolation or that every unusual
  workflow is supported; and
- not a B2C delivery product.

These boundaries keep the project focused on durable intent and organizational
handoffs rather than absorbing every adjacent infrastructure and operations
problem.

## Uncertainty and evolution

Several mission-shaping questions remain open. The precise invariants behind
“intent portability” and the minimum interoperability contract for replacement
tools and non-SEF supporting artifacts are not yet known. Neither are the exact
adoption layers and branch points, including whether particular layers create
useful value alone.

Operational work still has to establish the detailed stories and whether its
concept is best communicated as one document or a bundle of sequential
stories. It must also inform selection of the first prototype value stream and
demo. Blue/green concurrency under one solution identity, air-gapped bundle
patterns, workload modes beyond long-running applications and finite platform
jobs, and optional retry, compensation, and reconciliation interfaces all
remain unresolved.

Artifact questions are intentionally open: SEF container fields; the IR model;
mutation rules; textual or binary representations; the provenance mechanism;
and whether a later operational format should be layered. The CLI name,
measurable baselines, and numeric success thresholds are also unknown.

This list cannot contain the unknown unknowns, which are expected. At the
prototype stage, document evolution and breaking changes are acceptable when
new evidence exposes a mistaken boundary or assumption. Preserving uncertainty
here is a design input, not a failure to finish the mission analysis.
