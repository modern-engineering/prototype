// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"context"
	"flag"
	"io"

	"github.com/modern-engineering/prototype/application"
)

// An Element is one catalogue entry a solution unit can reference. The
// interface is sealed over the three element kinds: the application
// component packaged by [App], the provision type packaged by
// [Provision], and the symbol type packaged by [Symbol].
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

// A ProvisionType describes one way of holding backing services: code
// that runs at provisioning time against platform-guaranteed substrate
// (A-11). Its parameter surface binds at compile time like a
// component's; its outputs exist only once the provisioned resource
// does, so records reference them symbolically and the deployment
// environment resolves them at reconcile time.
type ProvisionType struct {
	// Doc documents what provisioning this type performs.
	Doc string

	// Make is the pure factory, the mirror of a component
	// descriptor's Make: it returns a fresh Provisioner on every
	// call, declares the instance's parameter flags, and does nothing
	// else — no I/O, no side effects, no failure. The provisioner
	// value is the instance: the linker dry-instantiates one for its
	// throwaway validation surface, and at enactment the host
	// wet-binds the same flags before running the driver, so the
	// surface the compiler checked is the surface the driver reads
	// (A-14). Nil declares a type that is parameterless and
	// driverless: its statements still compile — the registration
	// stays a complete dry citizen — and enactment refuses to run
	// them.
	Make func() Provisioner

	// Outputs is the scheme of reconcile-time values instances emit.
	// The scheme belongs to the type, not the kind: a slice and an
	// attachment of the same type emit the same outputs, so downstream
	// wiring cannot tell them apart (A-11).
	Outputs []Output

	// Kinds declares which provision kinds the type registers. The
	// declaration is explicit — zero kinds is a registration error —
	// and a solution statement may omit its kind word only while the
	// type registers exactly one.
	Kinds Kinds
}

// A Provisioner is one instance of a provision type: the parameter
// surface its Make declared, plus the driver that runs wet at the
// PROVISION phase of enactment. The seam is attach-only for now —
// drivers verify and plug into resources that already exist; the
// slice lifecycle (create, mutate, destroy, prune under delete
// protection) joins the interface when a type first registers a real
// slice driver.
type Provisioner interface {
	// Flags exposes the instance's parameter surface, declared when
	// Make built the value; nil means a flagless instance. The linker
	// validates statement bodies against a throwaway instance's
	// flags, and the host parses the record's resolved bindings into
	// the same flags before the driver runs, so the driver reads its
	// parameters the way a component service reads its own.
	Flags() *flag.FlagSet

	// Attach verifies the substrate resource the instance's
	// parameters describe and plugs into it without taking ownership
	// (A-11), writing every declared output through w. It runs wet —
	// I/O and failure are its business — and only at enactment;
	// nothing on the compile path ever calls it.
	Attach(ctx context.Context, w *OutputWriter) error
}

// An Output is one reconcile-time value of a provision type's scheme.
type Output struct {
	// Name is the output's name; statements reference it as
	// instance.name.
	Name string

	// Doc documents what the output carries.
	Doc string

	// Type is the output's scalar type, one of the [OutputType]
	// constants; empty means [OutputString], the permissive default.
	// The image pins the declaration, and consumers hold values to
	// it: the linker statically where a reference site is provably
	// boolean, the binding site's own flag.Value.Set everywhere else
	// once the value exists.
	Type OutputType

	// Sensitive marks outputs that must not be logged or exposed.
	// Bindings referencing a sensitive output carry the taint (A-10).
	Sensitive bool
}

// An OutputType names the scalar type a provision output carries,
// mirroring the SDL literal kinds. Outputs stay scalar for now;
// structured outputs wait on composite values.
type OutputType string

// The output types.
const (
	OutputString   OutputType = "string"
	OutputInt      OutputType = "int"
	OutputBool     OutputType = "bool"
	OutputDuration OutputType = "duration"
)

// valid reports whether t is inside the declared vocabulary: one of
// the [OutputType] constants or the empty permissive default.
// Registration validation and the [CheckProvisionType] harness share
// this one decider, so an image can never pin a type no consumer
// knows how to hold a value to.
func (t OutputType) valid() bool {
	switch t {
	case "", OutputString, OutputInt, OutputBool, OutputDuration:
		return true
	}
	return false
}

// Kinds is the bitset of provision kinds a [ProvisionType] registers.
type Kinds uint8

// The provision kinds. A slice owns a partition carved out of shared
// substrate: its driver creates it, mutates it, and — delete protection
// satisfied — destroys it. An attachment plugs into substrate without
// taking ownership: its driver verifies existence and compatibility at
// reconcile time and is never pruned (A-11).
const (
	Slice Kinds = 1 << iota
	Attach
)

// Provision packages a provision type as a catalogue element registered
// under name.
//
// As with [App] and [Symbol], the name is the Go identifier of the
// exported package variable holding p: a Go value cannot know the name
// of the variable that holds it, and generated code is the one place
// that sees the identifier and the value side by side.
func Provision(name string, p *ProvisionType) Element {
	return &provisionElement{name: name, typ: p}
}

// A provisionElement is a provision type under the exported identifier
// its defining package gives it.
type provisionElement struct {
	name string
	typ  *ProvisionType
}

func (*provisionElement) element() {}

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
	// declare. A unit without a clause at all — an empty or
	// comment-only unit — is a positioned link diagnostic, not a
	// config fault: the unit is the author's material.
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

	// Tool is the producing tool's own version, recorded verbatim as
	// the image's sdl.version build setting; empty records nothing.
	// The sdl CLI fills it when its binary knows an ordinary module
	// version, and a programmatic producer states here what it is.
	Tool string

	// Output receives the image JSON; nil means os.Stdout.
	Output io.Writer

	// Stderr receives diagnostics and warnings; nil means os.Stderr.
	Stderr io.Writer
}
