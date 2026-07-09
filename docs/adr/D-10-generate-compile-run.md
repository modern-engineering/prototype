---
status: proposed
since: 2026-07-09
refines:
  - A-14-staged-catalogue-compilation
---

# Compile Solutions by Generating a Program Against Imported Catalogues

## Context and Problem Statement

[A-09 Solution Layer] gave the compiler its charter — link definitions against
the catalogue, emit the image — and [D-06 Declarative Descriptor Values] made
the catalogue a set of ordinary Go packages exporting element values, registered
nowhere. Go has no package-level reflection: nothing can enumerate those values
without compiling against the packages that define them.
[A-14 Staged Catalogue Compilation] surveys the pathways by which a compiler can
come to know user-defined elements and the import semantics that name them. This
record fixes the mechanism and the import surface.

## Decision Drivers

- Safety comes from compiling: parameters must be checked against the schemes
  the elements themselves declare, never against a transcription
  ([D-07 Verb-First Directives]; [A-10 Value Binding]'s link-time checks).
- [D-06 Declarative Descriptor Values]'s identity model — values referenced
  directly, no `init()` registration, no blank imports — must extend to the
  solution layer unchanged.
- The authoring loop is local, in seconds, with errors positioned in the
  author's sources ([A-13 Mission Analysis]).
- Developers publish a catalogue once; no second description of what the code
  already declares may land on them.
- Definition units stay self-contained; the grammar stays closed under catalogue
  growth; every front-end remains a peer of the record shape.

## Considered Options

- **Init-time self-registration into global catalogues.** Blank-import packages
  that register their elements from `init()`.
- **External schema manifests.** A YAML/JSON description beside the code; the
  compiler reads manifests, never Go.
- **Dynamic loading of catalogue plugins.** Shared objects via Go's `plugin`
  package, loaded by a resident compiler.
- **Interpreting catalogue sources.** An embedded Go interpreter evaluates the
  packages inside the compiler.
- **Generate, compile, run.** Synthesize a program that imports the catalogue;
  compile it in the user's module context; run it to emit the image.

## Decision Outcome

Chosen option: **generate, compile, run**, with the import surface fixed
alongside it:

- **Per-file import blocks.** Every definition file opens with a Go-style import
  block (aliases as in Go) naming the catalogue packages it uses, and every
  element reference is package-qualified (`ff.Ping`), with no bare form even
  while unambiguous. Solution-local names — `var` and `extern` symbols, instance
  names — stay unqualified.
- **Ambient module resolution.** Import paths resolve exactly as the `go` CLI
  would in the solution's directory: an active module resolves through its own
  requirements and replaces, so local catalogue source is honoured; a workspace
  resolves through its union of modules; with no module context, the generated
  program's own module resolves each import at its latest version. No
  solution-own module file exists yet.
- **The build generates its own back half.** `sdl build` parses the units,
  synthesizes a main that imports the named packages and embeds the unit
  sources, compiles it with the ambient Go toolchain (the `go test`
  `_testmain.go` precedent), and runs it.
- **The image is emitted where values are live.** The generated binary resolves
  element references to the exported values, obtains parameter schemes by dry
  instantiation ([A-07 Documentation]'s invariant, guarded by `recover` into
  positioned diagnostics), validates literals through the same `flag.Value` code
  that parses real values at run time, builds the reference DAG, and emits the
  desired-state image of [D-08 Desired-State Image] itself.

### Consequences

- Good, schema drift is structurally impossible: the checking scheme and the
  running scheme are one piece of code, reached by the same dry call.
- Good, module semantics come for free: versioning, replaces, workspaces,
  proxies, and vendoring behave as every Go developer already expects.
- Good, units are self-contained, the generated program takes its import list
  verbatim, and two catalogues may export the same element name.
- Bad, a Go toolchain becomes a runtime dependency of solution builds; only
  formatting stays toolchain-free, and checking cannot be done by a standalone
  parser.
- Bad, generated code couples the tool's version to the library version in the
  user's module; skew must be detected, never miscompiled.
- Bad, a `go build` sits inside the authoring loop; the dev-loop latency measure
  of [A-13 Mission Analysis] is partly spent on build-cache warmth.

## Pros and Cons of the Options

### Init-Time Self-Registration into Global Catalogues

- Good, the familiar registry shape; enumeration comes for free.
- Bad, discovery is moved, not solved: something must still name the packages to
  blank-import, and a per-solution build is implied anyway.
- Bad, reintroduces everything [D-06 Declarative Descriptor Values] rejected:
  global mutable state, `init()` ordering, collision panics, and relationships
  the compiler cannot check.

### External Schema Manifests

- Good, no Go toolchain in the build; the compiler stays one static binary.
- Bad, the manifest restates what the type system knows and drifts from it;
  checking a transcription is the parallel-schema infrastructure
  [D-07 Verb-First Directives] already declined.
- Bad, generating manifests to fight the drift needs the very discovery
  mechanism they were meant to replace, plus distribution and versioning of
  their own.

### Dynamic Loading of Catalogue Plugins

- Good, discovered symbols are live Go values without a per-invocation build.
- Bad, platform support is partial, and plugin and host are locked to identical
  toolchain and dependency versions — the exact skew surface a many-team
  catalogue economy maximizes.

### Interpreting Catalogue Sources

- Good, near-live values with no toolchain dependency and no plugin fences.
- Bad, an interpreter is a reimplementation of Go: fidelity gaps around
  reflection, generics, and cgo are structural, and the values that validate are
  near — not identical to — the values that run. Behavioural drift by
  simulation, bought at heavy dependency weight.

### Generate, Compile, Run

- Good, the user's elements are live inside the compiler's back half — the only
  pathway where the checking code and the running code are the same code.
- Good, proven at ecosystem scale by `go test`; module resolution and positioned
  Go diagnostics are inherited rather than rebuilt.
- Bad, the toolchain dependency, the version coupling, and the in-loop build
  latency named above; each carries a door below.

## More Information

Doors this decision leaves open, each with its revisit trigger:

- **Version-skew strictness.** The build warns when the tool's library version
  differs from the user module's; the trigger to harden into refusal is the
  first miscompile traced to skew.
- **A solution manifest.** `sdl.mod`-style pinning (catalogue versions, file-set
  selection, identity ownership) if the ambient module proves too little; the
  trigger is the first solution that must build reproducibly outside its
  authoring module.
- **Composite literals.** Whole-value parameters only for now; the trigger is
  the first catalogue element whose parameter surface genuinely nests beyond
  dotted flag names.
- **Module-context completion.** Active-module resolution lands first; workspace
  union and module-less latest-resolution are follow-on rungs, each triggered by
  the first authoring setup that needs it.
- **Plumbing growth.** Re-rendering the image as one canonical definition file
  is the first plumbing verb; in-place amendment and queries (the `go mod edit`
  shape) follow as consumers appear. Whether site bindings enter the image
  through plumbing or live beside it stays open, together with
  [A-10 Value Binding]'s override-audit question.

[A-07 Documentation]: ../analyses/A-07-documentation.md
[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
[A-13 Mission Analysis]: ../analyses/A-13-mission-analysis.md
[A-14 Staged Catalogue Compilation]: ../analyses/A-14-staged-catalogue-compilation.md
[D-06 Declarative Descriptor Values]: D-06-declarative-descriptor-values.md
[D-07 Verb-First Directives]: D-07-verb-directive-syntax.md
[D-08 Desired-State Image]: D-08-desired-state-image.md
