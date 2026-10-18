package link_test

import (
	"errors"
	"testing"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/symbol"
)

// Occurrence identity belongs to the authored definition, while runtime values
// retain their identity in the registry. Resolving either must not instantiate
// descriptors or turn cycles into an execution-policy decision.
func TestOccurrencesResolveWithoutInvocation(t *testing.T) {
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		panic("descriptor instantiated during linking")
	})
	key := solution.Symbol{Package: "local", Name: "shared descriptor"}
	var values symbol.Table
	if err := values.Define(key.Package, key.Name, &factory); err != nil {
		t.Fatal(err)
	}
	if err := values.Define("local", "nil", nil); err != nil {
		t.Fatal(err)
	}
	var pointer *int
	if err := values.Define("local", "typed nil", pointer); err != nil {
		t.Fatal(err)
	}
	literal := solution.Literal("")
	reference := solution.Reference{Node: "producer", Output: "uninspected"}
	source := solution.Definition{
		Modules: []solution.Module{{Path: "example.org/catalogue", Version: "v1.0.0"}},
		Nodes: []solution.Node{
			{Name: "consumer", Definition: key},
			{Name: "producer", Definition: key},
			{Name: "opaque", Definition: solution.Symbol{Package: "local", Name: "nil"}},
			{Name: "typed-nil", Definition: solution.Symbol{Package: "local", Name: "typed nil"}},
		},
		Bindings: []solution.Binding{
			{Node: "consumer", Input: "forward", Value: &reference},
			{Node: "producer", Input: "cycle", Value: solution.Reference{Node: "consumer", Output: "uninspected"}},
			{Node: "producer", Input: "self", Value: solution.Reference{Node: "producer", Output: "uninspected"}},
			{Node: "consumer", Input: "empty", Value: &literal},
		},
	}
	resolved, err := link.Resolve(source, &values)
	if err != nil {
		t.Fatal(err)
	}
	consumer, producer := resolved.Nodes[0], resolved.Nodes[1]
	if consumer == producer || consumer.Name != "consumer" || producer.Name != "producer" {
		t.Fatalf("occurrences lost identity or order: %v, %v", consumer, producer)
	}
	if consumer.Definition != &factory || producer.Definition != &factory {
		t.Fatal("linked occurrences do not retain the registered descriptor")
	}
	if resolved.Nodes[2].Definition != nil || resolved.Nodes[3].Definition == nil || resolved.Nodes[3].Definition.(*int) != nil {
		t.Fatal("registered nil values lost presence or concrete type")
	}
	if resolved.Bindings[0].Node != consumer || resolved.Bindings[0].Value.(link.Reference).Node != producer {
		t.Fatal("forward reference does not point to the canonical occurrences")
	}
	if resolved.Bindings[1].Value.(link.Reference).Node != consumer || resolved.Bindings[2].Value.(link.Reference).Node != producer {
		t.Fatal("cyclic or self reference does not point to the canonical occurrence")
	}

	// Changes to authored data do not change an already resolved solution.
	source.Modules[0].Version = "v2.0.0"
	source.Nodes[0].Name = "changed"
	source.Bindings[0].Input = "changed"
	reference.Node, reference.Output = "changed", "changed"
	literal = "changed"
	if resolved.Modules[0].Version != "v1.0.0" || consumer.Name != "consumer" || resolved.Bindings[0].Input != "forward" {
		t.Fatal("source mutation changed linked structural records")
	}
	if got := resolved.Bindings[0].Value.(link.Reference); got.Node != producer || got.Output != "uninspected" {
		t.Fatalf("source mutation changed linked reference: %+v", got)
	}
	if got := resolved.Bindings[3].Value.(link.Literal); got != "" {
		t.Fatalf("source mutation changed explicit empty literal to %q", got)
	}
}

// Resolution reports incomplete or ambiguous structure without returning a
// partially linked solution, and without confusing a missing symbol with nil.
func TestResolutionFailuresHaveContext(t *testing.T) {
	key := solution.Symbol{Package: "local", Name: "present"}
	node := solution.Node{Name: "present", Definition: key}
	var values symbol.Table
	if err := values.Define(key.Package, key.Name, nil); err != nil {
		t.Fatal(err)
	}
	var nilLiteral *solution.Literal
	var nilReference *solution.Reference
	for _, test := range []struct {
		name   string
		source solution.Definition
		want   error
	}{
		{"missing symbol", solution.Definition{Nodes: []solution.Node{{Name: "missing", Definition: solution.Symbol{Package: "local", Name: "absent"}}}}, link.ErrMissingSymbol},
		{"duplicate occurrence", solution.Definition{Nodes: []solution.Node{node, node}}, link.ErrDuplicateNode},
		{"missing target", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "absent", Input: "input", Value: solution.Literal("")}}}, link.ErrMissingNode},
		{"missing source", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: solution.Reference{Node: "absent", Output: "output"}}}}, link.ErrMissingNode},
		{"nil value", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input"}}}, link.ErrInvalidValue},
		{"nil literal", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: nilLiteral}}}, link.ErrInvalidValue},
		{"nil reference", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: nilReference}}}, link.ErrInvalidValue},
		{"unknown form", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: struct{ solution.Value }{}}}}, link.ErrInvalidValue},
	} {
		resolved, err := link.Resolve(test.source, &values)
		if resolved != nil || !errors.Is(err, test.want) {
			t.Errorf("%s: solution = %v, error = %v; want nil, %v", test.name, resolved, err, test.want)
		}
		context, ok := errors.AsType[*link.Error](err)
		if !ok || context.Node == "" {
			t.Errorf("%s: error has no occurrence context: %v", test.name, err)
		}
	}
}
