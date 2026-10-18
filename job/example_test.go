package job_test

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

type greeting struct {
	Name    string
	Message string
}

func (g *greeting) Inputs() program.Inputs {
	var inputs input.List
	inputs.StringVar(&g.Name, "name", "Person to greet.")
	return &inputs
}

func (g *greeting) Outputs() program.Outputs {
	var outputs output.List
	outputs.StringVar(&g.Message, "message", "Composed greeting.")
	return &outputs
}

func (g *greeting) Run(context.Context) error {
	g.Message = "Hello, " + g.Name + "."
	return nil
}

// Your catalogue can use a type directly when its zero value is ready for input
// binding. The command still owns parsing, execution, and result formatting.
func ExampleMake() {
	descriptor := job.Descriptor{
		Name:     "greeting",
		Summary:  "Compose a greeting for a named person.",
		MakeFunc: job.Make[greeting],
	}

	runner, inputs, outputs := descriptor.Make()
	flags := flag.NewFlagSet(descriptor.Name, flag.ContinueOnError)
	if err := command.Parse(flags, inputs, []string{"-name", "Ada"}); err != nil {
		fmt.Println("configure:", err)
		return
	}
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println("run:", err)
		return
	}
	if err := command.WriteOutputs(os.Stdout, outputs); err != nil {
		fmt.Println("outputs:", err)
	}
	// Output: {"message":"Hello, Ada."}
}

// Keep defaults in your catalogue's constructor. From exposes the same
// initialized value through its runner and declared fields.
func ExampleFrom() {
	descriptor := job.Descriptor{
		Name: "greeting",
		MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
			return job.From(&greeting{Name: "World"})
		},
	}

	runner, _, outputs := descriptor.Make()
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println("run:", err)
		return
	}
	if err := command.WriteOutputs(os.Stdout, outputs); err != nil {
		fmt.Println("outputs:", err)
	}
	// Output: {"message":"Hello, World."}
}
