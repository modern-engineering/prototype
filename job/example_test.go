package job_test

import (
	"context"
	"flag"
	"fmt"

	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// A loader sets a job's inputs and reads its outputs after execution succeeds.
func ExampleDescriptor() {
	greeting := job.Descriptor{
		Name:    "greeting",
		Summary: "Compose a greeting for a named person.",
		MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
			var name, message string
			flags := flag.NewFlagSet("greeting", flag.ContinueOnError)
			flags.StringVar(&name, "name", "World", "Person to greet.")
			outputs := program.OutputFields{
				program.OutputField{
					Field:  program.Field{Name: "message", Help: "Composed greeting."},
					Source: program.GetterFunc(func() string { return message }),
				},
			}
			return program.RunnerFunc(func(context.Context) error {
				message = "Hello, " + name + "."
				return nil
			}), program.FromFlags(flags), outputs
		},
	}

	var factory program.Factory = greeting
	runner, inputs, outputs := factory.Make()
	name, ok := inputs.Lookup("name")
	if !ok {
		fmt.Println("missing name input")
		return
	}
	if err := name.Value().Set("Ada"); err != nil {
		fmt.Println("configure:", err)
		return
	}
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println("run:", err)
		return
	}
	message, ok := outputs.Lookup("message")
	if !ok {
		fmt.Println("missing message output")
		return
	}
	fmt.Println(message.Value().Get())
	// Output: Hello, Ada.
}
