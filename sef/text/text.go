// Package text reads and writes the experimental textual SEF encoding.
//
// A file contains line-oriented directives. The sef directive supplies the schema
// version; solution and revision supply artifact identity. Repeated require,
// node, and bind directives may be grouped in parenthesized blocks.
// Module context uses module, go, and toolchain headers with module-require,
// module-exclude, module-replace, and module-sum collections. The older require
// directive retains legacy Definition.Modules data; it does not select code.
// For example:
//
//	sef "0.1"
//	solution "cleanup"
//	revision "proposal-2"
//	node (
//		"trim" "example.com/catalogue" "Trim"
//		"print" "example.com/catalogue" "Print"
//	)
//	bind (
//		"trim" "text" literal "  hello  "
//		"print" "text" reference "trim" "text"
//	)
//
// Each statement and each block delimiter occupies its own line. Arguments may
// be bare words or double-quoted Go strings; quoted strings can carry arbitrary
// bytes, whitespace, empty values, and escaped newlines. A // comment continues
// to the end of its line. Encoders always quote data arguments.
//
// Parsing checks syntax, the SEF version, and binding forms. It does not resolve
// descriptors, vet occurrence names, inspect fields, or instantiate programs.
// The grammar is experimental and makes no compatibility promise.
package text

import (
	"fmt"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/solution"
)

// Position identifies a location in an input file. Line and Column are one-based;
// Column counts Unicode code points. Offset is a zero-based byte offset.
type Position struct {
	Filename string
	Line     int
	Column   int
	Offset   int
}

// Error is a syntax diagnostic at a source location.
type Error struct {
	Position Position
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Position.Filename, e.Position.Line, e.Position.Column, e.Message)
}

// ErrorList contains recoverable syntax diagnostics in discovery order. Missing
// versions and unterminated blocks are reported after statements. Its errors are
// available through errors.As.
type ErrorList []*Error

func (e ErrorList) Error() string {
	messages := make([]string, len(e))
	for i, diagnostic := range e {
		messages[i] = diagnostic.Error()
	}
	return strings.Join(messages, "\n")
}

// Unwrap returns the individual syntax diagnostics.
func (e ErrorList) Unwrap() []error {
	result := make([]error, len(e))
	for i, diagnostic := range e {
		result[i] = diagnostic
	}
	return result
}

// File retains parsed structural information and source comments. A File returned
// with a syntax error is incomplete and cannot be formatted. Artifact still
// returns the successfully parsed declarations for inspection.
type File struct {
	artifact solution.Artifact
	lines    []line
	nodes    []Position
	bindings []Position
	errors   ErrorList
}

// Artifact returns an independent structural snapshot. Changes to the returned
// artifact do not modify the source file or the result of formatting it.
func (f *File) Artifact() solution.Artifact {
	artifact := f.artifact
	artifact.Module = artifact.Module.Clone()
	artifact.Definition.Modules = slices.Clone(artifact.Definition.Modules)
	artifact.Definition.Nodes = slices.Clone(artifact.Definition.Nodes)
	artifact.Definition.Bindings = slices.Clone(artifact.Definition.Bindings)
	return artifact
}

// NodePosition locates the node at index in Artifact().Definition.Nodes.
// It reports false when index is outside that collection.
func (f *File) NodePosition(index int) (Position, bool) {
	if index < 0 || index >= len(f.nodes) {
		return Position{}, false
	}
	return f.nodes[index], true
}

// BindingPosition locates the binding at index in Artifact().Definition.Bindings.
// It reports false when index is outside that collection.
func (f *File) BindingPosition(index int) (Position, bool) {
	if index < 0 || index >= len(f.bindings) {
		return Position{}, false
	}
	return f.bindings[index], true
}

type line struct {
	directive string
	arguments []string
	comment   string
	kind      byte // blank, statement, block opening, or block closing
	inBlock   bool
}
