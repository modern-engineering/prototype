// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package ast declares the types used to represent syntax trees for SDL
// solution units. The node set covers the full grammar of the design
// specification so that later increments only grow the parser, never the
// tree.
//
// # Node model
//
// All nodes implement [Node], which exposes only the position of the
// node's first token; node kinds are distinguished by type switches.
// Sealed marker interfaces group the union types of the grammar: [Decl]
// for top-level declarations, [BodyItem] for the contents of a { } body,
// and [Value] for parameter values.
//
// # Comment model
//
// Comments follow the golang.org/x/mod modfile model rather than
// go/ast's: comments are attached where formatting needs them instead of
// being re-interleaved by position. Every statement-shaped node embeds
// [Comments] with two slots:
//
//	Before  the full-line comment groups above the statement
//	Suffix  the trailing // comment on the statement's final line
//
// A [CommentGroup] is a run of adjacent full-line comments; a blank line
// splits groups but does not detach them from the following statement:
// all groups above a statement land in its Before slot, in order.
//
// Container nodes carry the comments no statement claims: a factored
// (parenthesized) declaration stores the groups sitting before its
// closing ')' in After, a [Body] stores the groups before its closing '}'
// in After, and [File].After holds the groups trailing the last statement
// of the unit.
//
// For a single-spec declaration such as "deploy ff.Ping as Ping1 { ... }"
// the declaration line IS the spec, so Before and Suffix attach to the
// spec node and the declaration's own slots stay empty; for a factored
// declaration the keyword line's comments attach to the declaration and
// each spec carries its own. [SolutionClause] and [DefaultDecl] have no
// spec, so they carry their comments directly.
package ast
