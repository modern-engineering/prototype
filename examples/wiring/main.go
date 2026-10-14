// Command wiring demonstrates a finite job configuring a long-running
// application through their bound input and output handles.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

var Address = job.Descriptor{
	Name:    "address",
	Summary: "Construct a service address from a configured host",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		flags := flag.NewFlagSet("address", flag.ContinueOnError)
		host := flags.String("host", "localhost:8080", "service host and port")
		var address string

		run := program.RunnerFunc(func(context.Context) error {
			address = "https://" + *host
			return nil
		})
		outputs := program.OutputFields{
			program.OutputField{
				Field:  program.Field{Name: "url", Help: "service URL"},
				Source: program.GetterFunc(func() string { return address }),
			},
		}
		return run, program.FromFlags(flags), outputs
	},
}

var Backend = application.Descriptor{
	Name:    "backend",
	Summary: "Report the configured service address until stopped",
	MakeFunc: func() (program.Runner, program.Inputs) {
		flags := flag.NewFlagSet("backend", flag.ContinueOnError)
		address := flags.String("address", "", "upstream service URL")

		run := program.RunnerFunc(func(ctx context.Context) error {
			if *address == "" {
				return fmt.Errorf("upstream service URL is empty")
			}
			fmt.Println("backend configured:", *address)
			<-ctx.Done()
			return nil
		})
		return run, program.FromFlags(flags)
	},
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var factory program.Factory = Address
	address, inputs, outputs := factory.Make()
	host, _ := inputs.Lookup("host")
	if err := host.Value().Set("api.example.test:8443"); err != nil {
		return err
	}
	if err := address.Run(ctx); err != nil {
		return err
	}

	factory = Backend
	backend, backendInputs, _ := factory.Make()
	url, _ := outputs.Lookup("url")
	upstream, _ := backendInputs.Lookup("address")
	if err := program.Wire(upstream.Value(), url.Value()); err != nil {
		return err
	}
	return backend.Run(ctx)
}
