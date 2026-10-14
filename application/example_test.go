package application_test

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/program"
)

// A loader configures an application through the handles bound to its runner.
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
			}), program.FromFlags(flags)
		},
	}

	var factory program.Factory = app
	runner, inputs, outputs := factory.Make()
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
