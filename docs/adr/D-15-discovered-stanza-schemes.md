---
status: proposed
since: 2026-07-13
refines:
  - D-09-catalogue-citizenship
  - D-12-statement-body-anatomy
---

# Discover Stanza Schemes as Self-Naming Citizens

## Context and Problem Statement

[D-12 Statement-Body Anatomy] gave controller and community schemes their
compartment — named advisory `with` stanzas, recognized or ignored — and named
the cost advisory semantics carry: a misspelled key inside a stanza is an error
nowhere, invisible to the authoring loop [A-13 Mission Analysis] prices in
positioned errors and seconds. The remedy was designed in outline behind a door
— compilation discovering a value scheme for a qualifier it can resolve, checked
when the scheme's package is importable, tolerated opaque otherwise — and one
classification was deferred with it: whether such schemes are catalogue citizens
([D-09 Catalogue Citizenship]'s boundary) or a parallel discovered namespace.

The compartment has since shipped end to end: statements carry stanzas, images
carry extensions, and [D-13 Two-Phase Enactment]'s single-process hosts
dutifully ignore them — the advisory contract doing its job. That leaves stanzas
the one compartment nothing can check: parameters validate against the
catalogue, root fields against the verb's closed table, while the schemes
stanzas are written in remain lore, unpublishable as anything a compiler could
look up. This record opens D-12's door: it fixes how schemes are declared and
discovered, what a qualifier is a name of, and where checking begins and
advisory tolerance ends.

## Decision Drivers

- The named cost falls due: stanza typos must surface at authoring time,
  positioned in the sources ([A-13 Mission Analysis]'s authoring loop), not at
  whichever site first reads the compartment in earnest.
- The advisory contract is load-bearing ([D-12 Statement-Body Anatomy], carrying
  [A-09 Solution Layer]'s litmus): checking must never make a stanza mandatory —
  a definition tuned for one archetype must keep reifying on all four, so a
  qualifier nothing recognizes must keep riding opaquely.
- No registries: [D-06 Declarative Descriptor Values] retired `init()`-time
  tables, and [D-10 Generate-Compile-Run], [D-11 Self-Contained Services], and
  [D-14 Driver-on-Type] each upheld the stance; schemes must not reopen it.
- Vocabulary belongs to the convention's ecosystem: stanzas speak the language
  of the controllers reading them (`k8s.pod`, the way that ecosystem spells its
  kinds), so a scheme's name must be the scheme author's to declare — never a Go
  export surface leaking its capitalization into stanza text.
- What checks must be what runs ([A-14 Staged Catalogue Compilation]): stanza
  values must validate through the scheme's own declared surface, never a
  transcription of it.

## Considered Options

- **Package-referenced schemes.** Stanzas name schemes the way statements name
  elements: `with k8s.Pod`, resolved through the unit's imports.
- **An init-registered qualifier table.** Scheme packages claim their qualifiers
  in a process-global table at `init()`; blank imports assemble the recognized
  set.
- **Forever-opaque stanzas.** Keep [D-12 Statement-Body Anatomy]'s first cut
  verbatim: every stanza rides opaquely, and whatever checking exists lives in
  each controller, at enactment.
- **Discovered self-naming scheme citizens.** A new catalogue element kind
  carrying a self-declared qualifier and a dry key surface, discovered from the
  solution's imported packages like every other citizen.

## Decision Outcome

Chosen option: **discovered self-naming scheme citizens.**

An illustrative sketch — a conventions package declares, generated code
registers:

```go
// In a community conventions package:
var Pod = &solution.SchemeType{
    Doc:       "pod-level tuning for kubernetes controllers",
    Qualifier: "k8s.pod", // the name stanzas attach by
    Params: func(fs *flag.FlagSet) { // dry key surface; nil declares no keys
        fs.String("priorityClass", "", "scheduling class for the pods")
        fs.Int("replicas", 1, "desired replica count")
    },
}

// In the compiler's generated back half, beside the other citizens:
solution.Scheme("Pod", k8s.Pod)
```

- **Schemes are catalogue citizens — the deferred classification, settled.** A
  scheme is declaration alone: documentation, a self-declared qualifier, a dry
  key surface, nothing runnable. It enters the catalogue the way every element
  kind does — an exported package-level value, discovered by the generated
  program, validated at registration, pinned into the image — and earns the same
  instruments: a citizenship harness beside the existing ones asserts what
  tooling relies on (a well-formed qualifier; the same keys on every call, over
  fresh and distinct surfaces). [D-09 Catalogue Citizenship]'s boundary holds:
  that record fences out deployables that never consumed the library, and a
  scheme deploys nothing — admitting it widens the element vocabulary, the move
  [A-14 Staged Catalogue Compilation] already made when symbol and provision
  types joined, not the deployment scope.
- **The qualifier is self-declared, and it is the lookup key.** The exported
  identifier a scheme registers under (`Pod`) serves the pinned catalogue and
  diagnostics; stanza text never spells a package-qualified reference.
  `with k8s.pod` means whichever scheme among the solution's imports declares
  `Qualifier: "k8s.pod"`. This is [D-06 Declarative Descriptor Values]'s own
  two-name precedent — a descriptor's `Name` is the label runtime selection
  resolves at one explicit edge while the value stays the identity — replayed at
  the stanza edge: a namespace of declared names, parallel to element
  references, resolved in exactly one place.
- **Registration validates the namespace.** A qualifier is identifier segments
  joined by dots; the discovered qualifiers of one solution are unique, and a
  second package claiming a provided qualifier is refused at its registration
  (`scheme qualifier "k8s.pod" already provided by …`), like every other
  registration fault.
- **Scope is the solution, not the unit.** The lookup set is the union of every
  unit's imports. Element references stay per-unit
  ([D-10 Generate-Compile-Run]); scheme lookup deliberately is not, because
  checking is advisory: under per-unit scope the same stanza text would be
  checked in one file and opaque in its neighbour — a coin-flip diagnostic no
  author could predict.
- **Checking runs pre-fold, per source stanza, positioned.** When a qualifier
  resolves, an unknown key is an error naming both sides:
  `unknown key replica in with k8s.pod (scheme k8s.Pod)`. A literal value must
  satisfy the scheme's own flag surface, exercised fresh per stanza — the dry
  discipline every compartment validates with
  ([A-14 Staged Catalogue Compilation]), and the very validator a recognizing
  controller re-runs wet. A bare token passes unvalidated, staying the opaque
  profile word [D-12 Statement-Body Anatomy] made it. A qualifier no import
  provides keeps today's behaviour verbatim: the stanza rides the image
  opaquely, unchecked. Folding is untouched — a default's stanza is checked
  once, where it is written, and extensions merge per qualifier after checking.
- **The image pins the scheme, not the stanza.** Scheme elements pin into the
  image's catalogue block with their qualifier and keys, like any element's
  parameters; the extension compartment itself keeps its shape, so image
  consumers read stanzas without a live catalogue as before, and the pinned
  scheme records which vocabulary the build checked against.

### Consequences

- Checking is opt-in by import and exactly as good as the import set: a solution
  that imports a conventions package gets typed stanza values today and
  unknown-key errors tomorrow; one that imports nothing compiles unchanged.
- A misspelled qualifier is still no error — by construction, a typo and a
  foreign scheme are indistinguishable, and refusing unrecognized qualifiers is
  exactly the advisory breach this design refuses. The residual is priced: the
  cost [D-12 Statement-Body Anatomy] called "silently ignorable" closes for keys
  and values, not for the qualifier itself.
- Deployment conventions gain a publishable unit: a scheme package is versioned,
  documented building material that travels the supply chain
  [A-13 Mission Analysis] describes, beside the catalogue releases it rides
  with.

## Pros and Cons of the Options

### Package-Referenced Schemes

- Good, one reference model: stanzas would resolve the way element references
  do, through the unit's imports, and no second namespace would exist.
- Bad, it breaks the advisory contract at the text level: a stanza is legitimate
  with its scheme package nowhere in reach — tuning for a controller the
  authoring module does not depend on — and a package-qualified reference either
  fails to resolve, where opaque ride-along was promised, or demands the import
  in every unit that mentions the stanza.
- Bad, case mismatch: exported Go identifiers are capitalized (`k8s.Pod`),
  stanza vocabularies are the reading ecosystem's (`k8s.pod`); one meaning would
  need two spellings, or Go's export rules would dictate a community's
  vocabulary.
- Bad, aliasing splits the namespace: per-unit import aliases would let one
  scheme spell differently across units, and extensions folding per qualifier
  ([D-12 Statement-Body Anatomy]) would merge on the alias, not the scheme.

### An Init-Registered Qualifier Table

- Good, the familiar registry shape; enumerating the recognized set is free.
- Bad, it is the table [D-06 Declarative Descriptor Values] retired, costs
  intact: global mutable state, `init()` ordering, collisions as import-time
  panics instead of positioned registration errors. [D-10 Generate-Compile-Run]
  already declined registry-shaped discovery for the catalogue at large; schemes
  are no exception.

### Forever-Opaque Stanzas

- Good, nothing new to design, version, or teach; the compartment stays
  maximally permissive.
- Bad, the misspelling cost never falls due anywhere: a wrong key ships in every
  image and is ignored by every controller — which is the failure mode itself,
  not a diagnosis of it.
- Bad, conventions stay lore: without a publishable scheme there is no unit for
  community and platform vocabularies to travel as, so every controller
  reinvents and privately documents its keys.

### Discovered Self-Naming Scheme Citizens

- Good, checked exactly where checking is honest: importing a scheme package is
  opting into its vocabulary; everyone else keeps the opaque contract untouched.
- Good, the key surface is executable declaration — the same dry flag discipline
  as every citizen — so the vocabulary the linker checks is the one the scheme's
  package declares, drift-proof by construction.
- Good, self-naming keeps the namespace with the convention's owners: `k8s.pod`
  is spellable because the qualifier is data, and a collision is a positioned,
  solution-scoped error rather than a global panic.
- Bad, a second resolution rule exists: qualifiers resolve by declared name
  where elements resolve by package-qualified reference — two rules to teach,
  though [D-06 Declarative Descriptor Values] already runs the same pair (values
  as identity, declared names at one edge).
- Bad, silence is ambiguous: the same stanza text is checked in one solution and
  opaque in another, so an author can misread an un-imported scheme's silence as
  validation.

## More Information

Doors this decision leaves open, each with its reopening trigger:

- **Controller-side requirements.** Schemes are advisory end to end; nothing
  lets a controller declare the stanzas it requires so that their absence
  surfaces at plan time rather than deep in enactment. Trigger: the first
  controller that turns a missing stanza into a runtime surprise a plan-time
  declaration would have named.
- **Cross-catalogue qualifier policy.** Uniqueness is enforced per solution; no
  authority arbitrates `k8s.*` across independently published packages — the
  colliding solution gets its positioned error and chooses its imports. Trigger:
  the first collision between independently published scheme packages.
- **Scheme versioning.** A solution checks against the scheme version it
  imports; a controller reads by the version it was built with; stanzas ride
  between them unversioned. Trigger: the first scheme whose keys change meaning
  between releases while both sides are live.
- **Composite stanza values.** Scheme keys are scalar — the composite-value gap
  [D-12 Statement-Body Anatomy] and [D-14 Driver-on-Type] already share,
  reopened here by the first scheme that will not flatten.

[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-13 Mission Analysis]: ../analyses/A-13-mission-analysis.md
[A-14 Staged Catalogue Compilation]: ../analyses/A-14-staged-catalogue-compilation.md
[D-06 Declarative Descriptor Values]: D-06-declarative-descriptor-values.md
[D-09 Catalogue Citizenship]: D-09-catalogue-citizenship.md
[D-10 Generate-Compile-Run]: D-10-generate-compile-run.md
[D-11 Self-Contained Services]: D-11-self-contained-services.md
[D-12 Statement-Body Anatomy]: D-12-statement-body-anatomy.md
[D-13 Two-Phase Enactment]: D-13-two-phase-enactment.md
[D-14 Driver-on-Type]: D-14-driver-on-type.md
