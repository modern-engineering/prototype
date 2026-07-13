---
status: draft
since: 2026-07-09
refined-by:
  - D-13-two-phase-enactment
---

# Mission Analysis: Stakeholders, Operations, and Measures of Effectiveness

## Context

The solution layer now has shape: [A-09 Solution Layer] names the roles and
recommends the artifact, [A-10 Value Binding] stages its values,
[A-11 Substrate and Slices] grounds it in shared services, and
[A-12 Operator I/O] pins the process contract underneath. The analyses reached
that shape bottom-up, concern by concern. INCOSE approaches from the other end:
business or mission analysis fixes who operates a system and how success is
judged before the system that must serve them is designed. This analysis
performs that step for the layer — deliberately now, before toolchain work
begins, so the records to come trace to operational need rather than to
implementation appetite. It does for the layer what [A-01 Scope] did for the
library: fix the frame the later records are judged against.

The mission, in one sentence: an organization authors each deployed purpose
once, as a definition, and operates its whole estate — cloud, on-premises,
laptops — from those definitions. The analysis proceeds in the INCOSE order:
stakeholders, a concept of operations (ConOps — how the organization intends to
run its business around the layer), operational scenarios (OpsCon — how the
system is operated day to day), and measures of effectiveness. Nothing here
decides a mechanism; the field cases constrain whatever mechanism a later decision
record picks.

## Stakeholders

[A-09 Solution Layer] divides the authoring work among three roles; operating
the result adds a fourth; three more stand at the edges of the layer. For each:
the stake, the artifacts touched, the leading concerns, and what good looks
like.

**The application developer** writes components on the application library and
publishes them as catalogues. Artifacts: component code, descriptors, the
published catalogue. Concerns: the component must never learn where it runs (the
indifference [A-01 Scope] and [A-09 Solution Layer] establish); the declared
configuration surface is the component's entire interface
([A-03 Parameterization]), and its documentation is the downstream contract
([A-07 Documentation]); a published schema is compiled against by solutions the
developer never sees. Good looks like: publish once, special-case no
environment, and know before publishing which solutions a change will break.

**The solution engineer** composes catalogued elements into definitions and owns
the purpose each solution serves. Artifacts: definition sources and the compiled
artifact handed onward. Concerns: expressiveness is bounded by the deliberately
austere value language ([A-10 Value Binding]); the catalogue is consumed through
its documentation, trusted sight-unseen; a definition must mean the same thing
at every site that reifies it. Good looks like: the definition states intent and
nothing else; every inconsistency surfaces at authoring time, positioned in the
sources, never at deployment time.

**The platform engineer** builds the deployment systems that enact solutions and
guarantees the substrate they slice into ([A-11 Substrate and Slices]).
Artifacts: controllers and emitters, site configuration, the substrate itself.
Concerns: a consumable artifact contract that does not shift under them; safe
convergence and pruning over stateful holdings; each archetype re-implements
reification, so the shared core must stay small. Good looks like: standing up a
new deployment system is a bounded project, and site variance lives in
configuration they own — never in edited definitions.

**The operator** runs the deployed estate: watches health, rotates secrets,
answers pages. Artifacts: the running processes and the records around them —
site bindings, effective values, the process contract of [A-12 Operator I/O].
Concerns: today's estates answer "why is this process configured this way" from
tribal memory; the operator needs the answer to be mechanical. Good looks like:
from any running process to its definition statement, its bound values, and its
catalogue schema in bounded steps.

Three secondary stakeholders sharpen particular edges. **The security reviewer**
audits what crossed which boundary: secrets must never enter the artifact,
sensitivity must propagate as taint, and every site binding must be attributable
([A-10 Value Binding]); good is an audit that is a read, not an archaeology.
**The demo engineer** sells with the playground archetype: the entire solution
as one process on a laptop, no infrastructure within reach. The single-process
litmus of [A-09 Solution Layer] carries this commercial weight, not only the
testing story. **The framework maintainers** own the library, the catalogue
conventions, and any toolchain; their concern is that every concept added to the
layer lands in four archetypes at once, so the conceptual core must grow slower
than the estate it describes.

## Concept of Operations

At the organization level, the intended operation reads like a software supply
chain. Development teams publish catalogues the way they publish internal
libraries: versioned releases with documentation, consumed downstream on the
consumer's schedule. Solution engineering maintains a portfolio of living
solutions, each an instance anchored to one purpose ([A-09 Solution Layer]);
per-customer variants are distinct solutions, authored, or generated, above the
layer. Platform teams guarantee substrate and operate an archetype portfolio:
the same solution is proven as a single process on a developer laptop, shipped
to Kubernetes for the cloud offering, and delivered to systemd hosts
on-premises. Handovers between roles happen on artifacts, not on meetings: a
catalogue release, a compiled solution, a site binding.

A multi-tenant production system we studied shows the before picture. One Go
binary statically links roughly sixty components behind a hand-maintained list;
every production process runs that same binary, per-component enable flags
selecting the one component (rarely a few) it hosts, and a per-process options
file wiring the rest. Deployment is an umbrella chart with one near-identical
service per component, whose values blocks repeat the same boilerplate keys
dozens of times. The composition works — the same code runs one-per-pod in
production and co-located in a laptop demo — but operating it hurts in named
ways: the loader list, the chart inventory, and the values files are
synchronized by hand, so adding a component is a three-place edit; per-site
options files drift with no common source to diff against; no single artifact
states what a deployment as a whole should be, so the desired state is the union
of values files and memory; and runtime identity is derived from instance names
without being declared, so renames silently break message-consumer state — the
team once named a new component after a retired one purely to inherit its
identity, and migrations are feared accordingly.

The layer's operational promise is the inversion of those pains: the definition
becomes the unit the business operates on. Review and approval happen on
definitions; the per-process and per-chart boilerplate becomes derived output;
divergence between sites becomes a recorded set of bindings rather than an
archaeology across values files. The studied system already attributes its
wiring files to a "solution" in name; the layer makes that attribution the
primary object.

## Operational Scenarios

Five system-level scenarios describe the layer in operation. Each is
mechanism-agnostic: it names what the stakeholder experiences, and thereby
constrains any implementation a decision record later picks.

**The authoring loop.** A solution engineer edits definition sources, checks
them, and reads the result. Checking validates every statement against the
catalogue's registered schemas; errors point at the file and line that caused
them, in the author's vocabulary, not the artifact's. A formatter keeps layout
canonical so review diffs carry only meaning. The loop runs locally, in seconds,
with no environment standing by.

**A catalogue change.** A developer retypes a parameter and publishes a new
catalogue release. Recompiling each solution against the release turns the
change into positioned authoring errors before anything ships, and the
organization can enumerate which solutions the release touches. A deployed
artifact keeps naming the catalogue it was compiled against, so a deployment
system facing newer schemas detects the skew instead of misreading records.

**Site deployment.** An engineer takes a compiled solution to a site. The site
binds the values it must and overrides the defaults it may — the staging
[A-10 Value Binding] develops — and nothing else about the solution varies. The
deployment system shows its plan before acting, converges, and records the
effective value of every symbol it resolved, leaving the site's variance fully
inspectable after the fact.

**The single-process run.** The same solution, unchanged, runs as one process:
substrate bindings point at local stand-ins, and every component hosts in a
single binary. A demo engineer walks a customer through the product on a laptop;
a CI job proves solution-level behavior in an end-to-end harness the same way.
This is the litmus of [A-09 Solution Layer] exercised as a daily operation, not
a compliance test. [D-13 Two-Phase Enactment] commits the mechanism this
scenario and the dev-loop measure constrain: one command generates and runs a
tailored host, and the same wet core serves the prebuilt hosts production
operates.

**The 3am incident.** An operator is paged on a misbehaving process. From the
process they reach, mechanically: which solution and instance it is, which
artifact generation is deployed, the effective value of every symbol at that
site, and the catalogue schema that types each parameter — ending at the
definition statement that put the instance there. In the studied system today,
that walk crosses chart values, per-process wiring files, environment variables,
and the memory of whoever last touched them.

## Measures of Effectiveness

This analysis recommends judging the layer by five measures, stated from the
stakeholder's operational viewpoint. They are working measures: a later
requirement locks the survivors and sets their thresholds.

- **Round-trip fidelity.** Everything a deployment does is derivable from the
  definition plus the recorded site bindings, and every fact in the running
  estate traces back to one of the two. This measure kills the
  desired-state-as-union-of-files failure mode outright.
- **Deployment-indifference coverage.** Every solution the layer accepts reifies
  on all four archetypes; a definition that fails one archetype is a defect of
  the layer, not of the site. Inherited from the litmus of
  [A-09 Solution Layer].
- **Dev-loop latency.** One command takes edited sources to a running
  single-process solution, within a budget of seconds. The demo engineer and the
  end-to-end harness live or die by this number.
- **Blast-radius visibility.** Given a catalogue change, the set of affected
  solutions and statements is mechanically enumerable before anything deploys.
  This is what makes catalogue releases an operation rather than a gamble.
- **Variance-audit completeness.** The divergence between any two sites is fully
  expressible as their recorded bindings; variance invisible to the record
  counts against the layer. The security reviewer's measure as much as the
  operator's.

## Open Questions

- **Identity ownership.** Who mints solution identity and allocates its
  generation counter — solution engineering, the platform, or tooling acting for
  either — and where that authority is recorded.
- **The home of site configuration.** Organizationally, whether site bindings
  live with platform teams, with per-site operators, or with the solution
  engineer; each answer redraws the review boundary.
- **Catalogue publication policy.** Release cadence, deprecation windows, and
  who signs off a breaking schema change given the blast radius it creates
  downstream.
- **Artifact schema stewardship.** The artifact's own schema is consumed by
  every deployment system; which stakeholder signs off its changes — framework
  maintainers who evolve it or platform engineers who consume it.
- **The operator as a distinct role.** In small organizations the platform
  engineer wears the pager; whether the layer's tooling may assume that collapse
  or must serve a separate operations function.

[A-01 Scope]: A-01-scope.md
[A-03 Parameterization]: A-03-parameterization.md
[A-07 Documentation]: A-07-documentation.md
[A-09 Solution Layer]: A-09-solution-layer.md
[A-10 Value Binding]: A-10-value-binding.md
[A-11 Substrate and Slices]: A-11-substrate-slices.md
[A-12 Operator I/O]: A-12-operator-io.md
[D-13 Two-Phase Enactment]: ../adr/D-13-two-phase-enactment.md
