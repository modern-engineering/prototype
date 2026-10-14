// Package catalogue exports descriptors from the job and application catalogues.
package catalogue

import (
	"context"
	"errors"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// Database derives an address in the configured region.
var Database = job.Descriptor{
	Name: "database",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		var region, address string
		return program.RunnerFunc(func(context.Context) error {
			address = "database." + region + ".example"
			return nil
		}), program.InputList{
			{Field: program.Field{Name: "region"}, Value: bind.String(&region)},
		}, program.OutputList{
			{Field: program.Field{Name: "address"}, Value: bind.String(&address)},
		}
	},
}

// Backend requires a database address before serving until cancellation.
var Backend = application.Descriptor{
	Name: "backend",
	MakeFunc: func() (program.Runner, program.Inputs) {
		var address string
		return program.RunnerFunc(func(ctx context.Context) error {
			if address == "" {
				return errors.New("database address is empty")
			}
			<-ctx.Done()
			return ctx.Err()
		}), program.InputList{
			{Field: program.Field{Name: "database"}, Value: bind.String(&address)},
		}
	},
}
