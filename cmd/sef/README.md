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

## Author a bundle

A `solution.Bundle` is a directory with one root `go.mod`, an optional `go.sum`, and solution artifacts or supporting files.
Nested module manifests and symbolic links are rejected.
The directory is experimental and has no bundle version identifier yet.
Individual artifacts retain their string schema version.

```text
service/
    go.mod
    go.sum
    service.sef
```

Use `go mod init`, `go mod edit`, and normal manifest editing to declare the solution module, toolkit requirement, catalogue requirements, and replacements.
The solution module should have its own directory rather than reuse an application source module.
Every generated host requires `github.com/modern-engineering/prototype` at a declared version.
An unreleased checkout can use a placeholder requirement and a local replacement during development.

`sef mod tidy` gathers descriptor references across the bundle and prepares requirements and checksums through a generated Go package.
Ordinary `go mod tidy` cannot see references inside solution artifacts.
Dependency editing is separate from building: production builds require a complete context and do not fill in missing versions from the installed command.

```sh
go run ./cmd/sef mod tidy -dev internal/seftool/testdata/bundle
go run ./cmd/sef inspect -inputs -dev internal/seftool/testdata/bundle
```

This fixture uses a local toolkit replacement, so it requires `-dev`.
Versioned production bundles use published requirements and need no development flag.
Module editing retains authoring comments; exported module snapshots carry interpreted directives.

A directory or archive input takes its module context from the bundle root.
Use `-artifact path/to/service.sef` when the bundle contains multiple artifacts.
A single artifact file is standalone and never borrows a nearby module manifest.
Exporting a bundle member captures its root module context in the standalone artifact:

```sh
go run ./cmd/sef convert -to json -o /tmp/service.json.sef internal/seftool/testdata/bundle
go run ./cmd/sef convert -to gob -o /tmp/service.gob.sef /tmp/service.json.sef
```

The CUE frontend remains unchanged and can supply source artifacts inside a bundle.
CUE output does not yet carry the new module context; export standalone snapshots through JSON, gob, or text.

## Pack and unpack bundles

```sh
go run ./cmd/sef pack -o /tmp/service.tar internal/seftool/testdata/bundle
go run ./cmd/sef pack -o /tmp/service.tar.gz internal/seftool/testdata/bundle
go run ./cmd/sef inspect /tmp/service.tar.gz
go run ./cmd/sef unpack -outdir /tmp/service-bundle /tmp/service.tar.gz
```

The suffix chooses tar or gzip-compressed tar.
Both preserve artifact and supporting-file bytes and include the authoritative module context.
Packing does not compile descriptors or turn local replacements into portable source snapshots.
Development builds require an unpacked directory so local replacement paths have an explicit base.
Archive and unpack destinations must be new paths.

## Generate and build descriptor hosts

```sh
go run ./cmd/sef gen -main internal/seftool/testdata/bundle
go run ./cmd/sef check -linked -dev internal/seftool/testdata/bundle
go run ./cmd/sef build -dev -outdir /tmp/service-build -o /tmp/service.host internal/seftool/testdata/bundle
go run ./cmd/sef convert -to gob internal/seftool/testdata/bundle | /tmp/service.host inspect -inputs
```

Linked commands generate a small Go main that registers referenced descriptors and compile it in an isolated module.
The build consumes the artifact's module context, including requirements, replacements and checksums.
Production ignores caller workspaces and uses readonly module resolution.
Missing-package discovery cannot silently introduce another module or select a newer version outside that initial module graph.
Module downloads use the normal Go proxy and checksum configuration.
The independent module follows ordinary Go internal-package visibility rules.

`build` compiles without starting the host and requires a new binary output path.
`-outdir DIR` retains generated source, module files and binary in a new or empty directory, including failed build work for diagnosis.
Other temporary build directories are removed automatically.
A retained host reads gob artifacts from stdin and serves `check` or `inspect [-inputs]`.
It can inspect another artifact using the descriptors compiled into that host.

Compilation does not execute catalogue code.
Starting a host runs imported package initializers; `check` and ordinary inspection do not call `Make`.
Only `inspect -inputs` constructs invocations, and no command sets inputs, reads output values or runs programs.

## Use local modules during development

`-dev` enables local replacement directories from the authored manifest and local source substitutions from the project workspace or caller module.
Workspace discovery follows the project directory; `GOWORK=off` disables workspace discovery even in development mode.
Only already-required modules can receive automatic local substitutions.
Development never supplies missing requirement versions.
Source artifacts and project manifests remain untouched during builds.
`-dev=false` is equivalent to omitting the flag.

Replacements belong in `go.mod`; there are no per-command `-replace` or `-toolkit` overrides.
The former `-work` flag is replaced by `-outdir DIR`.

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
