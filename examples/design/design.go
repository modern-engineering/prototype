// Package design defines a solution without importing its catalogue.
// Your authoring tools can produce the same structural data and serialize it
// before a loading tool has the catalogue's executable code available.
package design

import "github.com/modern-engineering/prototype/solution"

// CataloguePath is the package owning the definitions used by Definition.
const CataloguePath = "github.com/modern-engineering/prototype/examples/catalogue"

// Definition connects a database job to an application.
// References name the catalogue's declared fields; instantiation checks them
// against the actual fields returned by each descriptor's Make function.
var Definition = solution.Definition{
	Nodes: []solution.Node{
		{Name: "database", Definition: solution.Symbol{Package: CataloguePath, Name: "Database"}},
		{Name: "backend", Definition: solution.Symbol{Package: CataloguePath, Name: "Backend"}},
	},
	Bindings: []solution.Binding{
		{Node: "database", Input: "region", Value: solution.Literal("eu-west")},
		{Node: "backend", Input: "database", Value: solution.Reference{Node: "database", Output: "address"}},
	},
}
