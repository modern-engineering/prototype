// Package catalogue is the importable set of operations your organization
// supports. Executables can reuse these descriptors with their own runtime policy.
package catalogue

import (
	"context"

	"example.com/acme/httpclient"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// Fetch binds package configuration and a computed response to fresh job state.
// Keep defaults here so CLI and non-CLI consumers receive the same configuration.
// Your executable supplies its HTTP client through UseHTTPClient before Run.
var Fetch = job.Descriptor{
	Name:    "fetch",
	Summary: "Fetch an authenticated HTTP resource",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		worker := &fetch{Endpoint: httpclient.Endpoint{URL: "http://localhost:8080"}}
		var inputs input.List
		var outputs output.List
		worker.BindInputs(&inputs)
		outputs.Struct(worker)
		return worker, &inputs, &outputs
	},
}

type fetch struct {
	httpclient.Endpoint
	Body []byte `output:"body" help:"response body"`
}

func (f *fetch) Run(ctx context.Context) error {
	body, err := f.Get(ctx)
	if err != nil {
		return err
	}
	f.Body = body
	return nil
}
