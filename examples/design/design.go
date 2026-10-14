// Package design supplies structural intent without importing its catalogue.
package design

import "github.com/modern-engineering/prototype/solution"

// CataloguePath is the package owning the definitions used by Definition.
const CataloguePath = "github.com/modern-engineering/prototype/examples/catalogue"

// Definition connects a database job to an application and includes a placement
// policy whose catalogue need not be understood by a program-aware consumer.
var Definition = solution.Definition{
	Nodes: []solution.Node{
		{Name: "database", Definition: solution.Symbol{Package: CataloguePath, Name: "Database"}},
		{Name: "backend", Definition: solution.Symbol{Package: CataloguePath, Name: "Backend"}},
		{Name: "placement", Definition: solution.Symbol{Package: CataloguePath, Name: "Europe"}},
	},
	Bindings: []solution.Binding{
		{Node: "database", Input: "region", Value: solution.Literal("eu-west")},
		{Node: "backend", Input: "database", Value: solution.Reference{Node: "database", Output: "address"}},
	},
}
