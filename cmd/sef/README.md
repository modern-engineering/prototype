# Working with solution artifacts

Run these commands from the module root.
The `.sef` text grammar and schema version `"0.1"` are experimental.
JSON and gob encode the same logical `solution.Artifact`; JSON requires Unicode strings, while text and gob also preserve arbitrary bytes.

## Inspect and edit without catalogue code

```sh
go run ./cmd/sef fmt internal/seftool/testdata/service.sef
go run ./cmd/sef check internal/seftool/testdata/service.sef
go run ./cmd/sef inspect internal/seftool/testdata/service.sef
go run ./cmd/sef rename -from database -to storage internal/seftool/testdata/service.sef
```

Formatting preserves source comments and statement order.
Renaming rewrites the occurrence and its references; it rejects ambiguous names and capture of unresolved endpoints.
Renaming and encoding conversion preserve logical content but do not retain source comments.
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

## Inspect compiled descriptors and actual fields

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
It does not download modules, verify the selected module versions, or execute catalogue programs.

Commands return status 0 on success, 1 for invalid artifacts or operation failures, and 2 for invalid usage.
Diagnostics go to stderr; successful tool output goes to stdout or the requested file.
