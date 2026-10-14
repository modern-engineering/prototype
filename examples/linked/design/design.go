// Package design supplies structural intent without importing its catalogue.
package design

import "github.com/modern-engineering/prototype/solution"

const cataloguePath = "github.com/modern-engineering/prototype/examples/linked/catalogue"

// Model connects a database job to an application and includes a placement
// policy whose catalogue need not be understood by a program-aware consumer.
var Model = solution.Model{
	Nodes: []solution.Node{
		{Name: "database", Definition: solution.Symbol{Package: cataloguePath, Name: "Database"}},
		{Name: "backend", Definition: solution.Symbol{Package: cataloguePath, Name: "Backend"}},
		{Name: "placement", Definition: solution.Symbol{Package: cataloguePath, Name: "Europe"}},
	},
	Bindings: []solution.Binding{
		{Node: "database", Input: "region", Value: solution.Literal("eu-west")},
		{Node: "backend", Input: "database", Value: solution.Reference{Node: "database", Output: "address"}},
	},
}
