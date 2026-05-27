---
status: draft
date: 2026-05-27
---

# A7. Component Documentation

## Context

Every component carries human-readable documentation. Loaders surface it
through help text, list views, doc subcommands, README generation, and
(eventually) IDE integration. The [R001-app-help](../reqs/R001-app-help.md)
requirement establishes that help text is extracted from the component's
package doc, displayed in two formats (a short one-line description and a
longer formatted description). This analysis surveys where documentation can
live, how it reaches the loader, and identifies the invariant that all the
options depend on.

## Survey of Carriers

### Field on the Descriptor

```go
var helloWorld = &Descriptor{
    Doc: "Hello, world.\n\nLong-form documentation that the loader formats.",
}
```

Simple. Duplicates content with the package's own godoc unless careful effort
is made to keep them synchronized. The format (title, blank line, body)
becomes a convention authors must respect. The `golang.org/x/tools/go/analysis`
framework's `Analyzer.Doc` field uses this shape.

### Runtime Extraction via `go/doc` Against the Package AST

The framework loads the package's source at runtime, walks the AST, extracts
the package doc comment. Lazy and accurate; the source is the single source
of truth. Pulls the AST parser into the binary, increases binary size,
complicates stripped binaries, requires source files at runtime. Disqualified
for distribution.

### The Analysis-Package Pattern: Embed `doc.go` at Compile Time, Extract at Init

The component's package contains a `doc.go` file with the package doc comment
in canonical form. The descriptor's package uses `//go:embed doc.go` to load
the file as a string at compile time. An init-time helper extracts the
package doc from the embedded source. The helper panics on malformed input
(hence the `Must`-prefixed naming convention, e.g., `MustExtractDoc`).

The package doc appears once (in `doc.go`), read by godoc directly and by the
framework via the extract helper. No duplication.

Illustration, modeled on the analysis framework's convention:

```go
// doc.go
// Package shadow checks for shadowed variables.
//
// A shadowed variable is one whose name is the same as a variable declared
// in an enclosing scope. ...
package shadow
```

```go
// shadow.go
//go:embed doc.go
var doc string

var Analyzer = &analysis.Analyzer{
    Name: "shadow",
    Doc:  analysisutil.MustExtractDoc(doc, "shadow"),
    Run:  run,
}
```

The `name` argument lets multiple analyzers share a `doc.go`; for this
framework, where each component is typically a separate package, the
single-package case suffices.

## Forces

Four forces, each addressed differently by the carriers above:

- **Single source of truth.** The package's godoc and the loader's help
  should never diverge.
- **Compile-time embedding.** Runtime AST parsing is disqualified by the
  binary-size and stripped-binary considerations.
- **Init-time extraction.** Lazy parsing on first help request adds latency
  without benefit; doing the work at init is cheap and deterministic.
- **Format flexibility.** The two output shapes from
  [R001-app-help](../reqs/R001-app-help.md) (one-line short, formatted long)
  imply structured extraction (title versus body, paragraph boundaries).

The embed-and-extract pattern addresses all four; the descriptor-field
pattern addresses three (single source of truth fails); the runtime-AST
pattern addresses two (compile-time embedding and binary-size fail).

## The Dry-Instantiation Invariant

Whichever carrier the framework chooses, help rendering walks the component's
slot declarations ([A3](A3-parameterization.md)) by calling the instantiation
function once. The function executes (registers flags, builds the closure),
the help renderer reads the resulting `Instance.Flags`, then discards the
runner without invoking it.

This requires the instantiation function to be:

- **Safe to call without intent to run.**
- **Side-effect-free** except for slot declaration and closure assembly.
- **I/O-free** (no logging, no network, no file reads).
- **Infallible** (no errors that depend on slot values, because no slot
  values have been bound).

The invariant binds the documentation work to the parameterization work
([A3](A3-parameterization.md)): both require the instantiation function to
be a clean declaration, separate from the runtime. The invariant is also why
the instantiation function does not receive a `context.Context`
([A8](A8-ambient-services.md)): if it did, it could reach for ambient
services (logging, stdout) during construction, breaking help rendering.

## Open Questions

- **Helper provision.** Whether the framework provides a canonical extract
  helper (importable as, say, `application/doc.MustExtract`) or relies on
  each component to populate the `Doc` field with the extracted string. The
  former centralizes the convention; the latter lets components use any
  source of documentation. Current direction: the framework provides a
  helper; the requirements name it.
- **Long-form formatting responsibility.** Whether paragraph wrap, terminal
  width, color, and other presentation concerns are the framework's
  responsibility or the loader's. R001 hints at "formatted to fit different
  output destinations"; the loader is the natural owner of output
  formatting.
- **Storage shape.** Whether the descriptor stores the entire embedded
  string (the embed source) or only the parsed title-and-body. Storing both
  preserves the raw form for future reformatting; storing only the parsed
  form saves no meaningful memory. Current direction: store the raw string
  and extract on demand, with a helper that caches the parse on first
  request.
- **Multi-component packages.** Whether multiple components can live in one
  package and share a `doc.go`. The analysis framework's `MustExtractDoc`
  takes a name argument to handle this; the framework may follow suit if a
  use case emerges.
