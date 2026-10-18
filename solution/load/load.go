// Package load instantiates linked descriptors and resolves their actual I/O
// handles. It provides construction primitives, not execution or transfer policy.
package load

import (
	"iter"
	"slices"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution/link"
)

// Instance holds fresh invocations for one linked solution. Declarations are
// snapshots; original Setter and Getter values retain their dynamic capabilities.
// Instances do not synchronize configuration, execution, or handle access.
type Instance struct {
	invocations []*Invocation
	byName      map[string]*Invocation
	byNode      map[*link.Node]*Invocation
	bindings    map[*link.Binding]Binding
}

// Invocations returns invocations in authored occurrence order.
func (i *Instance) Invocations() iter.Seq[*Invocation] { return slices.Values(i.invocations) }

// Lookup finds an invocation by its occurrence name.
func (i *Instance) Lookup(name string) (*Invocation, bool) {
	invocation, ok := i.byName[name]
	return invocation, ok
}

// Invocation finds the invocation for this exact linked occurrence. A node from
// a different linked solution does not match even if it has the same name.
func (i *Instance) Invocation(node *link.Node) (*Invocation, bool) {
	invocation, ok := i.byNode[node]
	return invocation, ok
}

// Binding resolves this exact linked assignment to its instantiated handles.
func (i *Instance) Binding(source *link.Binding) (Binding, bool) {
	binding, ok := i.bindings[source]
	return binding, ok
}

// Invocation holds the runner and I/O constructed by one call to Make.
type Invocation struct {
	node    *link.Node
	runner  program.Runner
	inputs  fields[program.Input]
	outputs fields[program.Output]
}

// Node returns the original linked occurrence and its descriptor provenance.
func (i *Invocation) Node() *link.Node { return i.node }

// Runner returns the runner bound to these input and output handles.
func (i *Invocation) Runner() program.Runner { return i.runner }

// Inputs returns the declared inputs in their original enumeration order.
// The returned surface is non-nil, including when Make returned nil inputs.
func (i *Invocation) Inputs() program.Inputs { return i.inputs }

// Outputs returns the declared outputs in their original enumeration order.
// The returned surface is non-nil, including when Make returned nil outputs.
func (i *Invocation) Outputs() program.Outputs { return i.outputs }

// Binding holds the destination and optional producer handles for a linked
// assignment. It performs no reads, writes, or conversion between those handles.
type Binding struct {
	source *link.Binding
	input  program.Input
	output program.Output
}

// Source returns the original linked binding.
func (b Binding) Source() *link.Binding { return b.source }

// Input returns the destination's original setter and declaration metadata.
func (b Binding) Input() program.Input { return b.input }

// Output returns the producer's original getter and declaration metadata for a
// reference binding. The boolean is false for a literal binding.
func (b Binding) Output() (program.Output, bool) {
	if b.source == nil {
		return program.Output{}, false
	}
	_, ok := b.source.Value().(link.Reference)
	return b.output, ok
}

// Literal returns the authored string for a literal binding. The boolean
// distinguishes a literal empty string from an output reference.
func (b Binding) Literal() (string, bool) {
	if b.source == nil {
		return "", false
	}
	literal, ok := b.source.Value().(link.Literal)
	return string(literal), ok
}

// Instantiate calls Make once per occurrence, snapshots each declaration surface,
// and validates the actual fields referenced by bindings. Independent findings
// are returned as Errors in occurrence order, followed by binding order. On any
// finding the result is nil; already constructed state is discarded.
//
// Nil I/O surfaces are empty. Duplicate field names, nil runner or value
// interfaces, and missing bound fields are errors. Typed nil values retain
// ordinary Go method semantics and are not rejected through reflection.
// Descriptors remain responsible for satisfying the pure, fresh-state Make
// contract; panics from an invalid implementation are not recovered.
//
// No Set, Get, or Run method is called, and no resources are acquired. Cycles and
// repeated assignments remain intact. Each call produces independent invocation
// state without changing the linked graph. source must be non-nil.
func Instantiate(source *link.Solution) (*Instance, error) {
	result := &Instance{
		byName: make(map[string]*Invocation), byNode: make(map[*link.Node]*Invocation),
		bindings: make(map[*link.Binding]Binding),
	}
	var findings Errors
	for node := range source.Nodes() {
		runner, inputs, outputs := node.Descriptor().Make()
		invocation := &Invocation{node: node, runner: runner}
		invocation.inputs.byName = make(map[string]program.Input)
		invocation.outputs.byName = make(map[string]program.Output)
		if runner == nil {
			findings = append(findings, &Error{Node: node, Err: ErrNilRunner})
		}
		if inputs != nil {
			for field := range inputs.All() {
				if _, found := invocation.inputs.byName[field.Name]; found {
					findings = append(findings, &Error{Node: node, Input: field.Name, Err: ErrDuplicateInput})
				} else {
					invocation.inputs.all = append(invocation.inputs.all, field)
					invocation.inputs.byName[field.Name] = field
				}
				if field.Value == nil {
					findings = append(findings, &Error{Node: node, Input: field.Name, Err: ErrNilInput})
				}
			}
		}
		if outputs != nil {
			for field := range outputs.All() {
				if _, found := invocation.outputs.byName[field.Name]; found {
					findings = append(findings, &Error{Node: node, Output: field.Name, Err: ErrDuplicateOutput})
				} else {
					invocation.outputs.all = append(invocation.outputs.all, field)
					invocation.outputs.byName[field.Name] = field
				}
				if field.Value == nil {
					findings = append(findings, &Error{Node: node, Output: field.Name, Err: ErrNilOutput})
				}
			}
		}
		result.invocations = append(result.invocations, invocation)
		result.byName[node.Name()] = invocation
		result.byNode[node] = invocation
	}
	for sourceBinding := range source.Bindings() {
		receiver := result.byNode[sourceBinding.Node()]
		input, found := receiver.inputs.Lookup(sourceBinding.Input())
		if !found {
			findings = append(findings, &Error{Node: receiver.node, Binding: sourceBinding, Input: sourceBinding.Input(), Err: ErrMissingInput})
		}
		binding := Binding{source: sourceBinding, input: input}
		if reference, ok := sourceBinding.Value().(link.Reference); ok {
			producer := result.byNode[reference.Node]
			output, found := producer.outputs.Lookup(reference.Output)
			if !found {
				findings = append(findings, &Error{Node: producer.node, Binding: sourceBinding, Output: reference.Output, Err: ErrMissingOutput})
			}
			binding.output = output
		}
		result.bindings[sourceBinding] = binding
	}
	if len(findings) > 0 {
		return nil, findings
	}
	return result, nil
}

type fields[T any] struct {
	all    []T
	byName map[string]T
}

func (f fields[T]) All() iter.Seq[T] { return slices.Values(f.all) }

func (f fields[T]) Lookup(name string) (T, bool) {
	field, ok := f.byName[name]
	return field, ok
}
