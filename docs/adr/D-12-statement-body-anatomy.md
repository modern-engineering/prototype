---
status: proposed
since: 2026-07-13
refines:
  - A-09-solution-layer
refined-by:
  - D-15-discovered-stanza-schemes
---

# Structure Statement Bodies as One Compartment per Audience

## Context and Problem Statement

[A-09 Solution Layer] observes that one `deploy` statement speaks to three
audiences — the application (catalogue-typed parameters), the environment's
controller (deployment intent), and nobody in particular (opaque metadata) — and
surveys, without settling, how a grammar keeps them apart.
[D-07 Verb-First Directives] fixed the statement shape around that survey and
left body compartments as an open grammar path.

The mockups since walked the survey's grammatical-separation option: parameters
sat bare at the body root, told apart by their colon, deployment intent lived in
one anonymous `on { ... }` section, and controller-specific tuning was confined
to site-side configuration. Building toward a deployable solution ended the
deferral. The linker must route every body item somewhere; a bare root key could
be a parameter of any element, so no diagnosis stood on its own before catalogue
resolution; and the single anonymous intent section had no story for several
controllers reading one statement, while the knowledge those controllers
legitimately need stayed outside the artifact. This record settles the anatomy —
the compartments, their audiences, and how defaults fold across them — and
resolves the paths [D-07 Verb-First Directives] kept open.

## Decision Drivers

- The three audiences are a fixture; the grammar must keep them separable as
  catalogues and controller ecosystems grow independently.
- Deployment indifference is the litmus ([A-09 Solution Layer]): nothing a
  definition carries — controller-specific knowledge included — may narrow which
  archetype environments can reify it.
- Closed under catalogue growth ([D-07 Verb-First Directives]): a parameter
  registered tomorrow must never collide with the grammar or with intent fields.
- Error quality: a misspelled key should fail against something closed and
  local, not cascade behind element resolution.
- Uniformity across layers: defaults fold per compartment under the settled
  merge order (catalogue < default verb < default type < instance); an anatomy
  that diverges between instance and default bodies multiplies folding rules.

## Considered Options

- **Grammatical separation alone.** Parameters bare at the body root
  (`key: value`), sections colon-less blocks (`on`, `metadata`),
  controller-specific tuning confined to site-side configuration — the previous
  mockups' anatomy.
- **Reserved section words.** Section names reserved inside one body; catalogue
  registration rejects colliding parameter names.
- **Explicit paired blocks.** Parameters in `params { ... }` beside a
  `deployment { ... }` block; each major audience a named block.
- **One compartment per audience, intent at the root.** Closed per-verb root
  fields, a `params { ... }` block, named advisory `with <qualifier> { ... }`
  stanzas, `metadata { ... }`.

## Decision Outcome

Chosen option: **one compartment per audience, intent at the root.**

- **Root fields carry deployment intent as a closed per-verb scheme**, the way a
  Kubernetes resource carries its own few top-level keys before the nested spec.
  `deploy` admits `location`; `provision` admits nothing yet. The table belongs
  to the verb, not the element: an unknown root key is an error whether or not
  the element resolved, and a root key that names a catalogue parameter earns a
  pointed move-into-`params` error. Root values are literals or opaque profile
  tokens — intent never references solution symbols, so the compartment stays
  outside the symbol namespace and the binding DAG ([A-10 Value Binding]'s
  machinery does not reach it).
- **`params { ... }` carries the application's parameters** — the spec, typed by
  the catalogue element, with the full value rules (literals, symbol references,
  provision outputs). Bare keys at the body root are no longer catalogue
  parameters.
- **`with <qualifier> { ... }` stanzas carry controller and community schemes,
  and they are advisory** (`on` is retired). The controller audience thereby
  splits in two: the portable intent every controller interprets (root fields)
  and the schemes particular controllers understand (stanzas). A dotted
  identifier names each scheme, a statement may carry several, and a repeated
  qualifier on one body is an error. A controller that recognizes a qualifier
  applies the stanza; one that does not ignores it, and unknown stanzas ride the
  image opaquely — so a definition tuned for one archetype still reifies on all
  four, and the litmus test survives. Stanza values stay opaque tokens (literals
  and bare identifiers, never symbol references) until schemes become
  discoverable.
- **`metadata { ... }` is unchanged**: portable keys and values carried through
  reification for whatever tooling cares to read them.
- **The anatomy is uniform across verbs and layers.** `provision` bodies and
  `default` bodies (verb- and type-level) nest the very same compartments; each
  compartment folds independently under the unchanged merge order, and stanzas
  fold per qualifier.

An illustrative sketch:

```
default deploy {
    location: awsUsEast1

    with k8s.pod {
        priorityClass: standard
    }
}

provision substrate.NATS slice as natsAccount {
    params {
        cluster: natsCluster
        adminAccount: natsAdmin
    }
}

deploy ff.Ping as Ping2 {
    location: euCentral1 // overrides `default deploy`

    params {
        count: 10
        target: echoSubject
        nats: natsAccount.config // reconcile-time output, as before
    }

    // Folded with the default-deploy stanza above, this instance ships
    // k8s.pod = {priorityClass: standard, replicas: 3}.
    with k8s.pod {
        replicas: 3
    }

    metadata {
        team: "search"
    }
}
```

One reversal is deliberate. The earlier anatomy kept definitions portable by
exclusion: controller-specific tuning (a priority class, a nice level) was
banished to site-side configuration, and [A-09 Solution Layer] already named the
cost — legitimate knowledge pushed out of the artifact. Advisory stanzas resolve
that force the other way: recognized-or-ignored semantics give the knowledge a
first-class home inside the artifact without narrowing where the artifact can
go.

## Pros and Cons of the Options

### Grammatical Separation Alone

- Good, no ceremony on the most common content; the audience split costs one
  character.
- Bad, a bare root key means nothing until the element resolves, so diagnostics
  cascade and tooling reads bodies catalogue-in-hand.
- Bad, the intent section is anonymous: one `on` block cannot serve several
  controllers with different vocabularies, which is how the tuning overflow
  ended up outside the artifact.

### Reserved Section Words

- Good, flat bodies, plain reading.
- Bad, a word reserved tomorrow breaks a catalogue registered yesterday
  ([A-09 Solution Layer]'s force). The chosen anatomy dissolves the collision
  instead of arbitrating it: parameters live inside `params`, a namespace no
  future section word can reach.

### Explicit Paired Blocks

- Good, maximum explicitness; both major audiences named.
- Bad, ceremony lands on intent too: `deployment { ... }` nests one or two
  scalars for the nesting's sake, where the Kubernetes precedent puts intent at
  the top level and nests the spec.
- Bad, one `deployment` block is as anonymous as `on`; growing per-controller
  sub-blocks inside it converges on the chosen option.

### One Compartment per Audience, Intent at the Root

- Good, the audience is readable from structure alone, catalogue in hand or not.
- Good, root-field validation is element-independent: the verb's closed table
  answers for the root even when the element fails to resolve.
- Good, each controller scheme gets a named, portable, per-qualifier-mergeable
  home; `with k8s.pod` and `with k8s.workload` coexist on one statement.
- Bad, ceremony on the most common content after all: every body wraps its
  parameters in `params { ... }` — the cost [A-09 Solution Layer] predicted for
  explicit blocks, paid here for the compartment clarity.
- Bad, advisory means silently ignorable: a misspelled qualifier is an error
  nowhere until stanza schemes become discoverable (below).

### Consequences

- The image record carries one compartment per audience — parameters, deployment
  intent, extensions keyed by qualifier, metadata — and the old `on` compartment
  disappears without a decoding shim: images regenerate at prototype pace, and
  an encoding version bump waits until an image consumer outlives its producer.
- The grammar stays closed and compartment-agnostic: syntactically any section
  may carry one dotted qualifier, and which section words require, tolerate, or
  forbid one is linker policy, so a new compartment never touches the parser.
- Qualifiers are dotted identifiers (`k8s.pod`) because dots already lex and
  namespace naturally; hyphenated bare words collide with signed-number
  scanning.

## More Information

Open doors, each with its reopening trigger:

- **Naming.** `params` sits beside [A-03 Parameterization]'s "parameters"
  wording; the section words (`params`, `with`, `metadata`) stay provisional,
  like every keyword in the notation, until a requirement locks names.
- **Composite values.** Parameter and stanza values are scalars, composites
  surviving only as flattened dotted keys; a real composite form is a named gap
  that richer stanza schemes and structured provision outputs
  ([D-14 Driver-on-Type]'s door, shared with this one) both wait on. Trigger:
  the first scheme or output that will not flatten.

The door this list once held first — discovered stanza schemes, and the
citizenship classification deferred with it — is closed by
[D-15 Discovered Stanza Schemes]: schemes join the catalogue as self-naming
citizens (settling the classification against [D-09 Catalogue Citizenship]'s
boundary), stanzas whose qualifier resolves are checked pre-fold, and unresolved
qualifiers keep the opaque ride-along this record promised.

[A-03 Parameterization]: ../analyses/A-03-parameterization.md
[A-09 Solution Layer]: ../analyses/A-09-solution-layer.md
[A-10 Value Binding]: ../analyses/A-10-value-binding.md
[D-07 Verb-First Directives]: D-07-verb-directive-syntax.md
[D-09 Catalogue Citizenship]: D-09-catalogue-citizenship.md
[D-14 Driver-on-Type]: D-14-driver-on-type.md
[D-15 Discovered Stanza Schemes]: D-15-discovered-stanza-schemes.md
