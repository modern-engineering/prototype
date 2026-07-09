---
status: draft
since: 2026-07-09
refined-by:
  - D-10-generate-compile-run
---

# Staged Catalogue Compilation: How the Compiler Knows User-Defined Elements

## Context

[A-09 Solution Layer] recommends the compiled path — "a compiler links
definitions against the catalogue and emits an intermediate language (IL) that
deployment systems consume" — and casts the catalogue in the role C gives
headers: "the catalogue's registered types". The records since fix what the
compiler consumes and produces: [D-07 Verb-First Directives] shapes the
statements and puts safety in compiling, [D-08 Desired-State Image] fixes the
artifact, [D-09 Catalogue Citizenship] guarantees every element honours the
library's full contract. No record yet designs how the compiler comes to _know_
the catalogue.

The gap is sharper than it looks. [D-06 Declarative Descriptor Values] made an
element's identity a Go value: an exported package-level struct literal,
referenced directly, registered nowhere. The catalogue is therefore not a
database or a file; it is a set of ordinary Go packages, and its contents exist
only inside programs that import those packages. Go has no package-level
reflection — no process can enumerate another package's exported values without
compiling against it — so whatever the compiler is, it must somehow contain the
user's packages.

The syntax scratchpad's closing ideas sketch one resolution — generate the
compiler's source per invocation, making the user's packages part of it for one
build — and pose two questions on the way: how catalogue packages are named from
definition files, and whether elements are Go types or Go values.
[A-13 Mission Analysis] fixes the frame any answer must serve: an authoring loop
that runs locally, in seconds, with errors positioned in the sources; catalogue
changes that surface before anything ships; a dev-loop latency measure hanging
over machinery placed inside the loop. This analysis surveys the discovery
pathways and the import semantics, maps the build's timelines onto
[A-10 Value Binding]'s binding moments, and weighs the narrowings the record
model needs. Working names (`sdl build`, `import`, `sdl.mod`) are provisional.

## Five Pathways to the Catalogue

**Init-time self-registration.** Catalogue packages insert their elements into a
process-global table from `init()`; the compiler blank-imports the packages and
reads the table. This is the registry [D-06 Declarative Descriptor Values]
rejected for application identity, and it fails here a step earlier: something
must still name the packages to blank-import, so discovery is only moved into an
import list, while every rejected cost returns — global mutable state, `init()`
ordering, collision panics, relationships the compiler cannot check. And because
the package set differs per solution, the importing program cannot be one
prebuilt binary anyway: a build per solution is already implied.

**External schema descriptions.** A manifest beside the code (YAML or JSON)
describes each element's name, parameters, and outputs; the compiler stays a
standalone binary and never touches Go. The manifest restates what the Go type
system already knows and drifts from it; maintenance doubles on the developer
[A-13 Mission Analysis] wants publishing once; and checking against a
transcription is the parallel-schema infrastructure [D-07 Verb-First Directives]
declined with the schema-validated YAML front-end. Generating the manifest from
source fights the drift, but the generator needs the very discovery mechanism it
was meant to replace.

**Dynamic loading.** Catalogue packages compile to shared objects a resident
compiler loads (Go's `plugin` package). Loaded symbols are live values, but the
mechanism is fenced: platform support is partial, and plugin and host must be
built by the same toolchain version against identical dependency versions —
precisely the skew surface a catalogue economy of independently publishing teams
maximizes.

**Source interpretation.** Embed a Go interpreter and evaluate the catalogue
packages inside the compiler. No toolchain dependency, no plugin fences — but
the interpreter's semantics are a reimplementation of Go: fidelity gaps around
reflection, generics, and cgo are structural; the dependency is heavy; and,
decisive here, the values it produces are _near_ the values the application runs
with, not the same ones. A parameter that validates under the interpreted
`flag.Value` may still fail under the compiled one.

**Generate, compile, run.** The Go toolchain's own answer to the problem:
`go test` synthesizes a `_testmain.go` that imports the packages under test,
compiles it in the user's module context, and runs the binary. Applied here:
`sdl build` synthesizes a main that imports the catalogue packages the
definition files name, compiles it in the solution's module context, and runs
it; the generated program is the compiler's back half and emits the image
itself.

The last pathway deserves the deepest look, because it is different in kind:
inside the generated binary the user's elements are live values — a reference
like `ff.Ping` resolves to the very descriptor value
[D-06 Declarative Descriptor Values] made the identity — and nowhere else in the
toolchain is that true. Three properties follow.

- **The parameter scheme is obtained, not transcribed.** The compiler calls the
  element's instantiation function once against a fresh instance and discards
  the runner — the dry-instantiation move [A-07 Documentation] established for
  help rendering, sound for the reasons [A-03 Parameterization]'s discipline
  gives (pure, infallible, instance-only, safe to call repeatedly). The slots
  that appear are the scheme.
- **Validation is the run-time code.** A literal parameter is checked by the
  very `flag.Value` implementation that will parse the real value at run time;
  drift between the checking schema and the running schema is not reduced but
  structurally impossible — they are one piece of code.
- **Violations are containable.** The invariant is a convention, and the
  compiler must survive its breach: dry instantiation runs under `recover`, and
  a panicking constructor becomes a diagnostic positioned at the statement that
  referenced the element, never a compiler crash. The invariant is also testable
  where it is authored: the library's `CheckParser`-style harness for custom
  flag values has a natural sibling — call the instantiation function twice on
  fresh instances; assert no panic, a non-nil runner, an identical slot schema —
  turning [D-09 Catalogue Citizenship]'s full-contract guarantee into a failing
  test.

The costs are equally structural: a Go toolchain becomes part of every solution
build; a `go build` sits inside the authoring loop [A-13 Mission Analysis]
budgets in seconds; and generated code couples the tool's version to the library
version in the user's module, so skew must be detected rather than miscompiled.
This analysis recommends generate-compile-run, with the costs carried into the
open questions below.

## Elements as Values, Not Types

The scratchpad's second question: are catalogue elements Go _types_ (structs the
compiler discovers and instantiates by convention) or Go _values_ (package
variables the generated program references directly)?

[D-06 Declarative Descriptor Values] answered it for applications — identity is
the value; the type that does the work sits behind `New` — and the newer element
kinds have the same shape: a provisioning type is data (name, kinds, the output
scheme [A-11 Substrate and Slices] registers on the type) plus a pure
parameter-declaration surface; a symbol type is data alone (name, and the
sensitivity attribute [A-10 Value Binding] wants declared on the type). Values
carry that as exported fields plus a pure constructor; types would need
reflection conventions or mandatory constructor shapes — exactly the machinery
values avoid — and the user's structs implementing the runner never surface to
the compiler at all. If types ever earn a place, a generic lift
(`NewFor[T]`-style) can package a type _as_ a value without the registration
machinery changing: the discovery surface stays "exported package-level values
of the catalogue's element types" either way.

This analysis recommends elements as values, extending
[D-06 Declarative Descriptor Values]'s identity model from applications to every
catalogue element, with the generic lift recorded as the door back.

## Naming the Catalogue: Imports

If elements live in packages, definition files must name packages. Two surfaces
compete.

**Per-file import blocks.** Each definition file opens with a Go-style import
block, and every element reference is package-qualified — `ff.Ping`, the way Go
reads exported identifiers. A unit's meaning is derivable from the unit alone,
mirroring the Go source-file model the directory-of-peers rule already borrows,
and the generated program takes its import list from the files verbatim.

**External type locations.** A side file (or invocation flags) maps bare element
names to packages, keeping references short (`Ping`). The unit stops being
self-contained; two files must now agree, and drift; and bare names collide as
catalogues multiply — two teams will eventually both publish a `Ping`, and only
qualification keeps both usable in one solution.

This analysis recommends per-file imports with references always qualified, even
while unambiguous; the record-model section below weighs that choice against the
corpus's bare-name sketches. The consequence worth naming: the authoring surface
reuses the ambient Go module or workspace, resolving imports exactly as the `go`
CLI would in the solution's directory — an active module through its own
requirements and replaces (a solution beside its catalogue's source sees that
source unreleased); a workspace through its union of modules; no module context
through the generated program's own module, at each import's latest version. A
solution-own manifest (`sdl.mod`-style: pinned catalogue versions, file-set
selection for [D-08 Desired-State Image]'s structural variance, identity
ownership) stays an open path; it affects only the authoring surface, never the
image contract.

## Four Timelines onto Three Binding Moments

[A-10 Value Binding] stages values across compile time, site configuration, and
reconcile time. The build machinery refines the first moment into two visible
timelines and leaves the later ones untouched:

| Timeline      | Runs               | Binds                                  |
| ------------- | ------------------ | -------------------------------------- |
| authoring     | editor, formatter  | nothing — syntax only                  |
| IR build      | generated program  | element refs, literals, `var` defaults |
| site config   | image consumers    | `extern` (MUST), `var` overrides (MAY) |
| reconcile-run | controllers, hosts | instance outputs; live slot parsing    |

The useful reading is what runs _dryly_ when. At IR build the instantiation
functions run dry — slots declared, runners discarded, literals passed through
`flag.Value` for validation only — and no other user code executes; `extern`
symbols and instance outputs stay symbolic in the image. At reconcile-run the
same functions run again, wet: loaders bind real values and runners are invoked.
The same code executing twice, once dry and once wet, is the drift-proofing:
what checked at build is what parses in production. Authoring binds nothing and
needs no catalogue — formatting stays editor-local — while _checking_ does need
the catalogue and therefore the toolchain; the seconds budget of
[A-13 Mission Analysis]'s authoring loop is spent exactly there.

## Narrowing the Record Model

[A-09 Solution Layer] states the isomorphism plainly — "each definition
statement compiles to one IL record" — and [D-07 Verb-First Directives] made it
a driver. Building the image presses three narrowings on a literal reading; each
is a trade-off to weigh, not a silent drift.

**Statements narrow to records and symbol rows.** `deploy` and `provision`
statements become records; `var` and `extern` statements become symbol-table
rows — still one statement, one addressable entry, and the table is where
[A-10 Value Binding] already put them; `default` statements fold into the
records they modify, each folded binding carrying its provenance; `solution` and
`import` clauses become headers. The alternative — reifying defaults as
free-standing records — makes every consumer reimplement merge order, and the
image stops being effective desired state, which [D-08 Desired-State Image]'s
diff-and-converge loop depends on. Front-end peerhood survives at the record
level: a designer emits records and symbols directly; `default` sugar is the
text front-end's own business. This analysis recommends the narrowing, with
provenance kept so folded defaults stay explainable.

**References are always qualified.** Every sketch in the corpus writes bare
`Ping`; qualification could be reserved for ambiguity, making the import table
part of name lookup. But optional qualification makes a reference's meaning a
function of the import table and of today's catalogue contents — adding an
import can ambiguate a bare name that compiled yesterday. Always-qualified
references cost a few characters per use and buy stability under catalogue
growth, a single resolution rule, and mechanical editability. This analysis
recommends always-qualified.

**Composite parameters defer.** The mockups promise `key: { ... }`, and
[A-10 Value Binding] fixed whole-value fields. Catalogue citizens declare
flag-shaped slots, and flags flatten structure into dotted names, so a composite
literal would need a mapping onto slots that does not yet exist. This analysis
recommends whole values only, with composites as a door; the trigger is the
first catalogue element whose parameter surface truly nests.

## Tooling Shape

One hierarchy question remains: the shape of the command surface. Git's
porcelain-and-plumbing split is the prior art — porcelain works the material
humans author, plumbing operates the artifact. Porcelain here covers the
sources: build, format, and eventually a mechanical statement-level edit command
for tools that must rewrite definitions without hand-patching text. Plumbing
covers the emitted image: re-rendering it as one canonical definition file (the
read side, doubling as the compiler's round-trip test), and in-place amendment
and queries (the write side — the `go mod edit` shape: the tool amends the
artifact it owns rather than making every caller parse it). This analysis
recommends the two-level structure and leaves open whether site bindings — the
values a site must supply for `extern` symbols — are written _into_ the image by
plumbing or live beside it; the question is continuous with
[A-10 Value Binding]'s override-audit question and [A-13 Mission Analysis]'s
open home for site configuration.

## Open Questions

- **Version skew between tool and library.** The generated program calls library
  entry points at whatever version the user's module pins; the tool was built at
  another. Warn versus refuse, and what a handshake pins, are open; strictness
  should follow the first observed miscompile.
- **A solution manifest.** Whether `sdl.mod`-style metadata — pinned catalogue
  versions, file-set selection, identity and generation ownership
  ([A-13 Mission Analysis] leaves the owner open) — earns existence, or the
  ambient module keeps carrying everything.
- **Composite literals.** The nesting trigger above, and the slot mapping a
  composite body would require.
- **The latency budget.** [A-13 Mission Analysis] prices the authoring loop in
  seconds; a `go build` now sits inside it. What budget a requirement sets, and
  whether warm-cache builds hold it, decides how much caching machinery the
  toolchain owes.
- **Where site configuration lives.** In the image via plumbing, or beside it —
  and validated by what, against the symbol table's types.

[A-03 Parameterization]: A-03-parameterization.md
[A-07 Documentation]: A-07-documentation.md
[A-09 Solution Layer]: A-09-solution-layer.md
[A-10 Value Binding]: A-10-value-binding.md
[A-11 Substrate and Slices]: A-11-substrate-slices.md
[A-13 Mission Analysis]: A-13-mission-analysis.md
[D-06 Declarative Descriptor Values]: ../adr/D-06-declarative-descriptor-values.md
[D-07 Verb-First Directives]: ../adr/D-07-verb-directive-syntax.md
[D-08 Desired-State Image]: ../adr/D-08-desired-state-image.md
[D-09 Catalogue Citizenship]: ../adr/D-09-catalogue-citizenship.md
