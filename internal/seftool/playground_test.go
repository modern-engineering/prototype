package seftool_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/internal/seftool"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// A compiled catalogue host uses the same artifact command while retaining
// control of cancellation and streams. Shutdown must join the application.
func TestCompiledPlayground(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		ready := make(chan string, 1)
		stopped := false
		var registry program.Registry
		prepare := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
			var outputs output.List
			address := outputs.String("address", "", "database endpoint", output.Sensitive)
			return program.RunnerFunc(func(context.Context) error {
				*address = "private-database.example"
				return nil
			}), nil, &outputs
		}}
		serve := application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
			var inputs input.List
			address := inputs.String("database", "", "database endpoint", input.Sensitive)
			return program.RunnerFunc(func(ctx context.Context) error {
				ready <- *address
				<-ctx.Done()
				stopped = true
				return ctx.Err()
			}), &inputs
		}}
		if err := registry.Register("private.example/catalogue", "Prepare", prepare); err != nil {
			t.Fatal(err)
		}
		if err := registry.Register("private.example/catalogue", "Serve", serve); err != nil {
			t.Fatal(err)
		}
		const source = `sef "0.1"
node (
 "database" "private.example/catalogue" "Prepare"
 "backend" "private.example/catalogue" "Serve"
)
bind "backend" "database" reference "database" "address"
`
		var output, diagnostics bytes.Buffer
		command := seftool.Command{Context: ctx, Registry: &registry, Stdin: strings.NewReader(source), Stdout: &output, Stderr: &diagnostics}
		finished := make(chan int, 1)
		go func() { finished <- command.Run([]string{"playground", "-job-timeout", "2s"}) }()
		if address := <-ready; address != "private-database.example" {
			t.Fatalf("application database = %q", address)
		}
		cancel()
		if status := <-finished; status != 1 {
			t.Fatalf("cancellation status = %d, want 1", status)
		}
		if !stopped {
			t.Fatal("command returned before application stopped")
		}
		if strings.Contains(output.String()+diagnostics.String(), "private-database.example") {
			t.Fatal("command disclosed a transferred value")
		}
	})
}

func TestPlaygroundFlagUsage(t *testing.T) {
	for _, args := range [][]string{
		{"playground", "-job-timeout", "0"},
		{"playground", "-job-timeout", "-1s"},
		{"playground", "-job-timeout", "invalid"},
	} {
		var diagnostics bytes.Buffer
		if status := (seftool.Command{Stderr: &diagnostics}).Run(args); status != 2 {
			t.Errorf("%v status = %d, want 2", args, status)
		}
	}
}
