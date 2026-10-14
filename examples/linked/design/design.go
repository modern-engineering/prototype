// Package design supplies structural intent without importing its catalogue.
package design

import "github.com/modern-engineering/prototype/solution/ast"

const cataloguePath = "github.com/modern-engineering/prototype/examples/linked/catalogue"

// Model connects a database job to an application and includes a placement
// policy whose catalogue need not be understood by a program-aware consumer.
var Model = ast.Model{
	Nodes: []ast.Node{
		{Name: "database", Definition: ast.Symbol{Package: cataloguePath, Name: "Database"}},
		{Name: "backend", Definition: ast.Symbol{Package: cataloguePath, Name: "Backend"}},
		{Name: "placement", Definition: ast.Symbol{Package: cataloguePath, Name: "Europe"}},
	},
	Bindings: []ast.Binding{
		{Node: "database", Input: "region", Value: ast.Literal("eu-west")},
		{Node: "backend", Input: "database", Value: ast.Reference{Node: "database", Output: "address"}},
	},
}
