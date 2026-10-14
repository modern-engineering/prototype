// Package link resolves structural intent into a graph of named descriptors and
// logical bindings. Checking and linking never instantiate or execute programs.
package link

import (
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
)

// Solution contains resolved occurrences and indexed logical relationships.
// Its structural records are snapshots; descriptor values retain their original
// concrete types and ordinary Go sharing semantics. It contains no invocations.
type Solution struct {
	modules  []solution.Module
	nodes    []*Node
	byName   map[string]*Node
	bindings []*Binding
}

// Modules returns authored requirements, not proof of the selected Go versions.
func (s *Solution) Modules() iter.Seq[solution.Module] { return slices.Values(s.modules) }

// Nodes returns occurrences in authored order, not execution order.
func (s *Solution) Nodes() iter.Seq[*Node] { return slices.Values(s.nodes) }

// Lookup finds an occurrence by its exact authored name.
func (s *Solution) Lookup(name string) (*Node, bool) {
	node, ok := s.byName[name]
	return node, ok
}

// Bindings returns every assignment in authored order, including repeated ones.
func (s *Solution) Bindings() iter.Seq[*Binding] { return slices.Values(s.bindings) }

// Node is a distinct named occurrence of a descriptor. Two occurrences may refer
// to the same descriptor without denoting the same eventual program invocation.
type Node struct {
	name         string
	symbol       solution.Symbol
	descriptor   program.Descriptor
	bindings     []*Binding
	byInput      map[string][]*Binding
	byOutput     map[string][]*Binding
	dependencies []*Node
}

// Name returns the occurrence's authored name.
func (n *Node) Name() string { return n.name }

// Symbol returns the address used to look up the descriptor.
func (n *Node) Symbol() solution.Symbol { return n.symbol }

// Descriptor returns the original descriptor, retaining its concrete type.
func (n *Node) Descriptor() program.Descriptor { return n.descriptor }

// Bindings returns this occurrence's assignments in authored order across all
// inputs. Repeated assignments are retained for the eventual setter to interpret.
func (n *Node) Bindings() iter.Seq[*Binding] { return slices.Values(n.bindings) }

// BindingsFor returns assignments to the named input in authored order. An empty
// sequence means no assignments; it does not say whether the descriptor has that
// input. Field existence is checked only after instantiation.
func (n *Node) BindingsFor(input string) iter.Seq[*Binding] { return slices.Values(n.byInput[input]) }

// Consumers returns assignments referring to the named output in authored order.
// Repeated assignments remain distinct. This describes authored references, not
// proof that the descriptor declares that output or has produced its value.
func (n *Node) Consumers(output string) iter.Seq[*Binding] { return slices.Values(n.byOutput[output]) }

// Dependencies returns distinct direct producers in first-binding order. It
// includes authored self-dependencies and makes no scheduling or cycle decision.
func (n *Node) Dependencies() iter.Seq[*Node] { return slices.Values(n.dependencies) }

// Binding relates an occurrence's named input to a literal or output reference.
// It carries no runtime handles and does not transfer a value.
type Binding struct {
	node  *Node
	input string
	value Value
}

// Node returns the receiving occurrence.
func (b *Binding) Node() *Node { return b.node }

// Input returns the authored input name, without claiming the field exists.
func (b *Binding) Input() string { return b.input }

// Value returns the authored literal or resolved occurrence reference.
func (b *Binding) Value() Value { return b.value }

// Value is a Literal or Reference in the same linked solution.
type Value interface {
	value()
}

// Literal is an authored string, including an explicit empty string.
type Literal string

func (Literal) value() {}

// Reference points to the canonical producing occurrence. Output is an authored
// field name; linking does not establish its existence or availability.
type Reference struct {
	Node   *Node
	Output string
}

func (Reference) value() {}

// Checking and resolution errors distinguish ambiguous identities, missing
// names, and unsupported structural binding forms.
var (
	ErrDuplicateNode = errors.New("duplicate occurrence name")
	ErrMissingNode   = errors.New("missing occurrence")
	ErrMissingSymbol = errors.New("missing descriptor")
	ErrInvalidValue  = errors.New("invalid binding value")
)

// Error records where checking failed. Symbol identifies a missing descriptor;
// Input is set for a binding failure. Err describes the cause.
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

// Unwrap returns the checking cause.
func (e *Error) Unwrap() error { return e.Err }

// CheckStructure checks occurrence identity, binding targets, referenced
// occurrences, and supported value forms without requiring catalogue code.
// It returns independent findings as Errors in the same order as Check, omitting
// registry findings. Missing symbols, fields, and execution policy are outside
// this pass.
func CheckStructure(source solution.Definition) error {
	_, _, err := preflight(source, nil, false)
	return err
}

// Check vets occurrence identity, registry presence, binding targets, referenced
// occurrences, and supported value forms. Independent failures are returned as
// Errors in authored order, with occurrence failures before binding
// failures. registry must be non-nil.
//
// Check does not call Make, Set, Get, or Run. It cannot establish input/output
// field existence, requiredness, or module versions. Cycles, self references, and
// repeated assignments are valid logical relationships. A successful check says
// nothing about later edits; Resolve checks again before constructing its graph.
func Check(source solution.Definition, registry *program.Registry) error {
	_, _, err := preflight(source, registry, true)
	return err
}

// Resolve performs Check's preflight, then constructs a linked graph. Each
// occurrence's descriptor is looked up once and retained with its concrete type.
// Names use exact lookup and need not be Go identifiers. Literal and reference
// forms accept non-nil pointers as well as values.
//
// The graph preserves authored occurrence and binding order. No Make, Set, Get,
// or Run methods are called. Actual I/O declarations and handles belong to a
// later instantiation step; execution and resource ownership belong to loaders.
//
// On failure Resolve returns a nil solution and Errors, without
// constructing a partial graph. The source and registry are unchanged; later
// structural edits do not change a linked solution. registry must be non-nil.
func Resolve(source solution.Definition, registry *program.Registry) (*Solution, error) {
	descriptors, values, err := preflight(source, registry, true)
	if err != nil {
		return nil, err
	}
	result := &Solution{
		modules: slices.Clone(source.Modules),
		byName:  make(map[string]*Node, len(source.Nodes)),
	}
	for i, item := range source.Nodes {
		node := &Node{
			name: item.Name, symbol: item.Definition, descriptor: descriptors[i],
			byInput: make(map[string][]*Binding), byOutput: make(map[string][]*Binding),
		}
		result.nodes = append(result.nodes, node)
		result.byName[item.Name] = node
	}
	for i, authored := range source.Bindings {
		node := result.byName[authored.Node]
		binding := &Binding{node: node, input: authored.Input}
		switch value := values[i].(type) {
		case solution.Literal:
			binding.value = Literal(value)
		case solution.Reference:
			producer := result.byName[value.Node]
			binding.value = Reference{Node: producer, Output: value.Output}
			producer.byOutput[value.Output] = append(producer.byOutput[value.Output], binding)
			if !slices.Contains(node.dependencies, producer) {
				node.dependencies = append(node.dependencies, producer)
			}
		}
		result.bindings = append(result.bindings, binding)
		node.bindings = append(node.bindings, binding)
		node.byInput[binding.input] = append(node.byInput[binding.input], binding)
	}
	return result, nil
}

func preflight(source solution.Definition, registry *program.Registry, checkSymbols bool) ([]program.Descriptor, []solution.Value, error) {
	names := make(map[string]bool, len(source.Nodes))
	descriptors := make([]program.Descriptor, 0, len(source.Nodes))
	var diagnostics Errors
	for _, node := range source.Nodes {
		if names[node.Name] {
			diagnostics = append(diagnostics, &Error{Node: node.Name, Err: ErrDuplicateNode})
		}
		names[node.Name] = true
		if checkSymbols {
			descriptor, found := registry.Lookup(node.Definition.Package, node.Definition.Name)
			if !found {
				diagnostics = append(diagnostics, &Error{Node: node.Name, Symbol: node.Definition, Err: ErrMissingSymbol})
			}
			descriptors = append(descriptors, descriptor)
		}
	}
	values := make([]solution.Value, 0, len(source.Bindings))
	for _, binding := range source.Bindings {
		if !names[binding.Node] {
			diagnostics = append(diagnostics, &Error{Node: binding.Node, Input: binding.Input, Err: ErrMissingNode})
		}
		value, err := normalizeValue(binding.Value)
		if err != nil {
			diagnostics = append(diagnostics, &Error{Node: binding.Node, Input: binding.Input, Err: err})
		}
		if reference, ok := value.(solution.Reference); ok && !names[reference.Node] {
			diagnostics = append(diagnostics, &Error{Node: binding.Node, Input: binding.Input, Err: fmt.Errorf("%w: %q", ErrMissingNode, reference.Node)})
		}
		values = append(values, value)
	}
	if len(diagnostics) > 0 {
		return nil, nil, diagnostics
	}
	return descriptors, values, nil
}

func normalizeValue(value solution.Value) (solution.Value, error) {
	switch value := value.(type) {
	case solution.Literal, solution.Reference:
		return value, nil
	case *solution.Literal:
		if value != nil {
			return *value, nil
		}
	case *solution.Reference:
		if value != nil {
			return *value, nil
		}
	}
	return nil, ErrInvalidValue
}
