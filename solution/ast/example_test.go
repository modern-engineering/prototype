package ast_test

import (
	"fmt"

	"github.com/modern-engineering/prototype/solution/ast"
)

// Structural inspection needs no imports from the referenced modules, including
// definitions whose catalogue type the inspecting tool does not understand.
func Example() {
	model := ast.Model{
		Modules: []ast.Module{{Path: "example.org/catalogue", Version: "v1.2.0"}},
		Nodes: []ast.Node{
			{Name: "database", Definition: ast.Symbol{Package: "example.org/catalogue/jobs", Name: "Database"}},
			{Name: "backend", Definition: ast.Symbol{Package: "example.org/catalogue/apps", Name: "Backend"}},
			{Name: "placement", Definition: ast.Symbol{Package: "example.org/catalogue/policy", Name: "Europe"}},
		},
		Bindings: []ast.Binding{
			{Node: "database", Input: "region", Value: ast.Literal("eu-west")},
			{Node: "backend", Input: "database", Value: ast.Reference{Node: "database", Output: "address"}},
		},
	}

	for _, node := range model.Nodes {
		fmt.Printf("%s: %s.%s\n", node.Name, node.Definition.Package, node.Definition.Name)
	}
	for _, binding := range model.Bindings {
		switch value := binding.Value.(type) {
		case ast.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node, binding.Input, value)
		case ast.Reference:
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
