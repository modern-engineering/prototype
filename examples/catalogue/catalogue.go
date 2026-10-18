// Package catalogue supplies the descriptors used by the solution examples.
// In your organization, a catalogue package can expose supported operations
// while keeping their dependencies and implementation behind ordinary Go APIs.
package catalogue

import (
	"context"
	"errors"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// Database models a finite setup operation that publishes an address.
// Replace its Run body with your provisioning code; keep acquisitions out of Make.
var Database = job.Descriptor{
	Name: "database",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		var region, address string
		var inputs input.List
		inputs.StringVar(&region, "region", "")
		var outputs output.List
		outputs.StringVar(&address, "address", "")
		return program.RunnerFunc(func(context.Context) error {
			address = "database." + region + ".example"
			return nil
		}), &inputs, &outputs
	},
}

// Backend models an application configured by Database's output.
// Your application's Run method would serve requests until its context ends.
var Backend = application.Descriptor{
	Name: "backend",
	MakeFunc: func() (program.Runner, program.Inputs) {
		var address string
		var inputs input.List
		inputs.StringVar(&address, "database", "")
		return program.RunnerFunc(func(ctx context.Context) error {
			if address == "" {
				return errors.New("database address is empty")
			}
			<-ctx.Done()
			return ctx.Err()
		}), &inputs
	},
}
