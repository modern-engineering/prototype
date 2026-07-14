---
status: draft
since: 2026-07-14
---

# Static Deployment Rendering: From Image to Kubernetes Manifests

## Context

The corpus pairs each controller shape with one host and leaves a quadrant
empty. [A-09 Solution Layer] names the Kubernetes archetype "applications on
Kubernetes, enacted by an operator"; [D-08 Desired-State Image] sets the other
shape against the other host, "static emitters (a compiler from the image to
systemd units) as peers". The pairings are illustrations, not commitments:
controllers are two shapes over one pure core, and "a static emitter is the same
function run once" ([D-08 Desired-State Image]) — nothing stops that one run
from emitting Kubernetes manifests instead of systemd units. Yet no record
develops the combination. Static deployment rendering — a compiled image plus
one site's bindings in, Kubernetes manifests out, deployed by nothing smarter
than `kubectl apply` — is sanctioned in principle and designed nowhere.

Field evidence says the empty quadrant is where production starts. The
multi-tenant production system [A-13 Mission Analysis] studies ships on the
order of a hundred applicative services from statically templated manifests,
synchronized by a GitOps engine; no operator stands between those manifests and
the running estate. Its ephemeral development environments are the same
structure again — the production manifest set rendered per namespace, values
varying while the structure holds. The operator the archetype names is a later
rung: [D-13 Two-Phase Enactment] already promises "the operator and
service-manager archetypes consume the same plan when they land", and a live
loop is that plan run on a watch instead of once. Static rendering is the entry
point.

This analysis develops that rung's problem space: the stakeholder cut, the
forces any rendering design must answer — granularity, custody, provisioning,
convergence, naming, derivability — and the viable paths through them. The
subject is a function, image and site in, manifests out; explicitly not an
operator. [A-13 Mission Analysis]'s measures frame the judging: this work fills
the Kubernetes cell of deployment-indifference coverage, and it must hold
round-trip fidelity (manifests derivable from image plus site alone) and
variance-audit completeness (two environments differing exactly by their
recorded bindings).

## Stakeholders

[A-13 Mission Analysis] maps the layer's stakeholders; five stakes specialize to
this rung.

**The platform engineer** owns the renderer, the host image the pods run, and
every convention the manifests follow. "Standing up a new deployment system is a
bounded project" ([A-13 Mission Analysis]) is the budget: the rung must run on
the layer's shared machinery — the image schema on the rendering side, the plan
gate of [D-13 Two-Phase Enactment] inside the pods — rather than re-implement
either, and the conventions it mints — names, mounts, annotations, probe shapes
— are the platform's to own and version.

**The solution engineer** holds deployment indifference: "a definition must mean
the same thing at every site that reifies it" ([A-13 Mission Analysis]).
Everything Kubernetes-specific must come from the site and the platform's
conventions, never from edited definitions; a definition that must change to
render here is a defect of the rung, charged against the coverage measure.

**The operator** gains a desired state readable with the tools the estate
already holds: rendered manifests are kubectl-visible, diffable, and
attributable. The concerns are mechanical. A configuration change must roll
exactly the pods it touches; process exits must meet restart policy so wet
failures retry while configuration faults surface; and the audit of every
resolved binding — the guardrail [A-10 Value Binding] calls not optional — must
appear where an operator looks, redacted where taint demands.

**The security reviewer** asks that taint drive custody. A value tainted
sensitive lands in a Secret, never a ConfigMap, never the environment — the
channel [A-12 Operator I/O] already fences ("platform policy may veto secrets in
it") while mapping the document class to "a mounted secret volume on
Kubernetes". Rendering adds a custody question of its own: a rendered Secret is
secret material at rest in an artifact, and where such artifacts may live —
committed beside the manifests or held apart — decides whether the audit stays
"a read, not an archaeology" ([A-13 Mission Analysis]).

**The application developer** is unaffected by construction: "the component must
never learn where it runs" ([A-13 Mission Analysis]). Nothing below may leak a
Kubernetes fact into a component's parameter surface.

## What Is One Pod?

The image's records fix what a solution is, not how its processes group. Three
granularities compete.

**The whole solution in one pod.** The single-process archetype, containerized:
one Deployment whose host enacts the full image — [D-13 Two-Phase Enactment]'s
host already runs all instances concurrently. The cheapest rendering there is,
and it forfeits what multi-process deployment exists for: no independent rollout
or scaling, one instance's failure recycles every instance, every configuration
change rolls everything.

**One deploy record per pod.** The studied estate's production shape: one
component per pod, across the whole applicative estate. Rollout, scaling,
restart policy, and audit narrow to the record, and the manifest set derives
record by record. Pod-level tuning already has a sketched vocabulary riding the
image — [D-15 Discovered Stanza Schemes] draws its example scheme as `k8s.pod`
(replicas, a priority class) — advisory to every other archetype; a renderer is
the first controller to read it in earnest.

**Arbitrary grouping.** Between the poles sits co-location — several records
sharing a pod. [A-09 Solution Layer] leaves exactly this open: "whether the
layer carries co-location constraints (never assignments — the single-process
archetypes forbid those) is open". Rendering does not force the answer; it needs
only a default until a grouping vocabulary exists.

A field fact keeps the middle path honest: the studied estate runs one shared
binary image for dozens of components, and each pod selects the one component it
hosts at runtime through an enable flag delivered as environment configuration.
Per-record pods do not imply per-record binaries — what an image contains and
what a pod enacts are separate axes, and the viable paths below diverge exactly
there.

## What the Pod Runs

[D-13 Two-Phase Enactment]'s Mode P is the natural per-pod process: "one
prebuilt host serves many solutions", built once on the catalogue's release
cadence, enacting any image compiled against that catalogue. The pod
specification is then nearly constant — one platform binary, one entrypoint —
and per-pod variance reduces to which desired-state fragment and which site
fragment reach the process as mounted files and arguments. The image's catalogue
pin turns skew between a stale host and a newer solution into a refusal instead
of a misreading ([D-08 Desired-State Image]).

The exit contract meets Kubernetes restart policy exactly.
[D-13 Two-Phase Enactment] fixes zero for a clean exit, one for a wet failure,
two for a configuration fault. Under an always-restart policy a wet failure
retries and may heal; a configuration fault — an unbound extern, an unreadable
image, a catalogue skew — fails identically on every start and crash-loops
visibly, which is "hold configuration faults for a human" expressed in the only
vocabulary a Deployment has. The audit transcript, one line per resolved binding
with taint redacted, lands in the pod log — where the operator's bounded walk
from process back to definition begins.

## Rendering Binds the Site

Rendering is a site-binding act: its output is per-environment by definition,
sitting at the second of [A-10 Value Binding]'s three moments. `extern` symbols
MUST bind — rendering refuses on any unbound one, the same total gate
[D-13 Two-Phase Enactment] runs at the host boundary, only earlier and against a
recorded document rather than invocation arguments. `var` symbols MAY rebind,
and only rebinds render: unrebound defaults already ride the image every pod
reads. Output references resolve wherever provisioning runs — the next section's
question.

Where bound values physically land is [A-12 Operator I/O]'s delivery question
worn by one archetype. The studied estate delivers options as config-map keys
wired one by one into environment variables — each key declared once in the map
and once in the wiring, the double bookkeeping whose boilerplate
[A-13 Mission Analysis] records — plus one mounted file handed to the process as
its argument. The environment is the weaker custody: readable at equal
privilege, vetoable for sensitive values, per-key by construction. A mounted
file delivers a whole fragment at one path, the path rides as an argument, and
it is where [A-12 Operator I/O]'s class mapping already lands ("a mounted secret
volume on Kubernetes"). The estate's own load-bearing channel is its one mounted
file; the per-key wiring is its boilerplate.

Sensitivity splits the fragment. Taint is a property of the symbol
([A-10 Value Binding]), so a renderer splits mechanically — untainted bindings
into a ConfigMap, tainted ones into a Secret, mounted apart:

```
kind: ConfigMap                # what Ping1's bindings reach, untainted
data:
  bindings: |
    echoSubject = "com.acme.Echo"
---
kind: Secret                   # tainted values, never the config map
stringData:
  bindings: |
    natsAdmin = <bound from the site's secret store>
```

The estate splits this way by convention — credentials in Secrets, coordinates
in config maps — and we still observed real credentials committed in plaintext
values files, duplicated into the mounted options file, and one database's
password riding a config map. The lesson is structural, not moral: custody
applied by hand erodes; custody derived from declared taint cannot misplace a
credential in the rendered set. This is where two measures earn their keep.
Variance-audit completeness: two environments' rendered sets differ exactly by
their recorded bindings, and the ephemeral-environment evidence shows the
structure can hold constant while values vary. Round-trip fidelity: everything
in the manifests derives from image plus site, so a hand edit to rendered output
is the failure mode a renderer must leave no reason for.

## Where Provisioning Runs

[D-13 Two-Phase Enactment] fixes the order — PROVISION, then DEPLOY — and
deliberately not the process: "a controller may provision in one place and
deploy in another", with the cross-binary door's trigger "the first controller
whose provisioning runs where its deployments do not". A static rendering is
that first controller the moment provisioning leaves the pods. Three placements
compete.

**At render time.** The renderer runs the attach drivers and bakes the reported
outputs into the rendered fragments. This is the purest reading of "static" —
the manifests carry everything, pods only bind — and it costs the renderer its
purity: a build step now touches substrate, so re-rendering is a side effect
rather than a function, and tainted outputs flow through the renderer into
rendered Secrets, compounding the custody question above. It also pulls the
cross-binary door in earnest: the manifests become the typed transport for
outputs the split requires.

**In init containers.** The archetype's own before-the-application slot: an init
container runs the pod's PROVISION phase and passes outputs to the main
container over a shared volume. The phase boundary maps onto a native seam and
rendering stays pure, but the typed transport must exist all the same, the
mechanism is Kubernetes-only by construction, and the provisioning work
duplicates per pod exactly as below.

**At pod startup.** Each pod's host runs the attach drivers its own records
reach, then deploys — [D-13 Two-Phase Enactment]'s present shape, both phases in
one process, no door pulled. The cost is repetition: many pods re-run
provisioning against shared substrate, so drivers must stay idempotent under
concurrent re-verification. Today that is tolerable by construction: "`Attach`
is the whole verb set, v0" ([D-14 Driver-on-Type]), a verification re-run
verifies again, and attachments "are never pruned, never destroyed"
([A-11 Substrate and Slices]), so nothing destructive multiplies. The first real
slice driver ends the tolerance — creation and destruction owned by many pods at
once is a coordination problem — and that trigger arrives together with
[D-14 Driver-on-Type]'s slice-lifecycle door.

Pod startup is the placement that needs nothing invented, and this analysis
treats it as the default posture; it is a posture, not an answer, and the
question returns with the first slice.

## Convergence Without a Controller

A static rendering leaves no resident process, so the convergence discipline of
[D-08 Desired-State Image] — observe, diff, converge, prune — compresses into
the apply; [A-09 Solution Layer] already describes static emitters as
"converging on the next apply". Creation and update come with the apply for
free. Three parts of the discipline lose their enforcement point.

**Rolling on configuration change.** Kubernetes restarts pods when the pod
template changes, not when the content behind a mount does. The estate's answer
is mechanical and derivable: a checksum of the rendered configuration rides the
pod template as an annotation, so a configuration change is a template change
and the rollout follows. A renderer emits this without ceremony; which fragments
the checksum covers is a platform convention — the estate's covers its config
map and not its Secret, so credential rotation rolls nothing there.

**Pruning.** `kubectl apply` never removes what left the image. A live
controller diffs desired against observed; a static set has no memory of the
previous set. The estate leans on its GitOps engine, whose ledger of what it
last applied yields the difference to prune; a bare apply flow holds no such
ledger. Pruning deleted records is unsolved on this rung, named below rather
than papered over.

**Generation monotonicity.** [D-08 Desired-State Image] has a controller refuse
"a generation older than what it last reconciled"; in a static flow nothing at
the site remembers what was last reconciled. The memory could ride the applied
objects — an annotation carrying the generation, checked before apply — or live
in the pipeline's history; neither is designed, and without one this guardrail
silently vanishes on the archetype.

## Two Names for One Instance

Instance names are the image's reconciliation keys ([D-08 Desired-State Image]),
spelled as Go-style identifiers (`Ping1`, `natsAccount`); Kubernetes object
names are DNS labels. The mapping is mechanical — lower the camel case,
hyphenate the word breaks — and a renderer must make it total and refuse what
does not fit: collisions after case-folding, names over the 63-character label
bound.

The harder problem is one thing with two names. One studied component keeps a
legacy runtime name so that its message-consumer offsets survive, while its
deployment name says what the component now is; [A-13 Mission Analysis] records
the fear behind the pattern — runtime identity derived from instance names
without being declared, renames silently breaking consumer state. The rendered
surface has room for both (an object named one thing, a process told another),
and the metadata compartment can carry the public spelling opaquely
([A-09 Solution Layer]'s third audience). What nothing yet does is declare
runtime identity as a first-class fact; until something does, renaming stays
delete-plus-create ([D-08 Desired-State Image]) and the two-name pattern stays a
workaround at the site.

## What Cannot Be Derived

Two Kubernetes fixtures have no source in the image — by design, not omission.

**Services.** The image carries no port or listener concept: [A-12 Operator I/O]
hands I/O in as handles — "Ports bind ambiently; handles are given" — so no
record says that an instance listens, on what, or for whom. A Service, a stable
name for exactly that, is underivable. The studied estate agrees from the other
side: its peers find one another through fixed in-namespace DNS coordinates
carried as ordinary configuration values, and only a handful of components
expose a Service at all. Coordinates-as-values is the honest present shape;
deriving Services would need listeners declared far enough into the catalogue's
semantic classes to be visible — a catalogue evolution, not a rendering choice.

**Probes.** A liveness or startup probe presupposes a health surface, and the
corpus has none to point at: [A-12 Operator I/O] leaves readiness unification
open — one health capability serving notify sockets, probe endpoints, and
in-process checks. The estate probes a conventional health port on every pod,
proof of the demand and its shape; rendering inherits the gap and must not close
it cosmetically by emitting probes against endpoints no host promises.

## Viable Paths

Five ways to fill the quadrant. The first two refuse the rung as stated; the
last three answer the granularity question three ways.

**A live operator first.** Build the archetype as named and skip static
rendering. The operator holds state, so pruning, generation refusal, and
continuous convergence regain their enforcement point. Against it: the estates
we can observe run production without one, so it answers needs not yet expressed
while costing cluster privileges, resident identity, and an upgrade story of its
own; and starting static forfeits nothing, because the operator wraps "the same
function run once" in a loop ([D-08 Desired-State Image]) — the rendering core
built now is the operator's core later.

**Helm-chart emission.** Emit a chart and its values, joining the ecosystem's
tooling; the studied estate's whole production flow consumes charts today. But a
chart is a template awaiting values, and the image is already-resolved desired
state; emitting a template from a resolution re-opens exactly the variance
[D-08 Desired-State Image] closed — "Value variance between sites is symbol
rebinding at the site", "never merging". The chart's values surface would stand
beside the image's symbol table as a second binding language, and what runs
would again be explainable only from base plus values in render order: the shape
the layer exists to invert.

**One pod, whole solution.** The first granularity above. As the contract it
degenerates the rung; as a profile it is nearly free — the same rendering with
grouping turned to "all" — and gives demos and small estates a legitimate shape
on the archetype.

**Per-record manifests, per-record sub-image.** One pod per deploy record, each
pod's host enacting a derived image (a sub-image, as a working name) that holds
exactly its record plus the symbols and provisions the record's bindings reach.
Derivation is ordinary image production — the image schema is a public
composition surface, and "a composed image is indistinguishable downstream from
a compiled one" ([A-09 Solution Layer]) — so every pod runs the ordinary plan
gate over exactly its slice: extern gate, audit, catalogue pin, per pod,
machinery unchanged ([D-13 Two-Phase Enactment]). Custody is minimal by
construction: a pod's Secret carries what its slice needs and nothing else. The
costs: a derivation step must exist and be trusted — sub-images must carry the
parent solution's identity and generation, or [D-08 Desired-State Image]'s
guardrails fragment across the set — and the solution-level view (one symbol
table, one reference DAG) must be reassembled for whole-solution audits.

**Per-record manifests, runtime selection.** One pod per record, one full
configuration everywhere, an enable switch selecting each pod's record — the
studied estate's production pattern, proven at its scale. One artifact renders
once and pods differ by a switch. The costs are custody and surface: every pod
receives the whole solution's configuration, secrets included, so "which pod can
read this credential" answers "all of them"; and the switch itself is a host
surface no record designs — selection semantics become load-bearing deployment
intent that lives nowhere in the image today.

This analysis recommends the fourth path as the shared basic contract — every
pod an ordinary enactment of an ordinary image, nothing new demanded of hosts —
with the fifth demonstrably reachable as a platform profile for estates that
prefer one shared configuration and a switch: the selection surface is
deployment intent a platform may add, never definition content. Committing is
not this analysis's job; a decision record takes this up.

## Open Questions

- **Ports and Services.** The missing listener declaration above: whether
  listeners enter the catalogue's semantic classes far enough to derive
  Services, and what carries peer coordinates meanwhile, stays open with
  [A-12 Operator I/O].
- **Pruning a static set.** What remembers the previously applied set — the
  pipeline's history, a ledger object at the site, a GitOps engine above the
  flow — and whether the layer owns any of them.
- **Probes.** Pending [A-12 Operator I/O]'s readiness unification; and which
  probe parameters are derivable at all versus platform convention.
- **Secret custody modes.** Baked Secret manifests versus references into
  external secret machinery — the studied estate materializes its Secrets from a
  vault through an external-secret operator even in ephemeral environments —
  and, either way, where rendered secret material may rest.
- **Co-location and grouping.** [A-09 Solution Layer]'s compute-isolation
  question made concrete: what vocabulary expresses grouping without breaking
  single-process reification. Advisory stanzas in
  [D-15 Discovered Stanza Schemes]'s mold are the natural remote collector; none is
  designed.
- **The home of the site document.** Rendering is the consumer
  [D-13 Two-Phase Enactment]'s site-file door predicts — a whole solution's
  extern surface outgrows a command line, and the rendered set is exactly
  "bindings as an artifact" — while [A-14 Staged Catalogue Compilation] leaves
  "Where site configuration lives" open (in the image via plumbing, or beside
  it) and [A-13 Mission Analysis] leaves its organizational home open. This rung
  cannot ship without an answer it can read.

[A-09 Solution Layer]: A-09-solution-layer.md
[A-10 Value Binding]: A-10-value-binding.md
[A-11 Substrate and Slices]: A-11-substrate-slices.md
[A-12 Operator I/O]: A-12-operator-io.md
[A-13 Mission Analysis]: A-13-mission-analysis.md
[A-14 Staged Catalogue Compilation]: A-14-staged-catalogue-compilation.md
[D-08 Desired-State Image]: ../adr/D-08-desired-state-image.md
[D-13 Two-Phase Enactment]: ../adr/D-13-two-phase-enactment.md
[D-14 Driver-on-Type]: ../adr/D-14-driver-on-type.md
[D-15 Discovered Stanza Schemes]: ../adr/D-15-discovered-stanza-schemes.md
