---
status: proposed
since: 2026-07-13
refines:
  - A-13-mission-analysis
  - A-14-staged-catalogue-compilation
  - D-08-desired-state-image
refined-by:
  - D-16-static-manifest-rendering
---

# Enact the Image in Two Phases on Any Host

## Context and Problem Statement

The dry half of the toolchain is decided. [D-10 Generate-Compile-Run] fixes how
definitions compile against live catalogue values, and
[D-08 Desired-State Image] fixes the artifact the compiler emits. The wet half
is not: no record says how an image becomes a running solution.
[A-09 Solution Layer]'s C analogy names the missing role — the image carries
"values the environment must bind", and the deployment environment plays the
dynamic loader — and [A-14 Staged Catalogue Compilation] walks the timelines up
to that boundary and stops: at reconcile-run "the same functions run again,
wet", run by controllers and hosts it does not design.

Three operational facts from [A-13 Mission Analysis] make the gap concrete. The
single-process run is a daily operation, not a compliance test, and the dev-loop
measure prices it: one command, edited sources to a running solution, in
seconds. Production runs on the opposite cadence: teams publish catalogues the
way they publish internal libraries, consumed downstream on the consumer's
schedule, so the machinery that runs solutions at a site cannot be rebuilt
whenever a solution changes. And deployment-indifference coverage forbids the
cheap resolution: machinery only one archetype could host would push its
assumptions into the image and narrow where definitions can go.

This record fixes the wet half: what an image may assume about its host
(nothing), the phases every enactment runs, and the two host shapes the
disparate cadences demand.

## Decision Drivers

- Deployment indifference ([A-09 Solution Layer]'s litmus, measured by
  [A-13 Mission Analysis]): nothing about enactment may leak into the image and
  narrow which archetypes can reify it.
- Disparate cadences: application teams release catalogues and solution teams
  release images on independent schedules ([A-13 Mission Analysis]'s concept of
  operations); the dev loop rebuilds everything per invocation, production
  rebuilds nothing per solution.
- One pure core: both controller shapes — live loops and static emitters —
  consume the artifact through the same desired-state-in, actions-out core
  ([D-08 Desired-State Image]); planning must be separable from acting.
- Order is semantics: reconcile-time outputs feed application parameters
  ([A-10 Value Binding]'s DAG), so backing services must be settled before the
  applications that consume their coordinates start.
- Nothing resolves unrecorded: the controller records the effective value of
  every symbol it binds, and sensitivity taints what must never appear in that
  record ([A-10 Value Binding]).
- The host is itself a process under some supervisor: its exit code must
  separate configuration faults from runtime failures, because restart policies
  read it ([A-12 Operator I/O]).

## Considered Options

- **Per-archetype machinery, whole cloth.** Each deployment system reads the
  image its own way; no shared plan, no named phases, no host library — the four
  archetypes own their enactment end to end.
- **One convergence pass, no phases.** Shared machinery orders every record,
  provisioning and application alike, by the reference DAG, and starts each as
  soon as its inputs exist.
- **A prebuilt host as the only shape.** One host binary form, built against a
  catalogue ahead of time; every context, dev loop included, hands it an encoded
  image.
- **A tailored host per enactment.** Every enactment generates and builds a host
  embedding the solution, the way the compiler's back half is generated
  ([D-10 Generate-Compile-Run]).
- **A pure plan, two phases, two host modes.** Planning is a pure function of
  image plus live catalogue; hosts execute the plan as a PROVISION phase then a
  DEPLOY phase; production hosts are prebuilt, dev-loop hosts are generated.

## Decision Outcome

Chosen option: **a pure plan, two phases, two host modes.**

- **The IR stays host-agnostic.** No record or symbol names a host, a mode, or
  an archetype. The image carries what any host must bind — literals, `var`
  defaults, `extern` symbols awaiting site values, output references awaiting
  provisioning ([A-10 Value Binding]'s three classes, unchanged) — and enactment
  machinery is one consumer of the image, never a dialect of it.
- **Enactment is two named phases: PROVISION, then DEPLOY.** Provisioning
  records run first, ordered by the output-reference DAG the linker already
  guarantees acyclic; each driver attaches to guaranteed substrate
  ([A-11 Substrate and Slices]) and reports the outputs its type declares. Only
  then do application records run: a fresh service per record
  ([D-11 Self-Contained Services]'s factory), its flags wet-bound to the very
  values the compiler validated dry, all instances hosted concurrently.
  "Reconcile" deliberately names no phase: it is the convergence discipline of
  [D-08 Desired-State Image] — observe, diff, converge, prune — that a live
  controller wraps around enactment; the phases are what one pass through the
  plan runs.
- **Planning is pure; acting is the host's.** The plan — externs to gate,
  provisions in dependency order, deploys after — is computed from the image and
  the live catalogue alone, no I/O, and validated eagerly: an element missing
  from the host's catalogue, a kind mismatch, a binding to a flag the element no
  longer declares, a driverless provision type, a slice record (lifecycle is
  unbuilt) are all refused before anything runs. The plan is data; invoking
  drivers and constructing services stays on the host's side of the seam, and
  nothing in the plan assumes the two phases share a binary — a controller may
  provision in one place and deploy in another.
- **Mode P, the production shape: one prebuilt host serves many solutions.** A
  platform team builds a host binary against its catalogue once, on the
  catalogue's release cadence; the binary enacts any image compiled against that
  catalogue, decoded from a file, on the solution's release cadence. The image's
  catalogue pin turns version skew into a detected refusal instead of a
  misreading ([D-08 Desired-State Image]'s guardrail).
- **Mode T, the tailored shape: a generated host serves one invocation.** The
  dev-loop verb (`sdl run`, a working name) generates a host main embedding the
  solution's sources, builds it in the ambient module context, and runs it — the
  same generate-compile-run mechanics that emit the image
  ([D-10 Generate-Compile-Run]), extended one step from emitting to running.
  Edited sources become a running solution in one command, and
  per-production-solution code generation never happens: generation buys the dev
  loop its exact-catalogue fidelity, nothing else.
- **Externs bind in-memory at the host boundary.** The invocation hands the host
  its extern values (repeatable `name=value` arguments today), and the host
  injects them straight into flag surfaces — the "direct in-memory injection"
  delivery [A-12 Operator I/O] maps for the single-process host, so no secret
  transits the environment or a temporary file. The gate is total and runs
  before any driver: every unbound `extern` is listed, then enactment refuses.
- **Every resolution is audited; taint redacts.** The host writes one line per
  binding it resolves and per output it stores — the guardrail
  [A-10 Value Binding] calls not optional — and a value tainted sensitive prints
  redacted, in the audit as in diagnostics.
- **The exit contract separates fault classes.** Zero is a clean exit, graceful
  shutdown included: the first termination signal starts an orderly stop under a
  grace budget, and completing it is success. One is a wet failure: a driver
  error, an application error, an overrun grace budget. Two is a configuration
  fault: an unreadable image, an unplannable solution, unbound externs, bad
  arguments. Restart policies can thereby retry runtime failures and hold
  configuration faults for a human — the distinction [A-12 Operator I/O] wants
  an exit taxonomy to carry.

An illustrative sketch — the same solution wet twice, tailored then prebuilt:

```
# Mode T: the dev loop. Edited sources to a running solution, one command.
$ sdl run -extern natsAdmin=file:/tmp/admin.creds ./sample

# Mode P: production. A host built once against the platform's catalogue
# enacts any image compiled against it.
$ platformhost -image sample.img -extern natsAdmin=file:/etc/creds/admin
```

Both print the same audit transcript, redaction included:

```
provision natsAccount: bound adminAccount = <redacted>
provision natsAccount: output config = <redacted>
deploy Ping1: bound target = "com.acme.Echo"
deploy Ping1: bound nats = <redacted>
```

## Pros and Cons of the Options

### Per-Archetype Machinery, Whole Cloth

- Good, each archetype stays maximally idiomatic, and no shared-core API has to
  be designed or versioned.
- Bad, enactment semantics — the extern gate, the ordering, the audit record —
  fork per archetype, so the same image means subtly different things at
  different sites, against the solution engineer's "a definition must mean the
  same thing at every site" ([A-13 Mission Analysis]).
- Bad, the archetype count works against it: the playground host and the e2e
  harness are two more full reimplementations, where [A-13 Mission Analysis]'s
  platform engineer needs standing up a deployment system to be a bounded
  project over a small shared core.

### One Convergence Pass, No Phases

- Good, maximal concurrency: an application whose parameters are all literal
  starts before unrelated provisioning settles.
- Good, one scheduling concept — the DAG — instead of order within phases plus
  order between them.
- Bad, parameters bind once, at process start ([A-12 Operator I/O]'s process
  contract), so every application that references an output waits for that
  provisioning anyway; the interleaving accelerates only literal-only instances,
  while costing the operator the stable narrative that provisioning settled
  before applications started.
- Bad, backing services and application processes have different owners, failure
  modes, and blast radii ([A-11 Substrate and Slices]); one undivided schedule
  has no boundary at which a controller can cut, hand off, or halt.

### A Prebuilt Host as the Only Shape

- Good, one shape to build, test, and operate; dev and production cannot
  diverge, being the same binary form.
- Bad, the dev loop breaks on catalogue fidelity: the loop compiles against the
  sources' own catalogue — local, possibly unreleased, possibly replaced
  ([D-10 Generate-Compile-Run]'s ambient-module argument) — and a binary built
  earlier holds yesterday's catalogue by construction.
- Bad, the inner loop grows artifact ceremony (compile, encode, hand the file
  over) where the measure is one command in seconds.

### A Tailored Host per Enactment

- Good, mechanics stay uniform with the compiler's back half, and the catalogue
  is exactly right at every invocation.
- Bad, a Go toolchain becomes a production dependency of every site and every
  restart — the cost [D-10 Generate-Compile-Run] accepted for the authoring
  loop, unacceptable on a host answering a page.
- Bad, cadences remerge: enacting a solution means building a binary, so
  solution releases queue behind a build farm the platform team never wanted to
  operate.

### A Pure Plan, Two Phases, Two Host Modes

- Good, each context gets the binding moment its cadence needs — production
  binds a prebuilt host to an image file at process start, the dev loop binds a
  generated host to its sources at build — and both execute the same wet core,
  so behavior cannot fork between them.
- Good, the plan is [D-08 Desired-State Image]'s plan-before-apply guardrail
  materialized: everything checkable without side effects is checked before the
  first driver runs.
- Good, the phase boundary is a stated contract: controllers may run both phases
  in one process today and in two binaries tomorrow without the record model
  moving.
- Bad, two modes must be documented, tested, and kept honest; features will land
  tailored-first, and the shared wet core is the only fence against divergence.
- Bad, a Mode-P host's catalogue is frozen at its build: a solution reaching
  outside it is refused at plan time — a detected skew, never a misreading, but
  serving that solution still means building a new host.

### Consequences

- The single-process archetypes stop being aspirational: the playground host and
  the e2e harness are the first consumers of the plan-and-host machinery, and
  the operator and service-manager archetypes consume the same plan when they
  land.
- The phase vocabulary is fixed: the build acts (generate, build, link, emit)
  end at the image, the enactment acts (PROVISION, DEPLOY) take it from there;
  "compile" keeps naming the whole dry half, "reconcile" keeps naming the
  convergence discipline.
- A definition's MUST-bind surface ([A-10 Value Binding]'s `extern` class) is
  enforced at every enactment, not only documented: hosts refuse to start
  partially bound.
- A single-process host consumes only the parameter compartment; deployment
  intent and advisory stanzas ([D-12 Statement-Body Anatomy]) address
  controllers with placement to decide, and ride ignored — the advisory
  semantics doing their job.
- How drivers come to run inside the PROVISION phase — where provisioning
  execution lives, and what its outputs are worth to the parameters they feed —
  is [D-14 Driver-on-Type]'s subject.

## More Information

Doors this decision leaves open, each with its reopening trigger:

- **A site file.** Externs arrive as invocation arguments; the recorded
  site-binding document [A-13 Mission Analysis]'s deployment scenario promises
  ("divergence between sites becomes a recorded set of bindings") is the next
  rung, continuous with [A-10 Value Binding]'s override-audit question and
  [A-14 Staged Catalogue Compilation]'s open home for site configuration.
  Trigger: the first site whose extern surface outgrows a command line, or the
  first audit that needs bindings as an artifact rather than a transcript.
- **Cross-binary phases.** The plan already refuses to assume one binary, but
  both phases run in one process today, and provisioning outputs cross to the
  DEPLOY phase in memory. Provisioning in one binary and deploying in another
  needs a typed transport for outputs between them, carrying the output contract
  [D-14 Driver-on-Type] fixes across processes. Trigger: the first controller
  whose provisioning runs where its deployments do not.
- **Multi-solution serving.** A Mode-P host binds one image per process; the
  prebuilt binary serves many solutions, one enactment at a time. One host
  process serving several solutions, or a resident host accepting new
  generations without a restart, is unexplored. Trigger: the first estate that
  co-hosts enough single-process solutions to feel the per-process cost, or the
  first controller that wants in-place generation handoff.
- **Slice lifecycle.** Enactment refuses slice records: with attachment the only
  provisioning act, nothing is created, mutated, destroyed, or pruned, and
  [D-08 Desired-State Image]'s delete protection has nothing to protect. The
  refusal is a teaching error at plan time, not a compile error — slice
  definitions stay compilable, so catalogues and definitions can lead their
  drivers. Trigger: the first real slice driver.

[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
[A-11 Substrate and Slices]: ../analyses/A-11-substrate-slices.md
[A-12 Operator I/O]: ../analyses/A-12-operator-io.md
[A-13 Mission Analysis]: ../analyses/A-13-mission-analysis.md
[A-14 Staged Catalogue Compilation]: ../analyses/A-14-staged-catalogue-compilation.md
[D-08 Desired-State Image]: D-08-desired-state-image.md
[D-10 Generate-Compile-Run]: D-10-generate-compile-run.md
[D-11 Self-Contained Services]: D-11-self-contained-services.md
[D-12 Statement-Body Anatomy]: D-12-statement-body-anatomy.md
[D-14 Driver-on-Type]: D-14-driver-on-type.md
