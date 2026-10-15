package host_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

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
