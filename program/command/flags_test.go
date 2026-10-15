package command_test

import (
	"context"
	"flag"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

// If your package already declares configuration with flag, expose those same
// variables as descriptor inputs without translating their value types.
func ExampleFromFlags() {
	makeGreeting := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		fs := flag.NewFlagSet("greeting", flag.ContinueOnError)
		name := fs.String("name", "world", "person to greet")
		prefix := fs.String("prefix", "Hello", "greeting prefix")
		var greeting string
		run := program.RunnerFunc(func(context.Context) error {
			greeting = *prefix + ", " + *name + "!"
			return nil
		})
		return run, command.FromFlags(fs), program.OutputList{
			program.Output{Field: program.Field{Name: "greeting"}, Value: program.GetterFunc(func() string { return greeting })},
		}
	})
	run, inputs, outputs := makeGreeting.Make()
	name, ok := inputs.Lookup("name")
	if !ok {
		panic("missing name input")
	}
	source := program.GetterFunc(func() string { return "Ada" })
	if err := name.Value.Set(source.Get()); err != nil {
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

// Loaders must see the live flag surface while retaining custom value semantics
// and capabilities.
func TestCustomFlagValues(t *testing.T) {
	fs := flag.NewFlagSet("parts", flag.ContinueOnError)
	value := partsValue{"seed"}
	fs.Var(&value, "part", "part to append")
	declaration := program.Field{Name: "part", Help: "private part", Sensitive: true}
	inputs := command.FromFlags(fs, declaration)
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
