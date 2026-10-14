package main

import (
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
)

func Example() {
	main()
	// Output:
	// backend depends on database (job.Descriptor)
	// backend.database <- database.address
	// database.address is used by backend.database
}

// The structural relationship can be resolved, but knowledge of the application
// catalogue rejects it before any invocation is constructed.
func TestApplicationCannotProduceConfiguration(t *testing.T) {
	var registry program.Registry
	descriptor := application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
		panic("verification called Make")
	}}
	if err := registry.Register("local", "Backend", descriptor); err != nil {
		t.Fatal(err)
	}
	source := solution.Definition{
		Nodes:    []solution.Node{{Name: "backend", Definition: solution.Symbol{Package: "local", Name: "Backend"}}},
		Bindings: []solution.Binding{{Node: "backend", Input: "database", Value: solution.Reference{Node: "backend", Output: "address"}}},
	}
	resolved, err := link.Resolve(source, &registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := verify(resolved); err == nil {
		t.Fatal("verification accepted an application as an output producer")
	}
}
