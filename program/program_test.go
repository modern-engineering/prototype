package program_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/program"
)

// A loader can configure flag-bound state through a getter before running and reading its output.
func Example() {
	makeGreeting := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		fs := flag.NewFlagSet("greeting", flag.ContinueOnError)
		name := fs.String("name", "world", "person to greet")
		prefix := fs.String("prefix", "Hello", "greeting prefix")
		var greeting string
		run := program.RunnerFunc(func(context.Context) error {
			greeting = *prefix + ", " + *name + "!"
			return nil
		})
		return run, program.FromFlags(fs), program.OutputList{
			program.Output{Field: program.Field{Name: "greeting"}, Value: program.GetterFunc(func() string { return greeting })},
		}
	})
	run, inputs, outputs := makeGreeting.Make()
	name, ok := inputs.Lookup("name")
	if !ok {
		panic("missing name input")
	}
	if err := program.Wire(name.Value, program.GetterFunc(func() string { return "Ada" })); err != nil {
		panic(err)
	}
	if err := run.Run(context.Background()); err != nil {
		panic(err)
	}
	greeting, ok := outputs.Lookup("greeting")
	if !ok {
		panic("missing greeting output")
	}
	fmt.Println(greeting.Value.Get())
	// Output: Hello, Ada!
}

// Loaders retain each invocation's handles, so construction must isolate their bound state.
func TestIndependentInvocations(t *testing.T) {
	makeRun := program.Adapt(func() *invocation { return &invocation{value: "default"} })
	first, firstInputs, firstOutputs := makeRun.Make()
	second, _, secondOutputs := makeRun.Make()
	input, ok := firstInputs.Lookup("value")
	if !ok {
		t.Fatal("missing value input")
	}
	if err := input.Value.Set("changed"); err != nil {
		t.Fatal(err)
	}
	if err := first.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	output, ok := firstOutputs.Lookup("value")
	if !ok || output.Value.Get() != "CHANGED" {
		t.Fatalf("first output = %v, %v; want CHANGED", output, ok)
	}
	output, ok = secondOutputs.Lookup("value")
	if !ok || output.Value.Get() != "default" {
		t.Fatalf("second output = %v, %v; want default", output, ok)
	}
	for _, runner := range []*invocation{first.(*invocation), second.(*invocation)} {
		if runner.inputsCalled != 1 || runner.outputsCalled != 1 {
			t.Errorf("surface calls = (%d, %d); want (1, 1)", runner.inputsCalled, runner.outputsCalled)
		}
	}
	_, inputs, outputs := program.Adapt(func() program.RunnerFunc {
		return func(context.Context) error { return nil }
	}).Make()
	if len(slices.Collect(inputs.All())) != 0 || len(slices.Collect(outputs.All())) != 0 {
		t.Error("runner without accessors has nonempty surfaces")
	}
	if _, ok := inputs.Lookup("value"); ok {
		t.Error("empty inputs contain value")
	}
	if _, ok := outputs.Lookup("value"); ok {
		t.Error("empty outputs contain value")
	}
}

type invocation struct {
	value                       string
	inputsCalled, outputsCalled int
}

func (r *invocation) Run(context.Context) error { r.value = strings.ToUpper(r.value); return nil }
func (r *invocation) Inputs() program.Inputs {
	r.inputsCalled++
	return program.InputList{program.Input{Field: program.Field{Name: "value"}, Value: program.SetterFunc(func(value string) error { r.value = value; return nil })}}
}
func (r *invocation) Outputs() program.Outputs {
	r.outputsCalled++
	return program.OutputList{program.Output{Field: program.Field{Name: "value"}, Value: program.GetterFunc(func() string { return r.value })}}
}

// Loaders must see the live flag surface while retaining custom value semantics
// and capabilities.
func TestCustomFlagValues(t *testing.T) {
	fs := flag.NewFlagSet("parts", flag.ContinueOnError)
	value := partsValue{"seed"}
	fs.Var(&value, "part", "part to append")
	declaration := program.Field{Name: "part", Help: "private part", Sensitive: true}
	inputs := program.FromFlags(fs, declaration)
	input, ok := inputs.Lookup("part")
	if !ok {
		t.Fatal("missing part input")
	}
	if got := input.Field; got != declaration {
		t.Errorf("declaration = %+v; want %+v", got, declaration)
	}
	getter, ok := input.Value.(flag.Getter)
	if !ok {
		t.Fatal("flag.Getter capability lost")
	}
	if got := getter.Get().([]string); !slices.Equal(got, []string{"seed"}) {
		t.Fatalf("default = %v; want [seed]", got)
	}
	if err := input.Value.Set("one"); err != nil {
		t.Fatal(err)
	}
	if err := input.Value.Set("two"); err != nil {
		t.Fatal(err)
	}
	if got := getter.Get().([]string); !slices.Equal(got, []string{"seed", "one", "two"}) {
		t.Errorf("assigned parts = %v; want [seed one two]", got)
	}
	if got := fs.Lookup("part").DefValue; got != "seed" {
		t.Errorf("flag default = %q; want seed", got)
	}

	fs.String("suffix", "end", "trailing part")
	suffix, ok := inputs.Lookup("suffix")
	if !ok || suffix.Value != fs.Lookup("suffix").Value {
		t.Fatal("flag declared after wrapping is missing or lost its original value")
	}
	if suffix.Help != "trailing part" {
		t.Errorf("flag help = %q; want trailing part", suffix.Help)
	}
	all := slices.Collect(inputs.All())
	if len(all) != 2 || all[0].Name != "part" || all[1].Name != "suffix" {
		t.Fatalf("enumerated inputs = %v; want part and suffix in lexical order", all)
	}
	yields := 0
	inputs.All()(func(program.Input) bool {
		yields++
		return false
	})
	if yields != 1 {
		t.Errorf("yields after early stop = %d; want 1", yields)
	}
}

type partsValue []string

func (v *partsValue) String() string         { return strings.Join(*v, ",") }
func (v *partsValue) Get() any               { return []string(*v) }
func (v *partsValue) Set(value string) error { *v = append(*v, value); return nil }

func TestAssignmentError(t *testing.T) {
	want := errors.New("assignment rejected")
	target := program.SetterFunc(func(value string) error {
		if value != "payload" {
			t.Errorf("assigned value = %q; want payload", value)
		}
		return want
	})
	err := program.Wire(target, program.GetterFunc(func() string { return "payload" }))
	if !errors.Is(err, want) {
		t.Errorf("Wire error = %v; want %v", err, want)
	}
}
