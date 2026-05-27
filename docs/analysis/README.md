# Analyses

INCOSE-style analyses that survey the problem space, name forces, and
enumerate alternatives. They precede requirements and do not commit to
specific design choices; requirements quote them.

Working names appear in brackets (e.g., `Descriptor`, `Instance`, `Runner`,
`NewFunc`). Names are not normative; the requirements lock them.

| Analysis | Scope |
| --- | --- |
| [A1. Scope: Long-Running Applications, Not Command-Line Tools](A1-scope.md) | Fixes the project's scope to long-running applications and explains how this scope shapes every subsequent analysis. |
| [A2. Architecture: Hosting Many Components in One Program](A2-architecture.md) | Surveys ecosystem inspirations (analysis framework, golangci-lint, ff), names what each contributes and lacks, and explains why a registry is necessary. |
| [A3. Parameterization: Blueprints and Instances](A3-parameterization.md) | Frames parameterization as the framework's defining need, uses Unix terminology (program / process / blueprint / instance), and develops the working three-value model. |
| [A4. Component Contract: Single-Method Interface with Capability Extensions](A4-component-contract.md) | Surveys interface shapes and frames capability interfaces as the growth mechanism for both third-party loaders and the unified loader itself. |
| [A5. Termination Signaling](A5-termination.md) | Separates the trigger (push, on context) from the confirmation (pull, return value), and surveys carriers for each. |
| [A6. Metadata on Components](A6-metadata.md) | Surveys mechanisms for carrying typed annotations on components and scores each against open-set extension, type safety, discoverability, and testability. |
| [A7. Component Documentation](A7-documentation.md) | Surveys documentation carriers, identifies the dry-instantiation invariant that all carriers depend on, and develops the embed-and-extract pattern. |
| [A8. Ambient Services, Concrete Dependencies, and the Discipline Against Magical DI](A8-ambient-services.md) | The complex synthesis. Distinguishes per-instance ambient values, optional discrete behaviors, and concrete required dependencies; catalogues anti-patterns with code. |

## Reading Order

A1 establishes scope. A2 establishes the topology that the scope demands. A3
establishes the parameterization model the topology hosts. A4 establishes the
component contract that parameterized instances satisfy. A5 specializes one
aspect of that contract (termination). A6 and A7 are perpendicular concerns
(metadata, documentation) that ride on the descriptor introduced in A3. A8
weaves A2 through A7 into the runtime-services discipline that prevents the
framework from degenerating into magical dependency injection.

## Status

All eight analyses are currently `draft`. They will graduate to `accepted`
once the requirements that quote them are written and reviewed.
