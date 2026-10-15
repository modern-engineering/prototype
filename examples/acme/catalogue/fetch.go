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
		return job.From(&fetch{Endpoint: httpclient.Endpoint{URL: "http://localhost:8080"}})
	},
}

type fetch struct {
	httpclient.Endpoint
	// Your result has its own surface so output binding cannot publish client
	// configuration or credentials alongside the response.
	result struct {
		Body []byte `json:"body"`
	}
}

func (f *fetch) Inputs() program.Inputs {
	var inputs input.List
	f.BindInputs(&inputs)
	return &inputs
}

func (f *fetch) Outputs() program.Outputs {
	var outputs output.List
	outputs.JSON(&f.result, program.Field{Name: "body", Help: "response body"})
	return &outputs
}

func (f *fetch) Run(ctx context.Context) error {
	body, err := f.Get(ctx)
	if err != nil {
		return err
	}
	f.result.Body = body
	return nil
}
