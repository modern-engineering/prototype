package input_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
)

// Constructor defaults and computed outputs live in ordinary Go variables.
// The loader sees handles bound to those same variables.
func Example() {
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		var greeting string
		var inputs input.List
		var outputs output.List
		name := inputs.String("name", "world", "recipient")
		outputs.StringVar(&greeting, "greeting", "uppercase greeting")
		run := program.RunnerFunc(func(context.Context) error {
			greeting = "Hello, " + strings.ToUpper(*name)
			return nil
		})
		return run, &inputs, &outputs
	})

	run, inputs, outputs := descriptor.Make()
	name, _ := inputs.Lookup("name")
	if err := name.Value.Set("Ada"); err != nil {
		fmt.Println(err)
		return
	}
	if err := run.Run(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	greeting, _ := outputs.Lookup("greeting")
	fmt.Println(greeting.Value.Get())
	// Output: Hello, ADA
}
