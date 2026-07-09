---
status: superseded
since: 2026-07-09
refines:
  - A-03-parameterization
superseded-by:
  - D-11-self-contained-services
---

# Define Applications as Declarative Descriptor Values

## Context and Problem Statement

[A-03 Parameterization] fixed the three-value model — blueprint (`Descriptor`),
`Instance`, `Runner` — but called the blueprint "a registered, immutable value"
without settling what _registered_ means or how an author writes one. Two
questions were left open, and they decide the shape of every application
definition and of the **solution-management** layer that will sit above them,
describing how applications relate to one another and to non-application
dependencies:

- **What carries an application type's identity** — a name resolved through a
  global namespace at runtime, or a concrete value the program references
  directly?
- **How is a descriptor defined** — by calling a constructor that returns an
  opaque value, or by filling the exported fields of a struct?

The forcing constraint is that the solution-management layer expresses
relationships _between_ application types (one application requires another) and
to concrete dependencies. A relationship is only as good as the thing it points
at: a name is a string the compiler cannot check, while a value is an edge the
compiler can. The end goal still includes runtime selection by string ("load one
of many programs by name"), so the decision must keep that affordance without
letting a name become the identity.

## Decision Drivers

- An application type's identity should be a concrete value the program
  references directly and the compiler checks, not a name resolved at runtime.
- Relationships between application types, and to non-application dependencies,
  must be first-class compiler-checked edges, so the solution-management layer
  can be navigated and validated rather than trusted.
- The descriptor's field set will grow — documentation ([A-07 Documentation]),
  metadata ([A-06 Metadata]), relationships — and adding a field must not break
  existing definitions.
- No global mutable state, no `init()`-time registration, no blank-import side
  effects. [A-03 Parameterization] already disqualifies global mutable state,
  and [A-06 Metadata] already rejects `init()`-time global maps; identity must
  not reintroduce what metadata and parameterization refused.
- Runtime selection by string stays possible, but confined to one explicit edge
  rather than pervading identity.

## Considered Options

- **Global name registry.** A package-global table populated by a side-effecting
  `Register` call in `init()`, reached by blank import. Identity is a name or
  token; lookup is dynamic and global. The model behind `gopacket` layer types,
  `database/sql` drivers, `encoding/gob` type names, and `image.RegisterFormat`.
- **Constructor-built opaque values.** A `Register`/`New` function returns a
  `Descriptor` with unexported fields, assigned to package variables and passed
  explicitly. No global table, but the value is opaque and the constructor's
  signature is the only definition surface.
- **Declarative descriptor values.** Exported-field `Descriptor` struct
  literals. Identity is the value; collections are assembled explicitly into a
  `Set` for runtime name lookup. The model behind `go/analysis`'s `Analyzer`.

## Decision Outcome

Chosen option: **declarative descriptor values.** An application type is a
`Descriptor` value defined by filling exported fields, referenced directly by
other code, and assembled into explicit collections where runtime string
selection is actually needed.

The decision separates two roles the way the `flag` package separates
[`flag.Flag`] from [`flag.Value`]: a declarative record that carries identity
and documentation, distinct from the behavioral interface that does the work.

> The snippets below are illustrative sketches, not references to committed
> code. The playground branch spells some of these provisionally (`Program` for
> `Runner`, `Environment` for `Instance`); the spellings a requirement locks may
> differ. What the snippets fix is the _shape_, not the names.

### Descriptor and Runner, Layered Like `flag.Flag` and `flag.Value`

`Runner` is the behavioral interface — the [`flag.Value`] of this design.
Concrete application _types_ implement it; a function adapter
([A-04 Component Contract]'s `RunFunc`) lets a bare function satisfy it without
a struct.

```go
// Runner is the unit of behavior. A stateful application is a struct with
// methods; a trivial one is a function adapted by RunFunc.
type Runner interface {
    Run(ctx context.Context) error
}

type RunFunc func(ctx context.Context) error

func (f RunFunc) Run(ctx context.Context) error { return f(ctx) }
```

`Descriptor` is the declarative identity — the [`flag.Flag`] of this design. It
ties a name and documentation to the instantiation function that makes fresh
`Runner` instances. The one divergence from `flag.Flag`, which holds its `Value`
in place: a `Descriptor` holds a _maker_, because [A-03 Parameterization]
requires a fresh, independently parameterized `Runner` per replica.

```go
// Descriptor is the declarative identity of an application type. Its exported
// fields are the first-class identity; open-set metadata rides separately via
// the Key[T] mechanism from A-06, never as more exported fields.
type Descriptor struct {
    Name   string // external reference surface: CLI words, URL segments, config keys
    Doc    string // see A-07 Documentation
    DocURL string

    // New is the instantiation function from A-03: it receives the Instance,
    // declares slots on Instance.Flags, captures locals by closure, and returns
    // the Runner. Infallible by A-03's contract; binding and validation happen
    // later (the loader binds, Run validates).
    New func(inst *Instance) Runner

    // Requires names other application types this one depends on — by value, so
    // every edge is a compiler-checked pointer rather than a string to resolve.
    Requires []*Descriptor
}
```

This resolves the app-as-value versus app-as-type question that the survey left
hanging: the application _type_ is the `Runner` implementation (a struct with
methods, or a function via `RunFunc`); the `Descriptor` is a value that _names_
and _makes_ it. The descriptor does not run; it points at what runs. That is
exactly the line `flag` draws between the `Flag` record and the `Value` it
carries.

### Definition Is Filling Fields, Not Calling a Constructor

An application is defined as a package-level value. There is no `init()`, no
registry, no side-effecting `Register`. The value _is_ the identity; other code
references it directly.

```go
var HelloWorld = &application.Descriptor{
    Name: "hello-world",
    Doc:  "greets the world on an interval",
    New:  newHelloWorld,
}

func newHelloWorld(inst *application.Instance) application.Runner {
    var (
        who      string
        interval time.Duration
    )
    inst.Flags.StringVar(&who, "who", "World", "the entity to greet")
    inst.Flags.DurationVar(&interval, "interval", time.Second, "interval between greetings")
    return application.RunFunc(func(ctx context.Context) error {
        t := time.NewTicker(interval)
        defer t.Stop()
        for {
            select {
            case <-t.C:
                fmt.Println("Hello", who)
            case <-ctx.Done():
                return nil
            }
        }
    })
}
```

### Where Names Resolve: The Solution-Management `Set`

Runtime string selection survives, confined to one explicit edge. The
solution-management layer assembles concrete descriptor values into a `Set`.
This is the only place a runtime string resolves to a descriptor, and the only
place relational invariants are checked — uniqueness, acyclic `Requires`, no
dangling edges, valid-identifier names. Those invariants are _relational_: they
exist only once the whole graph is assembled, which is why no single-descriptor
constructor could enforce them, and why deferring them to the `Set` costs
nothing the constructor would have saved.

```go
apps, err := application.NewSet(HelloWorld, Goodbye, Migrate)
if err != nil {
    return err // duplicate name, cycle, or dangling edge — reported with context
}

app, ok := apps.Lookup(os.Args[1]) // string -> concrete *Descriptor, at the edge only
```

A name stays a valid Go identifier because it still surfaces in CLI words, URL
segments, and config keys. But it is a _label_ resolved at this edge, not the
identity. The identity is the value.

## Pros and Cons of the Options

### Global Name Registry

- Good, the only model that works when the type is selected from data arriving
  from outside the program at runtime that the calling code cannot know at
  compile time: on-the-wire layer bytes, a DSN driver name, a `gob` type name.
- Good, enumeration ("list everything registered") and open-world third-party
  registration come for free.
- Bad, identity is a string the compiler cannot check; a relationship expressed
  by name is a typo away from a runtime miss.
- Bad, `init()` ordering, blank-import opacity, collision panics at startup, and
  process-global mutable state hostile to parallel tests — the costs
  [A-03 Parameterization] and [A-06 Metadata] already refuse. The application
  set here is known at build time, so the registry's one real strength does not
  apply.

### Constructor-Built Opaque Values

- Good, required arguments are enforced by the signature, and the value can be
  validated eagerly.
- Bad, eager validation buys little: a package-variable initializer cannot
  handle an error, and the invariants that matter are relational, visible only
  at the `Set`.
- Bad, does not scale as the descriptor grows the fields [A-06 Metadata] and
  [A-07 Documentation] add; a positional constructor bends its signature or
  retreats to functional options, which read less like data than a literal.
- Bad, relationships degrade to names or positional pointers, and the value is
  opaque rather than declarative.

### Declarative Descriptor Values

- Good, identity is a concrete value the compiler checks;
  `Requires []*Descriptor` makes every relationship a compiler-verified edge —
  the "concrete values over names" property generalized to the relationship
  graph.
- Good, a new optional field is invisible to literals that do not set it.
  `go/analysis`'s `Analyzer` grew its field set over years without breaking a
  single `var A = &analysis.Analyzer{...}`.
- Good, no global state, no `init()`, no blank imports; definitions are pure
  data and tests construct throwaway descriptors freely.
- Bad, the type cannot force `Name` and `New` to be set, and exported fields are
  mutable after construction. Both are met by convention and by validation at
  the `Set`, the same way [A-06 Metadata] freezes metadata by convention rather
  than by the type.

### Consequences

- A descriptor is definable as a plain package-level value: no `init()`, no
  function call, no blank import. A `Register` function that returns nothing and
  must run in `init()` is rejected outright; a constructor, if offered at all,
  must return the value.
- Relationships are compiler-checked pointer edges; a wrong reference is a build
  error, not a runtime miss. The solution-management layer is navigable by
  following values, and its `Set` is the single home for runtime name resolution
  and relational validation.
- The descriptor's exported surface stays small and stable: `Name`, `Doc`,
  `DocURL`, `New`, `Requires`. Open-set metadata rides via [A-06 Metadata]'s
  `Key[T]` mechanism, not as further exported fields, so loaders extend metadata
  without widening the struct.
- Identity is the value, not the name; names remain valid Go identifiers because
  they surface in CLI, URLs, and config, but they are resolved only at the `Set`
  edge.
- Mutability is by convention: fields are set at definition and then treated as
  frozen, consistent with the stance [A-06 Metadata] already takes.

## More Information

- This decision refines [A-03 Parameterization]: it locks the blueprint as a
  directly-referenced struct-literal value and clarifies that "registered" means
  assembled into an explicit `Set`, never inserted into a process-global table.
- [A-04 Component Contract] develops the `Runner` interface and its `RunFunc`
  adapter that the layering here depends on.
- [A-08 Ambient Services] frames the non-application dependencies the
  solution-management layer also describes; concrete dependencies stay visible
  in the import graph, never resolved by name.
- External precedents: [`go/analysis` Analyzer][analysis] for the declarative
  value; [`flag.Flag`] and [`flag.Value`] for the record-versus-behavior
  layering; [`gopacket`][gopacket] layer types, [`database/sql`][sql],
  [`encoding/gob`][gob], and [`image.RegisterFormat`][image] for the name
  registry this decision declines.

[A-03 Parameterization]: ../analyses/A-03-parameterization.md
[A-04 Component Contract]: ../analyses/A-04-component-contract.md
[A-06 Metadata]: ../analyses/A-06-metadata.md
[A-07 Documentation]: ../analyses/A-07-documentation.md
[A-08 Ambient Services]: ../analyses/A-08-ambient-services.md
[`flag.Flag`]: https://pkg.go.dev/flag#Flag
[`flag.Value`]: https://pkg.go.dev/flag#Value
[analysis]: https://pkg.go.dev/golang.org/x/tools/go/analysis#Analyzer
[gopacket]: https://pkg.go.dev/github.com/google/gopacket#RegisterLayerType
[sql]: https://pkg.go.dev/database/sql#Register
[gob]: https://pkg.go.dev/encoding/gob#Register
[image]: https://pkg.go.dev/image#RegisterFormat
