---
status: draft
date: 2026-05-27
---

# A2. Architecture: Hosting Many Components in One Program

## Context

Building on the scope established in [A1](A1-scope.md), the framework hosts
multiple components in one Go binary, with the binary's entry point delegating
to a loader that wires CLI arguments, environment, and configuration into
component parameter slots. This analysis surveys ecosystem inspirations for
that shape, names what each contributes and what each lacks against this
project's scope, and identifies why a registry value is necessary.

## Inspirations

### `golang.org/x/tools/go/analysis` (singlechecker, multichecker, internal/checker)

The drivers `singlechecker.Main(a *Analyzer)` and
`multichecker.Main(...*Analyzer)` are roughly fifty-line shells that delegate
to `internal/checker.Run` plus `internal/analysisflags.Parse`. Each `Analyzer`
is a value (not an interface) carrying `Name`, `Doc`, `Flags`, `Run`,
`Requires`, `FactTypes`. Introspection (help, list, doc) reads the struct
fields directly. The `multichecker` driver re-publishes each analyzer's flags
onto `flag.CommandLine` under a `<name>.<flag>` prefix and exposes each
analyzer as a `triState` flag bearing its `Name`. `internal/checker` is
`internal/` from this module's perspective: a downstream "enterprise checker"
cannot funnel through it without forking.

**What this project likes.** Component-as-introspectable-value gives discovery
and help for free; per-component `FlagSet` plus host-side re-publishing
decouples component authoring from host policy; the lifecycle is a clear
pipeline (parse, validate, run, report, exit).

**What this project dislikes** (against the [A1](A1-scope.md) scope): one-shot
batch lifecycle (analysis ends, process exits); globals on `flag.CommandLine`,
`os.Args`, and `os.Exit` make multi-instance hosting impossible; the
`internal/` packaging prevents the funnel from being a public seam for
downstream loaders.

### `golangci-lint`

The package `pkg/lint/lintersdb` organizes linters as a static slice produced
by `Builder.Build(*config.Config) []*linter.Config`. The `Manager` accepts
variadic builders so that built-ins, Go-plugin `.so` files, and module-plugin
sources are uniform. The package `pkg/lint/linter` defines a tiny `Linter`
interface (`Run`, `Name`, `Desc`) and a wrapper `linter.Config` carrying
metadata (presets, deprecation, load mode, since, URL, aliases). The
`pkg/commands` tree is a cobra command hierarchy (`run`, `linters`, `config`,
`cache`, `custom`, `version`), and `pkg/commands/flagsets.go` defines reusable
flag-group functions composed per subcommand. Configuration is one typed
struct per linter, aggregated into `LintersSettings`, defaulted via a single
literal, parsed via viper.

**What this project likes.** Static-slice-plus-builder beats `init()`-time map
magic for diff-friendliness, deterministic order, and testability; the
wrapper-with-metadata pattern (`linter.Config`) keeps the runtime interface
minimal while letting the host carry rich descriptive state; reusable
flag-group functions admit subcommand-per-concern without flag duplication.

**What this project dislikes** (against scope): batch analysis, not
long-running; the cobra command tree assumes a CLI-shaped entry point and
human invocation; the plugin transports (Go plugin `.so`, custom-gcl rebuild)
target a different audience than long-running services with multiple
instances.

### `peterbourgon/ff/v4`

The `Command` struct carries `Name`, `Usage`, `Flags`, `Exec func(ctx
context.Context, args []string) error`, and `Subcommands`. The package's
`ff.Parse` layers sources (flags, environment, config file) with documented
priority.

**What this project likes.** Per-`Command` `FlagSet` matches the per-instance
`FlagSet` needs that emerge in [A3](A3-parameterization.md); the
`(ctx, error)` entry-point shape matches the long-running contract
([A5](A5-termination.md)); layered-source parsing fits A3's
"parameterization comes from many places."

**What this project dislikes** (against scope): `Command`'s tree of commands is
CLI-shaped; multi-instance hosting in one process is not the design center;
nothing in `ff` anticipates that two instances of the same `Command` might
coexist in one program.

## Forces

The survey above surfaces four forces that any architecture for this project
must address:

- **Funnel as public seam.** Multiple loaders (built-in CLI, env-only,
  file-driven, control-plane-driven) must funnel through one set of types. The
  analysis framework's `internal/` mistake is the one this project cannot
  repeat.
- **Discoverability versus dynamism.** The framework needs to enumerate
  components (for help, list, dispatch) at runtime without sacrificing the
  testability of fresh registries.
- **Multi-instance hosting.** The hosting pattern must admit "one program, N
  instances of one component" (developed in [A3](A3-parameterization.md)).
  Globals on `flag.CommandLine` are disqualifying.
- **Long-running lifecycle.** The hosting pattern must admit signal-driven
  shutdown, structured observability surfaces, and components that grow
  capability over time ([A4](A4-component-contract.md),
  [A5](A5-termination.md)).

## Why a Registry Exists

The analysis framework gets away without one because its binaries fix the
analyzer list at compile time (`singlechecker.Main(a)` takes one;
`multichecker.Main(as...)` takes a slice from the caller) and never enumerate
by name at runtime beyond the help renderer and the `-NAME` triState filter.
The `Run` pipeline iterates the slice once and returns.

For this project's scope, several forces demand a registry value:

- **Multi-instance lookup.** A loader that creates two parameterized instances
  of `hello-world` needs to find the descriptor by name from the program's
  universe of descriptors.
- **Cross-cutting enumeration.** Help, list-style introspection, and per-loader
  dispatch all walk the same universe. A passed-around slice can serve this,
  but the slice has to come from somewhere.
- **Test isolation.** Tests instantiate fresh registries to avoid global-state
  contamination; package-level globals (`singlechecker`'s implicit "the binary
  owns one analyzer" pattern) cannot offer this.

The shape this analysis surfaces (without prescribing) is a typed `Registry`
value plus package-level convenience functions (`Register`, `Lookup`, `All`)
delegating to a process-wide default registry. Loaders accept `*Registry` or
variadic descriptors. The pattern parallels `http.DefaultServeMux` plus
`http.HandleFunc`.

## The Funnel-and-Loaders Topology

Two layers emerge.

**Core public seam** (working name: `application`):

- Component-as-blueprint and component-as-instance value types.
- Registry mechanics.
- The instantiation function contract.
- The Runner interface and the base capability set
  ([A4](A4-component-contract.md)).
- Help rendering ([A7](A7-documentation.md)).
- Metadata mechanism ([A6](A6-metadata.md)).
- Ambient-services helpers and their canonical key types
  ([A8](A8-ambient-services.md)).

**Peripheral loaders** (working names: `ffloader`, `enterpriseloader`, ...):

- Decide where parameter values come from (CLI, environment, file, control
  plane).
- Decide flag namespacing (subcommand-shaped, prefix-shaped, single-app).
- Decide how to enumerate instances (one per program, N from a config
  document, N from a command tree).
- Decide the dispatch model (one main, subcommand router, supervisor).
- Decide exit policy (first-failure, aggregate, restart).

The seam is the funnel. The choices on either side of it are independent.

## Open Questions

- The registry's exposure mechanism (package-level `Register` versus only via
  `*Registry` construction) is a usability/testability trade-off the
  requirements will weigh.
- The exact metadata schema on the descriptor.
  [A6](A6-metadata.md) surveys options; the requirements pick one.
- Whether plugin transports (Go plugin `.so`, build-a-custom-binary) belong in
  scope. Not addressed here; mentioned only to note that `golangci-lint`'s two
  transports each carry their own trade-offs and neither fits long-running
  multi-instance use directly.
