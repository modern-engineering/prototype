// Command curated fetches an HTTP resource using configuration declared by an
// adopting package and an HTTP client supplied separately by its loader.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/examples/internal/companyhttp"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// Fetch binds package configuration and a computed response to fresh job state.
// Its concrete runner also accepts a client through companyhttp.ClientReceiver.
var Fetch = job.Descriptor{
	Name:    "fetch",
	Summary: "Fetch an authenticated HTTP resource",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		worker := new(fetch)
		var inputs input.List
		var outputs output.List
		worker.BindInputs(&inputs)
		outputs.Struct(worker)
		return worker, &inputs, &outputs
	},
}

type fetch struct {
	companyhttp.Endpoint
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

func main() {
	if err := command(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func command() error {
	url := flag.String("url", "http://localhost:8080", "service URL")
	token := flag.String("token", "", "bearer credential")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	return run(ctx, client, *url, *token, os.Stdout)
}

// run loads one invocation while its caller keeps the shared client alive.
func run(ctx context.Context, client *http.Client, url, token string, out io.Writer) error {
	runner, inputs, outputs := Fetch.Make()
	address, _ := inputs.Lookup("url")
	if err := address.Value.Set(url); err != nil {
		return err
	}
	credential, _ := inputs.Lookup("token")
	if err := credential.Value.Set(token); err != nil {
		return err
	}

	// This capability belongs to companyhttp, independently of program inputs.
	runner.(companyhttp.ClientReceiver).UseHTTPClient(client)
	if err := runner.Run(ctx); err != nil {
		return err
	}
	body, _ := outputs.Lookup("body")
	_, err := io.WriteString(out, body.Value.Get())
	return err
}
