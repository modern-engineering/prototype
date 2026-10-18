// Command fetch shows how your organization can expose a catalogue descriptor
// as a CLI while keeping client configuration and shutdown in the executable.
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

	"example.com/acme/catalogue"
	"github.com/modern-engineering/prototype/program/command"
)

func main() {
	if err := runCommand(); err != nil && err != flag.ErrHelp {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCommand() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	return run(ctx, client, os.Args[1:], os.Stdout, os.Stderr)
}

// httpClientUser is the capability this command needs from its runner.
type httpClientUser interface {
	UseHTTPClient(*http.Client)
}

// Your command can expose new descriptor fields without repeating their names,
// defaults, or help text here. The supplied client remains owned by this caller.
func run(ctx context.Context, client *http.Client, args []string, out, diagnostics io.Writer) error {
	runner, inputs, outputs := catalogue.Fetch.Make()
	flags := flag.NewFlagSet(catalogue.Fetch.Name, flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	if err := command.Parse(flags, inputs, args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("fetch accepts flags only")
	}

	// Supply infrastructure through a capability your command consumes. The
	// catalogue does not choose your transport or its resource lifetime.
	runner.(httpClientUser).UseHTTPClient(client)
	if err := runner.Run(ctx); err != nil {
		return err
	}
	return command.WriteOutputs(out, outputs)
}
