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
		scheme := flags.String("scheme", "https", "service URL scheme")
		var address string

		run := program.RunnerFunc(func(context.Context) error {
			address = *scheme + "://" + *host
			return nil
		})
		outputs := program.OutputFields{
			program.OutputField{
				Field:  program.Field{Name: "url", Help: "service URL"},
				Source: program.GetterFunc(func() string { return address }),
			},
		}
		inputs := program.FromFlags(flags,
			program.Field{Name: "host", Requirement: program.Required},
			program.Field{Name: "scheme", Requirement: program.Optional},
		)
		return run, inputs, outputs
	},
}

var Backend = application.Descriptor{
	Name:    "backend",
	Summary: "Report the configured service address until stopped",
	MakeFunc: func() (program.Runner, program.Inputs) {
		var address string
		inputs := program.InputFields{
			suggestedInput{
				InputField: program.InputField{
					Field: program.Field{
						Name:        "address",
						Help:        "upstream service URL",
						Requirement: program.Required,
					},
					Target: program.SetterFunc(func(value string) error {
						address = value
						return nil
					}),
				},
				suggestions: []string{"https://api.example.test:8443"},
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

// suggestedInput adds a declaration capability independently of its Setter.
// Suggestions are display hints; the bound Setter controls value acceptance.
type suggestedInput struct {
	program.InputField
	suggestions []string
}

func (input suggestedInput) SuggestedValues() []string { return input.suggestions }

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	l := loader{trace: os.Stdout}
	address, err := l.load(Address, map[string]program.Getter{
		"host": literal("api.example.test:8443"),
	})
	if err != nil {
		return err
	}
	if err := address.runner.Run(ctx); err != nil {
		return err
	}
	url, err := address.output("url")
	if err != nil {
		return err
	}
	backend, err := l.load(Backend, map[string]program.Getter{
		"address": url,
	})
	if err != nil {
		return err
	}
	return backend.runner.Run(ctx)
}
