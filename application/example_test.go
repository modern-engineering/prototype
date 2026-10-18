package application_test

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

// Your package can expose an existing flag-based configuration as descriptor
// inputs while the executable chooses when to start and stop the application.
func ExampleDescriptor() {
	app := application.Descriptor{
		Name:    "server",
		Summary: "Wait for shutdown at a configured address.",
		MakeFunc: func() (program.Runner, program.Inputs) {
			var address string
			flags := flag.NewFlagSet("server", flag.ContinueOnError)
			flags.StringVar(&address, "address", ":8080", "Listen address.")
			return program.RunnerFunc(func(ctx context.Context) error {
				fmt.Println("configured address:", address)
				<-ctx.Done()
				return ctx.Err()
			}), command.FromFlags(flags)
		},
	}

	var descriptor program.Descriptor = app
	runner, inputs, outputs := descriptor.Make()
	address, ok := inputs.Lookup("address")
	if !ok {
		fmt.Println("missing address input")
		return
	}
	if err := address.Value.Set(":9090"); err != nil {
		fmt.Println("configure:", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runner.Run(ctx); !errors.Is(err, context.Canceled) {
		fmt.Println("run:", err)
		return
	}
	fmt.Println("stopped")
	for output := range outputs.All() {
		fmt.Println("output:", output.Name)
	}
	// Output:
	// configured address: :9090
	// stopped
}
