package link_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
)

// A linked graph indexes authored relationships without constructing programs,
// interpreting repeated assignments, or turning cycles into a scheduling policy.
func TestLogicalGraphPreservesIdentityAndAssignmentOrder(t *testing.T) {
	var registry program.Registry
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		panic("checking or linking called Make")
	})
	key := solution.Symbol{Package: "local", Name: "shared descriptor"}
	if err := registry.Register(key.Package, key.Name, &descriptor); err != nil {
		t.Fatal(err)
	}
	literal := solution.Literal("")
	reference := solution.Reference{Node: "producer", Output: "uninspected"}
	source := solution.Definition{
		Modules: []solution.Module{{Path: "example.org/catalogue", Version: "v1.0.0"}},
		Nodes: []solution.Node{
			{Name: "consumer", Definition: key},
			{Name: "producer", Definition: key},
		},
		Bindings: []solution.Binding{
			{Node: "consumer", Input: "a", Value: &literal},
			{Node: "consumer", Input: "b", Value: &reference},
			{Node: "producer", Input: "a", Value: solution.Reference{Node: "consumer", Output: "uninspected"}},
			{Node: "consumer", Input: "a", Value: solution.Reference{Node: "producer", Output: "uninspected"}},
			{Node: "producer", Input: "b", Value: solution.Reference{Node: "producer", Output: "uninspected"}},
		},
	}
	if err := link.Check(source, &registry); err != nil {
		t.Fatal(err)
	}
	resolved, err := link.Resolve(source, &registry)
	if err != nil {
		t.Fatal(err)
	}
	nodes := slices.Collect(resolved.Nodes())
	consumer, producer := nodes[0], nodes[1]
	if consumer == producer || consumer.Name() != "consumer" || producer.Name() != "producer" {
		t.Fatal("occurrences lost their authored identity or order")
	}
	if found, ok := resolved.Lookup("consumer"); !ok || found != consumer {
		t.Fatal("name lookup did not return the canonical occurrence")
	}
	if _, found := resolved.Lookup("absent"); found {
		t.Fatal("lookup found an absent occurrence")
	}
	if consumer.Descriptor() != &descriptor || producer.Descriptor() != &descriptor || consumer.Symbol() != key {
		t.Fatal("occurrence lost its original descriptor or provenance")
	}
	bindings := slices.Collect(resolved.Bindings())
	if bindings[0].Value() != link.Literal("") || bindings[0].Node() != consumer || bindings[0].Input() != "a" {
		t.Fatal("binding lost its receiver or explicit empty literal")
	}
	for _, i := range []int{1, 3, 4} {
		if got := bindings[i].Value().(link.Reference); got.Node != producer || got.Output != "uninspected" {
			t.Fatalf("binding %d did not reference the canonical producer: %+v", i, got)
		}
	}
	if got := slices.Collect(consumer.Bindings()); !slices.Equal(got, []*link.Binding{bindings[0], bindings[1], bindings[3]}) {
		t.Fatal("per-occurrence assignments lost authored order")
	}
	if got := slices.Collect(consumer.BindingsFor("a")); !slices.Equal(got, []*link.Binding{bindings[0], bindings[3]}) {
		t.Fatal("repeated input assignments were overwritten or reordered")
	}
	if got := slices.Collect(producer.Consumers("uninspected")); !slices.Equal(got, []*link.Binding{bindings[1], bindings[3], bindings[4]}) {
		t.Fatal("output consumers lost assignment identity or authored order")
	}
	if got := slices.Collect(consumer.Dependencies()); !slices.Equal(got, []*link.Node{producer}) {
		t.Fatal("direct producers were not deduplicated")
	}
	if got := slices.Collect(producer.Dependencies()); !slices.Equal(got, []*link.Node{consumer, producer}) {
		t.Fatal("cycle or self dependency lost first-binding order")
	}
	if len(slices.Collect(consumer.BindingsFor("unbound"))) != 0 || len(slices.Collect(producer.Consumers("unused"))) != 0 {
		t.Fatal("a query without authored relationships was not empty")
	}

	// Source and returned value edits cannot corrupt a linked graph's indexes.
	source.Modules[0].Version = "v2.0.0"
	source.Nodes[0].Name = "changed"
	source.Bindings[0].Input = "changed"
	literal = "changed"
	reference.Node, reference.Output = "changed", "changed"
	copy := bindings[1].Value().(link.Reference)
	copy.Node, copy.Output = consumer, "changed"
	if got := slices.Collect(resolved.Modules()); got[0].Version != "v1.0.0" || consumer.Name() != "consumer" || bindings[0].Input() != "a" {
		t.Fatal("source mutation changed linked structural records")
	}
	if bindings[0].Value() != link.Literal("") || bindings[1].Value().(link.Reference).Node != producer {
		t.Fatal("external mutation changed linked assignments")
	}
	if found, ok := resolved.Lookup("consumer"); !ok || found != consumer {
		t.Fatal("source mutation corrupted occurrence lookup")
	}
	if _, err := link.Resolve(source, &registry); !errors.Is(err, link.ErrMissingNode) {
		t.Fatalf("Resolve trusted the earlier Check after an edit: %v", err)
	}
}

// Check and Resolve share preflight diagnostics. Neither can instantiate a
// descriptor to discover fields, and a failure exposes no partial graph.
func TestPreflightFailuresHaveContext(t *testing.T) {
	key := solution.Symbol{Package: "local", Name: "present"}
	node := solution.Node{Name: "present", Definition: key}
	var registry program.Registry
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		panic("checking or linking called Make")
	})
	if err := registry.Register(key.Package, key.Name, descriptor); err != nil {
		t.Fatal(err)
	}
	var nilLiteral *solution.Literal
	var nilReference *solution.Reference
	for _, test := range []struct {
		name   string
		source solution.Definition
		want   error
	}{
		{"missing descriptor", solution.Definition{Nodes: []solution.Node{{Name: "missing", Definition: solution.Symbol{Package: "local", Name: "absent"}}}}, link.ErrMissingSymbol},
		{"duplicate occurrence", solution.Definition{Nodes: []solution.Node{node, node}}, link.ErrDuplicateNode},
		{"missing target", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "absent", Input: "input", Value: solution.Literal("")}}}, link.ErrMissingNode},
		{"missing source", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: solution.Reference{Node: "absent", Output: "output"}}}}, link.ErrMissingNode},
		{"nil value", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input"}}}, link.ErrInvalidValue},
		{"nil literal", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: nilLiteral}}}, link.ErrInvalidValue},
		{"nil reference", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: nilReference}}}, link.ErrInvalidValue},
		{"unknown form", solution.Definition{Nodes: []solution.Node{node}, Bindings: []solution.Binding{{Node: "present", Input: "input", Value: struct{ solution.Value }{}}}}, link.ErrInvalidValue},
	} {
		if err := link.Check(test.source, &registry); !errors.Is(err, test.want) {
			t.Errorf("%s: Check error = %v, want %v", test.name, err, test.want)
		}
		resolved, err := link.Resolve(test.source, &registry)
		if resolved != nil || !errors.Is(err, test.want) {
			t.Errorf("%s: solution = %v, error = %v; want nil, %v", test.name, resolved, err, test.want)
		}
		context, ok := errors.AsType[*link.Error](err)
		if !ok || context.Node == "" {
			t.Errorf("%s: error has no occurrence context: %v", test.name, err)
		}
	}
}

// Independent diagnostics let an editor report several problems in one pass.
// A declared occurrence with unavailable code is still known structurally.
func TestPreflightCollectsIndependentFailures(t *testing.T) {
	var registry program.Registry
	definition := solution.Definition{
		Nodes: []solution.Node{{Name: "present", Definition: solution.Symbol{Package: "missing", Name: "Descriptor"}}},
		Bindings: []solution.Binding{
			{Node: "present", Input: "first", Value: solution.Reference{Node: "present", Output: "output"}},
			{Node: "absent", Input: "second", Value: nil},
		},
	}
	err := link.Check(definition, &registry)
	findings, ok := errors.AsType[link.Errors](err)
	if !ok {
		t.Fatalf("Check error = %T, want link.Errors", err)
	}
	got := findings.Unwrap()
	want := []error{link.ErrMissingSymbol, link.ErrMissingNode, link.ErrInvalidValue}
	if len(got) != len(want) {
		t.Fatalf("diagnostics = %v, want three independent failures", got)
	}
	for i, cause := range want {
		if !errors.Is(got[i], cause) {
			t.Errorf("diagnostic %d = %v, want %v", i, got[i], cause)
		}
		if _, ok := errors.AsType[*link.Error](got[i]); !ok {
			t.Errorf("diagnostic %d has no structured context", i)
		}
	}
}

// Structural tools can inspect an artifact without importing its catalogue.
func TestStructuralCheckingDoesNotRequireCatalogueCode(t *testing.T) {
	definition := solution.Definition{
		Nodes:    []solution.Node{{Name: "unavailable", Definition: solution.Symbol{Package: "unknown", Name: "Descriptor"}}},
		Bindings: []solution.Binding{{Node: "unavailable", Input: "uninspected", Value: solution.Reference{Node: "unavailable", Output: "uninspected"}}},
	}
	if err := link.CheckStructure(definition); err != nil {
		t.Fatal(err)
	}
	var registry program.Registry
	if err := link.Check(definition, &registry); !errors.Is(err, link.ErrMissingSymbol) {
		t.Fatalf("registry check error = %v, want ErrMissingSymbol", err)
	}
	definition.Bindings[0].Node = "absent"
	if err := link.CheckStructure(definition); !errors.Is(err, link.ErrMissingNode) || errors.Is(err, link.ErrMissingSymbol) {
		t.Fatalf("structural check error = %v, want only ErrMissingNode", err)
	}
}
