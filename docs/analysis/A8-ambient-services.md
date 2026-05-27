---
status: draft
date: 2026-05-27
---

# A8. Ambient Services, Concrete Dependencies, and the Discipline Against Magical DI

## Context

Components in this framework reach for three categorically distinct things at
runtime:

1. Values that are **per-instance ambient** (stdout, stderr, logger, clock).
2. Optional **discrete behaviors** the framework or loader invokes on them
   (graceful shutdown, health check).
3. Concrete **required resources** owned by the loader and shared across the
   process (database pool, message broker, cache).

Conflating these is the failure mode this analysis exists to prevent. The
conflation has a name in dependency-injection literature: *ambient context*,
*service locator*, *magical DI*. The pattern smuggles required dependencies
through implicit channels, defers their existence checks to runtime, and
inverts the type system's job.

This analysis catalogues the three categories, names the carrier that fits
each, surveys the anti-patterns that emerge when carriers are misapplied, and
records the discriminator the requirements will quote.

## The Three Categories

### Per-Instance Ambient Values

Services with sane defaults whose specific instance differs per `Runner`.
Examples: stdin/stdout/stderr, the per-instance logger, a per-instance clock
for tests. Every runner can name what it wants; absence is meaningful only in
that the default applies. The category answers "what writer do I print to?",
where the question is a value query and the answer is structurally always
"yes, here is something."

### Optional Discrete Behaviors

Actions the framework or loader takes if the component supports them.
Examples: graceful drain (`Shutdowner`), liveness response (`Healthy`), hot
reload (`Reloader`). Absence is meaningful: the loader chooses a different
plan (rely on `ctx` cancellation alone, omit the `/healthz` endpoint, treat
SIGHUP as full restart). The category answers "can this component drain
gracefully?", where the question is binary and the absence informs the
loader.

### Concrete Required Dependencies

Process-shared resources the loader establishes once and components consume.
Examples: a SQL connection pool, a Kafka producer, a Redis client, a tracer
provider. Absence is a startup error; runtime fallback is not appropriate.
The category answers "where is the database I cannot work without?", where
the question is required and absence is a bug.

## Carrier Per Category

| Category | Carrier | Pattern | Default behavior |
| --- | --- | --- | --- |
| Ambient values | `context.Context` with helper functions | Loader populates `ctx` via `WithStdout`, `WithLogger`; consumer reads via `Stdout(ctx)`, `Logger(ctx)` | Helper returns a working default if unset |
| Discrete behaviors | Structural typing on the runner | Component implements `Shutdowner`; loader type-asserts | Loader skips the action if not implemented |
| Concrete dependencies | Explicit imports of loader-owned packages | Loader's package owns the resource; exposes accessor (`loaderdb.Pool`); consumer imports the loader's package and calls the accessor | Accessor fails early if the resource is not established |

### Why Ambient Values Fit `context.Context` + Helpers

The helper-with-default pattern collapses capability-detection ceremony into
the helper. There is no `if c, ok := rt.(Logger); ok { ... } else { ... }` at
every call site; the helper checks once. The pattern is the one OpenTelemetry
uses for `trace.SpanFromContext`, and the one slog would use if stdlib had
committed to context-attached loggers (it did not; the pattern is
community-standard outside stdlib).

The reading of the famous `context.go` comment:

> Use context Values only for request-scoped data that transits processes
> and APIs, not for passing optional parameters to functions.

is that the warning targets two specific antipatterns: (a) smuggling required
dependencies through context, and (b) kitchen-sink usage. Per-instance
ambient services accessed through helper functions with sensible defaults is
neither. It is the third pattern, exemplified by OpenTelemetry and the wider
ecosystem.

### Why Discrete Behaviors Fit Structural Typing

The question "does this component support behavior X?" is binary, the answer
is structurally meaningful (absent means "the loader does something
different"), and Go's structural typing is the natural mechanism.
`http.ResponseWriter`'s `Flusher` and `Hijacker` are the canonical
inspiration: the underlying transport may not support these actions, and the
absence informs the handler's plan.

### Why Concrete Dependencies Fit Explicit Imports

Imports are honest. The component's import graph declares what it needs from
the host environment. A missing dependency fails at startup (the loader's
accessor errors), not at first use deep inside an instance. The pattern is
what `http.Server` uses for its `Handler`, `Addr`, and `TLSConfig` fields:
explicit, required, set at construction.

## Inspirations and Where They Diverge

`http.ResponseWriter`. The canonical capability-detection model. The handler
holds the value and asks "can I flush?" by type-asserting to `Flusher`. The
answer is structurally meaningful (HTTP/1 plain transport differs from
HTTP/2 with a buffering proxy). Capability detection earns its keep because
the absence is informative.

OpenTelemetry `trace.SpanFromContext`. The canonical ambient-with-default
model. `SpanFromContext(ctx)` returns the active span if one is attached;
otherwise it returns a no-op span that satisfies the same interface and
discards all calls. Application code never checks for absence; the default
does the right thing.

`http.Server`. The canonical concrete-dependency model. The server's
dependencies (handler, address, TLS config) are fields on the struct, set at
construction. There is no `http.HandlerFromContext` lookup. The dependencies
are explicit because they are required.

12-factor app. Factor III declares config-in-environment. The application
reads env at startup, fails if required keys are missing, then runs. The
fail-fast discipline is the same one this framework applies to concrete
dependencies.

## The Runtime/Environment Object This Framework Did Not Adopt

The conversation that produced this analysis considered a `Runtime` (or
`Environment`) value passed as a parameter to `Run`, carrying stdin/stdout/
stderr/logger. The shape:

```go
// NOT ADOPTED.
type Runtime struct {
    Stdin          io.Reader
    Stdout, Stderr io.Writer
    Logger         *slog.Logger
}

type Runner interface {
    Run(ctx context.Context, rt Runtime) error
}
```

The shape has merits (explicit parameter, no hidden state) but two structural
problems.

**First, capability extension.** To let custom loaders add services (Clock,
Tracer, MetricsRegistry), `Runtime` would have to be either a struct (no
extension from another package) or an interface with capability detection
(the `http.ResponseWriter` shape). Either way, the framework now has *two*
capability vocabularies: one for component behaviors (`Shutdowner`) and one
for runtime services (`Clock-ProvidingRuntime`). Two vocabularies for one
design idea.

**Second, capability detection is wrong-fit here.** Asking "does this
runtime provide a logger?" is a degenerate question; the answer is
structurally always yes (every runtime can produce a logger, or a logger
that discards). The capability-detection pattern fits strong behaviors that
may meaningfully not exist, not weak value-providing where the answer is
always "yes, here is something." ([A4](A4-component-contract.md) develops
the behaviors-versus-values distinction.)

Consequence: runtime services move to context with helpers. Behaviors stay
on capability interfaces on the runner. The vocabularies stay disjoint.

## The Package-Level-Function-with-Instance Pattern (Preserved as Inspiration)

A discarded-but-valid alternative: the loader exposes per-instance services
via package-level functions taking the `Instance`. For example
`loader.PerComponentRuntime(inst *Instance) Runtime`, with the function
reading from an internal `sync.Map[*Instance]Runtime` populated by the
loader before `Run`.

The pattern has specific merits:

- Uniform access between `Run` and capability methods (both call the same
  function).
- Visible failure on pre-`Run` access (return zero, or panic, or return a
  typed error; the call site learns immediately).
- Per-instance keying with no global naming collisions.

The framework does not adopt it as the default because:

- It introduces a per-loader registry (a `sync.Map` keyed by pointer) whose
  lifetime the loader must manage explicitly.
- It pushes the pre-`Run` sentinel decision out of the type system into
  runtime behavior.
- It diverges from the standard `context.Context` propagation mechanism that
  the rest of the ecosystem speaks.

The pattern is preserved as a documented valid choice for custom loaders
whose services have lifetimes longer than a single `Run`, or that must be
accessible from goroutines without ctx-plumbing. It is not forbidden; it is
not the default.

## Anti-Patterns, with Code

The anti-patterns below all stem from applying the wrong carrier to a
category. Each is paired with the right shape.

### A. Context as DI Gateway for Required Dependencies

```go
// BAD. The connection pool is required, not ambient.
// A helper-with-default hides the dependency at the type level
// and turns "loader misconfigured" into a runtime nil panic.
func (a *myApp) Run(ctx context.Context) error {
    pool := application.DB(ctx)         // returns nil if unset
    rows, err := pool.QueryContext(ctx, "SELECT ...")  // panics on nil pool
    // ...
    _ = rows
    _ = err
    return nil
}
```

The failure: the type system loses its job. A required dependency should be
visible in the import graph, fail at startup, and survive `grep`. A
helper-with-default conceals it.

```go
// GOOD. The dependency is in the import graph; failure is at startup.
import "github.com/modern-engineering/prototype/.../loaderdb"

func (a *myApp) Run(ctx context.Context) error {
    pool, err := loaderdb.RequirePool(ctx)
    if err != nil {
        return fmt.Errorf("db: %w", err)
    }
    rows, err := pool.QueryContext(ctx, "SELECT ...")
    // ...
    _ = rows
    return err
}
```

The same pattern admits a `Must`-prefixed variant for cases where the caller
prefers to panic at startup:

```go
// Also GOOD. Process-level invariant; panic at the first call site that
// would have used a missing pool.
import "github.com/modern-engineering/prototype/.../loaderdb"

var pool = loaderdb.MustPool() // panics during package init if loader not ready
```

### B. Loader-Owned Shared Resources Injected Per-Instance Through Context

```go
// BAD. The pool is one shared value; routing it through each
// instance's ctx pretends it is per-instance. The same anti-pattern
// applies to "synchronized stdout" wrapped per-instance.
loaderctx := context.WithValue(ctx, dbKey{}, sharedPool)
loaderctx = context.WithValue(loaderctx, stdoutKey{}, syncedStdoutFor(inst))
go runner.Run(loaderctx)
```

The failure: the per-instance carrier is the wrong category for per-process
resources. The right shape is a loader-owned package exposing
process-globals:

```go
// GOOD. Process-level synchronized stdout owned by the loader.
package loaderos

// Stdout returns a writer that serializes writes across goroutines
// using a single internal mutex. All callers within the process share
// the same writer; per-instance interleaving is impossible.
func Stdout() io.Writer { /* ... */ return nil }
```

Any code in the process writes via `loaderos.Stdout()` and the loader's
mutex serializes the writes. No per-instance wrappers; no ctx-carried
writers. The discriminator: **per-instance values go on context;
per-process resources go in a package.**

### C. Reading Ambient Services During Instantiation

The instantiation function does not receive a context, so the failure cannot
occur with the current contract. The anti-pattern is recorded to motivate
why the contract is shaped this way.

```go
// STRUCTURALLY IMPOSSIBLE (and intended to remain so).
// If the instantiation function received ctx, it would be tempted
// to log "registered" during construction, breaking the dry-
// instantiation invariant from A7.
func newHelloWorld(inst *Instance, ctx context.Context) Runner {
    application.Logger(ctx).Info("constructing")  // breaks help
    return nil // illustrative
}
```

The contract: the instantiation function receives only the `Instance`. The
closure captures `ctx` at `Run` time, when ambient services are available
and the help renderer is not in flight.

```go
// GOOD. The closure receives ctx; instantiation is silent.
func newHelloWorld(inst *Instance) Runner {
    var who string
    inst.Flags.StringVar(&who, "who", "world", "the entity to greet")
    return application.RunFunc(func(ctx context.Context) error {
        application.Logger(ctx).Info("hello", "who", who)
        return nil
    })
}
```

### D. Bypassing the Helper's Default by Reading `ctx.Value` Directly

```go
// BAD. Reinvents the helper, uses the wrong key shape,
// skips the framework's documented default.
v, _ := ctx.Value("logger").(*slog.Logger) // wrong key type (string)
if v == nil {
    v = slog.Default()
}
v.Info("hello")
```

The failure: the framework's canonical keys are unexported types in their
helper's package:

```go
// Inside the application package, the key type is unexported.
type loggerKey struct{}
```

The misuse is structurally prevented by the key type's being inaccessible to
external packages. The right shape is to use the helper:

```go
// GOOD.
application.Logger(ctx).Info("hello")
```

Every canonical ambient service follows the same shape: an unexported key, a
`WithX` setter, and an `X(ctx)` getter that handles the default.

### E. Per-Call Mutation of Services

```go
// BAD. Context derivation is hierarchical: a parent's children
// see its values, but mutating the parent does not propagate to
// existing children. Trying to "swap stdout mid-Run" is fighting
// the model.
ctxMu.Lock()
ctx = context.WithValue(ctx, stdoutKey{}, newWriter)
ctxMu.Unlock()
```

The failure: context values are propagated by derivation, not by mutation.
There is no in-place update. The pattern reaches for context to do what
context cannot do.

If a loader genuinely needs runtime-mutable services, the right home is a
loader-owned package with explicit synchronization:

```go
// GOOD. Mutability is explicit and centralized.
package loaderos

import (
    "io"
    "os"
    "sync"
)

var (
    mu     sync.RWMutex
    stdout io.Writer = os.Stdout
)

func SetStdout(w io.Writer) {
    mu.Lock()
    stdout = w
    mu.Unlock()
}

func Stdout() io.Writer {
    mu.RLock()
    defer mu.RUnlock()
    return stdout
}
```

Mutability lives in one package, behind one mutex. The pattern is honest
about what is shared and how it is synchronized. Per-instance contexts stay
immutable.

## The Discriminator, Restated

This analysis hands the requirements one rule, suitable for direct quotation
by R002 (core seam), R004 (ambient services), and R007 (loader contract):

> Per-instance ambient values travel on `context.Context`, accessed via
> helper functions with sane defaults. Per-process shared resources are
> owned by loader packages and accessed through their package-level
> functions, with explicit fail-early semantics. Optional discrete
> behaviors that meaningfully may not exist are exposed as capability
> interfaces implemented by the component and detected by the loader via
> type assertion. Required dependencies are visible in the import graph;
> ambient services are not.

Forces this rule satisfies:

- **Type-system honesty.** Required things are typed and present; optional
  things are detected; defaulted things are silently available.
- **Multi-instance correctness.** Per-instance state lives in per-instance
  carriers (`ctx`); per-process state lives in per-process carriers
  (packages).
- **Fail-fast on misconfiguration.** Concrete dependencies error at startup,
  not deep inside instances.
- **Open extension.** Loaders add ambient helpers, capability interfaces,
  and concrete-dependency packages without coordinating with the core.

## How This Rule Feeds Requirements

- The loader-contract requirement uses it to constrain what loaders must
  populate on context versus what they must own at package scope.
- The capability-interface requirement uses it to discriminate when a new
  capability earns its place (must answer "is this a strong behavior or a
  weak value-providing").
- The ambient-services requirement uses it to enumerate canonical helpers
  and their default semantics.

## Open Questions

- **The canonical set of ambient helpers and their defaults.** Stdin
  (`io.Reader` returning EOF? `os.Stdin`?), Stdout (`io.Discard`?
  `os.Stdout`? `loaderos.Stdout()`?), Stderr (parallel choice), Logger
  (`slog.Default()`?). The requirements pick per-helper.
- **A helper-construction kit.** Whether the framework provides a generic
  `Key[T]`-keyed getter/setter pair so custom loaders can produce conforming
  helpers cheaply. Current direction: yes, with documented conventions.
- **Concrete-dependency accessor shape.** Whether concrete-dependency
  accessors return `(T, error)` or panic on absence (`MustPool` versus
  `RequirePool`). Both have valid use cases; the convention may admit both
  per loader's preference. The framework documents the trade-off; each
  loader chooses for its accessors.
- **Threading of context-carried services into capability methods.**
  Capability methods like `Shutdown(ctx) error` receive their own `ctx` from
  the loader. Whether the loader derives that `ctx` from the same parent as
  `Run`'s `ctx` (so ambient services propagate) is a loader contract item
  developed in R007. The current direction: yes, the loader is responsible
  for `ctx` parentage, and ambient services are available to capability
  methods through the same helpers `Run` uses.
