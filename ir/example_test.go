package ir_test

import (
	"fmt"

	"github.com/modern-engineering/prototype/ir"
)

// Structural inspection needs no imports from the referenced modules, including
// definitions whose catalogue type the inspecting tool does not understand.
func Example() {
	model := ir.Model{
		Modules: []ir.Module{{Path: "example.org/catalogue", Version: "v1.2.0"}},
		Nodes: []ir.Node{
			{Name: "database", Definition: ir.Symbol{Package: "example.org/catalogue/jobs", Name: "Database"}},
			{Name: "backend", Definition: ir.Symbol{Package: "example.org/catalogue/apps", Name: "Backend"}},
			{Name: "placement", Definition: ir.Symbol{Package: "example.org/catalogue/policy", Name: "Europe"}},
		},
		Bindings: []ir.Binding{
			{Node: "database", Input: "region", Value: ir.Literal("eu-west")},
			{Node: "backend", Input: "database", Value: ir.Reference{Node: "database", Output: "address"}},
		},
	}

	for _, node := range model.Nodes {
		fmt.Printf("%s: %s.%s\n", node.Name, node.Definition.Package, node.Definition.Name)
	}
	for _, binding := range model.Bindings {
		switch value := binding.Value.(type) {
		case ir.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node, binding.Input, value)
		case ir.Reference:
			fmt.Printf("%s.%s <- %s.%s\n", binding.Node, binding.Input, value.Node, value.Output)
		}
	}

	// Output:
	// database: example.org/catalogue/jobs.Database
	// backend: example.org/catalogue/apps.Backend
	// placement: example.org/catalogue/policy.Europe
	// database.region = "eu-west"
	// backend.database <- database.address
}
