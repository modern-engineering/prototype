# Analyses

Problem-space analyses that motivate the requirements; see the authoring guide
in `docs/CLAUDE.md`.

Working names appear in brackets (e.g., `Descriptor`, `Instance`, `Runner`,
`RunFunc`); they are provisional until a requirement locks them.

| Record                             | Analysis                                                                       |
| ---------------------------------- | ------------------------------------------------------------------------------ |
| [A-01](A-01-scope.md)              | Scope: Long-Running Applications, Not Command-Line Tools                       |
| [A-02](A-02-architecture.md)       | Architecture: Hosting Many Components in One Program                           |
| [A-03](A-03-parameterization.md)   | Parameterization: Blueprints and Instances                                     |
| [A-04](A-04-component-contract.md) | Component Contract: Single-Method Interface with Capability Extensions         |
| [A-05](A-05-termination.md)        | Termination Signaling                                                          |
| [A-06](A-06-metadata.md)           | Metadata on Components                                                         |
| [A-07](A-07-documentation.md)      | Component Documentation                                                        |
| [A-08](A-08-ambient-services.md)   | Ambient Services, Concrete Dependencies, and the Discipline Against Magical DI |
| [A-09](A-09-solution-layer.md)     | Solution Layer: From Definition to Reification                                 |
| [A-10](A-10-value-binding.md)      | Value Binding: Symbols Staged Across Compile, Site, and Reconcile Time         |
| [A-11](A-11-substrate-slices.md)   | Substrate, Slices, and Attachments                                             |

## Reading Order

A-01 establishes scope. A-02 establishes the topology that the scope demands.
A-03 establishes the parameterization model the topology hosts. A-04 establishes
the component contract that parameterized instances satisfy. A-05 specializes
one aspect of that contract (termination). A-06 and A-07 are perpendicular
concerns (metadata, documentation) that ride on the descriptor introduced in
A-03. A-08 weaves A-02 through A-07 into the runtime-services discipline that
prevents the framework from degenerating into magical dependency injection. A-09
opens the solution layer above the library: what a solution is, the compiled
artifact that carries one, and the reconciliation that keeps a deployed solution
converged on its definition. A-10 develops the value-binding model that staging
demands: the image's symbol table, override governance, secrets, and the value
language. A-11 grounds the isolation container in shared substrate: owned
slices, verified attachments, and the isolation-level spectrum between them.

## Status

The analyses are `draft`; each graduates to `accepted` once the requirements
that quote it are written and reviewed.
