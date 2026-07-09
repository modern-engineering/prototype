// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"io"

	"github.com/modern-engineering/prototype/application"
)

// An Element is one catalogue entry a solution unit can reference. The
// interface is sealed; this rung knows two element kinds: the
// application component packaged by [App] and the symbol type packaged
// by [Symbol].
type Element interface{ element() }

// App packages an application descriptor as a catalogue element
// registered under name.
//
// The name is the Go identifier of the exported package variable holding
// d — a Go value does not know the name of the variable that holds it,
// and generated code is the one place that sees the identifier and the
// value side by side, so it passes the binding explicitly:
//
//	solution.App("Ping", ff.Ping)
//
// SDL references resolve importName.identifier against this binding; the
// name need not match d.Name, which belongs to the application itself.
func App(name string, d *application.Descriptor) Element {
	return &appElement{name: name, desc: d}
}

// An appElement is a component: an application descriptor under the
// exported identifier its defining package gives it.
type appElement struct {
	name string
	desc *application.Descriptor
}

func (*appElement) element() {}

// A SymbolType classifies the late-bound values extern symbols carry:
// a solution unit declares "extern name pkg.Type" against a registered
// symbol type, and the deploying site binds the value at reification.
// The type is where sensitivity is declared (A-10): taint propagates
// through references, so every binding wired from a sensitive symbol
// is itself marked sensitive in the image.
type SymbolType struct {
	// Doc documents what values of this type carry.
	Doc string

	// Sensitive marks values that must not be logged or exposed.
	// Consumers redact anything the taint reaches.
	Sensitive bool
}

// Symbol packages a symbol type as a catalogue element registered
// under name.
//
// As with [App], the name is the Go identifier of the exported package
// variable holding t: a Go value cannot know the name of the variable
// that holds it, and generated code is the one place that sees the
// identifier and the value side by side. The design sketch carried a
// Name field on SymbolType instead; it is dropped so every element
// kind has the one identity rule — the exported identifier, supplied
// at registration — rather than a second, drift-prone spelling of the
// same fact.
func Symbol(name string, t *SymbolType) Element {
	return &symbolElement{name: name, typ: t}
}

// A symbolElement is a symbol type under the exported identifier its
// defining package gives it.
type symbolElement struct {
	name string
	typ  *SymbolType
}

func (*symbolElement) element() {}

// A Package registers the catalogue elements one Go package exports.
type Package struct {
	// Path is the package's Go import path; solution import specs
	// resolve against it.
	Path string

	// Name is the package's Go package name. It is not derivable from
	// Path (major-version suffixes, hyphenated repositories), so the
	// registration carries it; unaliased imports reference the package
	// by this name.
	Name string

	// Elements are the package's exported catalogue elements.
	Elements []Element
}

// A Unit is one SDL solution unit carried by content, not by path: the
// generated compiler embeds the exact sources it was generated from, so
// generation and execution can never skew. Name supplies the filename of
// diagnostic positions.
type Unit struct {
	Name   string
	Source string
}

// A CompileConfig carries one solution compilation: the units to link
// and the catalogue they compile against. The generated main fills it
// verbatim and calls [MainCompile].
//
// Command-line concerns stay outside this package: the -o and
// -generation flags land with the code generator in a later rung and
// translate into Output and Generation here. MainCompile itself parses
// no arguments.
type CompileConfig struct {
	// Solution is the solution name every unit's solution clause must
	// declare.
	Solution string

	// Dir is the solution directory the units were read from. It serves
	// diagnostics only; the compiler never reads it.
	Dir string

	// Units are the solution's units, in the producer's order (the
	// build driver feeds them sorted by filename). Records preserve
	// this order.
	Units []Unit

	// Catalogue registers the element packages the units may reference.
	// The image pins every registered package, referenced or not: the
	// registration is what this compilation was checked against.
	Catalogue []Package

	// Generation is the producer-supplied image generation; zero means
	// 1.
	Generation int64

	// Output receives the image JSON; nil means os.Stdout.
	Output io.Writer

	// Stderr receives diagnostics; nil means os.Stderr.
	Stderr io.Writer
}
