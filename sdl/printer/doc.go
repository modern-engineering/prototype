// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package printer renders SDL syntax trees in the one canonical form the
// toolchain emits: sdl fmt prints parsed units through it, and sdl echo
// prints units it synthesizes from a desired-state image.
//
// # Canonical form
//
// Statements print in tree order, one per line: the solution clause
// first, then the import declarations, then the remaining declarations,
// with one blank line between those three sections. Between consecutive
// declarations, and between the items inside a body or a factored
// block, the author's blank lines are collapsed: a gap of one or more
// blank lines in the source becomes exactly one, no gap stays none, and
// the lines directly after an opening brace and before a closing one
// are always snug. Bodies and factored blocks indent one tab per
// nesting level; a parameter prints as "key: value" with one space
// after the colon; instance naming prints with one space around "as".
// Factored declarations keep their factored form and single-form
// declarations stay single; the printer never regroups statements.
//
// # Lexemes
//
// Layout is normalized, lexemes are not: a literal that records the
// text the author wrote (a non-empty Raw field) reprints verbatim, so
// formatting never rewrites "1h30m" into "90m" or re-escapes a string.
// Synthesized nodes with no Raw text render canonically from the value:
// strings through strconv.Quote, integers in decimal, durations through
// time.Duration.String, and booleans as true or false. This is the
// asymmetry between sdl fmt and sdl echo: fmt prints parsed trees and
// preserves the author's lexemes, echo prints values recovered from an
// image, which stores no lexemes to preserve.
//
// # Comments
//
// Comments print from the slots the parser fills ([ast.Comments]):
// Before groups on their own lines directly above their statement,
// blank-separated when the source separated them; the Suffix comment at
// the end of the statement's line, one space before the "//" — on the
// opening line when the author wrote it beside the "(" or "{", on the
// closing line otherwise; After groups inside their block, above the
// closing ")" or "}"; and [ast.File].After groups at the end of the
// unit.
//
// # Synthesized trees
//
// Blank-line collapsing reads source positions, which synthesized nodes
// do not carry. Where positions are unknown the printer separates
// top-level declarations with one blank line and keeps block items
// snug, so a generated unit reads like an authored one; a multi-spec
// declaration without a recorded "(" position still prints factored,
// the only form that can hold it.
package printer
