// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package solution is the compile back half of the sdl toolchain: the
// part of solution compilation that must run where the catalogue
// packages are live Go imports rather than source text.
//
// # Generate, compile, run
//
// The sdl build command turns a solution directory into a small
// generated package main that embeds the solution units verbatim and
// registers the imported catalogue packages, then compiles that
// program in the solution's own module context and runs it. The
// generated main is this package's intended caller: it fills a
// [CompileConfig] with the embedded [Unit] sources and the [Package]
// registrations — each element packaged by [App] under the exported
// identifier its defining package gives it — and hands control to
// [MainCompile]. Running the compiler inside a program that imports
// the catalogue is the point: parameters are validated by the very
// flag.Value code that will parse them again at run time.
//
// # Linking
//
// [MainCompile] carries the compilation through its phases: parse
// every unit, collecting all syntax errors before giving up; link the
// units' headers (the shared solution clause and the union import
// table); collect every declared name — instances, vars, externs —
// into the solution's one flat namespace, so references resolve across
// units and forward; check each deploy statement — resolve its element
// through the imports into the catalogue, extract the element's
// parameter schema, and bind its parameters, literal values through
// the element's own flag surface and symbol references against the
// namespace, with sensitivity tainting bindings wired from sensitive
// symbol types; and emit the desired-state image of package
// [github.com/modern-engineering/prototype/solution/image] as
// canonical JSON, pinning the schema of every registered element
// alongside the symbol table and the deployment records.
//
// Schema extraction leans on the dry-instantiation invariant of
// package application: Make constructs a fresh service whose complete
// flag surface exists before Run — the property
// [application.CheckDescriptor] verifies for catalogue authors — so
// the compiler can instantiate elements freely (recover-guarded) and
// never runs anything.
//
// # Exit codes
//
// MainCompile returns the process exit code rather than exiting, and
// the sdl driver relays it process-wide:
//
//	0  the image was emitted
//	1  solution diagnostics: faults in the solution's own text,
//	   reported as positioned file:line:col lines on stderr
//	2  everything else: a malformed config, a broken catalogue
//	   registration, or a panicking element constructor
package solution
