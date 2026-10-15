# Working with solution artifacts

Run these commands from the module root.
The `.sef` text grammar, CUE frontend and schema version `"0.1"` are experimental.
All encodings produce the same logical `solution.Artifact`.
JSON requires Unicode strings; text, gob and CUE also preserve arbitrary bytes.

## Inspect and edit without catalogue code

```sh
go run ./cmd/sef fmt internal/seftool/testdata/service.sef
go run ./cmd/sef check internal/seftool/testdata/service.sef
go run ./cmd/sef inspect internal/seftool/testdata/service.sef
go run ./cmd/sef rename -from database -to storage internal/seftool/testdata/service.sef
```

Formatting preserves source comments.
Text formatting retains statement order; CUE formatting retains expressions and definitions without evaluating them.
Renaming rewrites the occurrence and its references; it rejects ambiguous names and capture of unresolved endpoints.
Renaming and encoding conversion preserve logical content but do not retain source comments or CUE expressions.
`fmt -w` replaces a named text file only after successful parsing and formatting.
Named output replacement preserves existing permissions and rejects symlinks.

## Exchange encodings

```sh
go run ./cmd/sef convert -to json internal/seftool/testdata/service.sef > /tmp/service.json.sef
go run ./cmd/sef convert -to gob -o /tmp/service.gob.sef /tmp/service.json.sef
go run ./cmd/sef convert -to text /tmp/service.gob.sef
```

A missing filename or `-` reads stdin; `-o -` writes stdout.
Input format defaults to the filename suffix, or text for stdin.
Use `-from json` or `-from gob` when reading those encodings from stdin.
For `rename`, `-from` and `-to` instead name occurrences; its input encoding follows the filename.
`dump` writes normalized JSON, including authored literal values.
`inspect` displays relationships without printing literal contents.


## Author with CUE

```sh
go run ./cmd/sef fmt internal/seftool/testdata/service.cue.sef
go run ./cmd/sef check internal/seftool/testdata/service.cue.sef
go run ./cmd/sef convert -to json internal/seftool/testdata/service.cue.sef
```

The CUE source shares a catalogue path and region using local declarations.
Evaluation produces concrete artifact records; output references remain ordinary node/output data.
The frontend embeds its information schema and requires an explicit version string.
It accepts a single source file and CUE builtin imports, without a CUE module loader.
Both `.cue` and `.cue.sef` select CUE; use `-from cue` for stdin.
`fmt -from cue` can format incomplete constraints, while checking and conversion require concrete data.

## Generate and build descriptor hosts

```sh
go run ./cmd/sef gen -main internal/seftool/testdata/service.sef
go run ./cmd/sef check -linked -toolkit . internal/seftool/testdata/service.sef
go run ./cmd/sef inspect -inputs -toolkit . internal/seftool/testdata/service.cue.sef
go run ./cmd/sef build -toolkit . -work -o /tmp/service.host internal/seftool/testdata/service.sef
go run ./cmd/sef convert -to gob internal/seftool/testdata/service.sef | /tmp/service.host inspect -inputs
```

Linked commands generate a small Go main that registers the referenced descriptors and compile it in a private module.
The generated `go.mod` uses the artifact's module requirements, with ordinary Go minimal version selection.
The caller's module and workspace do not select the build's dependencies.
Missing-package discovery cannot silently introduce another module or select a newer version outside that initial module graph.
Module downloads use the normal Go proxy and checksum configuration.

A versioned toolkit command supplies its own module version when the artifact does not specify one.
This unreleased source checkout uses the explicit `-toolkit .` development override in the examples.
Other local overrides use repeated `-replace module=directory` flags and require declared modules.
The independent module follows ordinary Go internal-package visibility rules.

`build` compiles without starting the host and requires a new output path.
`-work` prints and retains the generated source, module files and binary, including failed build work for diagnosis.
Other temporary work is removed automatically.
A retained host reads gob artifacts from stdin and serves `check` or `inspect [-inputs]`.
It can inspect another artifact using the descriptors compiled into that host.

Compilation does not execute catalogue code.
Starting a host runs imported package initializers; `check` and ordinary inspection do not call `Make`.
Only `inspect -inputs` constructs invocations, and no command sets inputs, reads output values or runs programs.

## Use caller descriptors during development

```sh
go run ./cmd/sef inspect -inputs -dev internal/seftool/testdata/service.cue.sef
```

The default build uses the artifact's requirements and a separate module.
`-dev` opts into the current caller module's source when its descriptors are referenced.
It adds a local `replace` in the generated `go.mod`, retaining any authored version requirement.
An undeclared caller module receives a placeholder version compatible with its module path, used only with that local replacement.
Running from this toolkit's source module also supplies the local toolkit implementation.
Explicit `-toolkit` and `-replace` overrides take precedence.

Development mode does not copy unrelated caller requirements, workspace modules or replacement directives.
Dependencies of the replaced module still participate in normal Go module selection.
The source artifact and caller manifests are left untouched.
`-dev=false` is equivalent to omitting the flag.

## Supply a compiled registry

```sh
go run ./cmd/solution-demo check -linked internal/seftool/testdata/service.sef
go run ./cmd/solution-demo inspect -linked internal/seftool/testdata/service.sef
go run ./cmd/solution-demo inspect -inputs internal/seftool/testdata/service.sef
go run ./examples/loading
```

The demonstration host compiles the example catalogue into a local `program.Registry`.
Structural checking needs no registry; linked checking resolves descriptor symbols without calling `Make`.
`inspect -inputs` additionally instantiates fresh programs and validates actual fields without setting inputs, reading outputs, or running programs.
The loading example explicitly configures and runs the finite database job, then transfers its result to the backend input.
Execution order, value transfer, and resource lifetimes belong to the adopting loader.

## Generate registration source

```sh
go run ./cmd/sef gen -package cataloguehost -func Register internal/seftool/testdata/service.sef
```

Generated registration is an optional way to name compiled Go descriptors.
Registration source generation does not resolve or download modules.
The generated-host build path above selects and compiles them.

Commands return status 0 on success, 1 for invalid artifacts or operation failures, and 2 for invalid usage.
Diagnostics go to stderr; successful tool output goes to stdout or the requested file.
