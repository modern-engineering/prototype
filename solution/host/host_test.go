package host_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
)

// Compiled tools receive a materialized artifact independently of its source
// language. A linked check inspects logical references without constructing an
// invocation.
func ExampleCommand() {
	var registry program.Registry
	var encoded, output bytes.Buffer
	if err := sef.EncodeGob(&encoded, solution.Artifact{Version: "0.1"}); err != nil {
		fmt.Println(err)
		return
	}
	command := host.Command{Registry: &registry, Stdin: &encoded, Stdout: &output}
	status := command.Run([]string{"check"})
	fmt.Print(output.String())
	fmt.Println("status:", status)
	// Output:
	// ok
	// status: 0
}

// Only declaration inspection crosses the Make boundary, and no inspection mode
// may invoke setters, getters, or runners.
func TestInspectionDoesNotExecute(t *testing.T) {
	makes := 0
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		makes++
		return program.RunnerFunc(func(context.Context) error { panic("Run") }),
			program.InputList{{Field: program.Field{Name: "name"}, Value: program.SetterFunc(func(string) error { panic("Set") })}},
			program.OutputList{{Field: program.Field{Name: "value"}, Value: program.GetterFunc(func() string { panic("Get") })}}
	})
	var registry program.Registry
	if err := registry.Register("example.test/catalogue", "Service", descriptor); err != nil {
		t.Fatal(err)
	}
	artifact := solution.Artifact{Version: "0.1", Definition: solution.Definition{Nodes: []solution.Node{{Name: "first", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Service"}}, {Name: "second", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Service"}}}}}
	var data bytes.Buffer
	if err := sef.EncodeGob(&data, artifact); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"check"}, {"inspect"}} {
		var diagnostics bytes.Buffer
		command := host.Command{Registry: &registry, Stdin: bytes.NewReader(data.Bytes()), Stderr: &diagnostics}
		if status := command.Run(args); status != 0 {
			t.Fatalf("%v status %d: %s", args, status, &diagnostics)
		}
	}
	if makes != 0 {
		t.Fatalf("linked operations called Make %d times", makes)
	}
	command := host.Command{Registry: &registry, Stdin: bytes.NewReader(data.Bytes())}
	if status := command.Run([]string{"inspect", "-inputs"}); status != 0 {
		t.Fatalf("input inspection status %d", status)
	}
	if makes != 2 {
		t.Fatalf("declaration inspection called Make %d times, want 2", makes)
	}
}

// Command execution supplies authored input and the caller's context to fresh
// invocations, with each job bounded by the selected timeout.
func TestPlaygroundExecution(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		type contextKey struct{}
		ctx := context.WithValue(t.Context(), contextKey{}, "caller")
		ran := false
		descriptor := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
			var assigned string
			return program.RunnerFunc(func(ctx context.Context) error {
				ran = true
				if assigned != "eu-west" || ctx.Value(contextKey{}) != "caller" {
					t.Errorf("invocation input = %q, caller = %v", assigned, ctx.Value(contextKey{}))
				}
				if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) != 23*time.Second {
					t.Errorf("job deadline = %v, %v", deadline, ok)
				}
				return nil
			}), program.InputList{{Field: program.Field{Name: "region"}, Value: program.SetterFunc(func(value string) error {
				assigned = value
				return nil
			})}}, nil
		}}
		var registry program.Registry
		if err := registry.Register("example.test/catalogue", "Database", descriptor); err != nil {
			t.Fatal(err)
		}
		artifact := solution.Artifact{Version: "0.1", Definition: solution.Definition{
			Nodes:    []solution.Node{{Name: "database", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Database"}}},
			Bindings: []solution.Binding{{Node: "database", Input: "region", Value: solution.Literal("eu-west")}},
		}}
		var input, diagnostics bytes.Buffer
		if err := sef.EncodeGob(&input, artifact); err != nil {
			t.Fatal(err)
		}
		command := host.Command{Context: ctx, Registry: &registry, Stdin: &input, Stderr: &diagnostics}
		if status := command.Run([]string{"playground", "-job-timeout", "23s"}); status != 0 {
			t.Fatalf("playground status %d: %s", status, &diagnostics)
		}
		if !ran {
			t.Fatal("job did not run")
		}
	})
}

// Canceling the caller's context reaches an application in the generated host's
// invocation path and completes the command with operation-failure status.
func TestPlaygroundCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	descriptor := application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
		return program.RunnerFunc(func(ctx context.Context) error {
			cancel()
			<-ctx.Done()
			return ctx.Err()
		}), nil
	}}
	var registry program.Registry
	if err := registry.Register("example.test/catalogue", "Backend", descriptor); err != nil {
		t.Fatal(err)
	}
	artifact := solution.Artifact{Version: "0.1", Definition: solution.Definition{
		Nodes: []solution.Node{{Name: "backend", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Backend"}}},
	}}
	var input, diagnostics bytes.Buffer
	if err := sef.EncodeGob(&input, artifact); err != nil {
		t.Fatal(err)
	}
	command := host.Command{Context: ctx, Registry: &registry, Stdin: &input, Stderr: &diagnostics}
	if status := command.Run([]string{"playground"}); status != 1 {
		t.Fatalf("canceled playground status %d: %s", status, &diagnostics)
	}
}

// Runner errors can contain input values, so command diagnostics must use the
// playground's value-free error rendering.
func TestPlaygroundDiagnostics(t *testing.T) {
	const secret = "private-value-not-in-diagnostics"
	failing := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		var value string
		return program.RunnerFunc(func(context.Context) error { return errors.New(value) }),
			program.InputList{{Field: program.Field{Name: "credential", Sensitive: true}, Value: program.SetterFunc(func(text string) error {
				value = text
				return nil
			})}}, nil
	}}
	var registry program.Registry
	if err := registry.Register("example.test/catalogue", "Failing", failing); err != nil {
		t.Fatal(err)
	}
	artifact := solution.Artifact{Version: "0.1", Definition: solution.Definition{
		Nodes:    []solution.Node{{Name: "backend", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Failing"}}},
		Bindings: []solution.Binding{{Node: "backend", Input: "credential", Value: solution.Literal(secret)}},
	}}
	var input, diagnostics bytes.Buffer
	if err := sef.EncodeGob(&input, artifact); err != nil {
		t.Fatal(err)
	}
	command := host.Command{Registry: &registry, Stdin: &input, Stderr: &diagnostics}
	if status := command.Run([]string{"playground"}); status != 1 {
		t.Fatalf("failed playground status %d: %s", status, &diagnostics)
	}
	if bytes.Contains(diagnostics.Bytes(), []byte(secret)) {
		t.Fatal("playground diagnostics disclosed a sensitive value")
	}
}

func TestPlaygroundUsage(t *testing.T) {
	var registry program.Registry
	for _, args := range [][]string{{"playground", "-job-timeout", "0"}, {"playground", "-job-timeout", "-1s"}, {"playground", "-inputs"}} {
		command := host.Command{Registry: &registry}
		if status := command.Run(args); status != 2 {
			t.Errorf("%v status %d, want usage failure", args, status)
		}
	}
}
