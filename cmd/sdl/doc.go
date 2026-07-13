// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Sdl is the solution definition language toolchain.
//
// Usage:
//
//	sdl <command> [arguments]
//
// The commands are:
//
//	build       compile a solution directory into its desired-state image
//	completion  emit a shell completion script for the sdl command
//	echo        render a desired-state image as canonical SDL
//	fmt         reformat solution units in canonical form
//	highlight   emit an editor syntax definition for solution units
//	image       amend and query desired-state image files
//	lsp         serve the language server protocol for solution units
//	run         compile and run a solution in a tailored host process
//	vet         report likely mistakes in solution units
//
// Use "sdl help <command>" for more information about a command.
//
// The remainder of this comment is maintainer documentation: how a
// build is put together across packages, and the phase glossary the
// code and its docs use.
//
// # Compilation architecture
//
// sdl build is a compiler in two halves separated by a process
// boundary. The front half runs in this process; the back half is a Go
// program the front half generates, builds, and runs. The split is
// forced by the catalogue: a solution's parameters must be validated
// by the very flag.Value code that will parse them again at run time,
// so linking has to happen where the catalogue packages are live Go
// imports — inside a program compiled in the solution's own module
// context — not inside whatever sdl binary the user installed. The
// version-skew handshake follows from the same split: the compiled
// semantics come from the solution's resolved copy of this framework,
// and the CLI only warns when its own version differs.
//
// The front half lives under cmd/sdl/internal:
//
//   - [github.com/modern-engineering/prototype/cmd/sdl/internal/load]
//     reads the solution directory: unit sources, solution name,
//     imported package paths.
//   - [github.com/modern-engineering/prototype/cmd/sdl/internal/gen]
//     discovers the catalogue and generates the back half's source.
//   - [github.com/modern-engineering/prototype/cmd/sdl/internal/work]
//     detects the module context and drives the go toolchain.
//
// The back half lives in the library proper:
//
//   - [github.com/modern-engineering/prototype/solution] links the
//     embedded units against the live catalogue (MainCompile).
//   - [github.com/modern-engineering/prototype/solution/image] defines
//     the desired-state image the two halves meet at.
//
// A build runs front to back as follows; package buildcmd wires
// exactly this and nothing more.
//
// Load units. load.Dir parses every .sdl file of the directory and
// runs the checks that must fail fast before any go invocation —
// syntax, import-path shape — collecting every fault into positioned
// diagnostics. Linking semantics deliberately stay out; they belong to
// the back half.
//
// Detect module context. work.Detect classifies the directory the way
// the go CLI itself would: an active go.work selects workspace mode,
// an enclosing go.mod module mode, and neither selects none.
//
// Discover catalogue values. gen.Discover resolves the imported
// packages in that same context via go/packages and scans their type
// surfaces for the exported pointer vars whose pointee types mark
// catalogue citizens. Discovery yields identifiers and kinds only; the
// values stay in their packages for the generated program to import.
//
// Generate the program source. gen.Source emits a package main that
// embeds the unit sources verbatim and registers the discovered
// catalogue — identifier beside live value, in the one place both are
// in scope — and hands them to solution.MainCompile.
//
// Drive the toolchain. work.Run synthesizes a temporary work module
// mirroring the solution's own resolution state (or joins a
// synthesized copy of its workspace), has the go toolchain build the
// generated program there, runs it, and relays its verdict.
//
// The generated program is the back half. It parses the embedded units
// again — embedding is verbatim precisely so generation and execution
// cannot skew, and each half trusts only its own parse — links them
// against the catalogue values it imports (schemas extracted by dry
// instantiation, references resolved, defaults folded), and emits the
// desired-state image as canonical JSON on its stdout; the driver
// routes the bytes and relays the exit code process-wide.
//
// One variant bends this shape. Outside any module context the
// solution's import paths mean nothing until a module exists to
// resolve them in, so resolution must precede discovery:
// work.ResolveModuleless tidies a probe program carrying the imports
// into the work module, discovery roots there, and the generated
// compiler then replaces the probe in the same solmain.go — probe,
// then compiler. Everywhere else discovery runs first, in a module
// context that exists independently of the build.
//
// sdl run reuses the whole pipeline with a different back half:
// gen.HostSource generates a tailored host in the compiler's place — a
// program that compiles the same embedded units in memory and then
// enacts the resulting image in-process, on the runtime of
// [github.com/modern-engineering/prototype/solution/host] — and the
// driver hands it the terminal and relays its exit code instead of
// routing an emitted image. One command takes edited sources to a
// running solution, the go test mechanics: a per-invocation binary
// built around the material under the verb.
//
// The image is the phase boundary. Compilation ends when the image is
// emitted; enactment — everything that makes the image true — starts
// from the image alone and never sees the solution's source text.
// sdl run keeps the boundary in memory: the tailored host enacts the
// image its embedded compile produced, never the source text around
// it.
//
// # Phase glossary
//
// Compile names the whole of sdl build, front half and back; no single
// act below is "the compile". The whole decomposes into four acts:
//
//	GENERATE  the front half emits the back half's source
//	BUILD     the go toolchain builds the generated program
//	LINK      the back half links units against the live catalogue
//	EMIT      the back half writes the desired-state image
//
// So sdl build generates, the toolchain builds, and the generated
// program links and emits; the module-less resolution above is a step
// inside GENERATE, not a fifth act. (Package solution predates this
// glossary in calling itself the "compile back half" and its entry
// point MainCompile; that naming is a recorded door in its doc, not a
// third sense of "compile".)
//
// Enactment decomposes into two acts, implemented by the
// single-process runtime of
// [github.com/modern-engineering/prototype/solution/host] — sdl run
// drives it tailored per invocation; prebuilt platform binaries
// (examples/host) embed it against a fixed catalogue:
//
//	PROVISION  backing-service access is provisioned against substrate
//	DEPLOY     applications are deployed and begin serving
//
// PROVISION precedes DEPLOY; whether one binary or several perform
// them is the deployment environment's choice, never an assumption for
// code here to bake in. Reconcile is not a phase: it names the
// convergence discipline — observe, diff, converge, prune (D-08) — a
// controller applies while enacting.
//
// # Tooling
//
// Beside the compiler verbs, sdl carries tooling that accompanies
// solution units in editors and shells. One rule governs the layer:
// tooling that describes the language is generated from the grammar
// code — sdl/token's vocabulary and the linker's body vocabulary
// (solution.Vocabulary) — never maintained by hand, and gate tests
// require every vocabulary word in every generated output, so a
// tool's view of the language cannot drift from the compiler's.
//
// sdl highlight emits editor syntax definitions (a Vim syntax file, a
// TextMate grammar for Visual Studio Code) on exactly this rule;
// installing the output into an editor is deliberately the user's
// affair, and 'sdl help highlight' carries the wiring snippets.
//
// sdl completion extends the same rule from the language to the CLI's
// own surface: the bash and zsh scripts are generated by walking the
// registered command tree — base.Commands and each verb's flag set,
// the material dispatch itself reads — so a script names exactly the
// verbs, subcommands, and flags of the binary that wrote it. The
// emitted script is static and never calls back into sdl;
// regenerating after an upgrade is the user's affair, and 'sdl help
// completion' carries the wiring.
//
// sdl vet bends the rule from description to judgement: the
// catalogue-free source checks judge section words and top-level
// fields against solution.Vocabulary, the same value the linker
// enforces, so vet cannot disagree with the compiler about what the
// language admits. Its remaining checks shadow the linker by hand —
// package vetcmd records that drift risk and the shared-checker door
// — and a corpus sweep holds vet to accepting whatever the build
// accepts.
package main
