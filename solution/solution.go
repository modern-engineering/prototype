// Package solution describes logical solution relationships without loading catalogue
// implementations. A Definition carries references to program descriptors, named
// occurrences of those descriptors, and authored input bindings.
//
// A catalogue is implicit in the concrete Go type of its exported values.
// Structural nodes do not classify those types or carry their I/O handles.
// A reference can be represented even when its package, symbol, or field does
// not exist. Go linkage and catalogue-aware inspection establish those facts.
//
// Definitions carry no invocation state or physical resource ownership. They are
// independent of the encoding, solution identity, and revision used to carry
// them in a solution artifact.
package solution

// Definition is structural intent. Its slices preserve authored order, not execution
// order. Retaining bindings does not define assignment cardinality or the
// effect of assigning an input more than once. Node names identify occurrences
// within the model; two nodes may name
// the same exported definition without sharing invocation state.
//
// A Definition is ordinary mutable data. Editing it does not update any linked or
// derived view produced from an earlier version.
type Definition struct {
	Modules  []Module
	Nodes    []Node
	Bindings []Binding
}

// Module is a Go module requirement for linkage. Version is a module version,
// independent of any artifact encoding or catalogue type schema version.
// Requirements participate in the Go build's module selection; they do not
// permit several versions of one module path in the same linked program.
type Module struct {
	Path    string
	Version string
}

// Symbol names a definition by package path and name. Exported Go values are
// the usual source; a runtime registry may also supply caller-defined aliases.
// A symbol's module is selected by the Go build, not by the symbol itself.
type Symbol struct {
	Package string
	Name    string
}

// Node is a named occurrence of an exported definition. The definition's Go
// type determines which catalogue it belongs to after linkage.
type Node struct {
	Name       string
	Definition Symbol
}

// Binding assigns a literal or an output reference to an occurrence's named
// input. The presence of a binding does not establish that either field exists
// or that the relationship can be realized. Absence of a binding expresses no
// assignment; a literal empty string is still an explicit assignment.
type Binding struct {
	Node  string
	Input string
	Value Value
}

// Value is one of the structural binding forms: Literal or Reference.
// These forms describe authored assignments, not runtime getters or values.
type Value interface {
	value()
}

// Literal is a string supplied directly by the solution description.
type Literal string

func (Literal) value() {}

// Reference names an output of an occurrence in the same model. It does not
// resolve the output, determine its availability, or choose a transport.
type Reference struct {
	Node   string
	Output string
}

func (Reference) value() {}
