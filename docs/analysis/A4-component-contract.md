---
status: draft
date: 2026-05-27
---

# A4. Component Contract: Single-Method Interface with Capability Extensions

## Context

The component (working name: `Runner`) is what the loader runs after the
instantiation function returns. This analysis surveys interface shapes for
that role and frames **capability interfaces** as the framework's growth
mechanism, both for third-party loader extensions and for the core or unified
loader's own evolution.

## Inspirations

### `http.Handler` + `http.HandlerFunc`

```go
type Handler interface {
    ServeHTTP(w ResponseWriter, r *Request)
}

type HandlerFunc func(w ResponseWriter, r *Request)
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }
```

The canonical Go pattern: single-method interface plus a function adapter.
Trivial implementations are functions wrapped via the named type; stateful
implementations are structs with methods. The runtime holds an interface; the
caller chooses shape.

### `http.ResponseWriter` and Its Capabilities

```go
type ResponseWriter interface {
    Header() Header
    Write([]byte) (int, error)
    WriteHeader(statusCode int)
}

type Flusher       interface { Flush() }
type Hijacker      interface { Hijack() (net.Conn, *bufio.ReadWriter, error) }
type CloseNotifier interface { CloseNotify() <-chan bool } // deprecated
```

The runtime hands the handler a `ResponseWriter`. The handler type-asserts to
discover optional behaviors. The pattern fits because the capabilities are
**strong behaviors** (actions the underlying transport may or may not
support), not value-providing concerns.

### `io.Reader` and the Capability Ladder

`io.Reader`, `io.ReaderAt`, `io.WriterTo`, `io.Seeker`, `io.Closer`.
Structural typing lets functions like `io.Copy` detect `WriterTo` for fast
paths without disrupting plain `Reader` users. Extension by structural typing
in stdlib.

### `fs.File` and Its Capabilities

`fs.File` (base), `fs.ReadDirFile` (lists entries), `fs.StatFS` (stat without
open). The same pattern in modern stdlib.

### `sort.Interface`

```go
type Interface interface {
    Len() int
    Less(i, j int) bool
    Swap(i, j int)
}
```

Pure interface, no function adapter. The counter-example: when an interface
has multiple required methods, stdlib does not provide a `HandlerFunc`-style
adapter. The adapter pattern fits single-method interfaces.

## What This Framework Takes from the Inspirations

- A **single-method base interface** for the runner. `Run(ctx) error` is
  enough; [A5](A5-termination.md) develops the signature.
- A **function adapter** (working name: `RunFunc`) so function-shaped
  components avoid struct boilerplate. Same shape as
  `http.Handler`/`http.HandlerFunc`.
- **Capability interfaces** (working names: `Shutdowner`, `Healthy`, `Ready`,
  ...) as optional extensions on the same runner value, discovered by the
  loader through type assertion.

## Capability Interfaces as a Growth Mechanism

This is the design-load-bearing observation. Capability interfaces serve two
distinct purposes in this framework:

### Third-Party Loader Extension

A custom loader (logging, tracing, sharding, leader-election, batching)
defines a capability interface in its own package. Components opt in by
implementing the interface; the loader type-asserts at use site. The loader
ships a new capability without coordinating with the framework core.

```go
package leaderloader

type Leadership interface {
    OnLeaderGained(ctx context.Context) error
    OnLeaderLost(ctx context.Context) error
}
```

A component that implements `Leadership` participates in the leader-election
protocol. A component that does not is run as a regular replica. The loader
type-asserts and dispatches accordingly.

### Unified Loader Evolution

The core or unified loader itself grows its capability set over time without
breaking the base `Runner` contract. The opening set may include `Shutdowner`;
later it may grow to include `Healthy`, `Ready`, `Reloader`, flusher-like
behaviors, and beyond. **Existing components continue to satisfy `Runner`
alone; new components opt in by implementing more.**

Both modes share the shape: interface declaration in the package that
consumes it, type assertion on the runner value, additive growth. Neither
mode requires registry coordination.

## Trade-Offs the Pattern Accepts

- **Strength: open-set extension.** The framework's capability vocabulary
  grows without breaking changes; loaders introduce vocabulary independently.
- **Weakness: grep-based discoverability.** To know what capabilities exist, a
  developer searches for interfaces defined in the framework and its loader
  packages. There is no central catalogue.

For a framework whose extensions are mostly authored by loaders rather than
by the framework itself, this trade-off is favorable: the core stays small,
the extension story is uniform, and the cost is a grepped survey rather than
a registry maintenance burden.

## Behaviors Versus Values: When Capability Interfaces Fit

A capability interface answers the question "does this component support
behavior X?" The answer is binary; the absence is meaningful (the loader
skips the action, or substitutes a different plan).

**Capability interfaces fit strong behaviors:**

- `Shutdowner`: graceful drain. The loader calls `Shutdown(ctx)` on signal;
  without it, the loader relies on context cancellation alone.
- `Healthy`: liveness probe response. The loader exposes `/healthz`; without
  it, the endpoint is omitted.
- `Reloader`: hot reconfiguration. The loader fires `Reload` on SIGHUP;
  without it, SIGHUP triggers full restart.

**Capability interfaces fit poorly for weak value-providing concerns:**

- A capability called `ProvidesLogger` answers a question whose answer is
  structurally always "yes" (every component can log, or log to nothing).
- A capability called `ProvidesStdout` is similar: the answer is yes by
  default; the question is degenerate.

[A8](A8-ambient-services.md) develops this distinction in detail. The summary:
capability interfaces are for **discrete actions**, not for value queries.
Value queries belong on context with helper functions and sane defaults.

## Open Questions

- **Naming of the opening capability set.** `Shutdowner` is unambiguous;
  `Healthy` versus `HealthChecker` versus `Probe` is a naming concern the
  requirements weigh.
- **Whether `Healthy` and `Ready` are separate interfaces or one interface
  with two methods.** The Kubernetes liveness/readiness distinction is real;
  the framework may want both, or may collapse them.
- **Whether the framework provides default implementations** (a
  `NopShutdowner`) or strictly relies on type assertion. The current direction
  favors type assertion: a missing capability is meaningful absence, not a
  candidate for a no-op default.
- **The fixup point.** A component with state needs the type-asserting
  capability methods to share that state with `Run`. The store-state-in-Run
  pattern is the current direction; capability methods read the receiver. The
  framework does not provide a built-in helper; component authors choose
  their own synchronization.
