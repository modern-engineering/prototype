package load_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

// Construction preserves bound capabilities, defaults, and authored assignments.
// Every occurrence and every instantiation gets its own constructor state.
func TestInstantiationPreservesHandlesWithoutTransfer(t *testing.T) {
	descriptor := new(trackedDescriptor)
	var registry program.Registry
	if err := registry.Register("local", "Shared", descriptor); err != nil {
		t.Fatal(err)
	}
	key := solution.Symbol{Package: "local", Name: "Shared"}
	definition := solution.Definition{
		Nodes: []solution.Node{{Name: "consumer", Definition: key}, {Name: "producer", Definition: key}},
		Bindings: []solution.Binding{
			{Node: "consumer", Input: "text", Value: solution.Literal("")},
			{Node: "producer", Input: "text", Value: solution.Reference{Node: "consumer", Output: "result"}},
			{Node: "consumer", Input: "text", Value: solution.Reference{Node: "producer", Output: "result"}},
			{Node: "consumer", Input: "text", Value: solution.Reference{Node: "consumer", Output: "result"}},
		},
	}
	graph, err := link.Resolve(definition, &registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.states) != 0 {
		t.Fatal("linking instantiated the descriptor")
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.states) != 2 {
		t.Fatalf("Make calls = %d, want one per occurrence", len(descriptor.states))
	}
	consumer, _ := instance.Lookup("consumer")
	producer, _ := instance.Lookup("producer")
	consumerNode, _ := graph.Lookup("consumer")
	if byNode, found := instance.Invocation(consumerNode); !found || byNode != consumer || consumer.Node() != consumerNode {
		t.Fatal("invocation lookup lost canonical linked identity")
	}
	if consumer.Runner() != descriptor.states[0] || producer.Runner() != descriptor.states[1] {
		t.Fatal("runner was detached from its constructor's I/O")
	}
	input, _ := consumer.Inputs().Lookup("text")
	if input.Value != descriptor.states[0] || input.Help != "content" || !input.Sensitive {
		t.Fatal("input lost its original handle or declaration metadata")
	}
	if _, ok := input.Value.(io.ReaderFrom); !ok {
		t.Fatal("original rich transfer capability was hidden")
	}
	if got := slices.Collect(consumer.Inputs().All()); len(got) != 2 || got[0].Name != "text" || got[1].Name != "spare" {
		t.Fatal("declaration enumeration order changed")
	}
	bindings := slices.Collect(graph.Bindings())
	for _, source := range bindings {
		binding, found := instance.Binding(source)
		if !found || binding.Source() != source {
			t.Fatal("an authored assignment was discarded")
		}
		if source.Node() == consumerNode && binding.Input().Value != descriptor.states[0] {
			t.Fatal("repeated assignment did not retain the same receiving handle")
		}
	}
	literal, _ := instance.Binding(bindings[0])
	if text, ok := literal.Literal(); !ok || text != "" {
		t.Fatal("literal empty string lost presence")
	}
	if _, ok := literal.Output(); ok {
		t.Fatal("literal binding reported a producer")
	}
	reference, _ := instance.Binding(bindings[2])
	if output, ok := reference.Output(); !ok || output.Value != descriptor.states[1] {
		t.Fatal("reference did not retain the producer's original getter")
	}
	if _, ok := reference.Literal(); ok {
		t.Fatal("output reference reported a literal")
	}
	for _, state := range descriptor.states {
		if state.text != "default" || state.sets != 0 || state.gets != 0 || state.runs != 0 {
			t.Fatal("instantiation transferred values, executed code, or changed defaults")
		}
	}

	// Index snapshots cannot be changed by mutating the original collection.
	descriptor.inputs[0][0].Name = "changed"
	if field, found := consumer.Inputs().Lookup("text"); !found || field.Name != "text" {
		t.Fatal("original declaration mutation changed the indexed surface")
	}
	if err := input.Value.Set("configured"); err != nil {
		t.Fatal(err)
	}
	if descriptor.states[0].text != "configured" || descriptor.states[1].text != "default" {
		t.Fatal("occurrences share constructor state")
	}
	again, err := load.Instantiate(graph)
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := again.Lookup("consumer")
	if len(descriptor.states) != 4 || fresh.Runner() == consumer.Runner() || descriptor.states[2].text != "default" {
		t.Fatal("separate instantiations reused state")
	}
	foreignGraph, err := link.Resolve(definition, &registry)
	if err != nil {
		t.Fatal(err)
	}
	foreign, _ := foreignGraph.Lookup("consumer")
	if _, found := instance.Invocation(foreign); found {
		t.Fatal("a foreign graph's occurrence matched by name")
	}
	for foreignBinding := range foreignGraph.Bindings() {
		if _, found := instance.Binding(foreignBinding); found {
			t.Fatal("a foreign graph's binding matched")
		}
	}
}

// Declaration faults and missing endpoints are independent findings with enough
// structured identity for a caller to present them without parsing error text.
func TestInstantiationCollectsDeclarationAndBindingFindings(t *testing.T) {
	calls := 0
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		calls++
		value := new(trackedState)
		input := program.Input{Field: program.Field{Name: "text"}, Value: value}
		output := program.Output{Field: program.Field{Name: "result"}, Value: value}
		return nil, program.InputList{input, input, {Field: program.Field{Name: "nil"}}},
			program.OutputList{output, output, {Field: program.Field{Name: "nil"}}}
	})
	var registry program.Registry
	if err := registry.Register("local", "Broken", descriptor); err != nil {
		t.Fatal(err)
	}
	definition := solution.Definition{
		Nodes:    []solution.Node{{Name: "broken", Definition: solution.Symbol{Package: "local", Name: "Broken"}}},
		Bindings: []solution.Binding{{Node: "broken", Input: "absent", Value: solution.Reference{Node: "broken", Output: "absent"}}},
	}
	graph, err := link.Resolve(definition, &registry)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := load.Instantiate(graph)
	findings, ok := errors.AsType[load.Errors](err)
	if instance != nil || !ok || calls != 1 {
		t.Fatalf("instance = %v, error = %v, Make calls = %d; want nil, Errors, 1", instance, err, calls)
	}
	want := []error{load.ErrNilRunner, load.ErrDuplicateInput, load.ErrNilInput, load.ErrDuplicateOutput, load.ErrNilOutput, load.ErrMissingInput, load.ErrMissingOutput}
	if len(findings) != len(want) {
		t.Fatalf("findings = %v, want %d", findings, len(want))
	}
	for i, cause := range want {
		if !errors.Is(findings[i], cause) || !errors.Is(err, cause) || findings[i].Node.Name() != "broken" {
			t.Errorf("finding %d = %v, want %v with occurrence context", i, findings[i], cause)
		}
	}
	if finding, ok := errors.AsType[*load.Error](err); !ok || finding != findings[0] {
		t.Fatal("collection did not preserve errors.As matching")
	}
	for _, finding := range findings[5:] {
		if finding.Binding == nil || finding.Binding.Input() != "absent" {
			t.Fatal("missing endpoint finding lost binding identity")
		}
	}
}

// Nil surfaces mean no fields. Typed nil handles can have valid methods and are
// governed by their ordinary Go contracts, rather than reflection-based policy.
func TestEmptySurfacesAndTypedNilHandles(t *testing.T) {
	var registry program.Registry
	var value *nilValue
	for name, descriptor := range map[string]program.Descriptor{
		"Empty": program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) { return value, nil, nil }),
		"TypedNil": program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
			return value, program.InputList{{Field: program.Field{Name: "input"}, Value: value}}, program.OutputList{{Field: program.Field{Name: "output"}, Value: value}}
		}),
	} {
		if err := registry.Register("local", name, descriptor); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := link.Resolve(solution.Definition{Nodes: []solution.Node{
		{Name: "empty", Definition: solution.Symbol{Package: "local", Name: "Empty"}},
		{Name: "typed", Definition: solution.Symbol{Package: "local", Name: "TypedNil"}},
	}}, &registry)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		t.Fatal(err)
	}
	empty, _ := instance.Lookup("empty")
	if len(slices.Collect(empty.Inputs().All())) != 0 || len(slices.Collect(empty.Outputs().All())) != 0 {
		t.Fatal("nil surfaces were not empty")
	}
	typed, _ := instance.Lookup("typed")
	input, _ := typed.Inputs().Lookup("input")
	if input.Value != value || typed.Runner() != value {
		t.Fatal("typed nil values lost their concrete types")
	}
}

type trackedDescriptor struct {
	states []*trackedState
	inputs []program.InputList
}

func (d *trackedDescriptor) Make() (program.Runner, program.Inputs, program.Outputs) {
	state := &trackedState{text: "default"}
	inputs := program.InputList{
		{Field: program.Field{Name: "text", Help: "content", Sensitive: true}, Value: state},
		{Field: program.Field{Name: "spare"}, Value: state},
	}
	d.states = append(d.states, state)
	d.inputs = append(d.inputs, inputs)
	return state, inputs, program.OutputList{{Field: program.Field{Name: "result"}, Value: state}}
}

type trackedState struct {
	text             string
	sets, gets, runs int
}

func (s *trackedState) Set(text string) error     { s.sets++; s.text = text; return nil }
func (s *trackedState) Get() string               { s.gets++; return s.text }
func (s *trackedState) Run(context.Context) error { s.runs++; return nil }
func (s *trackedState) ReadFrom(r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	s.text = string(data)
	return int64(len(data)), err
}

type nilValue struct{}

func (*nilValue) Set(string) error          { return nil }
func (*nilValue) Get() string               { return "" }
func (*nilValue) Run(context.Context) error { return nil }
