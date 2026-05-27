---
status: draft
date: 2026-05-27
---

# A6. Metadata on Components

## Context

The framework needs to attach typed, queryable **metadata** to descriptors so
that loaders can read it without coordinating through the core. Examples of
metadata that loaders might want to read or write: environment-variable
prefix, config-file subtree path, presets ("worker", "supervisor"),
deprecation notice, since-version, owner team, alternative names, scheduling
hints. This analysis frames the need and surveys ecosystem options for
carrying such metadata in Go.

## The Inspiration: Kubernetes Resource Metadata

Kubernetes resources carry three kinds of metadata:

- **Labels.** Indexed, used for selection and grouping (`matchLabels`, label
  selectors).
- **Annotations.** Not indexed, used for tooling-specific or human-readable
  data (`last-applied-configuration`, ownership notes, controller state).
- **OwnerReferences.** Typed pointers expressing structural relationships
  (replica sets to pods, services to endpoints).

Each is typed (string keys, structured values via schema definitions per
resource kind) and queryable through the standard API. The pattern is
extensible: tools add new annotation keys without coordinating with the core
API server.

For this framework, the parallel is: how does a loader attach (and another
loader read) typed values on a descriptor without making the descriptor's
struct definition the choke point?

## Survey of Go Carriers, with Their Downsides

### Struct Tags

```go
type Descriptor struct {
    Name string `prefix:"MYAPP" team:"platform"`
}
```

Compile-time placement, but reflection-bound and stringly-typed at retrieval.
Tag values are strings; consumers parse them. The schema is implicit. The
pattern is workable for marshaling (JSON, YAML), poorly suited for
"loader-specific schema growing over time" because every new loader-defined
metadata key has to ship in the tag space of a struct it does not own.

### Static Analysis of Source Code

A separate tool walks the AST to find annotations (magic comments,
helper-function calls). Heavy machinery: requires a separate toolchain
pipeline, slows builds, brittle to refactors. Works for code generation
(`stringer`, `mockgen`) but does not fit runtime introspection by loaders.

### Magic Strings or Naming Conventions

"If the descriptor's name starts with `worker-`, treat it as a worker." No
compile-time checks; refactors break silently; conventions diverge between
teams. Lossy and reverse-engineerable only by reading prose somewhere else.

### `init()`-Time Global Maps with String Keys

```go
var Annotations = map[string]map[string]any{
    "hello-world": {"team": "platform", "since": "v0.1"},
}
```

A common Go idiom for "tag a thing with arbitrary data." Type-erased at
retrieval (consumer must assert). Mutable global state, hostile to tests
because tests cannot easily run in parallel without registry contamination.

### `context.WithValue` with Stringly-Typed (or Struct-Typed) Keys

The runtime-values pattern, used legitimately in [A8](A8-ambient-services.md)
for ambient services. Works for request-scoped data and ambient services;
type-erased at retrieval; type assertion required. Conflates request-scoped
data with descriptor-level metadata if used here.

### Generic Typed Keys (`Key[T]`)

```go
package ffloader

var EnvPrefix = application.Key[string]{Name: "ffloader.envPrefix"}
```

```go
// Author of a component sets:
descriptor.Set(ffloader.EnvPrefix, "MYAPP")

// ffloader reads:
prefix, ok := application.Get(descriptor, ffloader.EnvPrefix) // (string, bool)
```

Each loader declares the keys it cares about in its own package. Setter:
`descriptor.Set(EnvPrefix, "MYAPP")`. Getter:
`application.Get(descriptor, EnvPrefix)` returning `(string, bool)`. Typed at
retrieval, open-set extension, no central registry. Trade-off: requires Go
1.18+ (no constraint for this project); the user-facing API is slightly less
familiar than the untyped `context.WithValue` idiom.

## Forces

Four forces emerge from the inspirations and the survey:

- **Open-set extension.** Loaders must add keys without coordinating with the
  core.
- **Type safety.** Consumers should not assert types at retrieval.
- **Discoverability.** The keys a loader cares about should be visible in
  that loader's package.
- **Testability.** Setting and inspecting metadata should not depend on
  package-level mutable state.

## Scoring the Carriers

| Carrier | Open extension | Type safe | Discoverable | Testable |
| --- | --- | --- | --- | --- |
| Struct tags | No (fixed schema) | No (strings) | Yes (in struct) | Yes |
| Static analysis | Yes | Variable | Variable | No (toolchain) |
| Magic strings | Yes | No | Variable | Yes |
| `init`-time global maps | Yes | No | No | No |
| `context.WithValue` | Yes | No | Variable | Yes |
| `Key[T]` | Yes | Yes | Yes (per package) | Yes |

`Key[T]` dominates the table. The requirements will likely adopt it. This
analysis records that the alternatives have specific failure modes against
the forces, not that one carrier is "correct" in some absolute sense.

## Open Questions

- **Mutability after registration.** Whether the descriptor's metadata is
  mutable after registration. The current direction: set during registration
  or shortly after, then frozen. The mechanism does not enforce immutability;
  the convention does.
- **Instance-level baggage.** Whether instances carry baggage in addition to
  descriptors. Currently deferred; the same `Key[T]` mechanism extends to
  the instance if a use case arises (a loader writing per-replica metadata
  for the instantiation function to read).
- **Discovery aids.** Whether the framework offers a `application.Keys()`
  function that enumerates registered keys for debugging or doc-generation
  tools. Currently deferred; the open-set extension property makes a global
  enumeration weakly meaningful.
