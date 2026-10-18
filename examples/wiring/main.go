// Command wiring demonstrates a finite job configuring a long-running
// application through their bound input and output handles.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/examples/internal/loader"
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
		outputs := program.OutputList{
			program.Output{
				Field: program.Field{Name: "url", Help: "service URL"},
				Value: bind.String(&address),
			},
		}
		return run, program.FromFlags(flags), outputs
	},
}

var Backend = application.Descriptor{
	Name:    "backend",
	Summary: "Report the configured service address until stopped",
	MakeFunc: func() (program.Runner, program.Inputs) {
		var address string
		inputs := program.InputList{
			program.Input{
				Field: program.Field{Name: "address", Help: "upstream service URL"},
				Value: bind.String(&address),
			},
		}

		run := program.RunnerFunc(func(ctx context.Context) error {
			if address == "" {
				return fmt.Errorf("upstream service URL is empty")
			}
			fmt.Println("backend configured:", address)
			<-ctx.Done()
			return nil
		})
		return run, inputs
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

func run(ctx context.Context) (err error) {
	l := loader.Loader{}
	address := l.Make(Address)
	defer func() { err = errors.Join(err, address.Close()) }()
	host, err := address.Input("host")
	if err != nil {
		return err
	}
	if err := host.Set("api.example.test:8443"); err != nil {
		return err
	}
	if err := address.Run(ctx); err != nil {
		return err
	}
	url, err := address.Output("url")
	if err != nil {
		return err
	}
	backend := l.Make(Backend)
	defer func() { err = errors.Join(err, backend.Close()) }()
	upstream, err := backend.Input("address")
	if err != nil {
		return err
	}
	if err := upstream.Set(url.Get()); err != nil {
		return err
	}
	return backend.Run(ctx)
}
