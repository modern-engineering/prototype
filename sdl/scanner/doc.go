// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package scanner implements a scanner for SDL source text. It takes a
// filename and a []byte as source which can then be tokenized through
// repeated calls to the Scan method.
//
// # Statement-terminating newlines
//
// SDL is a newline-terminated statement language in the go.mod mould. The
// scanner decides which line breaks matter so that the parser never has
// to: it emits a NEWLINE token for a line break only when the token most
// recently returned before that break could end a statement, namely
//
//	an identifier
//	a string, integer, or duration literal
//	the keywords true or false
//	one of the closing brackets ) or }
//
// Every other line break is folded into ordinary white space. In
// particular no NEWLINE is emitted after an opening ( or {, after a colon
// or dot or the verb keywords (so a statement may be continued on the
// next line after them), on blank lines, or on comment-only lines.
// Comments are transparent: a trailing // comment neither arms nor
// disarms the pending NEWLINE, so "count: 10 // note" still terminates.
// If the source ends without a final line break, a NEWLINE is synthesized
// before EOF under the same rule, so every statement the parser sees is
// NEWLINE-terminated.
//
// This is Go's insertSemi rule transplanted: the token classes that may
// end a statement are exactly those after which go/scanner would insert a
// semicolon, which keeps the parser free of newline special cases while
// letting authors break long statements after punctuation.
//
// # Literals
//
// Strings are Go interpreted string literals (strconv.Unquote semantics);
// raw strings do not exist in v1. Unterminated strings and malformed
// escapes are reported as positioned errors and scanning continues.
// Integers are an optional sign followed by decimal digits. A number
// followed immediately by a letter or '.' continues greedily through
// letters, digits, and dots and is returned as a DURATION lexeme; its
// validity is checked by the parser via time.ParseDuration, so "10x"
// scans as one DURATION token and fails later with a position. A
// fractional duration therefore must start with a digit (".5s" scans as
// '.' followed by "5s").
//
// Errors are reported through an [ErrorHandler]; the [ErrorList] type
// collects them in the pattern of go/scanner.
package scanner
