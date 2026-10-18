package solution_test

import (
	"fmt"

	"github.com/modern-engineering/prototype/solution"
)

// Structural inspection needs no imports from the referenced modules, including
// definitions whose catalogue type the inspecting tool does not understand.
func Example() {
	model := solution.Definition{
		Nodes: []solution.Node{
			{Name: "database", Definition: solution.Symbol{Package: "example.org/catalogue/jobs", Name: "Database"}},
			{Name: "backend", Definition: solution.Symbol{Package: "example.org/catalogue/apps", Name: "Backend"}},
			{Name: "placement", Definition: solution.Symbol{Package: "example.org/catalogue/policy", Name: "Europe"}},
		},
		Bindings: []solution.Binding{
			{Node: "database", Input: "region", Value: solution.Literal("eu-west")},
			{Node: "backend", Input: "database", Value: solution.Reference{Node: "database", Output: "address"}},
		},
	}

	for _, node := range model.Nodes {
		fmt.Printf("%s: %s.%s\n", node.Name, node.Definition.Package, node.Definition.Name)
	}
	for _, binding := range model.Bindings {
		switch value := binding.Value.(type) {
		case solution.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node, binding.Input, value)
		case solution.Reference:
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
