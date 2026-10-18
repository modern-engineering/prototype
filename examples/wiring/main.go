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
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

var Address = job.Descriptor{
	Name:    "address",
	Summary: "Construct a service address from a configured host",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		flags := flag.NewFlagSet("address", flag.ContinueOnError)
		host := flags.String("host", "localhost:8080", "service host and port")
		scheme := flags.String("scheme", "https", "service URL scheme")
		var address string

		run := program.RunnerFunc(func(context.Context) error {
			address = *scheme + "://" + *host
			return nil
		})
		var outputs output.List
		outputs.StringVar(&address, "url", "service URL")
		return run, program.FromFlags(flags), &outputs
	},
}

var Backend = application.Descriptor{
	Name:    "backend",
	Summary: "Report the configured service address until stopped",
	MakeFunc: func() (program.Runner, program.Inputs) {
		var address string
		var inputs input.List
		inputs.StringVar(&address, "address", "upstream service URL")

		run := program.RunnerFunc(func(ctx context.Context) error {
			if address == "" {
				return fmt.Errorf("upstream service URL is empty")
			}
			fmt.Println("backend configured:", address)
			<-ctx.Done()
			return nil
		})
		return run, &inputs
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
	address, inputs, outputs := Address.Make()
	host, _ := inputs.Lookup("host")
	if err := host.Value.Set("api.example.test:8443"); err != nil {
		return err
	}
	if err := address.Run(ctx); err != nil {
		return err
	}
	url, _ := outputs.Lookup("url")
	backend, inputs, _ := Backend.Make()
	upstream, _ := inputs.Lookup("address")
	if err := upstream.Value.Set(url.Value.Get()); err != nil {
		return err
	}
	return backend.Run(ctx)
}
