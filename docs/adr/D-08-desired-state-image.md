---
status: proposed
since: 2026-07-06
refines:
  - A-09-solution-layer
---

# Ship Solutions as a Reconciled Desired-State Image

## Context and Problem Statement

[A-09 Solution Layer] surveys what an update to a deployed solution ships:
append/patch instructions, per-site source overlays, or a complete desired-state
image that controllers reconcile. The layer's constraint is idempotence —
reifying the same artifact twice must be a no-op — and its production shape is a
reconciliation loop (a Kubernetes operator built from common tooling), with
static emitters (a compiler from the image to systemd units) as peers. An early
sketch of this layer tried the patch path far enough to learn why it fails; this
record fixes the alternative.

## Decision Drivers

- Idempotent reification: controllers converge on state, never replay
  instructions.
- One artifact of record per solution; what runs must be explainable from it.
- Both controller shapes — live loops and static emitters — must consume the
  same artifact through the same pure core (desired state in, actions or
  artifacts out).
- Multi-file authoring is a real need even with no overlay semantics.
- Pruning (deleting what left the definition) is required, and pruning accidents
  are unrecoverable, so guardrails are part of the decision.

## Considered Options

- **Append/patch instructions.** Updates ship fragments that modify the
  previously shipped solution, append-only.
- **Per-site source overlays.** A base definition plus site-specific patch
  layers, merged at reification.
- **A complete desired-state image, reconciled.** Every update ships the whole
  solution under the same identity; controllers diff and converge.

## Decision Outcome

Chosen option: **the complete desired-state image**. Its shape:

- **One image per solution.** The compiler links all source files of the
  solution — a directory of peers sharing the `solution` clause — into a single
  image; duplicate instance names are link errors, the way duplicate symbols
  fail a C link. There is no patch or override between files.
- **Deployed at most once.** The image's identity names one living solution
  ([A-09 Solution Layer]'s instance shape); updates ship a new image under the
  same identity with a monotonically increasing generation, and a controller
  refuses a generation older than what it last reconciled.
- **Reconciliation.** Controllers observe, diff, converge, and prune; instance
  names are the reconciliation keys, so renaming an instance is
  delete-plus-create until a move mechanism exists.
- **Guardrails.** Reification starts with a plan (the diff, before any action);
  owned provisionings carry delete protection ([A-11 Substrate and Slices] gives
  slices their lifecycle); the image pins the catalogue schemas it was compiled
  against, so a controller detects skew instead of misreading records.
- **Variance without overlays.** Value variance between sites is symbol
  rebinding at the site ([A-10 Value Binding]); structural variance is file-set
  selection at link time, in the spirit of Go build tags — still union linking,
  never merging.

## Pros and Cons of the Options

### Append/Patch Instructions

- Good, small updates ship small artifacts.
- Bad, an append is an instruction against presumed current state; the artifact
  becomes a log to replay, replay order becomes load-bearing, and idempotence
  dies. The early sketch was abandoned on exactly this.

### Per-Site Source Overlays

- Good, serves real per-site variance; strategic-merge tooling proves the
  demand.
- Bad, reintroduces the template shape [A-09 Solution Layer] declines: the
  artifact stops being one thing, and what runs at a site is explainable only
  from base-plus-patches in the right order.

### A Complete Desired-State Image

- Good, idempotent by construction; the image is the single artifact of record;
  diffing and pruning are well-defined.
- Good, one pure core serves both controller shapes; a static emitter is the
  same function run once.
- Bad, pruning by diff makes a truncated image dangerous — hence the
  directory-as-unit rule (no ad-hoc file lists), the plan step, generations, and
  delete protection, which are complexity this option must carry.

### Consequences

- The controller-facing SDK is built around the pure core: parse image, resolve
  symbols, diff against observed state, hand actions to drivers.
- Renaming stateful instances destroys and recreates them; a move-declaration
  analogue is named as future work in [A-11 Substrate and Slices]'s open
  questions.
- Fleet concerns (which image generation reaches which site cohort) stay above
  the layer, per [A-09 Solution Layer].

[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
[A-11 Substrate and Slices]: ../analyses/A-11-substrate-slices.md
