// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package parser implements a parser for SDL solution units. Input is
// provided as a filename and source text; the output is an [ast.File]
// whose nodes carry materialized positions and attached comments. The
// parser is a hand-written recursive-descent parser with one token of
// lookahead over sdl/scanner's token stream, in which the scanner has
// already folded non-terminating line breaks (see the sdl/scanner package
// documentation).
//
// # Grammar
//
// The parser accepts the full grammar of the design specification:
//
//	File           = SolutionClause { ImportDecl } { Decl } .
//	SolutionClause = "solution" ident .
//	ImportDecl     = "import" ( ImportSpec | "(" { ImportSpec } ")" ) .
//	ImportSpec     = [ ident ] string .
//	Decl           = ExternDecl | VarDecl | DefaultDecl | DeployDecl | ProvisionDecl .
//	ExternDecl     = "extern" ( ExternSpec | "(" { ExternSpec } ")" ) .
//	ExternSpec     = ident TypeRef .
//	VarDecl        = "var" ( VarSpec | "(" { VarSpec } ")" ) .
//	VarSpec        = ident ":" Literal .
//	DefaultDecl    = "default" ( TypeRef | "deploy" | "provision" ) Body .
//	DeployDecl     = "deploy" ( DeploySpec | "(" { DeploySpec } ")" ) .
//	DeploySpec     = TypeRef "as" ident [ Body ] .
//	ProvisionDecl  = "provision" ( ProvisionSpec | "(" { ProvisionSpec } ")" ) .
//	ProvisionSpec  = TypeRef [ ident ] "as" ident [ Body ] .
//	Body           = "{" { BodyItem } "}" .
//	BodyItem       = Param | Section .
//	Param          = Key ":" Value .
//	Key            = ident { "." ident } .
//	Section        = ident Body .
//	Value          = Literal | Ref .
//	Literal        = string | int | duration | bool .
//	Ref            = ident [ "." ident ] .
//	TypeRef        = ident "." ident .
//
// A dotted Key spells one composite parameter name — the flattened
// dotted flag names of the A-14/D-10 story — and joins into a single
// key string; section names stay plain identifiers.
//
// Statements are NEWLINE-terminated; the last statement before a closing
// ")" or "}" or before EOF needs no line break of its own. Because
// parameter keys are grammar identifiers, a keyword (solution, deploy,
// as, true, ...) cannot be used as a key segment or section name.
//
// # What the parser enforces
//
// Beyond token shapes, the parser reports as parse errors:
//
//   - a duplicate parameter key within one body (sections have their own
//     bodies and namespaces);
//   - a var value that is not a literal (the grammar admits literals
//     only; symbol aliasing was rejected by CP-A);
//   - an import declaration appearing after a non-import declaration;
//   - a factored "default" block ("default" does not factor);
//   - an unqualified TypeRef, but only in units that declare imports.
//     The design grammar wants TypeRef always package-qualified
//     (references resolve through the unit's imports); the historical
//     mockup corpus predates import blocks and writes bare names, so a
//     unit with no imports defers the failure to the linker, where every
//     element reference is unresolvable anyway.
//
// Everything else the design marks semantic (metadata value classes, "on"
// token values, kind-word validity, symbol resolution) is deliberately
// not checked here.
//
// # Error handling and recovery
//
// All errors carry file:line:col positions and are collected in a
// [scanner.ErrorList]; [ParseFile] returns the sorted list as its error
// value. After a statement-level error the parser resynchronizes by
// skipping to the next NEWLINE at the nesting depth where the error
// occurred (never escaping the enclosing block), so one malformed
// statement costs at most that statement.
//
// Comments attach to statements as described by the sdl/ast package. One
// lossy corner is made lossless by displacement: if both the "(" line and
// the ")" line of a factored block carry a trailing comment, the second
// one is kept as a trailing group (the single Suffix slot is taken) and
// flows to the following statement's Before or the enclosing After slot.
package parser
