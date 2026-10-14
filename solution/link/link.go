// Package link resolves structural intent through a descriptor registry.
// Resolved occurrences and bindings form a distinct representation without
// invoking their constructors or runners.
package link

import (
	"errors"
	"fmt"
	"slices"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
)

// Solution contains resolved occurrences and their logical bindings. Modules are
// copied requirements, not evidence that the registry satisfies those versions.
// Structural records are detached from the source; Definition values keep
// ordinary Go sharing semantics and are not cloned or instantiated.
type Solution struct {
	Modules  []solution.Module
	Nodes    []*Node
	Bindings []Binding
}

// Node is a distinct named occurrence of a registered definition. Two nodes may
// refer to the same definition without denoting the same program invocation.
type Node struct {
	Name       string
	Symbol     solution.Symbol
	Definition program.Descriptor
}

// Binding points to its receiving occurrence. Input names remain uninterpreted
// until a consumer understands the definition's configuration capabilities.
type Binding struct {
	Node  *Node
	Input string
	Value Value
}

// Value is a linked Literal or Reference. It describes a logical assignment,
// not a runtime getter, produced value, or transfer operation.
type Value interface {
	value()
}

// Literal supplies an authored string, including an explicit empty string.
type Literal string

func (Literal) value() {}

// Reference points to an occurrence in the same linked solution. Output remains a
// name; resolution does not establish that the field exists or is available.
type Reference struct {
	Node   *Node
	Output string
}

func (Reference) value() {}

// Resolution errors distinguish ambiguous identities, missing occurrences,
// missing runtime definitions, and unsupported structural binding forms.
var (
	ErrDuplicateNode = errors.New("duplicate occurrence name")
	ErrMissingNode   = errors.New("missing occurrence")
	ErrMissingSymbol = errors.New("missing symbol")
	ErrInvalidValue  = errors.New("invalid binding value")
)

// Error records where resolution failed. Symbol identifies a missing runtime
// definition; Input is set for a binding failure. Err describes the cause and
// may wrap one of the resolution errors.
type Error struct {
	Node   string
	Input  string
	Symbol solution.Symbol
	Err    error
}

func (e *Error) Error() string {
	location := fmt.Sprintf("occurrence %q", e.Node)
	if e.Input != "" {
		location += fmt.Sprintf(" input %q", e.Input)
	}
	if errors.Is(e.Err, ErrMissingSymbol) {
		return fmt.Sprintf("%s: %v %s.%s", location, e.Err, e.Symbol.Package, e.Symbol.Name)
	}
	return fmt.Sprintf("%s: %v", location, e.Err)
}

// Unwrap returns the resolution cause.
func (e *Error) Unwrap() error { return e.Err }

// Resolve resolves catalogue symbols and occurrence references using values.
// It copies structural records in authored order and normalizes literal and
// reference values, including their non-nil pointer forms. Registered Go values
// are retained with their concrete types and capabilities through the common
// program.Descriptor contract. values must be non-nil.
//
// Names use exact registry and occurrence lookup; they need not be Go
// identifiers. Forward, self, and cyclic references are allowed. Resolve does
// not inspect input/output fields, instantiate descriptors, transfer values,
// validate module versions, or determine execution order.
//
// On failure Resolve returns a nil solution and an *Error. It does not change the
// source or the registry. Mutating the source's structural records after a
// successful call does not change the returned solution.
func Resolve(source solution.Definition, values *program.Registry) (*Solution, error) {
	result := &Solution{
		Modules:  slices.Clone(source.Modules),
		Nodes:    make([]*Node, 0, len(source.Nodes)),
		Bindings: make([]Binding, 0, len(source.Bindings)),
	}
	nodes := make(map[string]*Node, len(source.Nodes))
	for _, item := range source.Nodes {
		if _, found := nodes[item.Name]; found {
			return nil, &Error{Node: item.Name, Err: ErrDuplicateNode}
		}
		definition, found := values.Lookup(item.Definition.Package, item.Definition.Name)
		if !found {
			return nil, &Error{Node: item.Name, Symbol: item.Definition, Err: ErrMissingSymbol}
		}
		node := &Node{Name: item.Name, Symbol: item.Definition, Definition: definition}
		result.Nodes = append(result.Nodes, node)
		nodes[item.Name] = node
	}
	for _, binding := range source.Bindings {
		node, found := nodes[binding.Node]
		if !found {
			return nil, &Error{Node: binding.Node, Input: binding.Input, Err: ErrMissingNode}
		}
		value, err := resolveValue(binding.Value, nodes)
		if err != nil {
			return nil, &Error{Node: binding.Node, Input: binding.Input, Err: err}
		}
		result.Bindings = append(result.Bindings, Binding{Node: node, Input: binding.Input, Value: value})
	}
	return result, nil
}

func resolveValue(value solution.Value, nodes map[string]*Node) (Value, error) {
	switch value := value.(type) {
	case solution.Literal:
		return Literal(value), nil
	case *solution.Literal:
		if value != nil {
			return Literal(*value), nil
		}
	case solution.Reference:
		node, found := nodes[value.Node]
		if !found {
			return nil, fmt.Errorf("%w: %q", ErrMissingNode, value.Node)
		}
		return Reference{Node: node, Output: value.Output}, nil
	case *solution.Reference:
		if value != nil {
			return resolveValue(*value, nodes)
		}
	}
	return nil, ErrInvalidValue
}
