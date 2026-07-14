---
status: proposed
since: 2026-07-14
refines:
  - A-15-static-deployment-rendering
  - D-13-two-phase-enactment
---

# Render Kubernetes Deployments as Static Manifest Sets

## Context and Problem Statement

[D-13 Two-Phase Enactment] fixed the wet half in the small: a pure plan over
image plus live catalogue, a PROVISION then a DEPLOY phase, and two host shapes
— the prebuilt Mode-P binary production operates and the generated host of the
dev loop. Above those hosts nothing is decided. The Kubernetes archetype heads
[A-09 Solution Layer]'s list, [D-08 Desired-State Image] promised static
emitters as peers of live loops over one pure core (desired state in, actions or
artifacts out), and no record yet says what a Kubernetes deployment of a
solution is: how many pods, what each pod carries, where site values rest, and
who validates what against which catalogue.

[A-15 Static Deployment Rendering] opens that problem space for the
manifests-at-rest shape: one solution image and one site's bindings in, static
Kubernetes objects out — held in a repository, reviewed, applied. It cuts the
stakeholders (the platform engineer builds it, the operator reads it, the
security reviewer audits it), names the forces — granularity, custody,
provisioning, convergence, naming — and closes recommending per-record sub-image
sharding, with the runtime-selection shape kept reachable as a platform profile.
This record commits to that recommendation and fixes the architecture around it.

The raw material is in place. The Mode-P host is a committed contract: one
prebuilt binary consuming an image file and repeatable name=value extern
arguments, exiting 0 clean, 1 on wet failure, 2 on configuration fault. The
community stanza vocabulary ([D-15 Discovered Stanza Schemes]) already spells
pod tuning (`k8s.pod`) and workload grouping (`k8s.workload`), and no controller
yet reads either. And the multi-tenant production system [A-13 Mission Analysis]
studied shows the target shape operating at scale — one shared binary behind
per-component pods, configuration files split by sensitivity — a shape a
validation exercise has since regenerated from an image and a site alone, no
definition source consulted. This record fixes the static half of the Kubernetes
archetype: what the renderer is, the grain of the manifest set, the payload each
pod carries, where site values rest, and the seam a platform profile may
re-skin.

## Decision Drivers

- Deployment indifference ([A-09 Solution Layer]'s litmus, measured by
  [A-13 Mission Analysis]): the Kubernetes shape may add nothing to the image —
  the renderer consumes what every archetype consumes, and the image gains no
  Kubernetes dialect.
- The dry/wet seam ([A-14 Staged Catalogue Compilation]'s boundary, hosted by
  [D-13 Two-Phase Enactment]): a renderer links no live catalogue, so whatever
  it checks is advisory; authoritative validation must stay at the point that
  links one — the pod's host.
- Taint custody is structural, not policy ([A-10 Value Binding]'s taint,
  [A-15 Static Deployment Rendering]'s custody force): which artifact a
  sensitive value may rest in must fall out of the artifact shapes themselves,
  never out of reviewer vigilance.
- Determinism: a manifest set is desired state at rest, committed and diffed
  (the GitOps posture), so one image and one site must render one byte string,
  generation after generation — [D-08 Desired-State Image]'s diffable-artifact
  discipline extended to what the image becomes.
- The platform engineer's bounded project ([A-13 Mission Analysis]): standing up
  the Kubernetes deployment system must be a bounded project over the shared
  enactment core, not a re-implementation of it.
- Disparate cadences ([A-13 Mission Analysis]'s concept of operations):
  rendering runs per site and per generation, the host binary builds per
  catalogue release, and the design must keep those loops independent.

## Considered Options

- **A live operator first.** An in-cluster reconciliation loop consumes images
  directly; manifests never exist at rest.
- **Helm-chart emission.** The renderer emits a chart; sites bind values at
  install time through the chart's own machinery.
- **A whole-solution single pod.** One Deployment runs the Mode-P host with the
  full image — the single-process archetype, containerized.
- **One full image per pod, selection at runtime.** Every pod mounts the whole
  image and a selector picks its instance — the studied estate's pattern.
- **Per-record sub-images on the stock host.** Each deploy record becomes its
  own pod carrying a derived image of exactly its closure; the unmodified Mode-P
  host enacts it.

## Decision Outcome

Chosen option: **per-record sub-images on the stock host.**

- **A dry rendering package in the solution layer.** A new package
  (`solution/kubernetes`, a working name) renders one solution image plus one
  site's bindings into static Kubernetes manifests. It consumes the image alone
  — decode and dry validation, no live catalogue, no cluster client, never an
  apply: rendering is packaging desired state for transport, not enacting it.
  Plans stay where [D-13 Two-Phase Enactment] put them: each pod's host loads
  its own plan against the catalogue its binary links, so catalogue-semantic
  validation happens at the authoritative point — pod startup, exit 2 on skew, a
  detected refusal instead of a misreading — and the renderer stays buildable
  anywhere, no catalogue in sight.
- **One deploy record, one manifest file: ConfigMap + optional Secret +
  Deployment.** Deploy records set the pod count; provision records get no
  workloads of their own. Each pod re-runs the attach drivers its record's
  closure needs at its own startup — with attachment the only provisioning act,
  a driver verifies substrate it never owns ([A-11 Substrate and Slices]), and
  verification bears repeating, so re-running it per pod is the cheap, honest
  choice. [D-13 Two-Phase Enactment]'s cross-binary-phases door — provisioning
  in one binary, deploying in another, outputs crossing on a typed transport —
  explicitly stays shut. No Services are emitted (the image carries no port or
  listener concept; [A-12 Operator I/O] hands components listeners, never port
  numbers to bind), no namespace objects, no umbrella packaging.
- **Per-record sub-images.** For each deploy record the renderer derives a
  sub-image: the record itself, the provision records its parameters reach
  transitively, the symbols that closure references, the full pinned catalogue
  (the skew guardrail, not desired state — slicing it would buy bytes and cost a
  dialect), and provenance — generation and build block — carried verbatim, so
  the generation a pod reports is the generation the pipeline shipped. A
  sub-image owes what any composed image owes: canonicalized, validated dry,
  encoded canonically. It rides the pod's ConfigMap, and the stock Mode-P host
  enacts it unmodified — desired state stays the artifact of record at every
  hop. [D-08 Desired-State Image]'s one image per solution holds: the solution
  image remains the artifact; shards are derived transport, regenerated with
  every render, never edited.
- **Site binding at render; externs stay late-bound files.** Render takes the
  site as data: extern values and var rebinds. A var rebind rewrites the
  sub-image's symbol table — the one place a rebind reaches every use site
  uniformly ([A-10 Value Binding]), exercised at exactly its intended edge.
  Extern values never enter any sub-image: plain externs land in an extern file
  inside the pod's ConfigMap, tainted externs in a Secret-mounted twin, and both
  feed the host as files — the delivery [A-12 Operator I/O] maps for this
  environment (a mounted secret volume on Kubernetes; the environment owns
  delivery) — which keeps taint custody structural: where a value may rest is
  decided by which file it renders into. The render gate refuses a site that
  leaves any reached extern unbound — the same MUST-bind gate the host re-runs,
  wet, as its first act at startup.
- **The host grows a file source for externs.** The door the host package
  records — a site-file extern source beside the in-memory map — is walked
  through deliberately minimally: a file of repeatable name=value lines
  (`-externs <file>`, a working name), the extern argument list in file form and
  nothing more. It is not a site format; the site-binding document proper stays
  [D-13 Two-Phase Enactment]'s open door.
- **Advisory stanzas are honored, never required.** The renderer is the first
  controller reading the community vocabulary: `k8s.pod`'s `replicas` and
  `priorityClass` shape the workload, its `cpu` and `memory` emit as resource
  requests equal to limits — both or neither, a half-bound pair refused at
  render — and `k8s.workload`'s `partOf` becomes the standard part-of label. A
  missing stanza means defaults, never an error:
  [D-15 Discovered Stanza Schemes]'s advisory contract is kept whole, and its
  controller-side-requirements door stays shut — the renderer declares no
  required stanzas.
- **A profile seam for platform conventions.** Object naming (instance
  identifier to DNS-1123 kebab), labels, the rollout-on-config-change annotation
  (a checksum over the pod's ConfigMap on the pod template), and the pod
  contract — image reference, payload file names and mounts, container env,
  args, probes — are defaults a platform team's profile may override. The
  namespace splits the same way the seam does: a solution may never name one
  (nothing in the image speaks of namespaces, and deployment indifference keeps
  it that way), a profile may bake one at render — the namespace-per-solution
  operational shape some estates prefer, at the price that a baked set routes
  itself and refuses any other destination — and the unset default leaves every
  object namespace-free for the apply to choose, so one rendered set serves any
  number of environments: the ephemeral-environment pattern the studied estate
  runs, structure constant while the destination varies. The bar this seam must
  clear is evidence-set: the estate [A-13 Mission Analysis] studied runs every
  pod off one shared binary with the component selected at runtime, and that
  shape must be reachable as a profile of this renderer, never a fork of it —
  the validation exercise above already regenerated that estate's deployment
  shape this way.

An illustrative sketch — one deploy record, one file, the whole pod at rest:

```
manifests/ping1.yaml
  ConfigMap  ping1-cm       # image.json (the sub-image) + externs.conf
  Secret     ping1-secret   # its externs.conf twin — only when taint flows
  Deployment ping1          # the stock prebuilt host, fed files:
    platformhost -image …/config/image.json -externs …/config/externs.conf
                 -externs …/secret/externs.conf
```

### Consequences

- The Kubernetes cell of deployment-indifference coverage
  ([A-13 Mission Analysis]'s measure) gets its first fill, and it costs the
  image nothing: the artifact the single-process hosts enact renders here
  unchanged.
- The exit taxonomy meets a supervisor that reads it: exit 2 — a configuration
  fault, reproducing until inputs change — surfaces as a crash loop pointing
  back at the render, exit 1 restarts toward convergence, and a rolling
  replacement is the clean wind-down the host already performs on its first
  termination signal.
- Each pod's extern surface is exactly its record's closure: least privilege
  stops being a policy to audit and becomes a property the render computes — no
  pod mounts a secret its record never reaches.
- N sub-images exist to keep regenerated: the manifest set is derived output,
  current only as of its last render. Determinism is the counterweight — a
  regeneration test byte-compares a fresh render against the committed set,
  turning drift into a test failure instead of an estate surprise.
- Slice-bearing solutions refuse at pod startup exactly as they refuse at every
  other host — the plan admits attach only. Unchanged and honest: the renderer
  does not pre-judge what the authoritative gate will say.
- Renaming an instance is delete-plus-create in the manifest set too — the file
  and its objects vanish and reappear under the new name —
  [D-08 Desired-State Image]'s known cost, compounded here by the pruning door
  below until it opens.

## Pros and Cons of the Options

### A Live Operator First

- Good, the convergence discipline arrives whole: observe, diff, converge, prune
  run in one loop, and pruning — the part manifests at rest handle worst — lives
  where it naturally belongs.
- Bad, it is the heaviest archetype first: cluster credentials, watch machinery,
  in-cluster upgrade and failure modes — the opposite of the bounded project
  [A-13 Mission Analysis] asks for, taken on before any Kubernetes deployment of
  a solution exists to learn from.
- Bad, it skips the artifact estates operate on: production Kubernetes runs on
  manifests held in repositories, reviewed and applied by standing machinery —
  and an operator's first act is computing desired manifests anyway, so the dry
  renderer is the operator's front half, not a detour
  ([D-08 Desired-State Image]'s one pure core serving both controller shapes).

### Helm-Chart Emission

- Good, it meets platform estates where they already deploy: chart repositories,
  release tooling, rollback verbs, a values file as the familiar site knob.
- Bad, it re-opens the variance door [D-08 Desired-State Image] closed: a chart
  is a template awaiting values, and per-site values files are the overlay shape
  rejected there — while the image is already resolved desired state, so
  emitting a template un-resolves it.
- Bad, two binding moments (render and install) split authority over the same
  values, and what runs at a site stops being explainable from image plus
  recorded bindings — [A-13 Mission Analysis]'s round-trip fidelity measure
  fails by construction, not by accident.

### A Whole-Solution Single Pod

- Good, the trivial mapping: the containerized single-process archetype, one
  manifest file, the stock host and the whole image verbatim, nothing derived at
  all.
- Bad, no independent rollout: every record shares one process, so any
  instance's change, crash, or restart is every instance's — the co-location
  question [A-09 Solution Layer] keeps open hardens into forced fact.
- Bad, it decides nothing: this shape is the degenerate case a co-location
  profile could always produce (every record grouped into one pod), so fixing it
  as the design forecloses granularity while buying nothing the profile door
  would not.

### One Full Image per Pod, Selection at Runtime

- Good, the proven pattern of the multi-tenant production system
  [A-13 Mission Analysis] studied: one shared binary and one configuration set
  behind per-component pods, operated at production scale.
- Bad, it needs a selection concept the host deliberately lacks: Mode P binds
  one image per process and enacts all of it, so per-pod selection means growing
  the host contract — new surface on every archetype's host for one archetype's
  convenience.
- Bad, configuration outlives its pod's needs: every pod mounts the whole
  solution's externs, so every pod sees every secret its solution uses — custody
  weaker than what the image itself can compute, against
  [A-15 Static Deployment Rendering]'s custody force.

### Per-Record Sub-Images on the Stock Host

- Good, the host contract is the whole pod contract: the renderer emits nothing
  the stock host does not already consume — an image and extern values — so a
  pod is explainable end to end from published contracts, and renderer and host
  keep their disparate cadences.
- Good, custody equals closure: the taint split and the extern surface are
  computed from the image's own references, making the security reviewer's audit
  a read rather than an archaeology ([A-13 Mission Analysis]).
- Good, desired state is the artifact at every hop: what a pod enacts is itself
  a valid image under the same contracts as the solution image, so every gate,
  diagnostic, and audit the enactment stack has works per pod unchanged.
- Bad, N derived artifacts to keep true: every render regenerates every
  sub-image, and a stale manifest set misrepresents the solution until the next
  render — the regeneration discipline is load-bearing.
- Bad, the closure computation is new load-bearing machinery: which provisions
  and symbols a record reaches must agree with the linker's own reference
  semantics, or a sub-image validates against a subtly different truth than the
  image it shards.

## More Information

Doors this decision leaves open, each with its reopening trigger:

- **Pruning deleted records.** A render emits the current record set and deletes
  nothing: a record removed from the solution leaves its last-rendered workload
  standing until something prunes it — the convergence discipline
  [D-08 Desired-State Image] assigns to reconciliation, deliberately not
  smuggled into a dry renderer. Trigger: the first stale workload an apply
  leaves behind in a real environment.
- **Services and ports.** No Service is derivable while the image carries no
  port or listener concept — [A-12 Operator I/O] models listeners as handed-in
  handles, and the catalogue declares no listener class yet. Trigger: a port or
  listener concept landing in the image, the door [A-12 Operator I/O]'s
  semantic-classes path holds open.
- **Probes.** The default pod contract emits none; a profile may. There is
  nothing honest to probe until the host exposes a health surface —
  [A-12 Operator I/O]'s readiness-unification question, one health capability
  serving notify sockets, probe endpoints, and in-process checks. Trigger: the
  host growing that surface.
- **Secret-reference custody.** Rendered Secrets carry values. A second custody
  mode would emit references into external secret machinery instead — the
  manifest names a key, the environment materializes it — keeping secret values
  out of the manifest set entirely. Trigger: the first environment whose custody
  policy forbids rendered secret values at rest; field evidence says the trigger
  is near — estates run external-secret operators even in development.
- **Render-time provisioning.** Every pod re-verifies its provisions at startup;
  moving provisioning to render time or a dedicated actor is exactly
  [D-13 Two-Phase Enactment]'s cross-binary-phases door, and it reopens on that
  record's own trigger, verbatim: the first controller whose provisioning runs
  where its deployments do not.
- **Co-location profiles.** One record, one pod is the fixed grain; a profile
  grouping several records into one pod (the whole-solution pod as its limit)
  waits on [A-09 Solution Layer]'s open compute-isolation question — co-location
  constraints, never assignments. Trigger: that question getting an answer.
- **Workload kinds.** Every deploy record renders as a Deployment — the
  archetype's serving assumption. A run-to-completion record completes into a
  restart loop: the kubelet backs off restarts on clean exits too, so finite
  work crash-loops on its own success. Nothing in the image marks a record
  finite ([D-13 Two-Phase Enactment]'s host deliberately imposes no
  long-runningness), so a Job is underivable without guessing. Trigger: the
  first real solution carrying run-to-completion work.

[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
[A-11 Substrate and Slices]: ../analyses/A-11-substrate-slices.md
[A-12 Operator I/O]: ../analyses/A-12-operator-io.md
[A-13 Mission Analysis]: ../analyses/A-13-mission-analysis.md
[A-14 Staged Catalogue Compilation]: ../analyses/A-14-staged-catalogue-compilation.md
[A-15 Static Deployment Rendering]: ../analyses/A-15-static-deployment-rendering.md
[D-08 Desired-State Image]: D-08-desired-state-image.md
[D-13 Two-Phase Enactment]: D-13-two-phase-enactment.md
[D-15 Discovered Stanza Schemes]: D-15-discovered-stanza-schemes.md
