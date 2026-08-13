# Working Vocabulary

This non-normative scaffold gives authors and reviewers consistent working language while the corpus evolves.
It remains a working vocabulary while any term or concept is still open.
Once every entry is settled through accepted corpus records, this scaffold will become the project glossary.
Until then, it neither changes the business mission nor makes provisional mission content authoritative.
Where an entry conflicts with [M-01-business-mission-analysis], the mission document governs according to its lifecycle status.
The scaffold may expose a sponsor-directed candidate correction, marked `provisional`, that requires a separate revision to the affected record before it governs the corpus.
The labels below describe the evidence for a vocabulary entry, not a document lifecycle state.

- `settled` — the term and concept are established by an accepted corpus record.
- `term open` — the concept is established, but its preferred term or exact keyword remains unsettled.
- `provisional` — the concept or its boundary still needs operational evidence.

- **Actor** — `provisional` — A person, organisation or automated tool that performs work in a responsibility domain; performing work does not by itself confer accountability or decision authority.
- **Adopting organisation** — `settled` — A B2B, software-centric organisation that incorporates the toolkit into the people, policies, tools and platforms through which it delivers tailored solutions.
  The present model encompasses integration and infrastructure-hosting responsibilities within this organisation's sociotechnical delivery system.
  _Adopting organisation_ is the preferred prose term; the one-word form is not recommended.
- **Application** — `provisional` — A reusable software capability made available through an application catalogue, from which solution intent selects a particular application and version for use in a logical solution.
  **Deploy** renders the intended application operational, while the application's exact catalogue and runtime boundaries remain open.
- **Approval** — `provisional` — An adopting-organisation governance activity or process state that accepts solution intent for some intended use.
  It does not define or change what the solution intent is.
  Its exact authority, timing, effects and place in the delivery lifecycle remain open, and the present model does not require it universally.
- **Authority** — `provisional` — The power to make a consequential decision, such as accepting solution intent for an intended use, authorising operational change, accepting deployment or accepting evidence; it must not be inferred merely from the actor that performs the work.
- **Backing capability / backing service** — `term open` — An operational capability prepared for or attached to a solution during **Provision**, whether dedicated to that solution or shared so that the solution receives a usable slice.
  The preferred umbrella term and the boundary between a capability, service and allocated slice remain open.
- **Capability gap** — `settled` — A delivery need that cannot be met using existing reusable software or platform behaviour and therefore requires an explicit capability change rather than routine delivery.
- **Catalogue** — `provisional` — An organisation's set of reusable application or backing-service capabilities considered available for composition or delivery.
  Its ownership, representation, discovery rules and relationship to solution intent remain open.
- **Cross-organisation product hypothesis** — `term open` — The proposition that materially different adopting organisations can reuse the same semantic contract and applicable reference tooling while retaining local execution authority and realisation.
- **Customer** — `settled` — The party that receives and uses the operational solution to achieve a business outcome; it is distinct from the adopting organisation.
- **Deploy** — `settled` — The activity that renders the intended applications operational.
- **Deployment slice** — `settled` — A bounded replacement of part of an incumbent delivery process in which solution intent produces an observable running service judged against criteria agreed in advance.
- **Design** — `provisional` — The activity that brings a logical solution into being by establishing its identity and solution intent, or later revises that intent without changing the solution's identity.
- **Document traceability** — `settled` — The property the corpus has when navigable document relationships let a reader move between related records, as established by [D-05-document-relationships].
  It is not a standalone field or index.
  Authors should qualify _traceability_ as _document traceability_ or _runtime traceability_ where the distinction matters.
- **Durable identity** — `settled` — The continuity by which a logical solution remains the same business and operational subject as its contents change, while two solutions remain distinct even when their present contents match.
- **Durable identity-and-intent handoff** — `provisional` — The mission's central handoff combines a logical solution's durable identity and solution intent with inspectable and reproducible know-how for transition into operation.
  The semantic contract preserves shared meaning at its boundaries but is not by itself the whole handoff.
- **Full customer thread** — `settled` — End-to-end evidence that a customer need becomes solution intent, is transitioned into operation and is validated through customer use with less routine dependence on specialist intervention.
- **Intermediate representation (IR)** — `settled` — A low-level actionable description of the intended deployment footprint that can be understood without solution identity.
  It is not the logical solution, its durable identity or its lifecycle record.
- **Location** — `settled` — An opaque, organisation-defined identity for a place where a solution may be available, such as a site, cluster, namespace, machine or region.
  The same solution can relocate between Locations, but simultaneous availability at two Locations is currently modelled as two logical solutions.
- **Logical solution** — `settled` — The composed whole an organisation commits to meet a customer need and the durable, identity-bearing subject of delivery, lifecycle, history and accountability.
- **Operate** — `settled` — The customer's use of the operational solution to meet its needs and produce evidence about whether it remains fit for those needs.
- **Operational isolation** — `settled` — The property by which each logical solution can be changed, placed and retired on its own terms.
  It does not promise security, compute, network, data, resource or failure isolation.
- **Operational solution** — `provisional` — Current corpus prose for a logical solution as realised and available for customer use, not a second identity-bearing subject.
  Operational stories may refine the boundary between the logical solution, its realised contents and customer use.
- **Owning entity** — `settled` — The business attribution recorded for a logical solution, such as a customer, account or shared pool.
  Ownership does not by itself grant lifecycle or deployment-acceptance authority.
- **Partial adoption** — `settled` — Legitimate use of a bounded subset of toolkit capabilities that demonstrates only the outcomes exercised and must not be presented as the full mission result.
- **Platform** — `settled` — An organisation-owned operational capability on which solutions run; it is part of the adopting organisation's delivery system, not part of the toolkit.
- **Platform readiness** — `settled` — The condition in which a platform, including an existing brownfield platform, can accept provision or deploy work.
- **Provision** — `settled` — The activity that renders the required backing capabilities usable by the solution.
- **Provisioning procedure / provisioning job** — `term open` — Inspectable and reproducible, target-specific know-how that prepares or attaches backing capabilities.
  The mission establishes the need for that know-how, but not whether its executable unit is a procedure, job, driver or another mechanism.
- **Reconciliation** — `provisional` — The organisation-controlled work of comparing intended and observed operational state and deciding or acting on differences.
  The adopting organisation retains reconciliation authority, but the mission does not prescribe a reconciler, control loop or execution model.
- **Relevant software provenance** — `provisional` — The mission's narrow provenance boundary identifies the reusable application and version selected in solution intent and the application and version observed in operation.
  Unqualified _software provenance_ may describe a broader chain outside this mission.
- **Responsibility domain** — `settled` — One of the enduring accountability domains of solution engineering, software engineering or platform/service engineering.
  Solution engineering is accountable for understanding the customer need, curating solution intent and validating that the operational solution fulfils that need.
  Software engineering is accountable for developing and verifying reusable software capabilities and their supported means of configuration and integration.
  Platform/service engineering is accountable for making platforms and backing services ready to receive solution work and providing the organisation-specific means to provision, deploy and sustain solutions there.
  A domain is not a prescribed team, and one actor may perform work in several domains while the accountable domain, performing actor, authorising authority and owning entity remain distinguishable.
- **Retire** — `settled` — The terminal activity that ends a logical solution's operational life while retaining its identity and history and never reassigning them.
- **Routine delivery** — `settled` — Tailoring, provisioning or deployment performed with reusable software, platform capabilities and established procedures that already exist.
- **Runtime traceability** — `provisional` — The connection between the application and version selected in solution intent and the application and version observed in operation within the mission's relevant software provenance boundary.
  It need not reconstruct every intermediate transformation.
- **Semantic contract** — `settled` — The shared meaning the toolkit requires participants to preserve at its handoffs while allowing tools and realisations specific to each adopting organisation.
  Its precise schemas, compatibility rules and versioning remain open.
- **Semantic handoff compatibility** — `term open` — A contextual relation in which two participants or implementations preserve the meaning required at one named handoff.
  “The proprietary consumer is semantically compatible with the reference producer at the solution-artefact handoff because it preserves solution identity and solution intent” is a complete relational use; unqualified “B is compatible” is incomplete.
  Semantic handoff compatibility does not make tools or artefacts interchangeable, and precise compatibility and versioning contracts remain open.
- **Semantic portability** — `provisional` — Preservation of a solution's identity and solution intent across handoffs and materially different targets, even when local representations and mechanisms differ.
  It does not imply byte-for-byte artefacts, equivalent operational behaviour or a complete record of every transformation.
- **Sociotechnical delivery system** — `settled` — The adopting organisation's combination of people, policies, tools, automation and platforms through which tailored solutions move from intent into operation and sustainment.
- **Solution artefact** — `settled` — The identity-bearing umbrella for material exchanged about a logical solution.
  A solution artefact is not the solution itself, and neither SEF nor IR should be used as its synonym.
- **Solution Exchange Format (SEF)** — `settled` — The project's serialised exchange format and one kind of solution artefact.
  Its schema, containment, carrier and custody remain open.
- **Solution intent** — `provisional` — The curated selection of software, its intended configuration and relationships for a logical solution.
  This definition is a sponsor-directed candidate correction to the accepted mission's use of _approved intent_ and does not govern the corpus until [M-01-business-mission-analysis] is separately revised and accepted.
- **Specialist intervention** — `provisional` — Delivery-specific work that the established routine-delivery path cannot perform because it requires specialist-held expertise, access or judgement, or a change to reusable software or platform capability.
  For example, a platform specialist uses restricted production control-plane access to add a customer-specific firewall route that the established provisioning procedure cannot express, without changing a reusable capability.
- **Sustain** — `settled` — The adopting organisation's work to keep an operational solution fit for customer use through monitoring, maintenance, support, repair, change and capacity planning.
- **System of interest** — `settled` — The toolkit when analysed within the next-larger context of the adopting organisation's sociotechnical delivery system.
- **Toolkit** — `settled` — The opinionated, open-source Go packages, supported extension mechanisms, semantic contract and reference tooling supplied by this project.
  It does not own the adopting organisation's infrastructure or replace its execution, reconciliation or deployment-acceptance authority.
- **Transition into operation** — `settled` — The direction of work from **Design** through **Provision** and **Deploy** that makes a logical solution available for customer use, without prescribing a mandatory linear workflow.

[D-05-document-relationships]: adr/D-05-document-relationships.md
[M-01-business-mission-analysis]: mission/M-01-business-mission-analysis.md
