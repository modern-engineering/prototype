package playground_test

import (
	"context"
	"fmt"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/playground"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

// A bespoke host can supply its own catalogue, then run authored occurrences
// and carry a successful job's output into another job's input.
func ExampleRun() {
	greet := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		var name, message string
		var inputs input.List
		inputs.StringVar(&name, "name", "")
		var outputs output.List
		outputs.StringVar(&message, "message", "")
		return program.RunnerFunc(func(context.Context) error {
			message = "Hello, " + name + "."
			return nil
		}), &inputs, &outputs
	}}
	print := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		var message string
		var inputs input.List
		inputs.StringVar(&message, "message", "")
		return program.RunnerFunc(func(context.Context) error {
			fmt.Println(message)
			return nil
		}), &inputs, nil
	}}
	var registry program.Registry
	if err := registry.Register("example.org/catalogue", "Greet", greet); err != nil {
		fmt.Println(err)
		return
	}
	if err := registry.Register("example.org/catalogue", "Print", print); err != nil {
		fmt.Println(err)
		return
	}
	graph, err := link.Resolve(solution.Definition{
		Nodes: []solution.Node{
			{Name: "print", Definition: solution.Symbol{Package: "example.org/catalogue", Name: "Print"}},
			{Name: "greet", Definition: solution.Symbol{Package: "example.org/catalogue", Name: "Greet"}},
		},
		Bindings: []solution.Binding{
			{Node: "print", Input: "message", Value: solution.Reference{Node: "greet", Output: "message"}},
			{Node: "greet", Input: "name", Value: solution.Literal("Ada")},
		},
	}, &registry)
	if err != nil {
		fmt.Println(err)
		return
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := playground.Run(context.Background(), instance, playground.Options{}); err != nil {
		fmt.Println(err)
	}
	// Output: Hello, Ada.
}
