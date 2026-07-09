// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package token defines constants representing the lexical tokens of the
// SDL solution-description language and basic operations on tokens
// (printing, predicates), together with the source-position types shared
// by the sdl/scanner, sdl/ast, and sdl/parser packages.
//
// The package mirrors go/token at a reduced scale. Positions are carried
// as materialized [Position] values (filename, line, column, offset)
// rather than compact go/token.Pos integers: one solution unit is small
// enough that the indirection buys nothing. A [File] maps byte offsets to
// positions for a single unit; there is no global file set, but the File
// API is shaped (NewFile with an explicit size, offset-based queries) so
// that a FileSet could be introduced later without reshaping callers.
package token
